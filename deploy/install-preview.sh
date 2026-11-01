#!/bin/sh
# Run as root: sh deploy/install-preview.sh /opt/mirrorouc/releases/<release>
set -eu
release=$(readlink -f "$1")
case "$release" in /opt/mirrorouc/releases/*) ;; *) echo 'Invalid release path' >&2; exit 1;; esac
test -x "$release/mirrorouc"
test -f "$release/dist/index.html"
deploy="$release/deploy"
id mirrorouc >/dev/null 2>&1 || useradd --system --home-dir /var/lib/mirrorouc --shell /usr/sbin/nologin mirrorouc
install -d -m 0700 -o mirrorouc -g mirrorouc /var/lib/mirrorouc
install -d -m 0755 /etc/mirrorouc /usr/local/lib/mirrorouc
install -d -m 0700 /var/lib/mirrorouc-config
install -m 0755 "$deploy/config-helper.py" /usr/local/lib/mirrorouc/config-helper.py
install -m 0644 "$deploy/mirrorouc.service" "$deploy/mirrorouc-config.service" /etc/systemd/system/
if [ ! -f /etc/mirrorouc/mirrorouc.env ]; then
    umask 077
    cat > /etc/mirrorouc/mirrorouc.env <<'EOF'
MIRROROUC_LISTEN=127.0.0.1:8081
MIRROROUC_DATA=/var/lib/mirrorouc
MIRROROUC_STATIC=/opt/mirrorouc/current/dist
MIRROROUC_MANAGER=http://127.0.0.1:12345
MIRROROUC_WORKER=srv-software-mirror
MIRROROUC_WORKER_CONFIG=/etc/tunasync/worker.conf
MIRROROUC_LOGS=/var/log/tunasync
MIRROROUC_METRICS=/var/lib/mirrorouc/metrics/history.json
MIRROROUC_CONFIG_SOCKET=/run/mirrorouc-config/control.sock
MIRROROUC_SECURE_COOKIES=false
EOF
fi
# Leave current tunasync configurations, ownership and permissions intact.
if [ ! -s /var/lib/mirrorouc/mirrorouc.db ]; then
    umask 077
    runuser -u mirrorouc -- "$release/mirrorouc" -data /var/lib/mirrorouc -init-admin admin > /etc/mirrorouc/initial-admin-password
fi
ln -sfn "$release" /opt/mirrorouc/current.new
mv -Tf /opt/mirrorouc/current.new /opt/mirrorouc/current
install -m 0644 "$deploy/nginx-preview.conf" /etc/nginx/sites-available/mirrorouc-preview
ln -sfn /etc/nginx/sites-available/mirrorouc-preview /etc/nginx/sites-enabled/mirrorouc-preview
nginx -t
systemctl daemon-reload
systemctl enable --now mirrorouc-config.service
systemctl restart mirrorouc-config.service
systemctl enable mirrorouc.service
systemctl restart mirrorouc.service
systemctl reload nginx
echo 'MirrorOUC preview installed at http://100.69.2.2:8080'
