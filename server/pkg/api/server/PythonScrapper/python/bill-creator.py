import sys
import json
import time
from datetime import datetime
import random

def process_orders(orders):
    """
    Dummy logic to process orders and create bills
    """
    # Debug: print what we received
    print(f"DEBUG: Received {len(orders)} orders", file=sys.stderr)
    for i, order in enumerate(orders):
        print(f"DEBUG: Order {i}: {json.dumps(order, indent=2)}", file=sys.stderr)
    
    bills = []
    
    for order in orders:
        # Debug: Print what we're extracting from each order
        print(f"DEBUG: Processing order - IdClient: {order.get('IdClient')}, Address: {order.get('Address')}", file=sys.stderr)
        print(f"DEBUG: Products count: {len(order.get('Products', []))}", file=sys.stderr)
        
        # Simulate some processing time
        time.sleep(0.1)
        
        # Calculate totals for the order
        total_amount = 0
        processed_products = []
        
        for product in order.get('Products', []):
            # Access Go struct fields with correct casing
            price = float(product.get('Price', 0)) if product.get('Price') else random.uniform(10.0, 100.0)
            quantity = int(product.get('Quantity', 1)) if product.get('Quantity') else 1
            product_total = price * quantity
            total_amount += product_total
            
            processed_products.append({
                'id': product.get('IdProduct', 'N/A'),
                'name': product.get('Details', 'Unknown Product'),
                'price': round(price, 2),
                'quantity': quantity,
                'total': round(product_total, 2),
                'discount': product.get('Discount', '0'),
                'tax': product.get('Tax', '0')
            })
        
        # Generate a dummy bill (match Go struct field names)
        bill = {
            'bill_id': f"BILL-{random.randint(1000, 9999)}-{order.get('IdClient', 'UNKNOWN')}",
            'client_id': order.get('IdClient', 'UNKNOWN'),
            'client_address': order.get('Address', 'No address provided'),
            'products': processed_products,
            'subtotal': round(total_amount, 2),
            'tax': round(total_amount * 0.15, 2),  # 15% tax
            'total_amount': round(total_amount * 1.15, 2),
            'created_at': datetime.now().isoformat(),
            'status': 'pending',
            'payment_method': random.choice(['credit_card', 'debit_card', 'cash', 'bank_transfer'])
        }
        
        bills.append(bill)
    
    # Return summary and bills
    result = {
        'summary': {
            'total_orders_processed': len(orders),
            'total_bills_created': len(bills),
            'total_revenue': round(sum(bill['total_amount'] for bill in bills), 2),
            'processing_time': f"{len(orders) * 0.1:.1f} seconds"
        },
        'bills': bills
    }
    
    return result

def main():
    try:
        # Check if we have the required argument
        if len(sys.argv) < 2:
            raise ValueError("No orders data provided as argument")
        
        orders_json = sys.argv[1]
        
        # Parse orders data
        orders = json.loads(orders_json)
        
        # Validate that we have a list
        if not isinstance(orders, list):
            raise ValueError("Orders data must be a list")
        
        # Log to stderr for debugging (will show up in Go logs)
        print(f"Processing {len(orders)} orders...", file=sys.stderr)
        
        # Process orders
        result = process_orders(orders)
        
        # Log processing completion
        print(f"Successfully created {result['summary']['total_bills_created']} bills", file=sys.stderr)
        
        # Return success response to stdout
        response = {
            "success": True,
            "data": result,
            "message": f"Successfully processed {len(orders)} orders and created {len(result['bills'])} bills"
        }
        
        print(json.dumps(response, indent=2))
        
    except json.JSONDecodeError as e:
        error_response = {
            "success": False,
            "error": f"Invalid JSON data: {str(e)}"
        }
        print(json.dumps(error_response))
        
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