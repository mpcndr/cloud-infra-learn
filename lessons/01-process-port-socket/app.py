"""
Pixbin v0 — แอปที่เล็กที่สุดที่ยังเป็น "server" จริง
ไม่มี dependency นอกจาก Python standard library
"""
import os
import socket
from http.server import BaseHTTPRequestHandler, HTTPServer

HOST = os.environ.get("HOST", "127.0.0.1")
PORT = int(os.environ.get("PORT", "8000"))


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        client_ip, client_port = self.client_address
        body = (
            f"hello from Pixbin\n"
            f"server pid   : {os.getpid()}\n"
            f"server host  : {socket.gethostname()}\n"
            f"listening on : {HOST}:{PORT}\n"
            f"you came from: {client_ip}:{client_port}\n"
        ).encode()
        self.send_response(200)
        self.send_header("Content-Type", "text/plain; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


if __name__ == "__main__":
    print(f"[pid {os.getpid()}] listening on {HOST}:{PORT}", flush=True)
    HTTPServer((HOST, PORT), Handler).serve_forever()
