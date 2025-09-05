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
from datetime import datetime

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
    serial_number: List[Dict[str, Any]] = None
    error_message: str = ""
    
    def __post_init__(self):
        if self.serial_number is None:
            self.serial_number = []

class KeyInvoiceBillBot:
    """Enhanced bot class for KeyInvoice bill/invoice automation"""
    
    def __init__(self, credentials: LoginCredentials, headless: bool = False):
        self.credentials = credentials
        self.headless = headless
        self.base_url = "https://www.keyinvoice.com/"
        self.demo_url = "https://login.keyinvoice.com/"
        self.browser = None
        self.context = None
        self.page = None
        self.step_counter = 0
        self.playwright = None
    
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
        """Navigate to serial numbers section"""
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
        """Collect serial numbers with detailed debug logs"""
        try:
            self.log_step(f"STEP 0: Starting to collect serial numbers for product {product.name}")
        
            # Initialize browser session
            self.launch_browser()
            self.navigate_to_homepage()
            self.login()
            self.enter_demo_mode()
            self.navigate_to_serial_numbers()

            # --- Search input ---
            self.log_step(f"STEP 1: Filling search input with: {product.name}")
            self.page.fill('input[name="AR"]', product.name)

            # --- Stock checkbox ---
            self.log_step("STEP 2: Toggling stock checkbox (label[for='XVS'])")
            checkbox = self.page.locator('input#XVS')
            try:
                checkbox.wait_for(state="attached", timeout=3000)
                if not checkbox.is_checked():
                    self.page.click('label[for="XVS"]')
                    self.log_step("Stock checkbox clicked (was unchecked)")
                else:
                    self.log_step("Stock checkbox already checked")
            except Exception as e:
                self.log_step(f"WARNING: Could not toggle stock checkbox: {e}")

            # --- Submit search ---
            self.log_step("STEP 3: Clicking Atualizar button (input#PESQUISAR)")
            try:
                self.page.click('input#PESQUISAR')
                self.page.wait_for_load_state('networkidle')
                self.log_step("Search submitted successfully")
            except Exception as e:
                self.log_step(f"ERROR: Could not click Atualizar button: {e}")
                raise

            # --- Locate table ---
            self.log_step("STEP 4: Locating results table")
            table = self.page.locator("table.table-docs-venc")
            try:
                table.wait_for(state="visible", timeout=5000)
                self.log_step("Results table found")
            except Exception as e:
                self.log_step(f"ERROR: Results table not found: {e}")
                raise

            rows = table.locator("tbody tr")
            row_count = rows.count()
            self.log_step(f"STEP 5: Found {row_count} rows in results table")

            # --- Parse rows ---
            serial_numbers = []
            i = 0
            while i < row_count:
                row = rows.nth(i)
                text = (row.text_content() or "").strip()
                self.log_step(f"Row {i} text: {text}")

                if "Nº série:" in text:
                    serial_number = text.split("Nº série:")[1].strip()
                    self.log_step(f"Found serial number: {serial_number}")

                    if i + 1 < row_count:
                        detail_row = rows.nth(i + 1)

                        def safe_text(n):
                            try:
                                return detail_row.locator(f"td:nth-child({n})").text_content().strip()
                            except Exception:
                                return ""

                        product_text = safe_text(2)
                        date_text = safe_text(3)
                        doc_text = safe_text(4)
                        doc_number = safe_text(5)
                        entity = safe_text(6)

                        self.log_step(
                            f"Detail row: product='{product_text}', date='{date_text}', "
                            f"doc='{doc_text}', doc_number='{doc_number}', entity='{entity}'"
                        )

                        try:
                            parsed_date = datetime.strptime(date_text, "%Y-%m-%d")
                            iso_date = parsed_date.strftime("%Y-%m-%dT00:00:00Z")
                        except Exception:
                            iso_date = date_text  # fallback in case format is unexpected

                        serial_data = {
                            "serial_number": serial_number,
                            "product_name": product_text,
                            "date": iso_date,
                            "document": doc_text,
                            "doc_number": doc_number,
                            "entity": entity,
                        }
                        serial_numbers.append(serial_data)

                    i += 2
                else:
                    i += 1

            self.log_step(
                f"STEP 6: Collecting serial numbers completed for {product.name} "
                f"→ {len(serial_numbers)} serials found"
            )

            return ProductsResult(
                success=True,
                product_name=product.name,
                serial_number=serial_numbers,
            )

        except Exception as e:
            error_msg = f"Error collecting serial numbers for product {product.name}: {str(e)}"
            self.log_step(error_msg)
            return ProductsResult(success=False, product_name=product.name, error_message=error_msg)

        finally:
            if not self.headless:
                time.sleep(5)
            self.close()


def parse_products_from_json(products: Any) -> List[Product]:
    """Parse products from JSON structure"""
    try:
        parsed_products = []
        for product in products:
            name = product.get("ProductName", "")
            parsed_products.append(Product(name=name))
        return parsed_products
    except Exception as e:
        raise ValueError(f"Failed to parse products JSON: {str(e)}")

def get_serial_numbers(email: str, senha: str, products: Any) -> Dict[str, Any]:
    """Extract serial numbers for products from the serial number search page"""
    try:
        product_list = parse_products_from_json(products)
        print(f"Processing {len(product_list)} products...", file=sys.stderr)

        credentials = LoginCredentials(
            email=email,
            password=senha
        )

        all_serial_numbers = []

        for i, product in enumerate(product_list, 1):
            print(f"Processing product {i}/{len(product_list)}: {product.name}", file=sys.stderr)

            bot = KeyInvoiceBillBot(credentials, headless=False)
            result = bot.collect_serial_numbers(product)

            if result.success:
                serial_entry = {
                    "product_name": product.name,
                    "serial_numbers": result.serial_number
                }
                all_serial_numbers.append(serial_entry)
            else:
                print(f"Failed to collect serial numbers for {product.name}: {result.error_message}", file=sys.stderr)

        return {
            "success": True,
            "serial_numbers": all_serial_numbers,
            "count": len(all_serial_numbers)
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
            raise RuntimeError("Serial number processing failed: no result returned")

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