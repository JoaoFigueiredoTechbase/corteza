from playwright.sync_api import sync_playwright
from urllib.parse import urljoin
from datetime import datetime
import time
from dataclasses import dataclass

step_number = 0

@dataclass
class Product:
    IdProduct: str
    Name: str
    ShortName: str
    TaxValue: str
    IsService: str
    HandlingType: str
    Price: str
    TaxIncluded: str
    Family: str
    BrandName: str
    BrandModels: str
    DirectDiscount: str

def log_step(message):
    global step_number

    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    print(f"[{timestamp}] STEP {step_number}: {message}")
    step_number = step_number + 1
    #time.sleep(2)


# def extract_id_product(page):
#     try:
#         selector = "//div[contains(text(), 'Código') or contains(., 'Código')]/following-sibling::input[1]"
#         page.wait_for_selector(f"xpath={selector}", timeout=5000)
#         return page.locator(f"xpath={selector}").inputValue() or "N/A"
#     except Exception as e:
#         print(f"Error extracting IdProduct: {e}")
#         return "N/A"

# def extract_name(page):
#     try:
#         selector = "//div[contains(text(), 'Designação') or contains(., 'Designação')]/following-sibling::input[1]"
#         page.wait_for_selector(f"xpath={selector}", timeout=5000)
#         return page.locator(f"xpath={selector}").inputValue() or "N/A"
#     except Exception as e:
#         print(f"Error extracting Name: {e}")
#         return "N/A"

# def extract_short_name(page):
#     try:
#         selector = "//div[contains(text(), 'Abreviatura') or contains(., 'Abreviatura')]/following-sibling::input[1]"
#         page.wait_for_selector(f"xpath={selector}", timeout=5000)
#         return page.locator(f"xpath={selector}").inputValue() or "N/A"
#     except Exception as e:
#         print(f"Error extracting ShortName: {e}")
#         return "N/A"

# def extract_tax_value(page):
#     try:
#         selector = "//div[contains(text(), 'Taxa de IVA') or contains(., 'Taxa de IVA')]/following-sibling::select[1]"
#         page.wait_for_selector(f"xpath={selector}", timeout=5000)
#         return page.locator(f"xpath={selector}").evaluate("el => el.options[el.selectedIndex]?.textContent.trim()") or "N/A"
#     except Exception as e:
#         print(f"Error extracting TaxValue: {e}")
#         return "N/A"

# def extract_is_service(page):
#     try:
#         page.locator('text=Stocks').click()  # Navigate to Stocks tab
#         selector = "//div[contains(text(), 'Artigo reconhecido como Serviços?') or contains(., 'Artigo reconhecido como Serviços?')]/preceding-sibling::input[@type='checkbox'][1]"
#         page.wait_for_selector(f"xpath={selector}", timeout=5000)
#         is_checked = page.locator(f"xpath={selector}").is_checked()
#         return "Yes" if is_checked else "No"
#     except Exception as e:
#         print(f"Error extracting IsService: {e}")
#         return "No"

# def extract_handling_type(page):
#     try:
#         page.locator('text=Stocks').click()  # Ensure Stocks tab
#         selector = "//div[contains(text(), 'Modo de Tratamento') or contains(., 'Modo de Tratamento')]/following-sibling::select[1] | //div[contains(text(), 'Modo de Tratamento')]/following-sibling::input[1]"
#         page.wait_for_selector(f"xpath={selector}", timeout=5000)
#         return page.locator(f"xpath={selector}").evaluate("el => el.tagName === 'SELECT' ? el.options[el.selectedIndex]?.textContent.trim() : el.value") or "N/A"
#     except Exception as e:
#         print(f"Error extracting HandlingType: {e}")
#         return "N/A"

# def extract_price(page):
#     try:
#         page.locator('text=Preços').click()  # Navigate to Preços tab
#         selector = "//div[contains(text(), 'Preços')]/following::table//td[contains(text(), 'PVP s/IVA')]/following-sibling::td/input"
#         page.wait_for_selector(f"xpath={selector}", timeout=5000)
#         return page.locator(f"xpath={selector}").inputValue() or "0.00"
#     except Exception as e:
#         print(f"Error extracting Price: {e}")
#         return "0.00"

# def extract_tax_included(page):
#     try:
#         page.locator('text=Preços').click()  # Ensure Preços tab
#         selector = "//div[contains(text(), 'Preços')]/following::table//th[contains(text(), 'IVA incluído?')]/following-sibling::td/input[@type='checkbox']"
#         page.wait_for_selector(f"xpath={selector}", timeout=5000)
#         is_checked = page.locator(f"xpath={selector}").is_checked()
#         return "Yes" if is_checked else "No"
#     except Exception as e:
#         print(f"Error extracting TaxIncluded: {e}")
#         return "No"

# def extract_family(page):
#     try:
#         page.locator('text=Geral').click()  # Back to Geral tab
#         selector = "//div[contains(text(), 'Família') or contains(., 'Família')]/following-sibling::select[1]"
#         page.wait_for_selector(f"xpath={selector}", timeout=5000)
#         return page.locator(f"xpath={selector}").evaluate("el => el.options[el.selectedIndex]?.textContent.trim()") or "N/A"
#     except Exception as e:
#         print(f"Error extracting Family: {e}")
#         return "N/A"

