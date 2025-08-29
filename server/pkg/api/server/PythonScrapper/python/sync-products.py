from playwright.sync_api import sync_playwright

def scrape_example():
    with sync_playwright() as p:
        # Launch Chromium browser in headless mode
        browser = p.chromium.launch(headless=True)
        page = browser.new_page()
        
        # Go to example.com
        page.goto("https://example.com")
        
        # Print page title
        title = page.title()
        print("Page title:", title)
        
        # Extract the paragraph text
        paragraph = page.locator("p").nth(0).inner_text()
        print("Paragraph text:", paragraph)
        
        browser.close()

if __name__ == "__main__":
    scrape_example()
