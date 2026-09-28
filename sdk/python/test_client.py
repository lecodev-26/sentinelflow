import json
from http.server import BaseHTTPRequestHandler, HTTPServer
from threading import Thread
from sentinelflow import Client

class H(BaseHTTPRequestHandler):
    def do_POST(self):
        self.send_response(200); self.send_header("Content-Type","application/json"); self.end_headers()
        self.wfile.write(json.dumps({"id":"r1","object":"chat.completion","choices":[{"message":{"role":"assistant","content":"ok"}}]}).encode())
    def log_message(self,*a): pass

s=HTTPServer(("127.0.0.1",0),H); Thread(target=s.serve_forever,daemon=True).start()
c=Client(f"http://127.0.0.1:{s.server_port}","k")
assert c.chat("m",[{"role":"user","content":"hi"}])["id"]=="r1"
s.shutdown()
print("python sdk test: ok")