# def extract_brand_name(page):
#     try:
#         page.locator('text=Geral').click()  # Ensure Geral tab
#         selector = "//div[contains(text(), 'Marca') or contains(., 'Marca')]/following-sibling::select[1]"
#         page.wait_for_selector(f"xpath={selector}", timeout=5000)
#         return page.locator(f"xpath={selector}").evaluate("el => el.options[el.selectedIndex]?.textContent.trim()") or "N/A"
#     except Exception as e:
#         print(f"Error extracting BrandName: {e}")
#         return "N/A"

# def extract_brand_models(page):
#     try:
#         page.locator('text=Geral').click()  # Ensure Geral tab
#         selector = "//div[contains(text(), 'Modelos') or contains(., 'Modelos')]/following-sibling::input[1]"
#         page.wait_for_selector(f"xpath={selector}", timeout=5000)
#         return page.locator(f"xpath={selector}").inputValue() or "N/A"
#     except Exception as e:
#         print(f"Error extracting BrandModels: {e}")
#         return "N/A"

# def extract_direct_discount(page):
#     try:
#         page.locator('text=Descontos').click()  # Navigate to Descontos tab
#         selector = "//div[contains(text(), 'Desconto direto (%)') or contains(., 'Desconto direto (%)')]/following-sibling::input[1]"
#         page.wait_for_selector(f"xpath={selector}", timeout=5000)
#         return page.locator(f"xpath={selector}").inputValue() + "%" or "0%"
#     except Exception as e:
#         print(f"Error extracting DirectDiscount: {e}")
#         return "0%"

def scrape_example():

    #Variaveis

    links = []

    log_step("Starting WebBot Fatura")

    log_step("Launching browser")
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        context = browser.new_context(accept_downloads=True)
        page = context.new_page()
        
        log_step(f"Navigating to {base_url}")
        page.goto(base_url)
        page.wait_for_load_state('networkidle')
        log_step("Homepage loaded successfully")
        
        log_step("Clicking login button")
        page.press('text=Login', 'Enter')
        page.wait_for_load_state('networkidle')
        log_step("Login page loaded")

        log_step("Filling login form")
        page.fill('input[name="xfldEmail"]', email) 
        page.fill('input[name="xfldSenha"]', senha)
        log_step("Credentials entered")

        log_step("Submitting login form")
        page.press('input[name="ENTRAR"]', 'Enter')
        page.wait_for_load_state('networkidle')
        log_step("Login successful")
        
        log_step("Entering demo mode")
        page.click('text=DEMO')
        page.click('text=Utilizador Demo')
        page.wait_for_load_state('networkidle')
        log_step("Demo mode activated")
        time.sleep(2)

        log_step("Navigating to Tabelas section")
        page.hover('a[title="Tabelas"]')
        page.click('a[title="Tabelas"]')
        page.wait_for_load_state('domcontentloaded')
        log_step("Tabelas section loaded")

        log_step("Opening Artigos")
        page.wait_for_selector('a:has-text("Artigos"):visible')
        page.click('a[href*="ListaArtigos.php?SID=wrs1gheptgf9yjcztfhq9"]')
        page.wait_for_load_state('networkidle')
        log_step("Artigos page loaded")

        # Print page title
        title = page.title()
        print("Page title:", title)

        page.wait_for_selector("table.tbl_artz tbody tr")

        a_elements = page.locator("table.tbl_artz tbody tr td a")
        count = a_elements.count()
        
        for i in range(count):
            href = a_elements.nth(i).get_attribute("href")
            if href:
                links.append(href)

        print("Found links:", links)

        results = []
        products = []

        for link in links:
            try:
                link = f"{demo_url}{link}"
                
                page.goto(link, timeout=3000)  # 10 sec timeout
                # Optionally, wait for some element on the target page
                page.wait_for_load_state("domcontentloaded")

                page.add_style_tag(content="* { all: unset !important; }")

                # product = Product(
                #     IdProduct=extract_id_product(page),
                #     Name=extract_name(page),
                #     ShortName=extract_short_name(page),
                #     TaxValue=extract_tax_value(page),
                #     IsService=extract_is_service(page),
                #     HandlingType=extract_handling_type(page),
                #     Price=extract_price(page),
                #     TaxIncluded=extract_tax_included(page),
                #     Family=extract_family(page),
                #     BrandName=extract_brand_name(page),
                #     BrandModels=extract_brand_models(page),
                #     DirectDiscount=extract_direct_discount(page)
                # )
                # products.append(product)
                # log_step(f"Extracted product: {product.IdProduct}")
                # print(f"Product extracted: {product}")

                
                print(f"Visited successfully: {link}")
                results.append((link, True))
            except Exception as e:
                print(f"Failed to visit: {link}, Error: {e}")
                results.append((link, False))
        
        for link, success in results:
            print(f"{link} -> {'OK' if success else 'FAILED'}")

        browser.close()

if __name__ == "__main__":
    scrape_example()
    # products = scrape_example()
    # for i, product in enumerate(products, 1):
    #     print(f"\nProduct {i}:")
    #     print(f"  IdProduct: {product.IdProduct}")
    #     print(f"  Name: {product.Name}")
    #     print(f"  ShortName: {product.ShortName}")
    #     print(f"  TaxValue: {product.TaxValue}")
    #     print(f"  IsService: {product.IsService}")
    #     print(f"  HandlingType: {product.HandlingType}")
    #     print(f"  Price: {product.Price}")
    #     print(f"  TaxIncluded: {product.TaxIncluded}")
    #     print(f"  Family: {product.Family}")
    #     print(f"  BrandName: {product.BrandName}")
    #     print(f"  BrandModels: {product.BrandModels}")
    #     print(f"  DirectDiscount: {product.DirectDiscount}")    