from playwright.sync_api import sync_playwright, TimeoutError as PlaywrightTimeoutError
from urllib.parse import urljoin
from datetime import datetime
import time
from dataclasses import dataclass, asdict
import json
import re
import sys
import unicodedata
import threading
from contextlib import contextmanager
import platform

# Timeout configuration
SCRIPT_TIMEOUT = 3000  # 50 minutes total timeout
PAGE_TIMEOUT = 10000  # 10 seconds per page
NAVIGATION_TIMEOUT = 15000  # 15 seconds for navigation

class TimeoutException(Exception):
    pass

@contextmanager
def timeout_context(seconds):
    """Context manager for script-level timeout - cross-platform"""
    if platform.system() == "Windows":
        # On Windows, we'll use threading.Timer instead of signal
        timeout_occurred = threading.Event()
        
        def timeout_handler():
            timeout_occurred.set()
        
        timer = threading.Timer(seconds, timeout_handler)
        timer.start()
        
        try:
            yield timeout_occurred
        finally:
            timer.cancel()
    else:
        # On Unix systems, use signal-based timeout
        import signal
        
        def timeout_handler(signum, frame):
            raise TimeoutException("Script execution timed out")
        
        signal.signal(signal.SIGALRM, timeout_handler)
        signal.alarm(seconds)
        try:
            yield None
        finally:
            signal.alarm(0)

@dataclass
class Product:
    IdProduct: str = ""
    Name: str = ""
    ShortName: str = ""
    TaxValue: str = ""
    IsService: bool = False
    HandlingType: str = ""
    Price: str = ""
    TaxIncluded: str = ""
    Family: str = ""
    BrandName: str = ""
    BrandModels: str = ""
    DirectDiscount: str = ""

    def to_dict(self):
        return asdict(self)

def clean_text(text):
    """Clean and normalize text data"""
    if not text or text == "null" or text == "undefined":
        return ""
    
    # Convert to string and strip whitespace
    text = str(text).strip()
    
    # Normalize unicode characters
    text = unicodedata.normalize('NFKD', text)
    
    # Remove control characters and replace with space
    text = ''.join(char if ord(char) >= 32 else ' ' for char in text)
    
    # Clean up multiple spaces
    text = re.sub(r'\s+', ' ', text).strip()
    
    return text

def safe_parse_number(value, default="0"):
    """Safely parse numeric values"""
    if not value:
        return default
    
    # Clean the value
    cleaned = clean_text(value)
    
    # Extract numbers and decimal points
    numeric_match = re.search(r'[\d,]+\.?\d*', cleaned)
    if numeric_match:
        result = numeric_match.group().replace(',', '')
        return result if result else default
    
    return default

def safe_parse_boolean(value, default=False):
    """Safely parse boolean values"""
    if isinstance(value, bool):
        return value
    
    if isinstance(value, str):
        return value.lower() in ('true', '1', 'yes', 'on', 'checked')
    
    return bool(value) if value is not None else default

def safe_get_text(page, selector, timeout=3000):
    """Safely get text from an element with timeout"""
    try:
        element = page.wait_for_selector(selector, timeout=timeout)
        if element:
            return element.inner_text()
    except (PlaywrightTimeoutError, Exception):
        pass
    return ""

def safe_get_input_value(page, selector, timeout=3000):
    """Safely get input value from an element with timeout"""
    try:
        element = page.wait_for_selector(selector, timeout=timeout)
        if element:
            return element.input_value()
    except (PlaywrightTimeoutError, Exception):
        pass
    return ""

def safe_is_checked(page, selector, timeout=3000):
    """Safely check if checkbox is checked"""
    try:
        element = page.wait_for_selector(selector, timeout=timeout)
        if element:
            return element.is_checked()
    except (PlaywrightTimeoutError, Exception):
        pass
    return False

