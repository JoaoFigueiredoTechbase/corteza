# from server.pkg.api.server.PythonScrapper.python.new.keyinvoice_base import *
# from typing import List
# import re
# import unicodedata

# @dataclass
# class ProductBill:
#     """Data class for product bill items"""
#     details: str = ""
#     discount: str = "0"
#     id_product: str = ""
#     price: str = "0"
#     quantity: str = "1"
#     tax: str = "0"
    
#     def get_price_float(self) -> float:
#         return safe_float(self.price)
    
#     def get_quantity_float(self) -> float:
#         return safe_float(self.quantity, 1.0)
    
#     def get_discount_float(self) -> float:
#         return safe_float(self.discount)

# @dataclass
# class Order:
#     """Data class for order"""
#     address: str = ""
#     avenca_record_id: str = ""
#     doc_date: str = ""
#     id_client: str = ""
#     products: List[ProductBill] = None
    
#     def __post_init__(self):
#         if self.products is None:
#             self.products = []

# @dataclass
# class BillResult:
#     """Result of bill creation"""
#     success: bool
#     bill_id: str = ""
#     client_id: str = ""
#     avenca_record_id: str = ""
#     total_amount: float = 0.0
#     error_message: str = ""
#     pdf_filename: str = ""
#     pdf_content: bytes = b""

# class BillCreator(KeyInvoiceBotBase):
#     """Specialized bot for creating bills/invoices"""
    
#     def __init__(self, credentials: LoginCredentials, orders: List[Order], headless: bool = True):
#         super().__init__(credentials, headless)
#         self.orders = orders
#         self.results = []
    
#     def navigate_to_invoices(self) -> None:
#         """Navigate to invoices section"""
#         try:
#             self.log_step("Navigating to Vendas section")
#             self.page.hover('a[title="Vendas"]')
#             self.page.click('a[title="Vendas"]')
#             self.page.wait_for_load_state('domcontentloaded')
            
#             self.log_step("Opening Faturas")
#             self.page.wait_for_selector('a:has-text("Faturas"):visible')
#             self.page.click('a[href*="ListaFacturas.php"]')
#             self.wait_for_stable_page()
            
#         except Exception as e:
#             raise NavigationError(f"Failed to navigate to invoices: {e}")
    
#     def create_new_invoice(self) -> None:
#         """Start creating a new invoice"""
#         try:
#             self.log_step("Creating new invoice")
#             self.page.press('text=Nova Fatura', 'Enter')
#             self.wait_for_stable_page()
#         except Exception as e:
#             raise KeyInvoiceError(f"Failed to create new invoice: {e}")
    
#     def select_client_by_id(self, client_id: str) -> bool:
#         """Select a client by ID"""
#         try:
#             self.log_step(f"Selecting client: {client_id}")
            
#             # Open client popup
#             self.page.click('a.fancybox[href*="CodigoTerceiro"]', force=True)
#             self.page.wait_for_selector('#fancybox-content', state='visible', timeout=10000)
            
#             # Filter for the client
#             filter_input = self.page.locator('td.nowrap.left input.filter_column.filter_text').first
#             filter_input.wait_for(state="visible", timeout=5000)
#             filter_input.fill(client_id)
#             filter_input.dispatch_event("input")
#             filter_input.dispatch_event("change")
            
#             # Select the client
#             row = self.page.locator(f'tbody tr:has-text("{client_id}")').first
#             row.wait_for(state="visible", timeout=5000)
#             client_link = row.locator('a[href*="pick"]').first
#             client_link.click()
            
#             # Wait for popup to close
#             self.page.wait_for_selector('#fancybox-content', state='hidden', timeout=10000)
#             self.wait_for_stable_page()
            
#             self.log_step("Client selected successfully")
#             return True
            
#         except Exception as e:
#             self.logger.error(f"Failed to select client {client_id}: {e}")
#             # Try to close popup if still open
#             try:
#                 if self.page.locator('#fancybox-content').count() > 0:
#                     self.page.keyboard.press('Escape')
#             except:
#                 pass
#             return False
    
#     def add_product_by_id(self, product_id: str) -> bool:
#         """Add a product to the invoice"""
#         try:
#             self.log_step(f"Adding product: {product_id}")
            
#             # Open product menu
#             self.page.click('#adicionar_artigo')
#             self.wait_for_stable_page()
            
