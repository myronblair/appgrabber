#!/usr/bin/env python3
"""Publish appgrabber.tomtomenterprises.com: Cloudflare DNS record + NPM access list,
proxy host and Let's Encrypt cert. You run it and type the secrets when prompted
(nothing is stored or printed). Safe to re-run: existing pieces are reused.

  python3 tools/publish.py
"""
import getpass, json, ssl, sys, urllib.request, urllib.error

HOST = "appgrabber.tomtomenterprises.com"
BACKEND = ("10.48.200.5", 8090)
LAN = "10.48.200.0/24"
CF_ZONE = "029dadea1c40ffa04f71614649f77954"      # tomtomenterprises.com
CF_COPY_IP_FROM = "jarvis.tomtomenterprises.com"  # reuse the public IP the other hosts use
NPM = "http://10.48.200.111:81"
ACL_NAME = "Gated + LAN open"

def call(method, url, token=None, body=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", "Bearer " + token)
    try:
        with urllib.request.urlopen(req, timeout=120, context=ssl.create_default_context()) as r:
            return json.load(r)
    except urllib.error.HTTPError as e:
        sys.exit(f"{method} {url} -> HTTP {e.code}: {e.read().decode()[:400]}")

# ---- Cloudflare -------------------------------------------------------------
cf_token = getpass.getpass("Cloudflare API token (DNS edit): ")
cf = lambda m, p, b=None: call(m, "https://api.cloudflare.com/client/v4" + p, cf_token, b)
recs = cf("GET", f"/zones/{CF_ZONE}/dns_records?name={CF_COPY_IP_FROM}&type=A")["result"]
if not recs:
    sys.exit("could not find the jarvis A record to copy the public IP from")
ip = recs[0]["content"]
existing = cf("GET", f"/zones/{CF_ZONE}/dns_records?name={HOST}")["result"]
if existing:
    print("DNS: record already exists ->", existing[0]["content"])
else:
    cf("POST", f"/zones/{CF_ZONE}/dns_records",
       {"type": "A", "name": HOST, "content": ip, "ttl": 1, "proxied": False})
    print("DNS: created", HOST, "->", ip, "(DNS only)")

# ---- NPM --------------------------------------------------------------------
email = input("NPM admin email: ").strip()
npm_pw = getpass.getpass("NPM admin password: ")
tok = call("POST", NPM + "/api/tokens", None, {"identity": email, "secret": npm_pw})["token"]
npm = lambda m, p, b=None: call(m, NPM + "/api" + p, tok, b)

acls = npm("GET", "/nginx/access-lists")
acl = next((a for a in acls if a["name"] == ACL_NAME), None)
if acl:
    print("NPM: access list exists, reusing")
else:
    gu = input("Username for the outside-access password: ").strip()
    gp = getpass.getpass("Password for that user: ")
    acl = npm("POST", "/nginx/access-lists", {
        "name": ACL_NAME, "satisfy_any": True, "pass_auth": False,
        "items": [{"username": gu, "password": gp}],
        "clients": [{"address": LAN, "directive": "allow"}]})
    print("NPM: created access list")

hosts = npm("GET", "/nginx/proxy-hosts")
ph = next((h for h in hosts if HOST in h["domain_names"]), None)
base = {"domain_names": [HOST], "forward_scheme": "http", "forward_host": BACKEND[0],
        "forward_port": BACKEND[1], "access_list_id": acl["id"], "block_exploits": True,
        "caching_enabled": False, "allow_websocket_upgrade": False, "advanced_config": "",
        "locations": [], "enabled": True}
if ph:
    print("NPM: proxy host exists, updating")
    ph = npm("PUT", f"/nginx/proxy-hosts/{ph['id']}", base)
else:
    ph = npm("POST", "/nginx/proxy-hosts", {**base, "meta": {"letsencrypt_agree": False, "dns_challenge": False}})
    print("NPM: created proxy host")

if not ph.get("certificate_id"):
    le_email = input("Email for Let's Encrypt notices [%s]: " % email).strip() or email
    print("NPM: requesting certificate (DNS must already point here; can take ~30s)...")
    cert = npm("POST", "/nginx/certificates", {
        "provider": "letsencrypt", "domain_names": [HOST],
        "meta": {"letsencrypt_email": le_email, "letsencrypt_agree": True}})
    npm("PUT", f"/nginx/proxy-hosts/{ph['id']}", {
        **base, "certificate_id": cert["id"], "ssl_forced": True, "http2_support": True, "hsts_enabled": False})
    print("NPM: certificate attached, Force SSL on")
print("Done: https://" + HOST)
