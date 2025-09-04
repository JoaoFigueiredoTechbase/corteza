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
class ProductBill:
    """Data class for product bill items (matches Go structure)"""
    details: str = ""
    discount: str = "0"
    id_product: str = ""
    price: str = "0"
    quantity: str = "1"
    tax: str = "0"
    
    def get_price_float(self) -> float:
        """Get price as float"""
        try:
            return float(self.price) if self.price else 0.0
        except (ValueError, TypeError):
            return 0.0
    
    def get_quantity_float(self) -> float:
        """Get quantity as float"""
        try:
            return float(self.quantity) if self.quantity else 1.0
        except (ValueError, TypeError):
            return 1.0
    
    def get_discount_float(self) -> float:
        """Get discount as float"""
        try:
            return float(self.discount) if self.discount else 0.0
        except (ValueError, TypeError):
            return 0.0


@dataclass
class Order:
    """Data class for order (matches Go structure)"""
    address: str = ""
    doc_date: str = ""
    id_client: str = ""
    products: List[ProductBill] = None
    
    def __post_init__(self):
        if self.products is None:
            self.products = []


@dataclass
class BillResult:
    """Result of bill creation"""
    success: bool
    bill_id: str = ""
    client_id: str = ""
    total_amount: float = 0.0
    error_message: str = ""
    pdf_filename: str = ""
    pdf_content: bytes = b""


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
    
    def navigate_to_invoices(self) -> None:
        """Navigate to invoices section"""
        self.log_step("Navigating to Vendas section")
        self.page.hover('a[title="Vendas"]')
        self.page.click('a[title="Vendas"]')
        self.page.wait_for_load_state('domcontentloaded')
        self.log_step("Vendas section loaded")

        self.log_step("Opening Faturas")
        self.page.wait_for_selector('a:has-text("Faturas"):visible')
        self.page.click('a[href*="ListaFacturas.php"]')
        self.page.wait_for_load_state('networkidle')
        self.log_step("Faturas page loaded")
    
    def create_new_invoice(self) -> None:
        """Start creating a new invoice"""
        self.log_step("Creating new invoice")
        self.page.press('text=Nova Fatura', 'Enter')
        self.page.wait_for_load_state('networkidle')
        self.log_step("Nova Fatura page loaded")

    def select_client_by_id(self, client_id: str) -> bool:
        """Select a client by ID for the invoice"""
        try:
            self.log_step("Opening client popup")
            self.page.click('a.fancybox[href*="CodigoTerceiro"]', force=True)
            self.page.wait_for_selector('#fancybox-content', state='visible', timeout=10000)
            self.log_step("Client popup opened")

            # Locate and use the filter input
            filter_input_selector = 'td.nowrap.left input.filter_column.filter_text'
            try:
                filter_input = self.page.locator(filter_input_selector).first
                filter_input.wait_for(state="visible", timeout=5000)

                self.log_step(f"Searching for client ID: {client_id}")
                filter_input.fill(client_id)

                # Trigger filtering (some tables listen for input/change instead of Enter)
                filter_input.dispatch_event("input")
                filter_input.dispatch_event("change")

                # Wait for a row that contains the client_id
                row = self.page.locator(f'tbody tr:has-text("{client_id}")').first
                row.wait_for(state="visible", timeout=5000)

                self.log_step("Row with client ID found")
            except Exception as filter_error:
                self.log_step(f"Could not filter table: {filter_error}")
                return False

            # Try clicking the client link inside that row
            client_link = row.locator('a[href*="pick"]').first
            try:
                client_name = client_link.text_content(timeout=2000)
                self.log_step(f"Selecting client: {client_name}")
            except:
                pass

            client_link.wait_for(state="visible", timeout=5000)
            client_link.click()

            # Wait for popup to close (indicates success)
            self.page.wait_for_selector('#fancybox-content', state='hidden', timeout=10000)
            self.log_step("Client popup closed - selection successful")

            self.page.wait_for_load_state('networkidle')
            self.log_step("Client selection completed")
            return True

        except Exception as e:
            self.log_step(f"Error selecting client {client_id}: {str(e)}")
            # Try to close the popup if it's still open
            try:
                if self.page.locator('#fancybox-content').count() > 0:
                    self.page.keyboard.press('Escape')
            except:
                pass
            return False

    def add_product_by_id(self, product_id: str) -> bool:
        """Add a product to the invoice by its ID"""
        try:
            self.log_step(f"Adding product with ID: {product_id}")
            
            # Open the product menu
            self.log_step("Opening product menu")
            self.page.click('#adicionar_artigo')
            self.page.wait_for_load_state('networkidle')
            self.log_step("Product menu opened")
            
            # --- Attempt 1: Use table filter ---
            self.log_step(f"Searching for product ID in table filter: {product_id}")
            filter_input = self.page.locator("table#tbl_psqart2 thead tr.filter th:nth-child(2) input")
            filter_input.fill("")
            filter_input.fill(product_id)
            filter_input.press('Enter')
            self.log_step("Pressed Enter to trigger table filter search")
            
            # Wait for table to update
            self.page.wait_for_selector("#tbl_psqart2_processing", state="hidden", timeout=10000)
            
            # Check table rows
            rows = self.page.locator("table#tbl_psqart2 tbody tr[role='row']")
            self.log_step(f"Found {rows.count()} rows in product catalog")
            for i in range(rows.count()):
                row_text = rows.nth(i).text_content()
                self.log_step(f"Row {i}: {row_text}")
            
            matching_row = rows.filter(has=self.page.locator(f"td:has-text('{product_id}')")).first
            if matching_row.count() > 0:
                matching_row.wait_for(state="visible", timeout=5000)
                matching_row.locator("a.artpickr").click()
                self.log_step(f"Product {product_id} added to invoice using table filter")
                return True
            
            # --- Attempt 2: Use input + search button if table filter fails ---
            self.log_step(f"Product not found in table filter, trying search input method")
            search_input = self.page.locator("input[type='search'].form-control.input-sm")
            search_input.fill(product_id)
            
            search_button = self.page.locator("button.SrxArtsPesquisar")
            search_button.click()
            self.log_step("Clicked 'Pesquisar' button")
            
            # Wait for table to update again
            self.page.wait_for_selector("#tbl_psqart2_processing", state="hidden", timeout=10000)
            
            # Check rows again
            rows = self.page.locator("table#tbl_psqart2 tbody tr[role='row']")
            matching_row = rows.filter(has=self.page.locator(f"td:has-text('{product_id}')")).first
            if matching_row.count() > 0:
                matching_row.wait_for(state="visible", timeout=5000)
                matching_row.locator("a.artpickr").click()
                self.log_step(f"Product {product_id} added to invoice using search input")
                return True
            else:
                self.log_step(f"Product {product_id} not found in catalog after both attempts")
                return False
            
        except Exception as e:
            self.log_step(f"Error adding product {product_id}: {str(e)}")
            return False

        
    def configure_product_line(self, product: ProductBill, line_number: int) -> bool:
        """Configure a product line with all its properties"""
        try:
            product_id = product.id_product
            self.log_step(f"Configuring product line {line_number} for product {product_id}")
            
            # Select the product line first
            self.select_product_by_id(product_id)
            
            # Set price if specified and valid
            price = product.get_price_float()
            if price > 0:
                self.set_product_price_by_line(price, line_number)
            
            # Set description/details if specified
            if product.details and product.details.strip():
                self.set_product_description_by_line(product.details, line_number)
            
            # Set discount if specified
            discount = product.get_discount_float()
            if discount > 0:
                self.set_product_discount_by_line(discount, line_number)
            
            # Set quantity if different from default
            quantity = product.get_quantity_float()
            if quantity != 1.0:
                self.set_product_quantity_by_line(quantity, line_number)
            
            self.log_step(f"Product line {line_number} configured successfully")
            return True
            
        except Exception as e:
            self.log_step(f"Error configuring product line {line_number}: {str(e)}")
            return False
    
    def select_product_by_id(self, product_id: str) -> bool:
        """Select a product in the invoice by its ID"""
        try:
            self.log_step(f"Selecting product with ID: {product_id}")
            
            # Use JavaScript to find and click the exact product
            result = self.page.evaluate(f'''() => {{
                const items = Array.from(document.querySelectorAll('li.col-article'));
                const target = items.find(el => el.innerText.includes("{product_id}"));
                if (target) {{
                    target.click();
                    return true;
                }}
                return false;
            }}''')
            
            if result:
                self.log_step(f"Product {product_id} selected")
                return True
            else:
                self.log_step(f"Could not find product {product_id}")
                return False
                
        except Exception as e:
            self.log_step(f"Error selecting product {product_id}: {str(e)}")
            return False
    
    def set_product_price_by_line(self, price: float, line_number: int) -> bool:
        """Set the price for a specific product line"""
        try:
            self.log_step(f"Setting price {price} for line {line_number}")
            price_field = f'input[name="COL_PRE_{line_number:03d}"]'
            self.page.fill(price_field, str(price))
            self.page.press('text=Gravar', 'Enter')
            self.page.wait_for_load_state('networkidle')
            self.log_step(f"Price set to {price}")
            return True
        except Exception as e:
            self.log_step(f"Error setting price for line {line_number}: {str(e)}")
            return False
        
    def set_product_discount_by_line(self, discount: float, line_number: int) -> bool:
        """Set the discount for a specific product line"""
        try:
            self.log_step(f"Setting discount {discount}% for line {line_number}")
            discount_field = f'input[name="COL_DSC_{line_number:03d}"]'
            self.page.fill(discount_field, str(discount))
            self.page.press('text=Gravar', 'Enter')
            self.page.wait_for_load_state('networkidle')
            self.log_step(f"Discount set to {discount}%")
            return True
        except Exception as e:
            self.log_step(f"Error setting discount for line {line_number}: {str(e)}")
            return False
    
    def set_product_quantity_by_line(self, quantity: float, line_number: int) -> bool:
        """Set the quantity for a specific product line"""
        try:
            self.log_step(f"Setting quantity {quantity} for line {line_number}")
            quantity_field = f'input[name="COL_QTD_{line_number:03d}"]'
            self.page.fill(quantity_field, str(quantity))
            self.page.press('text=Gravar', 'Enter')
            self.page.wait_for_load_state('networkidle')
            self.log_step(f"Quantity set to {quantity}")
            return True
        except Exception as e:
            self.log_step(f"Error setting quantity for line {line_number}: {str(e)}")
            return False
    
    def set_invoice_observations(self, observations: str) -> bool:
        """Set general observations for the invoice"""
        try:
            if not observations or not observations.strip():
                return True
                
            self.log_step("Setting invoice observations")
            self.page.press('text=Observações', 'Enter')
            self.page.fill('#Observacoes', observations)
            self.page.wait_for_load_state('networkidle')
            self.log_step("Observations set")
            return True
        except Exception as e:
            self.log_step(f"Error setting observations: {str(e)}")
            return False
    
    def finalize_invoice(self) -> bool:
        """Finalize the invoice"""
        try:
            self.log_step("Finalizing invoice")
            self.page.press('#BOTAO1', 'Enter')
            self.page.press('#popup_ok', 'Enter')
            self.page.wait_for_load_state('networkidle')
            self.log_step("Invoice finalized")
            return True
        except Exception as e:
            self.log_step(f"Error finalizing invoice: {str(e)}")
            return False
    
    def get_invoice_details(self) -> Dict[str, Any]:
        """Extract invoice details after creation"""
        try:
            invoice_details = {
                'invoice_number': '',
                'total_amount': 0.0,
                'creation_date': datetime.now().isoformat()
            }
            
            # Try to extract invoice number and total
            try:
                invoice_number_element = self.page.locator('div.controls', has_text="Nº Fatura").locator('p.md-subhead')
                if invoice_number_element.count() > 0:
                    invoice_details['invoice_number'] = invoice_number_element.text_content().strip()
                    self.log_step(f"aquiiiiii, fatura id {invoice_details['invoice_number']}")
            except:
                pass

            try:
                total_element = self.page.locator('input[name*="TOTAL"]')
                if total_element.count() > 0:
                    total_value = total_element.get_attribute('value')
                    if total_value:
                        invoice_details['total_amount'] = float(total_value.replace(',', '.'))
            except:
                pass
            
            return invoice_details
        except Exception as e:
            self.log_step(f"Error extracting invoice details: {str(e)}")
            return {'invoice_number': '', 'total_amount': 0.0, 'creation_date': datetime.now().isoformat()}
    
    # def print_pdf_file(self, client_id: str) -> bool:
    #     self.log_step("Clicking Imprimir button to print invoice")

    #     with self.page.expect_download() as download_info:
    #         self.page.click('#BOTAO4')

    #     download = download_info.value

    #     # Save file with custom name (e.g., per client)
    #     file_path = f"invoice_{client_id}.pdf"
    #     download.save_as(file_path)

    #     self.log_step(f"Invoice downloaded successfully as {file_path}")

    def print_pdf_file(self, client_id: str) -> tuple[str, bytes]:
        """Download PDF file and return filename and content"""
        self.log_step("Clicking Imprimir button to download invoice PDF")

        try:
            with self.page.expect_download() as download_info:
                self.page.click('#BOTAO4')

            download = download_info.value

            # Create temporary file to store the PDF
            with tempfile.NamedTemporaryFile(delete=False, suffix='.pdf') as temp_file:
                temp_path = temp_file.name

            # Save the downloaded file to temp location
            download.save_as(temp_path)

            # Read the PDF content as bytes
            with open(temp_path, 'rb') as pdf_file:
                pdf_content = pdf_file.read()

            # Clean up temp file
            os.unlink(temp_path)

            filename = f"invoice_{client_id}.pdf"
            self.log_step(f"Invoice downloaded successfully as {filename}")
            
            return filename, pdf_content

        except Exception as e:
            self.log_step(f"Error downloading PDF: {str(e)}")
            return "", b""


    def set_product_description_by_line(self, description: str, line_number: int) -> bool:
        """Set the description for a specific product line"""
        try:
            timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            self.log_step(f"[{timestamp}] STEP: Starting to set description for line {line_number}")

            # Skip if description is empty or whitespace
            if not description or not description.strip():
                self.log_step(f"[{timestamp}] STEP: No description provided for line {line_number}, skipping")
                return True

            # Click the "Detalhes" button using JavaScript evaluation
            self.log_step(f"[{timestamp}] STEP: Attempting to click 'Detalhes' button for line {line_number} using JS")
            success = self.page.evaluate(f'''() => {{
                const buttons = Array.from(document.querySelectorAll('input[type="button"][value="Detalhes"]'));
                const target = buttons.find(btn => btn.getAttribute('data-lin') === "{line_number}");
                if (target) {{
                    target.click();
                    return true;
                }}
                return false;
            }}''')

            if not success:
                self.log_step(f"[{timestamp}] STEP: Failed to find or click 'Detalhes' button for line {line_number}")
                return False

            self.log_step(f"[{timestamp}] STEP: Clicked 'Detalhes' button for line {line_number} using JS")

            # Wait for the textarea to be visible
            desc_field = f'#DocLinObsObs1'
            self.log_step(f"[{timestamp}] STEP: Waiting for textarea {desc_field} to be visible")
            self.page.wait_for_selector(desc_field, state='visible', timeout=10000)
            self.log_step(f"[{timestamp}] STEP: Textarea {desc_field} is visible")

            # Fill the textarea
            self.log_step(f"[{timestamp}] STEP: Filling description in textarea for line {line_number} with value: {description}")
            self.page.fill(desc_field, description + " - Este po é referente aos itens apresentados a cima.")
            self.log_step(f"[{timestamp}] STEP: Filled description in textarea for line {line_number}")

            # Click the save button
            self.log_step(f"[{timestamp}] STEP: Clicking save button")
            self.page.click('button.btn_sv')
            self.log_step(f"[{timestamp}] STEP: Description saved")

            # Wait for network to be idle
            self.log_step(f"[{timestamp}] STEP: Waiting for network to be idle")
            self.page.wait_for_load_state('networkidle')
            self.log_step(f"[{timestamp}] STEP: Details set and network idle")

            return True
        except Exception as e:
            timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            self.log_step(f"[{timestamp}] STEP: Error setting description for line {line_number}: {str(e)}")
            return False

    def create_bill_from_order(self, order: Order) -> BillResult:
        """Create a bill/invoice from an order"""
        try:
            self.log_step(f"Starting bill creation for client {order.id_client}")
            
            # Initialize browser session
            self.launch_browser()
            self.navigate_to_homepage()
            self.login()
            self.enter_demo_mode()
            
            # Navigate to invoices and create new one
            self.navigate_to_invoices()
            self.create_new_invoice()
            
            # Select client
            if not self.select_client_by_id(order.id_client):
                return BillResult(success=False, error_message=f"Failed to select client {order.id_client}")
            
            # Add all products first
            added_products = []
            for product in order.products:
                if self.add_product_by_id(product.id_product):
                    added_products.append(product)
                else:
                    self.log_step(f"Failed to add product {product.id_product}, skipping...")
            
            if not added_products:
                return BillResult(success=False, error_message="No products could be added to the invoice")
            
            # Configure each added product
            total_amount = 0.0
            for line_number, product in enumerate(added_products, 1):
                if self.configure_product_line(product, line_number):
                    # Calculate line total for tracking
                    line_total = product.get_price_float() * product.get_quantity_float()
                    line_total = line_total * (1 - product.get_discount_float() / 100)
                    total_amount += line_total
            
            # Set observations based on order address or other info
            # observations = f"Order Date: {order.doc_date}"
            # if order.address:
            #     observations += f"\nAddress: {order.address}"
            
            # self.set_invoice_observations(observations)
            
            # Finalize the invoice
            if not self.finalize_invoice():
                return BillResult(success=False, error_message="Failed to finalize invoice")
            

            # Get invoice details
            invoice_details = self.get_invoice_details()

            # self.print_pdf_file(order.id_client)

            pdf_filename, pdf_content = self.print_pdf_file(order.id_client)
            
            self.log_step(f"Bill creation completed successfully for client {order.id_client}")
            
            result = BillResult(
                success=True,
                bill_id=invoice_details.get('invoice_number', f"INV-{order.id_client}-{int(time.time())}"),
                client_id=order.id_client,
                total_amount=invoice_details.get('total_amount', total_amount),
                error_message=""
            )
            
            # Add PDF data to result
            result.pdf_filename = pdf_filename
            result.pdf_content = pdf_content

            return result
            
        except Exception as e:
            error_msg = f"Error creating bill for client {order.id_client}: {str(e)}"
            self.log_step(error_msg)
            return BillResult(success=False, client_id=order.id_client, error_message=error_msg)
        
        finally:
            # Keep browser open briefly for review in non-headless mode
            if not self.headless:
                time.sleep(5)
            self.close()
    
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


