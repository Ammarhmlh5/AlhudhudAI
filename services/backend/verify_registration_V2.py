import requests
import json
import time

url = "http://localhost:8081/api/v1/tenants/register"
timestamp = int(time.time())
payload = {
    "email": f"testuser_{timestamp}@example.com",
    "phone": f"55{timestamp}",
    "country_code": "+966",
    "password": "Password123!",
    "confirm_password": "Password123!"
}

try:
    print(f"Sending request to {url} with payload: {payload}")
    response = requests.post(url, json=payload, timeout=5)
    print(f"Status Code: {response.status_code}")
    print("Response Body:")
    try:
        print(json.dumps(response.json(), indent=2))
    except:
        print(response.text)
        
    if response.status_code == 201:
        print("SUCCESS: Registration completed.")
    else:
        print("FAILURE: Registration failed.")
except Exception as e:
    print(f"ERROR: {e}")
