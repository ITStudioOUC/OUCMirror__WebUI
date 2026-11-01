#!/bin/sh
set -eu
cd "$(dirname "$0")"
if ! id mirrorouc >/dev/null 2>&1; then
    useradd --system --home-dir /var/lib/mirrorouc --shell /usr/sbin/nologin mirrorouc
fi
install -d -o mirrorouc -g mirrorouc -m 0700 /var/lib/mirrorouc
install -d -o root -g mirrorouc -m 0750 /var/lib/mirrorouc/metrics
install -d -o root -g root -m 0755 /usr/local/lib/mirrorouc
install -d -o root -g root -m 0755 /etc/mirrorouc
install -m 0644 collect_metrics.py /usr/local/lib/mirrorouc/collect_metrics.py
install -m 0644 collect_job_speeds.py /usr/local/lib/mirrorouc/collect_job_speeds.py
install -m 0755 ensure-counters.sh /usr/local/lib/mirrorouc/ensure-counters.sh
install -m 0644 protocol-counters.nft /etc/mirrorouc/protocol-counters.nft
install -m 0644 mirrorouc-counters.service mirrorouc-metrics.service mirrorouc-metrics.timer mirrorouc-speeds.service /etc/systemd/system/
if ! /usr/sbin/nft list table inet mirrorouc_monitor >/dev/null 2>&1; then
    /usr/sbin/nft -c -f /etc/mirrorouc/protocol-counters.nft
fi
systemctl daemon-reload
systemctl enable --now mirrorouc-counters.service mirrorouc-metrics.timer
systemctl start mirrorouc-metrics.service
systemctl enable --now mirrorouc-speeds.service
systemctl restart mirrorouc-speeds.service
