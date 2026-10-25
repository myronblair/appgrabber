#!/usr/bin/env bash
# Run on the Ubuntu VM as root (sudo bash setup.sh) from the unpacked appgrabber folder.
set -euo pipefail
cd "$(dirname "$0")/.."
SRC=$(pwd)

apt-get update -qq
apt-get install -y golang-go python3 ufw >/dev/null

id appgrabber >/dev/null 2>&1 || useradd --system --home /opt/appgrabber --shell /usr/sbin/nologin appgrabber
mkdir -p /opt/appgrabber/dist
python3 tools/gen_catalog.py
cp -r public /opt/appgrabber/

# real Windows .exe (cross-compiled) and the Linux server
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -ldflags "-s -w" -o /opt/appgrabber/dist/appgrabber-stub.exe ./stub
CGO_ENABLED=0 go build -buildvcs=false -ldflags "-s -w" -o /opt/appgrabber/appgrabber-server ./server

chown -R root:root /opt/appgrabber
install -m 644 deploy/appgrabber.service /etc/systemd/system/appgrabber.service
systemctl daemon-reload
systemctl enable --now appgrabber
sleep 1
curl -fsS http://127.0.0.1:8090/api/health && echo " <- appgrabber is up on :8090"
