#!/usr/bin/env bash
# Run once as root: (1) weekly winget-id check, (2) nightly git-pull + rebuild, (3) OS security auto-updates.
set -euo pipefail
SRC=$(cd "$(dirname "$0")/.." && pwd)
chmod +x "$SRC"/deploy/*.sh

# (3) automatic Ubuntu security updates
apt-get install -y unattended-upgrades >/dev/null
cat > /etc/apt/apt.conf.d/20auto-upgrades <<CONF
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
APT::Periodic::AutocleanInterval "7";
CONF

# (1) + (2) systemd timers
cat > /etc/systemd/system/appgrabber-update.service <<UNIT
[Unit]
Description=AppGrabber pull and rebuild
[Service]
Type=oneshot
ExecStart=$SRC/deploy/update.sh
UNIT
cat > /etc/systemd/system/appgrabber-update.timer <<UNIT
[Unit]
Description=AppGrabber nightly update
[Timer]
OnCalendar=*-*-* 03:30:00
RandomizedDelaySec=600
Persistent=true
[Install]
WantedBy=timers.target
UNIT
cat > /etc/systemd/system/appgrabber-idcheck.service <<UNIT
[Unit]
Description=AppGrabber winget id check
[Service]
Type=oneshot
ExecStart=$SRC/deploy/idcheck.sh
UNIT
cat > /etc/systemd/system/appgrabber-idcheck.timer <<UNIT
[Unit]
Description=AppGrabber weekly winget id check
[Timer]
OnCalendar=Sun *-*-* 04:00:00
Persistent=true
[Install]
WantedBy=timers.target
UNIT
systemctl daemon-reload
systemctl enable --now appgrabber-update.timer appgrabber-idcheck.timer
"$SRC/deploy/update.sh" -f
"$SRC/deploy/idcheck.sh"
systemctl list-timers 'appgrabber-*' --no-pager
