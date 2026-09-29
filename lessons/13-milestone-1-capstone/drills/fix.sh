#!/usr/bin/env bash
# คืนสภาพ Pixbin บน VM ให้กลับเป็นแบบที่ Lesson 12 สร้างไว้ (ใช้หลังทำ drill แต่ละรอบ)
set -uo pipefail
[[ $EUID -eq 0 ]] || { echo "รันด้วย sudo"; exit 1; }

PKI=/home/ubuntu/pki
DROPIN=/etc/systemd/system/pixbin.service.d/zz-drill.conf

systemctl stop pixbin 2>/dev/null
pkill -CONT -x pixbin 2>/dev/null

rm -f "$DROPIN" /etc/pixbin/drill.env
systemctl daemon-reload

if mountpoint -q /var/lib/pixbin; then
  umount /var/lib/pixbin
fi
chown pixbin:pixbin /var/lib/pixbin
chmod 750 /var/lib/pixbin

ufw delete deny 443/tcp >/dev/null 2>&1

if [[ -f $PKI/pixbin.crt ]]; then
  install -m 644 "$PKI/pixbin.crt" /etc/pixbin/tls/pixbin.crt
fi

systemctl start pixbin
systemctl start nginx
systemctl reload nginx

sleep 1
echo "pixbin : $(systemctl is-active pixbin)"
echo "nginx  : $(systemctl is-active nginx)"
echo "health : $(curl -s -m 3 http://127.0.0.1:8000/healthz || echo FAIL)"
