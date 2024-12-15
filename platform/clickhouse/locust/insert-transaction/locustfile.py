from locust import HttpUser, task, between
import random
import uuid
from datetime import datetime
import time


class LoadTestTrx(HttpUser):
    wait_time = between(1, 5)  # Simulate user wait time between 1 to 5 seconds

 
    @task
    def send_post_request(self):
        # Generate random data for the request
        transaction = {
            "id": str(uuid.uuid4()),
            "amount": round(random.uniform(1, 1000), 2),
            "currency": random.choice(["USD", "EUR", "GBP"]),
            "status": random.choice(["pending", "completed", "failed"]),
            "user_id": f"user_{random.randint(1, 100)}",
            "merchant_id": f"merchant_{random.randint(1, 20)}",
            "payment_method": random.choice(["credit_card", "debit_card", "paypal"])
        }
        self.client.post("/transactions", json=transaction)

        # Sleep for 1 second before sending the request
        time.sleep(1)

        
