# from server.pkg.api.server.PythonScrapper.python.new.keyinvoice_base import *
# from typing import List
# import re
# import unicodedata

# @dataclass
# class Product:
#     """Data class for Product information"""
#     IdProduct: str = ""
#     Name: str = ""
#     ShortName: str = ""
#     TaxValue: str = ""
#     IsService: bool = False
#     HandlingType: str = ""
#     Price: str = ""
#     TaxIncluded: str = ""
#     ShortDescription: str = ""
#     LongDescription: str = ""
#     Family: str = ""
#     BrandName: str = ""
#     BrandModels: str = ""
#     DirectDiscount: str = ""

#     def to_dict(self) -> Dict[str, Any]:
#         return asdict(self)
    
#     def is_valid(self) -> bool:
#         """Check if product has meaningful data"""
#         return bool(self.IdProduct.strip() or self.Name.strip())

# class ProductScraper(KeyInvoiceBotBase):
#     """Specialized bot for scraping products from KeyInvoice"""
    
#     def __init__(self, credentials: LoginCredentials, headless: bool = True, max_products: int = None):
#         super().__init__(credentials, headless)
#         self.max_products = max_products # or 50  # Reasonable default
        
#     def navigate_to_products_section(self) -> None:
#         """Navigate to the products/articles section"""
#         try:
#             self.log_step("Navigating to Tabelas section")
#             self.page.hover('a[title="Tabelas"]', timeout=5000)
#             self.page.click('a[title="Tabelas"]', timeout=5000)
#             self.page.wait_for_load_state('domcontentloaded', timeout=10000)
            
#             self.log_step("Opening Artigos page")
#             self.page.wait_for_selector('a:has-text("Artigos"):visible', timeout=10000)
#             self.page.click('a[href*="ListaArtigos.php"]', timeout=5000)
#             self.wait_for_stable_page(15000)
            
#         except Exception as e:
#             raise NavigationError(f"Failed to navigate to products section: {e}")
    
#     def get_product_links(self) -> List[str]:
#         """Extract product links from the table"""
#         try:
#             self.log_step("Waiting for product table")
#             self.page.wait_for_selector("table.tbl_artz tbody tr", timeout=15000)
            
#             self.log_step("Extracting product links")
#             a_elements = self.page.locator("table.tbl_artz tbody tr td a")
#             count = min(a_elements.count(), self.max_products)
            
#             links = []
#             for i in range(count):
#                 href = a_elements.nth(i).get_attribute("href", timeout=2000)
#                 if href:
#                     links.append(href)
            
#             self.log_step(f"Extracted {len(links)} product links")
#             return links
            
#         except Exception as e:
#             raise KeyInvoiceError(f"Failed to extract product links: {e}")
    
#     def scrape_product_details(self, link: str) -> Optional[Product]:
#         """Scrape details from a single product page"""
#         try:
#             full_link = f"{Config.DEMO_URL}{link}"
#             self.page.goto(full_link, timeout=Config.NAVIGATION_TIMEOUT, wait_until='domcontentloaded')
            
#             # Extract product information using safe methods
#             product = Product(
#                 IdProduct=clean_text(self.safe_get_input_value("#CodigoArtigo")),
#                 Name=clean_text(self.safe_get_input_value("#Designacao")),
#                 ShortName=clean_text(self.safe_get_input_value("#Abreviatura")),
#                 TaxValue=self._extract_tax_value(),
#                 IsService=self._is_service_checked(),
#                 HandlingType=clean_text(self.safe_get_text("div.controls #CodigoTratamento ~ p")),
#                 Price=self._safe_parse_number(self.safe_get_input_value("#COL_PRE_001")),
#                 Family=clean_text(self.safe_get_text("#select2-CodigoFam_1-container")),
#                 BrandName=clean_text(self.safe_get_text("#select2-CodigoMarca-container")),
#                 BrandModels=clean_text(self.safe_get_text("#select2-CodigoTipoArtigo-container")),
#                 DirectDiscount=self._safe_parse_number(self.safe_get_input_value("#DescontoDirecto")),
#                 ShortDescription=clean_text(self.safe_get_text("#DescricaoCurta")),
#                 LongDescription=clean_text(self.safe_get_text("#DescricaoLonga")),
#             )
            