def parse_orders_from_json(orders_json: str) -> List[Order]:
    """Parse orders from JSON string"""
    try:
        orders_data = json.loads(orders_json)
        orders = []
        
        for order_data in orders_data:
            products = []
            for product_data in order_data.get('Products', []):
                product = ProductBill(
                    details=product_data.get('Details', ''),
                    discount=product_data.get('Discount', '0'),
                    id_product=product_data.get('IdProduct', ''),
                    price=product_data.get('Price', '0'),
                    quantity=product_data.get('Quantity', '1'),
                    tax=product_data.get('Tax', '0')
                )
                products.append(product)
            
            order = Order(
                address=order_data.get('Address', ''),
                doc_date=order_data.get('DocDate', ''),
                id_client=order_data.get('IdClient', ''),
                products=products
            )
            orders.append(order)
        
        return orders
    except Exception as e:
        raise ValueError(f"Failed to parse orders JSON: {str(e)}")


def process_bills(orders_json: str, email: str, senha: str) -> Dict[str, Any]:
    """Main function to process orders and create bills"""
    try:
        # Parse orders from JSON
        orders = parse_orders_from_json(orders_json)
        print(f"Processing {len(orders)} orders for bill creation...", file=sys.stderr)

        # Credentials passed from Go (fallback to env if missing)
        credentials = LoginCredentials(
            email=email,
            password=senha
        )

        results = []
        successful_bills = 0
        total_revenue = 0.0
        pdf_files = []

        for i, order in enumerate(orders, 1):
            print(f"Processing order {i}/{len(orders)} for client {order.id_client}...", file=sys.stderr)

            bot = KeyInvoiceBillBot(credentials, headless=True)
            result = bot.create_bill_from_order(order)

            if result.success:
                successful_bills += 1
                total_revenue += result.total_amount
                
                # Convert PDF content to base64 for JSON transport
                pdf_base64 = ""
                if result.pdf_content:
                    pdf_base64 = base64.b64encode(result.pdf_content).decode('utf-8')
                
                bill_data = {
                    'bill_id': result.bill_id,
                    'client_id': result.client_id,
                    'total_amount': result.total_amount,
                    'status': 'created',
                    'products_count': len(order.products),
                    'creation_date': datetime.now().isoformat(),
                    'pdf_filename': result.pdf_filename,
                    'pdf_content': pdf_base64  # Base64 encoded PDF
                }
                
                # Also add to separate PDF files array for easier access
                if result.pdf_content:
                    pdf_files.append({
                        'client_id': result.client_id,
                        'bill_id': result.bill_id,
                        'filename': result.pdf_filename,
                        'content': pdf_base64
                    })
                    
            else:
                bill_data = {
                    'bill_id': '',
                    'client_id': result.client_id,
                    'total_amount': 0.0,
                    'status': 'failed',
                    'error': result.error_message,
                    'products_count': len(order.products),
                    'creation_date': datetime.now().isoformat(),
                    'pdf_filename': '',
                    'pdf_content': ''
                }

            results.append(bill_data)
            time.sleep(2)

            # if result.success:
            #     successful_bills += 1
            #     total_revenue += result.total_amount
            #     bill_data = {
            #         'bill_id': result.bill_id,
            #         'client_id': result.client_id,
            #         'total_amount': result.total_amount,
            #         'status': 'created',
            #         'products_count': len(order.products),
            #         'creation_date': datetime.now().isoformat()
            #     }
            # else:
            #     bill_data = {
            #         'bill_id': '',
            #         'client_id': result.client_id,
            #         'total_amount': 0.0,
            #         'status': 'failed',
            #         'error': result.error_message,
            #         'products_count': len(order.products),
            #         'creation_date': datetime.now().isoformat()
            #     }

            # results.append(bill_data)
            # time.sleep(2)

        return {
            'summary': {
                'total_orders_processed': len(orders),
                'successful_bills': successful_bills,
                'failed_bills': len(orders) - successful_bills,
                'total_revenue': round(total_revenue, 2),
                'processing_time': f"{len(orders) * 2:.1f} seconds (estimated)"
            },
            'bills': results

            # 'summary': {
            #     'total_orders_processed': len(orders),
            #     'successful_bills': successful_bills,
            #     'failed_bills': len(orders) - successful_bills,
            #     'total_revenue': round(total_revenue, 2),
            #     'processing_time': f"{len(orders) * 2:.1f} seconds (estimated)"
            # },
            # 'bills': results
        }

    except Exception as e:
        error_msg = f"Error processing bills: {str(e)}"
        print(error_msg, file=sys.stderr)
        raise

def run_process_bills(orders_json, email, senha):
    return process_bills(orders_json, email, senha)

def main():
    try:
        if len(sys.argv) < 4:
            raise ValueError("Usage: script.py <email> <senha> <orders_json>")
        email = sys.argv[1]
        senha = sys.argv[2]
        orders_json = sys.argv[3]

        result_container = {}

        def worker():
            result_container['result'] = run_process_bills(orders_json, email, senha)

        thread = threading.Thread(target=worker)
        thread.start()
        thread.join()

        result = result_container.get('result')
        if result is None:
            raise RuntimeError("Bill processing failed: no result returned")

        response = {
            "success": True,
            "data": result,
            "message": f"Successfully processed {result['summary']['total_orders_processed']} orders, "
                       f"created {result['summary']['successful_bills']} bills"
        }
        print(json.dumps(response, indent=2))

    except ValueError as e:
        error_response = {
            "success": False,
            "error": str(e)
        }
        print(json.dumps(error_response))

    except Exception as e:
        error_response = {
            "success": False,
            "error": f"Unexpected error: {str(e)}"
        }
        print(json.dumps(error_response))


if __name__ == "__main__":
    main()