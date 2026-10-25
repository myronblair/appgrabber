import json, re, sys
apps, seen, cats = [], set(), []
for line in open('tools/catalog.txt', encoding='utf-8'):
    line = line.strip()
    if not line or line.startswith('#'):
        continue
    cat, name, wid, desc = line.split('|')
    if wid in seen:
        continue
    seen.add(wid)
    if cat not in cats:
        cats.append(cat)
    slug = re.sub(r'[^a-z0-9]+', '-', name.lower()).strip('-')
    apps.append({'id': slug, 'name': name, 'cat': cat, 'desc': desc, 'winget': wid})
ids = {a['id'] for a in apps}
bundles = [
 {'id': 'new-workstation', 'name': 'New Workstation', 'desc': 'Everyday office PC baseline',
  'apps': ['chrome', 'firefox', 'adobe-reader', '7-zip', 'vlc', 'zoom', 'microsoft-teams', 'notepad', 'libreoffice', 'vc-redistributable-2015-2022-x64', 'vc-redistributable-2015-2022-x86', 'java-temurin-21-jre', 'net-desktop-runtime-8', 'webview2-runtime']},
 {'id': 'new-server', 'name': 'New Server', 'desc': 'Admin essentials for Windows Server',
  'apps': ['7-zip', 'notepad', 'powershell-7', 'windows-terminal', 'sysinternals-suite', 'putty', 'winscp', 'wireshark', 'windows-admin-center', 'vc-redistributable-2015-2022-x64', 'net-runtime-8', 'webview2-runtime', 'chrome', 'treesize-free']},
 {'id': 'tech-toolkit', 'name': 'MSP Tech Toolkit', 'desc': 'Everything a tech needs on a USB-less job',
  'apps': ['anydesk', 'teamviewer', 'putty', 'winscp', 'advanced-ip-scanner', 'angry-ip-scanner', 'wireshark', 'nmap', 'sysinternals-suite', 'crystaldiskinfo', 'crystaldiskmark', 'hwinfo', 'cpu-z', 'rufus', 'ventoy', 'balena-etcher', 'everything', 'windirstat', '7-zip', 'notepad', 'bitwarden', 'tailscale']},
 {'id': 'developer', 'name': 'Developer', 'desc': 'Coding workstation',
  'apps': ['git', 'github-desktop', 'visual-studio-code', 'windows-terminal', 'powershell-7', 'python-3-12', 'node-js-lts', 'postman', 'docker-desktop', 'winmerge', 'windows-subsystem-for-linux', 'putty', 'notepad']},
 {'id': 'home', 'name': 'Home PC', 'desc': 'Browsing, media and chat',
  'apps': ['chrome', 'firefox', 'vlc', 'spotify', 'discord', 'zoom', 'adobe-reader', '7-zip', 'libreoffice', 'steam', 'malwarebytes', 'bitwarden']},
]
for b in bundles:
    b['apps'] = [a for a in b['apps'] if a in ids]
    missing = [a for a in b['apps'] if a not in ids]
json.dump({'categories': cats, 'apps': apps, 'bundles': bundles}, open('public/catalog.json', 'w', encoding='utf-8'), indent=1)
print(len(apps), 'apps', len(cats), 'categories')
for b in bundles:
    print(b['id'], len(b['apps']))
