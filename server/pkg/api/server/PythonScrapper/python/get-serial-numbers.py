import sys
import json
import threading
from playwright.sync_api import sync_playwright
import os
import json
import sys
from urllib.parse import urljoin
from datetime import datetime
import time
from typing import Dict, List, Optional, Any
from dataclasses import dataclass
from pathlib import Path
import threading
import base64
import tempfile

@dataclass
class LoginCredentials:
    """Data class for login credentials"""
    email: str
    password: str

@dataclass
class Product:
    """Data class for Product"""
    name: str = ""

@dataclass
class Products:
    """Data class for Products"""
    products: List[Product] = None
    
    def __post_init__(self):
        if self.products is None:
            self.products = []

@dataclass
class SerialNumber:
    name: str = ""

@dataclass
class ProductsResult:
    """Result of bill creation"""
    success: bool
    product_name: str = ""
    serial_number: List[SerialNumber]

class KeyInvoiceBillBot:
    """Enhanced bot class for KeyInvoice bill/invoice automation"""
    
    def __init__(self, credentials: LoginCredentials, headless: bool = True):
        self.credentials = credentials
        self.headless = headless
        self.base_url = "https://www.keyinvoice.com/"
        self.demo_url = "https://login.keyinvoice.com/"
        self.browser = None
        self.context = None
        self.page = None
        self.step_counter = 0
    
    def log_step(self, message: str) -> None:
        """Log a step with timestamp"""
        timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        print(f"[{timestamp}] STEP {self.step_counter}: {message}", file=sys.stderr)
        self.step_counter += 1
    
    def launch_browser(self) -> None:
        """Launch browser and create context"""
        self.log_step("Launching browser")
        self.playwright = sync_playwright().start()
        self.browser = self.playwright.chromium.launch(headless=self.headless)
        self.context = self.browser.new_context(accept_downloads=True)
        self.page = self.context.new_page()
        self.log_step("Browser launched successfully")
    
    def navigate_to_homepage(self) -> None:
        """Navigate to KeyInvoice homepage"""
        self.log_step(f"Navigating to {self.base_url}")
        self.page.goto(self.base_url)
        self.page.wait_for_load_state('networkidle')
        self.log_step("Homepage loaded successfully")
    
    def login(self) -> None:
        """Login to KeyInvoice"""
        self.log_step("Clicking login button")
        self.page.press('text=Login', 'Enter')
        self.page.wait_for_load_state('networkidle')
        self.log_step("Login page loaded")

        self.log_step("Filling login form")
        self.page.fill('input[name="xfldEmail"]', self.credentials.email)
        self.page.fill('input[name="xfldSenha"]', self.credentials.password)
        self.log_step("Credentials entered")

        self.log_step("Submitting login form")
        self.page.press('input[name="ENTRAR"]', 'Enter')
        self.page.wait_for_load_state('networkidle')
        self.log_step("Login successful")
    
    def enter_demo_mode(self) -> None:
        """Enter demo mode"""
        self.log_step("Entering demo mode")
        self.page.click('text=DEMO')
        self.page.click('text=Utilizador Demo')
        self.page.wait_for_load_state('networkidle')
        self.log_step("Demo mode activated")
        time.sleep(2)
    
    def navigate_to_serial_numbers(self) -> None:
        """Navigate to seial numbers section"""
        self.log_step("Navigating to Stocks section")
        self.page.hover('a[title="Stocks"]')
        self.page.click('a[title="Stocks"]')
        self.page.wait_for_load_state('domcontentloaded')
        self.log_step("Stocks section loaded")

        self.log_step("Opening Pesquisa de Núm.Série/IMEI")
        self.page.wait_for_selector('a:has-text("Pesquisa de Núm.Série/IMEI"):visible')
        self.page.click('a[href*="ListaNumerosSerie.php"]')
        self.page.wait_for_load_state('networkidle')
        self.log_step("Pesquisa de Núm.Série/IMEI page loaded")
    
    def close(self) -> None:
        """Close browser and cleanup"""
        if self.browser:
            self.browser.close()
            self.browser = None
        if self.context:
            self.context.close()
            self.context = None
        if hasattr(self, 'playwright') and self.playwright:
            self.playwright.stop()
            self.playwright = None
        self.log_step("Browser closed")

    def collect_serial_numbers(self, product: Product) -> ProductsResult:
        """Collect serial numbers"""
        try:
            
            self.log_step(f"Starting to collect  serial numbers for product {product.name}")
         
            
            # Initialize browser session
            self.launch_browser()
            self.navigate_to_homepage()
            self.login()
            self.enter_demo_mode()
            
            # Navigate to invoices and create new one
            self.navigate_to_serial_numbers()

            self.log_step(f"Searching for product: {product.name}")
            self.page.fill('input[name="AR"]', product.name)

            stock_checkbox = self.page.locator('input[name="XVS"][type="checkbox"]')
            if not stock_checkbox.is_checked():
                stock_checkbox.check()

            self.page.click('input[name="ATUALIZAR"]')
            self.page.wait_for_load_state('networkidle')
            self.log_step("Search submitted")

            serial_numbers = []
            table = self.page.locator('table.table-docs-venc')
            
            # Process each serial number section
            section_rows = table.locator('tbody tr:has(td[colspan="6"])')

            for i in range(section_rows.count()):
                section_row = section_rows.nth(i)
                section_text = section_row.text_content()
                
                # Extract serial number from section header
                if "Nº série:" in section_text:
                    serial_number = section_text.split("Nº série:")[1].strip()
                    
                    # Get the detail row following this section header
                    detail_row = section_row.locator('xpath=following-sibling::tr[1]')
                    
                    if detail_row.count() > 0:                      
                        serial_data = {
                            'serial_number': serial_number,
                            'product_name': product.name
                        }
                        
                        serial_numbers.append(serial_data)
                                
            self.log_step(f"Collecting serial numbers process completed successfully for product {product.name}")
            
            result = ProductsResult(
                success=True,
                product_name= product.name,
                serial_number= serial_numbers,
            )
            

            return result
            
        except Exception as e:
            error_msg = f"Error collecting serial numbers for product {product.name}: {str(e)}"
            self.log_step(error_msg)
            return ProductsResult(success=False, product_name=product.name, error_message=error_msg)
        
        finally:

            if not self.headless:
                time.sleep(5)
            self.close()

