import urllib.request
import urllib.error
import json
import time
import base64

# Configuration
BASE_URL = "http://localhost:8080"
AUTH_URL = f"{BASE_URL}/v1/auth/login"
EMAIL = "d@bytebase.com"
PASSWORD = "Hello123"
PROJECT = "projects/new-project"
INSTANCE = "instances/new-instance"
DATABASE = "d1"
DB_RESOURCE = f"{INSTANCE}/databases/{DATABASE}"

def make_request(url, method="GET", data=None, headers=None):
    if headers is None:
        headers = {}
    
    if data:
        data_json = json.dumps(data).encode("utf-8")
        headers["Content-Type"] = "application/json"
    else:
        data_json = None

    req = urllib.request.Request(url, data=data_json, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req) as response:
            if response.status != 200:
                print(f"Request failed: {response.reason}")
                return None
            return json.loads(response.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        print(f"HTTP Error {e.code}: {e.read().decode('utf-8')}")
        return None
    except Exception as e:
        print(f"Error: {e}")
        return None

def login():
    payload = {"email": EMAIL, "password": PASSWORD}
    response = make_request(AUTH_URL, method="POST", data=payload)
    if not response:
        raise Exception("Login failed")
    token = response["token"]
    return {"Authorization": f"Bearer {token}"}

def create_bulk_release(headers, file_count=500):
    files = []
    print(f"Generating {file_count} files...")
    for i in range(file_count):
        statement = f"CREATE TABLE IF NOT EXISTS demo_{i} (id INT);"
        files.append({
            "path": f"migration_{i}.sql",
            "statement": base64.b64encode(statement.encode('utf-8')).decode('utf-8'),
            "type": "VERSIONED",
            "version": f"1.0.{i}"
        })

    payload = {
        "title": f"Bulk Release urllib {time.time()}",
        "files": files
    }

    print(f"Creating release with {file_count} files...")
    start_time = time.time()
    response = make_request(f"{BASE_URL}/v1/{PROJECT}/releases", method="POST", headers=headers, data=payload)
    end_time = time.time()
    
    if not response:
        return None

    duration = end_time - start_time
    print(f"Release creation ({file_count} files) took: {duration:.4f} seconds")
    return response

def list_releases(headers):
    print("Listing releases...")
    start_time = time.time()
    response = make_request(f"{BASE_URL}/v1/{PROJECT}/releases", headers=headers)
    end_time = time.time()
    
    if not response:
        return
        
    duration = end_time - start_time
    print(f"List releases took: {duration:.4f} seconds")
    
    releases = response.get("releases", [])
    print(f"Found {len(releases)} releases")
    
    if releases and "files" in releases[0]:
        files = releases[0]["files"]
        if files:
            first_file = files[0]
            if "statement" in first_file:
                 print("WARNING: 'statement' field is PRESENT")
                 if first_file["statement"]:
                      print(f"  Value (truncated): {str(first_file['statement'])[:20]}...")
            else:
                 print("SUCCESS: 'statement' field is ABSENT")

            if "statementSize" in first_file:
                 print(f"WARNING: 'statementSize' field is PRESENT: {first_file['statementSize']}")
            else:
                 print("SUCCESS: 'statementSize' field is ABSENT")

if __name__ == "__main__":
    try:
        headers = login()
        release = create_bulk_release(headers, 500)
        if release:
            list_releases(headers)
    except Exception as e:
        print(f"Error: {e}")
