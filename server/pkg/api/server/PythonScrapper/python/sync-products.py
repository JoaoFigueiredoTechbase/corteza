from playwright.sync_api import sync_playwright
from urllib.parse import urljoin
from datetime import datetime
import time
from dataclasses import dataclass, asdict
import json
import re
import sys

step_number = 0

@dataclass
class Product:
    IdProduct: str = ""
    Name: str = ""
    ShortName: str = ""
    TaxValue: str = ""
    IsService: str = ""
    HandlingType: str = ""
    Price: str = ""
    TaxIncluded: str = ""
    Family: str = ""
    BrandName: str = ""
    BrandModels: str = ""
    DirectDiscount: str = ""

    def to_dict(self):
        return asdict(self)

# def log_step(message):
#     global step_number

#     timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
#     print(f"[{timestamp}] STEP {step_number}: {message}")
#     step_number = step_number + 1
#     #time.sleep(2)

def scrape_example(email, senha):
    #Variaveis
    base_url = "https://www.keyinvoice.com/"
    demo_url = "https://login.keyinvoice.com/"
    links = []

    # log_step("Starting WebBot Fatura")

    # log_step("Launching browser")
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        context = browser.new_context(accept_downloads=True)
        page = context.new_page()
        
        # log_step(f"Navigating to {base_url}")
        page.goto(base_url)
        page.wait_for_load_state('networkidle')
        # log_step("Homepage loaded successfully")
        
        # log_step("Clicking login button")
        page.press('text=Login', 'Enter')
        page.wait_for_load_state('networkidle')
        # log_step("Login page loaded")

        # log_step("Filling login form")
        page.fill('input[name="xfldEmail"]', email) 
        page.fill('input[name="xfldSenha"]', senha)
        # log_step("Credentials entered")

        # log_step("Submitting login form")
        page.press('input[name="ENTRAR"]', 'Enter')
        page.wait_for_load_state('networkidle')
        # log_step("Login successful")
        
        # log_step("Entering demo mode")
        page.click('text=DEMO')
        page.click('text=Utilizador Demo')
        page.wait_for_load_state('networkidle')
        # log_step("Demo mode activated")
        time.sleep(2)

        # log_step("Navigating to Tabelas section")
        page.hover('a[title="Tabelas"]')
        page.click('a[title="Tabelas"]')
        page.wait_for_load_state('domcontentloaded')
        # log_step("Tabelas section loaded")

        # log_step("Opening Artigos")
        page.wait_for_selector('a:has-text("Artigos"):visible')
        page.click('a[href*="ListaArtigos.php?SID=wrs1gheptgf9yjcztfhq9"]')
        page.wait_for_load_state('networkidle')
        # log_step("Artigos page loaded")

        page.wait_for_selector("table.tbl_artz tbody tr")

        a_elements = page.locator("table.tbl_artz tbody tr td a")
        count = a_elements.count()
        
        for i in range(count):
            href = a_elements.nth(i).get_attribute("href")
            if href:
                links.append(href)

        # print("Found links:", links)

        results = []
        products = []

        for link in links:
            try:
                link = f"{demo_url}{link}"
                
                page.goto(link, timeout=3000)  # 10 sec timeout
                # Optionally, wait for some element on the target page
                page.wait_for_load_state("domcontentloaded")

                page.add_style_tag(content="* { all: unset !important; }")

                id_product = page.locator("#CodigoArtigo").input_value()
                name = page.locator("#Designacao").input_value() 
                short_name = page.locator("#Abreviatura").input_value()
                tax_value = page.locator("#select2-CodigoTaxaIVA-container").inner_text()
                tax_value = tax_value.split("%")[0]

                is_service = page.locator("#Servicos").is_checked()
                handling_type = page.locator("#CodigoTratamento ~ p").inner_text()
                # price = page.locator("#PrecoCusto").input_value()
                price = page.locator("#COL_PRE_001").input_value()
                # tax_included = page.locator("#TaxIncluded").input_value()  # example
                family = page.locator("#select2-CodigoFam_1-container").inner_text()
                brand_name = page.locator("#select2-CodigoMarca-container").inner_text()
                brand_models = page.locator("#select2-CodigoTipoArtigo-container").inner_text()
                direct_discount = page.locator("#DescontoDirecto").input_value()  # example

                product = Product(
                    IdProduct=id_product,
                    Name=name,
                    ShortName=short_name,
                    TaxValue=tax_value,
                    IsService=is_service,
                    HandlingType=handling_type,
                    Price=price,
                    # TaxIncluded=tax_included,
                    Family=family,
                    BrandName=brand_name,
                    BrandModels=brand_models,
                    DirectDiscount=direct_discount
                )
                        
                # print(f"Visited successfully: {link}")
                results.append((link, True))
                products.append(product)
            except Exception as e:
                # print(f"Failed to visit: {link}, Error: {e}")
                results.append((link, False))
        
        # for link, success in results:
        #     print(f"{link} -> {'OK' if success else 'FAILED'}")

        browser.close()

        # clean = []
        # for item in products:
        #     # Convert all values to strings and ensure UTF-8 encoding
        #     product_dict = {k: str(v).encode('utf-8').decode('utf-8') for k, v in item.items()}
        #     clean.append(Product(**product_dict))

        return products, results

if __name__ == "__main__":

    if len(sys.argv) != 3:
        print(json.dumps({"success": False, "error": "Email and senha are required", "products": []}, ensure_ascii=False, indent=2))
        sys.exit(1)

    email = sys.argv[1]
    senha = sys.argv[2]

    products, results = scrape_example(email, senha)

    #scrape_example()
    # product = scrape_example()
    # print(product)

    # products, results = scrape_example()
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

    # products, results = scrape_example()

    output = {
        "success": True,
        "products": [p.to_dict() for p in products],
        # "results": [{"url": link, "success": success} for link, success in results]
    }

    print(json.dumps(output, ensure_ascii=False, indent=2))