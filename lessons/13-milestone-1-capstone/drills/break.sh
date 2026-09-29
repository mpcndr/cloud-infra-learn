#!/usr/bin/env bash
# สร้าง incident จำลองบน VM โดยไม่บอกว่าพังตรงไหน
#   sudo ./break.sh        สุ่ม 1 จาก 8 แบบ
#   sudo ./break.sh 3      เลือกแบบที่ 3
# เมื่อหาต้นเหตุได้แล้ว: sudo ./reveal.sh   คืนสภาพ: sudo ./fix.sh
set -uo pipefail
[[ $EUID -eq 0 ]] || { echo "รันด้วย sudo"; exit 1; }

HERE=$(cd "$(dirname "$0")" && pwd)
PKI=/home/ubuntu/pki
DROPIN_DIR=/etc/systemd/system/pixbin.service.d

"$HERE/fix.sh" >/dev/null 2>&1

N=${1:-$(( RANDOM % 8 + 1 ))}

case "$N" in
  1)
    systemctl stop pixbin
    REPORT="เว็บขึ้นหน้า error ทุกหน้า"
    ;;
  2)
    mkdir -p "$DROPIN_DIR"
    echo "PORT=8081" > /etc/pixbin/drill.env
    printf '[Service]\nEnvironmentFile=/etc/pixbin/drill.env\n' > "$DROPIN_DIR/zz-drill.conf"
    systemctl daemon-reload
    systemctl restart pixbin
    REPORT="เว็บขึ้นหน้า error ทุกหน้า แต่ทีม dev ยืนยันว่า app รันอยู่ ไม่มีใครแตะ"
    ;;
  3)
    chmod 550 /var/lib/pixbin
    REPORT="user upload รูปไม่ได้ แต่ health check บน dashboard เขียว"
    ;;
  4)
    ufw insert 1 deny 443/tcp >/dev/null
    REPORT="เว็บหมุนค้างนานแล้วเข้าไม่ได้"
    ;;
  5)
    systemctl stop nginx
    REPORT="เว็บเข้าไม่ได้เลย browser บอกทันทีว่าเชื่อมต่อไม่ได้"
    ;;
  6)
    kill -STOP "$(systemctl show -p MainPID --value pixbin)"
    REPORT="เว็บหมุนประมาณ 5 วินาทีแล้วขึ้น error"
    ;;
  7)
    if [[ ! -f $PKI/pixbin.csr || ! -f $PKI/ca.key || ! -f $PKI/san.ext ]]; then
      echo "ไม่พบไฟล์ใน $PKI — ต้องทำ Lesson 12 ก่อน"; exit 1
    fi
    command -v faketime >/dev/null || apt-get install -y -qq faketime >/dev/null
    tmp=$(mktemp)
    faketime '2020-01-01 00:00:00' openssl x509 -req -in "$PKI/pixbin.csr" -CA "$PKI/ca.crt" \
      -CAkey "$PKI/ca.key" -CAcreateserial -days 1 -extfile "$PKI/san.ext" -out "$tmp" 2>/dev/null
    install -m 644 "$tmp" /etc/pixbin/tls/pixbin.crt
    rm -f "$tmp"
    systemctl reload nginx
    REPORT="browser ขึ้นหน้าเตือนว่าการเชื่อมต่อไม่ปลอดภัย"
    ;;
  8)
    systemctl stop pixbin
    mount -t tmpfs -o size=64k,uid="$(id -u pixbin)",gid="$(id -g pixbin)",mode=0750 tmpfs /var/lib/pixbin
    systemctl start pixbin
    REPORT="upload รูปเล็กได้ แต่รูปใหญ่ไม่ได้"
    ;;
  *)
    echo "เลือกได้ 1–8"; exit 1
    ;;
esac

echo "$N $(date -Is)" > /root/.pixbin-drill
echo
echo "🚨 INCIDENT: $REPORT"
echo
echo "เริ่ม investigate ได้เลย จดทุกขั้น: สังเกต → hypothesis → วัด → ตัดทิ้ง → ต้นเหตุ"
echo "หาเจอแล้ว: sudo $HERE/reveal.sh   คืนสภาพ: sudo $HERE/fix.sh"
