#!/usr/bin/env bash
# Weekly: verify every winget id in the catalog still exists. Report only.
SRC=$(cd "$(dirname "$0")/.." && pwd)
mkdir -p /var/lib/appgrabber
python3 "$SRC/tools/check_ids.py" /opt/appgrabber/public/catalog.json /var/lib/appgrabber/idcheck.json \
  | logger -t appgrabber-idcheck
cat /var/lib/appgrabber/idcheck.json | python3 -c 'import json,sys;d=json.load(sys.stdin);print("idcheck:",d.get("total"),"apps,",len(d["dead"]),"dead", d.get("error") or "")'
exit 0