#             # Search using table filter
#             filter_input = self.page.locator("table#tbl_psqart2 thead tr.filter th:nth-child(2) input")
#             filter_input.fill("")
#             filter_input.fill(product_id)
#             filter_input.press('Enter')
            
#             # Wait for table to update
#             self.page.wait_for_selector("#tbl_psqart2_processing", state="hidden", timeout=10000)
            
#             # Find and click product
#             rows = self.page.locator("table#tbl_psqart2 tbody tr[role='row']")
#             matching_row = rows.filter(has=self.page.locator(f"td:has-text('{product_id}')")).first
            
#             if matching_row.count() > 0:
#                 matching_row.locator("a.artpickr").click()
#                 self.log_step(f"Product {product_id} added successfully")
#                 return True
            
#             # Fallback: try search input method
#             search_input = self.page.locator("input[type='search'].form-control.input-sm")
#             search_input.fill(product_id)
#             search_button = self.page.locator("button.SrxArtsPesquisar")
#             search_button.click()
            
#             self.page.wait_for_selector("#tbl_psqart2_processing", state="hidden", timeout=10000)
            
#             matching_row = rows.filter(has=self.page.locator(f"td:has-text('{product_id}')")).first
#             if matching_row.count() > 0:
#                 matching_row.locator("a.artpickr").click()
#                 self.log_step(f"Product {product_id} added via search")
#                 return True
            
#             self.log_step(f"Product {product_id} not found")
#             return False
            
#         except Exception as e:
#             self.logger.error(f"Failed to add product {product_id}: {e}")
#             return False
    
#     def configure_product_line(self, product: ProductBill, line_number: int) -> bool:
#         """Configure a product line with properties"""
#         try:
#             self.log_step(f"Configuring product line {line_number}")
            
#             # Set price if specified
#             price = product.get_price_float()
#             if price > 0:
#                 self._set_line_field(f'COL_PRE_{line_number:03d}', str(price))
            
#             # Set description if specified
#             if product.details and product.details.strip():
#                 self._set_product_description(product.details, line_number)
            
#             # Set discount if specified
#             discount = product.get_discount_float()
#             if discount > 0:
#                 self._set_line_field(f'COL_DSC_{line_number:03d}', str(discount))
            
#             # Set quantity if different from default
#             quantity = product.get_quantity_float()
#             if quantity != 1.0:
#                 self._set_line_field(f'COL_QTD_{line_number:03d}', str(quantity))
            
#             return True
            
#         except Exception as e:
#             self.logger.error(f"Failed to configure line {line_number}: {e}")
#             return False
    
#     def _set_line_field(self, field_name: str, value: str) -> bool:
#         """Set a field value for a product line"""
#         try:
#             field_selector = f'input[name="{field_name}"]'
#             if self.safe_fill(field_selector, value):
#                 self.page.press('text=Gravar', 'Enter')
#                 self.wait_for_stable_page()
#                 return True
#             return False
#         except Exception as e:
#             self.logger.error(f"Failed to set field {field_name}: {e}")
#             return False
    
#     def _set_product_description(self, description: str, line_number: int) -> bool:
#         """Set description for a product line"""
#         try:
#             # Click details button using JavaScript
#             success = self.page.evaluate(f'''() => {{
#                 const buttons = Array.from(document.querySelectorAll('input[type="button"][value="Detalhes"]'));
#                 const target = buttons.find(btn => btn.getAttribute('data-lin') === "{line_number}");
#                 if (target) {{
#                     target.click();
#                     return true;
#                 }}
#                 return false;
#             }}''')
            
#             if not success:
#                 return False
            
#             # Fill description
#             desc_field = '#DocLinObsObs1'
#             self.page.wait_for_selector(desc_field, state='visible', timeout=10000)
#             enhanced_desc = description + "\n - Esta informação é referente aos itens apresentados a cima."
#             self.page.fill(desc_field, enhanced_desc)
            
#             # Save
#             self.page.click('button.btn_sv')
#             self.wait_for_stable_page()
            
#             return True
            
#         except Exception as e:
#             self.logger.error(f"Failed to set description for line {line_number}: {e}")
#             return False
    