#             return product if product.is_valid() else None
            
#         except Exception as e:
#             self.logger.error(f"Failed to scrape product from {link}: {e}")
#             return None
    
#     def _extract_tax_value(self) -> str:
#         """Extract and clean tax value"""
#         tax_text = self.safe_get_text("#select2-CodigoTaxaIVA-container")
#         if "%" in tax_text:
#             return self._safe_parse_number(tax_text.split("%")[0])
#         return self._safe_parse_number(tax_text)
    
#     def _is_service_checked(self) -> bool:
#         """Check if the service checkbox is checked"""
#         try:
#             element = self.page.wait_for_selector("#Servicos", timeout=3000, state='attached')
#             if element:
#                 # Try multiple methods to determine if checked
#                 checked_attr = element.get_attribute('checked')
#                 if checked_attr is not None:
#                     return True
#                 # Use JavaScript evaluation as fallback
#                 return element.evaluate("el => el.checked")
#             return False
#         except Exception:
#             return False
    
#     def _safe_parse_number(self, value: str) -> str:
#         """Safely parse and format numeric values"""
#         try:
#             if not value:
#                 return "0"
#             # Extract numeric part using regex
#             numeric_match = re.search(r'[\d,]+\.?\d*', value)
#             if numeric_match:
#                 result = numeric_match.group().replace(',', '')
#                 return result if result else "0"
#             return "0"
#         except Exception:
#             return "0"
    
#     def execute_main_task(self) -> Dict[str, Any]:
#         """Main task: scrape all products"""
#         self.navigate_to_products_section()
        
#         links = self.get_product_links()
#         if not links:
#             raise KeyInvoiceError("No product links found")
        
#         products = []
#         failed_count = 0
#         max_failures = min(len(links) // 2, 10)
        
#         self.log_step(f"Starting to scrape {len(links)} products")
        
#         for i, link in enumerate(links):
#             if failed_count > max_failures:
#                 self.log_step(f"Max failures ({max_failures}) reached, stopping")
#                 break
            
#             product = self.scrape_product_details(link)
#             if product:
#                 products.append(product.to_dict())
#                 self.log_step(f"Product {i+1}/{len(links)} scraped: {product.IdProduct} - {product.Name}")
#             else:
#                 failed_count += 1
#                 self.log_step(f"Product {i+1}/{len(links)} failed or invalid")
        
#         self.log_step(f"Scraping completed: {len(products)} products, {failed_count} failures")
        
#         return {
#             "products": products,
#             "debug": f"Successfully scraped {len(products)} products"
#         }

# def run_product_scraper(email: str, senha: str) -> str:
#     """Run the product scraper and return JSON result"""
#     try:
#         credentials = LoginCredentials(email=email, password=senha)
        
#         def scraper_task():
#             scraper = ProductScraper(credentials, headless=True)
#             return scraper.run()
        
#         response = run_with_timeout(scraper_task, Config.MAX_SCRIPT_TIMEOUT)
#         return response.to_json()
        
#     except Exception as e:
#         error_response = ScriptResponse(success=False, error=str(e))
#         return error_response.to_json()

# def main_product_scraper():
#     """Entry point for product scraper script"""
#     try:
#         if len(sys.argv) != 3:
#             raise ValueError("Usage: script.py <email> <senha>")
        
#         email, senha = sys.argv[1], sys.argv[2]
#         result = run_product_scraper(email, senha)
#         print(result)
        
#     except Exception as e:
#         error_response = ScriptResponse(success=False, error=str(e))
#         print(error_response.to_json())
#         sys.exit(1)
