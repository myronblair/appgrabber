# AppGrabber

Ninite-style installer builder for appgrabber.tomtomenterprises.com. Tick apps, get a small
compiled Windows `.exe` that installs them silently via winget (direct-download fallback).

- `public/` site (Blair HQ look) + `catalog.json` (generated from `tools/catalog.txt`)
- `server/` Go server: static files + `POST /api/build` (appends the app list to the stub exe)
- `stub/` Go Windows installer stub (cross-compiled, stdlib only)
- `deploy/setup.sh` one-shot VM install (Go, build, systemd on :8090)

Add an app: append a line to `tools/catalog.txt` (`category|name|winget id|description`),
then `python3 tools/gen_catalog.py` and restart the service.

## Access (Nginx Proxy Manager, VM111)
One proxy host, `appgrabber.tomtomenterprises.com` -> `http://<vm-ip>:8090`, SSL (Let's Encrypt, Force SSL).
Access List with **Satisfy Any**: allow `10.48.200.0/24`, plus a Basic Auth user.
LAN clients get in with no prompt; everyone else gets the password prompt.
Direct LAN URL (ungated): `http://<vm-ip>:8090`.

DNS: Cloudflare A record `appgrabber` -> the public IP NPM sits behind (same as the other
proxied hosts); optionally a FortiGate dns-database entry pointing it at 10.48.200.111 for LAN.

## Self-updating
- `deploy/extras.sh` (run once as root) installs: nightly `update.sh` (git pull, rebuild, restart),
  weekly `idcheck.sh` (report in `/var/lib/appgrabber/idcheck.json`), and Ubuntu unattended security updates.
- Set a git remote in this folder to enable the nightly pull:
  `git remote add origin <gitea-url>` then `git push -u origin master`. Edit `tools/catalog.txt`, push, done.
- Force a rebuild now: `sudo deploy/update.sh -f`.