# def parse_products_from_json(products: str) -> List[Products]:
#     """Parse orders from JSON string"""
#     try:
#         products_data = json.loads(products)
#         products_array = []
        
#         for product_data in products_data:
#             products = []
#             for product_data in product_data.get('Products', []):
#                 product = Product(
#                     name=product_data.get('product_name', ''),
#                 )
#                 products.append(product)
            
#             product_list = Products(
#                 products=products
#             )
#             products_array.append(product_list)
        
#         return products_array
#     except Exception as e:
#         raise ValueError(f"Failed to parse orders JSON: {str(e)}")
    

def parse_products_from_json(products: Any) -> List[Product]:
    """Parse products from JSON structure"""
    try:
        parsed_products = []
        for product in products:
            name = product.get("product_name", "")
            parsed_products.append(Product(name=name))
        return parsed_products
    except Exception as e:
        raise ValueError(f"Failed to parse products JSON: {str(e)}")

def get_serial_numbers(email: str, senha: str, products: str) -> Dict[str, Any]:
    """Extract serial numbers for a specific product from the serial number search page"""
    try:
        product_list = parse_products_from_json(products)
        print(f"Processing {len(product_list)} products...", file=sys.stderr)

        credentials = LoginCredentials(
            email=email,
            password=senha
        )

        for i, product in enumerate(product_list, 1):
            print(f"Processing product {i}/{len(product)}...", file=sys.stderr)

            bot = KeyInvoiceBillBot(credentials, headless=True)
            result = bot.collect_serial_numbers(product_list)






        self.log_step(f"Found {len(serial_numbers)} serial numbers for product {product_name}")
        return serial_numbers
        
    except Exception as e:
        self.log_step(f"Error extracting serial numbers: {str(e)}")
        return []
    

def get_serial_numbers(email: str, senha: str, products: Any) -> Dict[str, Any]:
    """Extract serial numbers for a specific product from the serial number search page"""
    try:
        product_list = parse_products_from_json(products)
        print(f"Processing {len(product_list)} products...", file=sys.stderr)

        credentials = LoginCredentials(
            email=email,
            password=senha
        )

        serial_numbers = []

       for i, product in enumerate(product_list, 1):
            print(f"Processing product {i}/{len(product)}...", file=sys.stderr)

            bot = KeyInvoiceBillBot(credentials, headless=True)
            result = bot.collect_serial_numbers(product_list)

            serial_entry = {
                "product_name": product.name,
                "serial_number": result.serial_number
            }
            serial_numbers.append(serial_entry)

            return {
                "success": True,
                "serial_numbers": serial_numbers,
                "count": len(serial_numbers)
            }
    except Exception as e:
        return {
            "success": False,
            "serial_numbers": [],
            "count": 0,
            "error": str(e)
        }


def run_get_serial_numbers(email, senha, products):
    return get_serial_numbers(email, senha, products)

def main():
    try:
        # Accept JSON either from argument or stdin
        if len(sys.argv) > 1 and sys.argv[1].strip().startswith("{"):
            raw_data = sys.argv[1]
        else:
            raw_data = sys.stdin.read()

        if not raw_data.strip():
            raise ValueError("No input JSON provided")

        data = json.loads(raw_data)

        # Expected GSNPythonCommand structure
        email = data.get("email")
        senha = data.get("senha")
        products = data.get("Products", [])

        if not email or not senha:
            raise ValueError("Missing email or senha")
        
        result_container = {}

        def worker():
            result_container['result'] = run_get_serial_numbers(email, senha, products)

        thread = threading.Thread(target=worker)
        thread.start()
        thread.join()

        result = result_container.get('result')
        if result is None:
            raise RuntimeError("Bill processing failed: no result returned")

        print(json.dumps(result, indent=2, ensure_ascii=False))

    except Exception as e:
        error_result = {
            "success": False,
            "serial_numbers": [],
            "error": str(e)
        }
        print(json.dumps(error_result, ensure_ascii=False))
        sys.exit(1)

if __name__ == "__main__":
    main()
