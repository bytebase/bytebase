import base64
import http.server
import json
from http.server import BaseHTTPRequestHandler

# We can use {{http://localhost:1137/data}} for testing external secret manager.

class JSONHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/data':
            password = 'ktPYr0bQixOHzCux'
            response_data = {
                'payload': {
                	'data': base64.b64encode(password.encode('utf-8')).decode('utf-8'),
                },
            }
            response_json = json.dumps(response_data)

            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(response_json.encode('utf-8'))
        else:
            self.send_response(404)
            self.wfile.write(b'Not Found')

server_address = ('', 1137)
httpd = http.server.HTTPServer(server_address, JSONHandler)

print("Server is running on port 1137...")
httpd.serve_forever()