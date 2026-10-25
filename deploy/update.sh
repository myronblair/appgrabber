#!/usr/bin/env bash
# Pull the latest catalog/code (if a git remote is set), rebuild, restart.
# Runs as root from the appgrabber-update timer; safe to run by hand.
set -euo pipefail
SRC=$(cd "$(dirname "$0")/.." && pwd)
cd "$SRC"

OWNER=$(stat -c %U "$SRC")
as_owner() { runuser -u "$OWNER" -- "$@"; }

changed=0
if as_owner git remote get-url origin >/dev/null 2>&1; then
  before=$(as_owner git rev-parse HEAD)
  as_owner git pull --ff-only --quiet
  [ "$before" != "$(as_owner git rev-parse HEAD)" ] && changed=1
else
  echo "no git remote set; rebuilding from local files only"
fi

# Rebuild when new commits arrived, or when forced with -f
if [ "$changed" = 1 ] || [ "${1:-}" = "-f" ]; then
  python3 tools/gen_catalog.py
  go_env=(env HOME=/root GOCACHE=/root/.cache/go-build CGO_ENABLED=0)
  "${go_env[@]}" GOOS=windows GOARCH=amd64 go build -buildvcs=false -ldflags "-s -w" -o /opt/appgrabber/dist/appgrabber-stub.exe.new ./stub
  "${go_env[@]}" go build -buildvcs=false -ldflags "-s -w" -o /opt/appgrabber/appgrabber-server.new ./server
  mv /opt/appgrabber/dist/appgrabber-stub.exe.new /opt/appgrabber/dist/appgrabber-stub.exe
  mv /opt/appgrabber/appgrabber-server.new /opt/appgrabber/appgrabber-server
  rm -rf /opt/appgrabber/public.new && cp -r public /opt/appgrabber/public.new
  rm -rf /opt/appgrabber/public.old; mv /opt/appgrabber/public /opt/appgrabber/public.old
  mv /opt/appgrabber/public.new /opt/appgrabber/public; rm -rf /opt/appgrabber/public.old
  systemctl restart appgrabber
  echo "rebuilt and restarted"
else
  echo "already up to date"
fi
