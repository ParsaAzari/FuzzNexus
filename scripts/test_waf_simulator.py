#!/usr/bin/env python3
"""
FuzzNexus Anti-Ban & WAF Simulator (TestLab)
============================================
A lightweight, zero-dependency mock HTTP server to safely test and validate:
1. CircuitBreaker state transitions (Closed -> Open)
2. Safe Stop modal trigger (IP Block popup on 403 Forbidden)
3. Dynamic Rate Limiting & 429 backoff
4. Connection testing and scanning resumption after unban
"""

import sys
import time
import threading
from http.server import ThreadingHTTPServer, BaseHTTPRequestHandler

PORT = 8888
BAN_AFTER_REQUESTS = 10
BLOCK_STATUS_CODE = 403

request_counter = 0
is_banned = False
counter_lock = threading.Lock()

class WAFSimulatorHandler(BaseHTTPRequestHandler):
    def log_message(self, format, *args):
        # Suppress default server log to show custom colored output
        pass

    def do_GET(self):
        global request_counter, is_banned

        # Control endpoints for easy automated or script testing
        if self.path == "/unban":
            with counter_lock:
                is_banned = False
                request_counter = 0
            self.send_response(200)
            self.send_header("Content-Type", "text/plain")
            self.end_headers()
            self.wfile.write(b"Simulator unbanned (200 OK mode active)")
            return

        if self.path == "/ban":
            with counter_lock:
                is_banned = True
            self.send_response(200)
            self.send_header("Content-Type", "text/plain")
            self.end_headers()
            self.wfile.write(b"Simulator banned (403 Forbidden mode active)")
            return

        # Health probe check on root path when in banned state
        if is_banned and (self.path == "/" or self.path == ""):
            print(f"\033[93m[PROBE]\033[0m Target health probe received during IP block -> Returning 403 Forbidden")
            self.send_response(403)
            self.send_header("Content-Type", "text/plain")
            self.end_headers()
            self.wfile.write(b"WAF Forbidden: IP Blocked")
            return

        with counter_lock:
            request_counter += 1
            current_num = request_counter

            if current_num > BAN_AFTER_REQUESTS and not is_banned:
                is_banned = True
                print("\n" + "=" * 65)
                print(f"\033[91m[WAF ALERT] Threshold ({BAN_AFTER_REQUESTS} requests) reached!\033[0m")
                print("\033[91m[WAF ALERT] Now returning HTTP 403 Forbidden to trigger Safe Stop!\033[0m")
                print("\033[96m[TEST TIP]  Check FuzzNexus UI: The Safe Stop popup should now appear.\033[0m")
                print("\033[92m[ACTION]    Press [ENTER] in this terminal anytime to simulate VPN/Unban!\033[0m")
                print("=" * 65 + "\n")

        if is_banned:
            print(f"\033[91m[BLOCKED #{current_num}]\033[0m {self.command} {self.path} -> \033[91m403 Forbidden\033[0m")
            self.send_response(403)
            self.send_header("Content-Type", "text/html")
            self.end_headers()
            self.wfile.write(b"<html><body><h1>403 Forbidden</h1><p>WAF IP Block Simulator</p></body></html>")
        else:
            print(f"\033[92m[ALLOW #{current_num}]\033[0m {self.command} {self.path} -> \033[92m200 OK\033[0m")
            self.send_response(200)
            self.send_header("Content-Type", "text/plain")
            self.end_headers()
            self.wfile.write(b"Endpoint content OK")

    def do_HEAD(self):
        self.do_GET()

    def do_POST(self):
        self.do_GET()

def key_listener():
    global is_banned, request_counter
    while True:
        try:
            line = sys.stdin.readline()
            if not line:
                break
            with counter_lock:
                if is_banned:
                    is_banned = False
                    request_counter = 0
                    print("\n" + "*" * 65)
                    print("\033[92m[UNBAN SIMULATED] IP Unblocked! Server will now return 200 OK.\033[0m")
                    print("\033[96m[TEST TIP] In FuzzNexus Safe Stop modal, click 'Test Connection' then 'Resume'.\033[0m")
                    print("*" * 65 + "\n")
                else:
                    is_banned = True
                    print("\n" + "*" * 65)
                    print("\033[91m[MANUAL BAN] Server forced into 403 Forbidden state!\033[0m")
                    print("*" * 65 + "\n")
        except Exception:
            break

def main():
    server_address = ("127.0.0.1", PORT)
    httpd = ThreadingHTTPServer(server_address, WAFSimulatorHandler)

    print("\n" + "=" * 65)
    print("   \033[1;34mFuzzNexus WAF & Safe-Stop Validation Server\033[0m")
    print("=" * 65)
    print(f"Server running at: \033[1;32mhttp://127.0.0.1:{PORT}/\033[0m")
    print(f"Target URL for FuzzNexus: \033[1;33mhttp://127.0.0.1:{PORT}/FUZZ\033[0m")
    print(f"Mode: Normal for first {BAN_AFTER_REQUESTS} requests, then simulates WAF 403 Block.")
    print("=" * 65)
    print("Starting server... Press Ctrl+C to stop.\n")

    listener = threading.Thread(target=key_listener, daemon=True)
    listener.start()

    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("\nStopping WAF Simulator server...")
        httpd.server_close()
        sys.exit(0)

if __name__ == "__main__":
    main()
