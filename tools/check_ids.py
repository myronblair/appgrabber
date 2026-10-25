#!/usr/bin/env python3
"""Check every winget id in public/catalog.json against the official winget community
source index and write a report. Report only: it never edits the catalog.

usage: check_ids.py [catalog.json] [report.json]
"""
import io, json, os, sqlite3, sys, tempfile, time, urllib.request, zipfile

SRC = "https://cdn.winget.microsoft.com/cache/source.msix"
catalog = sys.argv[1] if len(sys.argv) > 1 else "public/catalog.json"
report = sys.argv[2] if len(sys.argv) > 2 else "idcheck.json"

def fetch_index_ids():
    req = urllib.request.Request(SRC, headers={"User-Agent": "AppGrabber-idcheck"})
    data = urllib.request.urlopen(req, timeout=120).read()
    z = zipfile.ZipFile(io.BytesIO(data))
    name = next(n for n in z.namelist() if n.lower().endswith("index.db"))
    with tempfile.NamedTemporaryFile(suffix=".db", delete=False) as t:
        t.write(z.read(name))
        path = t.name
    try:
        db = sqlite3.connect(path)
        return {r[0].lower() for r in db.execute("SELECT id FROM ids")}
    finally:
        os.unlink(path)

out = {"checked": int(time.time()), "ok": False, "dead": [], "error": None}
try:
    known = fetch_index_ids()
    if len(known) < 1000:
        raise RuntimeError(f"index looks wrong ({len(known)} ids)")
    apps = json.load(open(catalog, encoding="utf-8"))["apps"]
    out["dead"] = [{"id": a["id"], "name": a["name"], "winget": a["winget"]}
                   for a in apps if a["winget"].lower() not in known]
    out["total"] = len(apps)
    out["ok"] = True
except Exception as e:
    out["error"] = f"{type(e).__name__}: {e}"

json.dump(out, open(report, "w"), indent=1)
if out["ok"]:
    print(f"{out['total']} apps checked, {len(out['dead'])} dead id(s)")
    for d in out["dead"]:
        print("  DEAD:", d["name"], "->", d["winget"])
else:
    print("check failed:", out["error"])
sys.exit(0 if out["ok"] and not out["dead"] else 1)