#     def finalize_invoice(self) -> bool:
#         """Finalize the invoice"""
#         try:
#             self.log_step("Finalizing invoice")
#             self.page.press('#BOTAO1', 'Enter')
#             self.page.press('#popup_ok', 'Enter')
#             self.wait_for_stable_page()
#             return True
#         except Exception as e:
#             self.logger.error(f"Failed to finalize invoice: {e}")
#             return False
    
#     def get_invoice_details(self) -> Dict[str, Any]:
#         """Extract invoice details"""
#         details = {
#             'invoice_number': '',
#             'total_amount': 0.0,
#             'creation_date': datetime.now().isoformat()
#         }
        
#         try:
#             # Extract invoice number
#             invoice_element = self.page.locator('div.controls', has_text="Nº Fatura").locator('p.md-subhead')
#             if invoice_element.count() > 0:
#                 details['invoice_number'] = invoice_element.text_content().strip()
            
#             # Extract total amount
#             total_element = self.page.locator('input[name*="TOTAL"]')
#             if total_element.count() > 0:
#                 total_value = total_element.get_attribute('value')
#                 if total_value:
#                     details['total_amount'] = safe_float(total_value.replace(',', '.'))
                    
#         except Exception as e:
#             self.logger.warn(f"Could not extract all invoice details: {e}")
        
#         return details
    
#     def download_pdf(self, client_id: str) -> tuple[str, bytes]:
#         """Download invoice PDF"""
#         try:
#             self.log_step("Downloading invoice PDF")
            
#             with self.page.expect_download() as download_info:
#                 self.page.click('#BOTAO4')
            
#             download = download_info.value
#             temp_path = create_temp_file(b'', '.pdf')
#             download.save_as(temp_path)
            
#             # Read PDF content
#             with open(temp_path, 'rb') as pdf_file:
#                 pdf_content = pdf_file.read()
            
#             # Clean up
#             os.unlink(temp_path)
            
#             filename = f"invoice_{client_id}.pdf"
#             self.log_step(f"PDF downloaded: {filename}")
            
#             return filename, pdf_content
            
#         except Exception as e:
#             self.logger.error(f"Failed to download PDF: {e}")
#             return "", b""
    
#     def process_single_order(self, order: Order) -> BillResult:
#         """Process a single order to create a bill"""
#         try:
#             self.log_step(f"Processing order for client {order.id_client}")
            
#             # Create new invoice
#             self.create_new_invoice()
            
#             # Select client
#             if not self.select_client_by_id(order.id_client):
#                 return BillResult(
#                     success=False, 
#                     client_id=order.id_client,
#                     avenca_record_id=order.avenca_record_id,
#                     error_message=f"Failed to select client {order.id_client}"
#                 )
            
#             # Add products
#             added_products = []
#             for product in order.products:
#                 if self.add_product_by_id(product.id_product):
#                     added_products.append(product)
            
#             if not added_products:
#                 return BillResult(
#                     success=False,
#                     client_id=order.id_client,
#                     avenca_record_id=order.avenca_record_id,
#                     error_message="No products could be added"
#                 )
            
#             # Configure products
#             total_amount = 0.0
#             for line_number, product in enumerate(added_products, 1):
#                 if self.configure_product_line(product, line_number):
#                     # Calculate line total
#                     line_total = product.get_price_float() * product.get_quantity_float()
#                     line_total *= (1 - product.get_discount_float() / 100)
#                     total_amount += line_total
            
#             # Finalize invoice
#             if not self.finalize_invoice():
#                 return BillResult(
#                     success=False,
#                     client_id=order.id_client,
#                     avenca_record_id=order.avenca_record_id,
#                     error_message="Failed to finalize invoice"
#                 )
            
#             # Get details and PDF
#             invoice_details = self.get_invoice_details()
#             pdf_filename, pdf_content = self.download_pdf(order.id_client)
            
#             return BillResult(
#                 success=True,
#                 bill_id=invoice_details.get('invoice_number', f"INV-{order.id_client}-{int(time.time())}"),
#                 client_id=order.id_client,
#                 avenca_record_id=order.avenca_record_id,
#                 total_amount=invoice_details.get('total_amount', total_amount),
#                 pdf_filename=pdf_filename,
#                 pdf_content=pdf_content
#             )
            
#         except Exception as e:
#             error_msg = f"Error processing order for client {order.id_client}: {e}"
#             self.logger.error(error_msg)
#             return BillResult(
#                 success=False,
#                 client_id=order.id_client,
#                 avenca_record_id=order.avenca_record_id,
#                 error_message=error_msg
#             )
    
