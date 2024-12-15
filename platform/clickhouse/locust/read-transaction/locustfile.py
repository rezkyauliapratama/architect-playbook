from locust import HttpUser, task, between
import random
import uuid
from datetime import datetime
import time


class LoadTestTrx(HttpUser):
    wait_time = between(1, 5)  # Simulate user wait time between 1 to 5 seconds

 
    @task(1)
    def get_mysql_totals(self):
        self.client.get("/totals/mysql")

    @task(1)
    def get_clickhouse_totals(self):
        self.client.get("/totals/clickhouse")

        
