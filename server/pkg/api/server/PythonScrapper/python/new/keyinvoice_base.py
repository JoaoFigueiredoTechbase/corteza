# from playwright.sync_api import sync_playwright, TimeoutError as PlaywrightTimeoutError
# from abc import ABC, abstractmethod
# from dataclasses import dataclass, asdict
# from datetime import datetime
# from typing import Dict, List, Optional, Any, Union
# from contextlib import contextmanager
# import json
# import sys
# import time
# import threading
# import tempfile
# import os
# import base64

# # Configuration constants
# class Config:
#     DEFAULT_TIMEOUT = 10000  # 10 seconds
#     NAVIGATION_TIMEOUT = 15000  # 15 seconds
#     MAX_SCRIPT_TIMEOUT = 3600  # 1 hour
#     BASE_URL = "https://www.keyinvoice.com/"
#     DEMO_URL = "https://login.keyinvoice.com/"

# # Custom exceptions
# class KeyInvoiceError(Exception):
#     """Base exception for KeyInvoice operations"""
#     pass

# class LoginError(KeyInvoiceError):
#     """Login related errors"""
#     pass

# class NavigationError(KeyInvoiceError):
#     """Navigation related errors"""
#     pass

# class ScriptTimeoutError(KeyInvoiceError):
#     """Script timeout errors"""
#     pass

# # Data classes
# @dataclass
# class LoginCredentials:
#     """Data class for login credentials"""
#     email: str
#     password: str
    
#     def __post_init__(self):
#         if not self.email or not self.password:
#             raise ValueError("Email and password are required")

# @dataclass
# class ScriptResponse:
#     """Standard response format for all scripts"""
#     success: bool
#     data: Optional[Dict[str, Any]] = None
#     error: Optional[str] = None
#     message: Optional[str] = None
    
#     def to_json(self) -> str:
#         return json.dumps(asdict(self), ensure_ascii=False, indent=2)

# # Logger interface
# class Logger:
#     """Enhanced logging with different levels"""
    
#     @staticmethod
#     def info(message: str, *args) -> None:
#         timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
#         print(f"[{timestamp}] INFO: {message.format(*args) if args else message}", file=sys.stderr)
    
#     @staticmethod
#     def error(message: str, *args) -> None:
#         timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
#         print(f"[{timestamp}] ERROR: {message.format(*args) if args else message}", file=sys.stderr)
    
#     @staticmethod
#     def debug(message: str, *args) -> None:
#         timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
#         print(f"[{timestamp}] DEBUG: {message.format(*args) if args else message}", file=sys.stderr)
    
#     @staticmethod
#     def warn(message: str, *args) -> None:
#         timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
#         print(f"[{timestamp}] WARN: {message.format(*args) if args else message}", file=sys.stderr)

# # Base browser automation class
# class KeyInvoiceBotBase(ABC):
#     """Base class for KeyInvoice browser automation"""
    
#     def __init__(self, credentials: LoginCredentials, headless: bool = True):
#         self.credentials = credentials
#         self.headless = headless
#         self.browser = None
#         self.context = None
#         self.page = None
#         self.step_counter = 0
#         self.playwright = None
#         self.logger = Logger()
    
#     def __enter__(self):
#         self.launch_browser()
#         return self
    
#     def __exit__(self, exc_type, exc_val, exc_tb):
#         self.close()
    
#     def log_step(self, message: str) -> None:
#         """Log a step with counter"""
#         self.step_counter += 1
#         self.logger.debug(f"STEP {self.step_counter}: {message}")
    
#     def launch_browser(self) -> None:
#         """Launch browser and create context"""
#         try:
#             self.log_step("Launching browser")
#             self.playwright = sync_playwright().start()
            
#             self.browser = self.playwright.chromium.launch(
#                 headless=self.headless,
#                 args=[
#                     '--no-sandbox',
#                     '--disable-dev-shm-usage',
#                     '--disable-gpu',
#                     '--disable-extensions',
#                     '--disable-images' if self.headless else '',
#                     '--disable-plugins'
#                 ] if self.headless else []
#             )
            
#             self.context = self.browser.new_context(
#                 accept_downloads=True,
#                 ignore_https_errors=True
#             )
            