#     def execute_main_task(self) -> Dict[str, Any]:
#         """Main task: process all orders"""
#         self.navigate_to_invoices()
        
#         results = []
#         successful_bills = 0
#         total_revenue = 0.0
#         pdf_files = []
        
#         for i, order in enumerate(self.orders, 1):
#             self.log_step(f"Processing order {i}/{len(self.orders)}")
            
#             result = self.process_single_order(order)
            
#             if result.success:
#                 successful_bills += 1
#                 total_revenue += result.total_amount
                
#                 pdf_base64 = encode_pdf_content(result.pdf_content)
                
#                 bill_data = {
#                     'bill_id': result.bill_id,
#                     'client_id': result.client_id,
#                     'avenca_record_id': result.avenca_record_id,
#                     'total_amount': result.total_amount,
#                     'status': 'created',
#                     'products_count': len(order.products),
#                     'creation_date': datetime.now().isoformat(),
#                     'pdf_filename': result.pdf_filename,
#                     'pdf_content': pdf_base64
#                 }
                
#                 if result.pdf_content:
#                     pdf_files.append({
#                         'client_id': result.client_id,
#                         'bill_id': result.bill_id,
#                         'avenca_record_id': result.avenca_record_id,
#                         'filename': result.pdf_filename,
#                         'content': pdf_base64
#                     })
#             else:
#                 bill_data = {
#                     'bill_id': '',
#                     'client_id': result.client_id,
#                     'avenca_record_id': result.avenca_record_id,
#                     'total_amount': 0.0,
#                     'status': 'failed',
#                     'error': result.error_message,
#                     'products_count': len(order.products),
#                     'creation_date': datetime.now().isoformat(),
#                     'pdf_filename': '',
#                     'pdf_content': ''
#                 }
            
#             results.append(bill_data)
#             time.sleep(2)  # Rate limiting
        
#         return {
#             'summary': {
#                 'total_orders_processed': len(self.orders),
#                 'successful_bills': successful_bills,
#                 'failed_bills': len(self.orders) - successful_bills,
#                 'total_revenue': round(total_revenue, 2),
#                 'processing_time': f"{len(self.orders) * 2:.1f} seconds (estimated)"
#             },
#             'bills': results,
#             'pdf_files': pdf_files
#         }


# def run_bill_creator(email: str, senha: str, orders_json: str) -> str:
#     """Run the bill creator and return JSON result"""
#     try:
#         credentials = LoginCredentials(email=email, password=senha)
        
#         # Parse orders
#         orders_data = json.loads(orders_json)
#         orders = []
        
#         for order_data in orders_data:
#             products = []
#             for product_data in order_data.get('Products', []):
#                 product = ProductBill(
#                     details=product_data.get('Details', ''),
#                     discount=product_data.get('Discount', '0'),
#                     id_product=product_data.get('IdProduct', ''),
#                     price=product_data.get('Price', '0'),
#                     quantity=product_data.get('Quantity', '1'),
#                     tax=product_data.get('Tax', '0')
#                 )
#                 products.append(product)
            
#             order = Order(
#                 address=order_data.get('Address', ''),
#                 avenca_record_id=order_data.get('AvencaRecordID', ''),
#                 doc_date=order_data.get('DocDate', ''),
#                 id_client=order_data.get('IdClient', ''),
#                 products=products
#             )
#             orders.append(order)
        
#         def bill_creator_task():
#             creator = BillCreator(credentials, orders, headless=True)
#             return creator.run()
        
#         response = run_with_timeout(bill_creator_task, Config.MAX_SCRIPT_TIMEOUT)
#         return response.to_json()
        
#     except Exception as e:
#         error_response = ScriptResponse(success=False, error=str(e))
#         return error_response.to_json()

# def main_bill_creator():
#     """Entry point for bill creator script"""
#     try:
#         if len(sys.argv) < 4:
#             raise ValueError("Usage: script.py <email> <senha> <orders_json>")
        
#         email, senha, orders_json = sys.argv[1], sys.argv[2], sys.argv[3]
#         result = run_bill_creator(email, senha, orders_json)
#         print(result)
        
#     except Exception as e:
#         error_response = ScriptResponse(success=False, error=str(e))
#         print(error_response.to_json())
#         sys.exit(1)