def scrape_products(email, senha):
    """Main scraping function with improved error handling"""
    base_url = "https://www.keyinvoice.com/"
    demo_url = "https://login.keyinvoice.com/"
    
    print(f"[DEBUG] Starting scrape with base_url: {base_url}", file=sys.stderr)
    
    try:
        with timeout_context(SCRIPT_TIMEOUT):
            print("[DEBUG] Entering timeout context", file=sys.stderr)
            
            with sync_playwright() as p:
                print("[DEBUG] Playwright context created", file=sys.stderr)
                browser = None
                try:
                    print("[DEBUG] Launching browser", file=sys.stderr)
                    # Launch browser with optimized settings
                    browser = p.chromium.launch(
                        headless=True,
                        args=[
                            '--no-sandbox',
                            '--disable-dev-shm-usage',
                            '--disable-gpu',
                            '--disable-extensions',
                            '--disable-images',  # Disable images for faster loading
                            '--disable-plugins',
                            '--disable-javascript-harmony-shipping'
                        ]
                    )
                    print("[DEBUG] Browser launched successfully", file=sys.stderr)
                    
                    context = browser.new_context(
                        accept_downloads=False,
                        ignore_https_errors=True,
                        java_script_enabled=True
                    )
                    
                    page = context.new_page()
                    page.set_default_timeout(PAGE_TIMEOUT)
                    page.set_default_navigation_timeout(NAVIGATION_TIMEOUT)
                    print("[DEBUG] Page created and configured", file=sys.stderr)
                    
                    # Navigate to homepage
                    print(f"[DEBUG] Navigating to {base_url}", file=sys.stderr)
                    page.goto(base_url, wait_until='domcontentloaded')
                    page.wait_for_load_state('networkidle', timeout=10000)
                    print("[DEBUG] Homepage loaded", file=sys.stderr)
                    
                    # Click login button
                    try:
                        print("[DEBUG] Looking for login button", file=sys.stderr)
                        page.click('text=Login', timeout=5000)
                        page.wait_for_load_state('networkidle', timeout=10000)
                        print("[DEBUG] Login button clicked", file=sys.stderr)
                    except PlaywrightTimeoutError as e:
                        print(f"[ERROR] Failed to find login button: {str(e)}", file=sys.stderr)
                        raise Exception("Failed to find or click login button")
                    
                    # Fill login form
                    try:
                        print("[DEBUG] Filling login form", file=sys.stderr)
                        page.fill('input[name="xfldEmail"]', email, timeout=5000)
                        page.fill('input[name="xfldSenha"]', senha, timeout=5000)
                        page.click('input[name="ENTRAR"]', timeout=5000)
                        page.wait_for_load_state('networkidle', timeout=15000)
                        print("[DEBUG] Login form submitted", file=sys.stderr)
                    except PlaywrightTimeoutError as e:
                        print(f"[ERROR] Failed to fill login form: {str(e)}", file=sys.stderr)
                        raise Exception("Failed to fill login form or login")
                    
                    # Enter demo mode
                    try:
                        print("[DEBUG] Entering demo mode", file=sys.stderr)
                        page.click('text=DEMO', timeout=5000)
                        page.click('text=Utilizador Demo', timeout=5000)
                        page.wait_for_load_state('networkidle', timeout=10000)
                        time.sleep(1)  # Reduced sleep time
                        print("[DEBUG] Demo mode activated", file=sys.stderr)
                    except PlaywrightTimeoutError as e:
                        print(f"[ERROR] Failed to enter demo mode: {str(e)}", file=sys.stderr)
                        raise Exception("Failed to enter demo mode")
                    
                    # Navigate to Tabelas section
                    try:
                        print("[DEBUG] Navigating to Tabelas", file=sys.stderr)
                        page.hover('a[title="Tabelas"]', timeout=5000)
                        page.click('a[title="Tabelas"]', timeout=5000)
                        page.wait_for_load_state('domcontentloaded', timeout=10000)
                        print("[DEBUG] Tabelas section opened", file=sys.stderr)
                    except PlaywrightTimeoutError as e:
                        print(f"[ERROR] Failed to navigate to Tabelas: {str(e)}", file=sys.stderr)
                        raise Exception("Failed to navigate to Tabelas section")
                    
                    # Open Artigos
                    try:
                        print("[DEBUG] Opening Artigos page", file=sys.stderr)
                        page.wait_for_selector('a:has-text("Artigos"):visible', timeout=10000)
                        page.click('a[href*="ListaArtigos.php?SID=wrs1gheptgf9yjcztfhq9"]', timeout=5000)
                        page.wait_for_load_state('networkidle', timeout=15000)
                        print("[DEBUG] Artigos page loaded", file=sys.stderr)
                    except PlaywrightTimeoutError as e:
                        print(f"[ERROR] Failed to open Artigos: {str(e)}", file=sys.stderr)
                        raise Exception("Failed to open Artigos page")
                    
                    # Wait for product table
                    try:
                        print("[DEBUG] Waiting for product table", file=sys.stderr)
                        page.wait_for_selector("table.tbl_artz tbody tr", timeout=15000)
                        print("[DEBUG] Product table found", file=sys.stderr)
                    except PlaywrightTimeoutError as e:
                        print(f"[ERROR] Product table not found: {str(e)}", file=sys.stderr)
                        raise Exception("Failed to find product table")
                    
                    # Get product links
                    links = []
                    try:
                        print("[DEBUG] Extracting product links", file=sys.stderr)
                        a_elements = page.locator("table.tbl_artz tbody tr td a")
                        count = a_elements.count()
                        print(f"[DEBUG] Found {count} potential product links", file=sys.stderr)
                        
                        for i in range(count):  # Limit to 50 products for performance
                            href = a_elements.nth(i).get_attribute("href", timeout=2000)
                            if href:
                                links.append(href)
                        
                        print(f"[DEBUG] Extracted {len(links)} valid product links", file=sys.stderr)
                    except Exception as e:
                        print(f"[ERROR] Failed to extract links: {str(e)}", file=sys.stderr)
                        raise Exception(f"Failed to extract product links: {str(e)}")
                    
                    if not links:
                        print("[ERROR] No product links found", file=sys.stderr)
                        raise Exception("No product links found")
                    
                    # Scrape individual products
                    products = []
                    failed_count = 0
                    max_failures = min(len(links) // 2, 10)  # Allow up to half failures or 10, whichever is smaller
                    print(f"[DEBUG] Starting to scrape {len(links)} products, max failures: {max_failures}", file=sys.stderr)
                    
                    for i, link in enumerate(links):
                        if failed_count > max_failures:
                            print(f"[DEBUG] Max failures ({max_failures}) reached, stopping", file=sys.stderr)
                            break
                            
                        try:
                            print(f"[DEBUG] Processing product {i+1}/{len(links)}: {link}", file=sys.stderr)
                            full_link = f"{demo_url}{link}"
                            page.goto(full_link, timeout=NAVIGATION_TIMEOUT, wait_until='domcontentloaded')
                            
                            # Extract product data with safe methods
                            id_product = clean_text(safe_get_input_value(page, "#CodigoArtigo"))
                            name = clean_text(safe_get_input_value(page, "#Designacao"))
                            short_name = clean_text(safe_get_input_value(page, "#Abreviatura"))
                            
                            # Tax value extraction with cleaning
                            tax_text = safe_get_text(page, "#select2-CodigoTaxaIVA-container")
                            tax_value = safe_parse_number(tax_text.split("%")[0] if "%" in tax_text else tax_text)
                            
                            is_service = safe_is_checked(page, "#Servicos")
                            handling_type = clean_text(safe_get_text(page, "#CodigoTratamento ~ p"))
                            price = safe_parse_number(safe_get_input_value(page, "#COL_PRE_001"))
                            family = clean_text(safe_get_text(page, "#select2-CodigoFam_1-container"))
                            brand_name = clean_text(safe_get_text(page, "#select2-CodigoMarca-container"))
                            brand_models = clean_text(safe_get_text(page, "#select2-CodigoTipoArtigo-container"))
                            direct_discount = safe_parse_number(safe_get_input_value(page, "#DescontoDirecto"))
                            
                            # Skip products with no meaningful data
                            if not id_product and not name:
                                failed_count += 1
                                print(f"[DEBUG] Product {i+1} skipped - no ID or name", file=sys.stderr)
                                continue
                            
                            product = Product(
                                IdProduct=id_product,
                                Name=name,
                                ShortName=short_name,
                                TaxValue=tax_value,
                                IsService=is_service,
                                HandlingType=handling_type,
                                Price=price,
                                TaxIncluded="",  # Not available in the form
                                Family=family,
                                BrandName=brand_name,
                                BrandModels=brand_models,
                                DirectDiscount=direct_discount
                            )
                            
                            products.append(product)
                            print(f"[DEBUG] Product {i+1} scraped successfully: {id_product} - {name}", file=sys.stderr)
                            
                        except Exception as e:
                            failed_count += 1
                            print(f"[DEBUG] Product {i+1} failed: {str(e)}", file=sys.stderr)
                            continue
                    
                    print(f"[DEBUG] Scraping completed: {len(products)} products, {failed_count} failures", file=sys.stderr)
                    return products
                    
                except Exception as e:
                    print(f"[ERROR] Browser operation failed: {str(e)}", file=sys.stderr)
                    raise Exception(f"Browser operation failed: {str(e)}")
                finally:
                    if browser:
                        try:
                            print("[DEBUG] Closing browser", file=sys.stderr)
                            browser.close()
                            print("[DEBUG] Browser closed", file=sys.stderr)
                        except Exception as e:
                            print(f"[DEBUG] Error closing browser: {str(e)}", file=sys.stderr)
                            pass
                            
    except TimeoutException:
        print("[ERROR] Script execution timed out", file=sys.stderr)
        raise Exception("Script execution timed out")
    except Exception as e:
        print(f"[ERROR] Scraping failed: {str(e)}", file=sys.stderr)
        raise Exception(f"Scraping failed: {str(e)}")

def main():
    try:
        # Log script start to stderr for debugging (won't interfere with JSON output)
        print(f"[DEBUG] Script started with {len(sys.argv)} arguments", file=sys.stderr)
        
        if len(sys.argv) != 3:
            print(f"[DEBUG] Invalid arguments count: {len(sys.argv)}", file=sys.stderr)
            output = {
                "success": False,
                "error": "Email and senha are required",
                "products": [],
                "debug": f"Arguments received: {len(sys.argv)}"
            }
            print(json.dumps(output, ensure_ascii=False, indent=2))
            sys.exit(1)
        
        email = sys.argv[1].strip()
        senha = sys.argv[2].strip()
        
        print(f"[DEBUG] Email length: {len(email)}, Senha length: {len(senha)}", file=sys.stderr)
        
        if not email or not senha:
            print(f"[DEBUG] Empty credentials - Email: '{email}', Senha: '{senha}'", file=sys.stderr)
            output = {
                "success": False,
                "error": "Email and senha cannot be empty",
                "products": [],
                "debug": f"Email empty: {not email}, Senha empty: {not senha}"
            }
            print(json.dumps(output, ensure_ascii=False, indent=2))
            sys.exit(1)
        
        print("[DEBUG] Starting scraping process", file=sys.stderr)
        
        # Check if playwright is available
        try:
            from playwright.sync_api import sync_playwright
            print("[DEBUG] Playwright import successful", file=sys.stderr)
        except ImportError as e:
            output = {
                "success": False,
                "error": f"Playwright not installed: {str(e)}",
                "products": [],
                "debug": "Import error"
            }
            print(json.dumps(output, ensure_ascii=False, indent=2))
            sys.exit(1)
        
        products = scrape_products(email, senha)
        print(f"[DEBUG] Scraping completed, found {len(products)} products", file=sys.stderr)
        
        if not products:
            output = {
                "success": True,
                "error": "No products found",
                "products": [],
                "debug": "Scraping successful but no products"
            }
        else:
            output = {
                "success": True,
                "products": [p.to_dict() for p in products],
                "debug": f"Successfully scraped {len(products)} products"
            }
        
        print(json.dumps(output, ensure_ascii=False, indent=2))
        
    except Exception as e:
        print(f"[ERROR] Exception in main: {str(e)}", file=sys.stderr)
        print(f"[ERROR] Exception type: {type(e).__name__}", file=sys.stderr)
        
        import traceback
        print(f"[ERROR] Traceback: {traceback.format_exc()}", file=sys.stderr)
        
        output = {
            "success": False,
            "error": str(e),
            "products": [],
            "debug": f"Exception: {type(e).__name__} - {str(e)}"
        }
        print(json.dumps(output, ensure_ascii=False, indent=2))
        sys.exit(1)

if __name__ == "__main__":
    main()