#             self.page = self.context.new_page()
#             self.page.set_default_timeout(Config.DEFAULT_TIMEOUT)
#             self.page.set_default_navigation_timeout(Config.NAVIGATION_TIMEOUT)
            
#             self.log_step("Browser launched successfully")
            
#         except Exception as e:
#             self.logger.error(f"Failed to launch browser: {e}")
#             raise KeyInvoiceError(f"Browser launch failed: {e}")
    
#     def navigate_to_homepage(self) -> None:
#         """Navigate to KeyInvoice homepage"""
#         try:
#             self.log_step(f"Navigating to {Config.BASE_URL}")
#             self.page.goto(Config.BASE_URL, wait_until='domcontentloaded')
#             self.page.wait_for_load_state('networkidle', timeout=10000)
#             self.log_step("Homepage loaded successfully")
#         except Exception as e:
#             raise NavigationError(f"Failed to navigate to homepage: {e}")
    
#     def login(self) -> None:
#         """Login to KeyInvoice"""
#         try:
#             self.log_step("Clicking login button")
#             self.page.click('text=Login', timeout=5000)
#             self.page.wait_for_load_state('networkidle', timeout=10000)
            
#             self.log_step("Filling login form")
#             self.page.fill('input[name="xfldEmail"]', self.credentials.email, timeout=5000)
#             self.page.fill('input[name="xfldSenha"]', self.credentials.password, timeout=5000)
            
#             self.log_step("Submitting login form")
#             self.page.click('input[name="ENTRAR"]', timeout=5000)
#             self.page.wait_for_load_state('networkidle', timeout=15000)
            
#             self.log_step("Login successful")
            
#         except PlaywrightTimeoutError as e:
#             raise LoginError(f"Login timeout: {e}")
#         except Exception as e:
#             raise LoginError(f"Login failed: {e}")
    
#     def enter_demo_mode(self) -> None:
#         """Enter demo mode"""
#         try:
#             self.log_step("Entering demo mode")
#             self.page.click('text=DEMO', timeout=5000)
#             self.page.click('text=Utilizador Demo', timeout=5000)
#             self.page.wait_for_load_state('networkidle', timeout=10000)
#             time.sleep(1)
#             self.log_step("Demo mode activated")
#         except Exception as e:
#             raise NavigationError(f"Failed to enter demo mode: {e}")
    
#     def safe_click(self, selector: str, timeout: int = 5000) -> bool:
#         """Safely click an element with error handling"""
#         try:
#             self.page.click(selector, timeout=timeout)
#             return True
#         except PlaywrightTimeoutError:
#             self.logger.warn(f"Timeout clicking selector: {selector}")
#             return False
#         except Exception as e:
#             self.logger.error(f"Error clicking selector {selector}: {e}")
#             return False
    
#     def safe_fill(self, selector: str, value: str, timeout: int = 5000) -> bool:
#         """Safely fill an input with error handling"""
#         try:
#             self.page.fill(selector, value, timeout=timeout)
#             return True
#         except PlaywrightTimeoutError:
#             self.logger.warn(f"Timeout filling selector: {selector}")
#             return False
#         except Exception as e:
#             self.logger.error(f"Error filling selector {selector}: {e}")
#             return False
    
#     def safe_get_text(self, selector: str, timeout: int = 3000) -> str:
#         """Safely get text from an element"""
#         try:
#             element = self.page.wait_for_selector(selector, timeout=timeout)
#             return element.inner_text() if element else ""
#         except (PlaywrightTimeoutError, Exception):
#             return ""
    
#     def safe_get_input_value(self, selector: str, timeout: int = 3000) -> str:
#         """Safely get input value from an element"""
#         try:
#             element = self.page.wait_for_selector(selector, timeout=timeout)
#             return element.input_value() if element else ""
#         except (PlaywrightTimeoutError, Exception):
#             return ""
    
#     def wait_for_stable_page(self, timeout: int = 10000) -> None:
#         """Wait for page to be stable (no network activity)"""
#         try:
#             self.page.wait_for_load_state('networkidle', timeout=timeout)
#         except PlaywrightTimeoutError:
#             self.logger.warn("Page stability timeout - continuing anyway")
    
#     def close(self) -> None:
#         """Close browser and cleanup resources"""
#         try:
#             if self.browser:
#                 self.browser.close()
#                 self.browser = None
#             if self.context:
#                 self.context.close()
#                 self.context = None
#             if self.playwright:
#                 self.playwright.stop()
#                 self.playwright = None
#             self.log_step("Browser closed")
#         except Exception as e:
#             self.logger.error(f"Error during cleanup: {e}")
    
#     @abstractmethod
#     def execute_main_task(self) -> Any:
#         """Execute the main task - must be implemented by subclasses"""
#         pass
    
#     def run(self) -> ScriptResponse:
#         """Main execution method with error handling"""
#         try:
#             with self:
#                 self.navigate_to_homepage()
#                 self.login()
#                 self.enter_demo_mode()
                
#                 result = self.execute_main_task()
                
#                 if not self.headless:
#                     time.sleep(5)  # Keep browser open for review
                
#                 return ScriptResponse(success=True, data=result)
                
#         except KeyInvoiceError as e:
#             self.logger.error(f"KeyInvoice operation failed: {e}")
#             return ScriptResponse(success=False, error=str(e))
#         except Exception as e:
#             self.logger.error(f"Unexpected error: {e}")
#             return ScriptResponse(success=False, error=f"Unexpected error: {e}")

# # Utility functions
# def clean_text(text: str) -> str:
#     """Clean and normalize text"""
#     if not text:
#         return ""
#     return ' '.join(text.strip().split())

# def safe_float(value: Any, default: float = 0.0) -> float:
#     """Safely convert value to float"""
#     try:
#         if isinstance(value, (int, float)):
#             return float(value)
#         if isinstance(value, str):
#             cleaned = value.replace(',', '').strip()
#             return float(cleaned) if cleaned else default
#         return default
#     except (ValueError, TypeError):
#         return default

# def safe_int(value: Any, default: int = 0) -> int:
#     """Safely convert value to int"""
#     try:
#         if isinstance(value, int):
#             return value
#         if isinstance(value, (float, str)):
#             return int(float(str(value).replace(',', '').strip()))
#         return default
#     except (ValueError, TypeError):
#         return default

# @contextmanager
# def timeout_handler(seconds: int):
#     """Context manager for script timeout"""
#     timeout_occurred = threading.Event()
    
#     def timeout_callback():
#         timeout_occurred.set()
    
#     timer = threading.Timer(seconds, timeout_callback)
#     timer.start()
    
#     try:
#         yield timeout_occurred
#     finally:
#         timer.cancel()

# def run_with_timeout(func, timeout_seconds: int = Config.MAX_SCRIPT_TIMEOUT):
#     """Run a function with timeout protection"""
#     result_container = {}
#     exception_container = {}
    
#     def worker():
#         try:
#             result_container['result'] = func()
#         except Exception as e:
#             exception_container['exception'] = e
    
#     thread = threading.Thread(target=worker)
#     thread.start()
#     thread.join(timeout_seconds)
    
#     if thread.is_alive():
#         # Thread is still running - timeout occurred
#         raise ScriptTimeoutError(f"Operation timed out after {timeout_seconds} seconds")
    
#     if 'exception' in exception_container:
#         raise exception_container['exception']
    
#     return result_container.get('result')

# def parse_date_safely(date_str: str) -> str:
#     """Parse date string safely to ISO format"""
#     try:
#         if not date_str:
#             return ""
#         # Try common date formats
#         for fmt in ["%Y-%m-%d", "%d/%m/%Y", "%d-%m-%Y"]:
#             try:
#                 parsed = datetime.strptime(date_str.strip(), fmt)
#                 return parsed.strftime("%Y-%m-%dT00:00:00Z")
#             except ValueError:
#                 continue
#         # If no format matches, return as is
#         return date_str
#     except Exception:
#         return date_str

# def encode_pdf_content(pdf_bytes: bytes) -> str:
#     """Encode PDF content to base64"""
#     if not pdf_bytes:
#         return ""
#     return base64.b64encode(pdf_bytes).decode('utf-8')

# def create_temp_file(content: bytes, suffix: str = '.tmp') -> str:
#     """Create a temporary file with content"""
#     with tempfile.NamedTemporaryFile(delete=False, suffix=suffix) as temp_file:
#         temp_file.write(content)
#         return temp_file.name