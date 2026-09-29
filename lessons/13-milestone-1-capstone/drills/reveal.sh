#!/usr/bin/env bash
# เฉลย drill ล่าสุด — เปิดดูหลังจากเขียนต้นเหตุของตัวเองแล้วเท่านั้น
set -uo pipefail
[[ $EUID -eq 0 ]] || { echo "รันด้วย sudo"; exit 1; }
[[ -f /root/.pixbin-drill ]] || { echo "ยังไม่มี drill"; exit 1; }
read -r N WHEN < /root/.pixbin-drill

case "$N" in
  1) A="pixbin ถูก stop
หลักฐาน: nginx error log 'connect() failed (111: Connection refused)' · systemctl status pixbin = inactive" ;;
  2) A="pixbin ถูกตั้ง PORT=8081 ผ่าน EnvironmentFile ใน drop-in (zz-drill.conf) — app รันอยู่ แต่ไม่ได้ฟังที่ 8000 ที่ nginx ส่งไป
หลักฐาน: 502 + 'Connection refused' · ss -tlnp เห็น pixbin ที่ 8081 · systemctl cat pixbin เห็น drop-in" ;;
  3) A="/var/lib/pixbin ถูก chmod 550 — สร้างไฟล์ใหม่ไม่ได้ แต่ .healthz เดิมยังเขียนทับได้ health check จึงผ่าน (Lesson 4 ข้อ 5)
หลักฐาน: upload ได้ 500 · journal 'permission denied' · ls -ld /var/lib/pixbin" ;;
  4) A="ufw มีกฎ deny 443/tcp อยู่บนสุด — packet ถูก DROP
หลักฐาน: https timeout (ไม่ใช่ refused) · http ยังได้ 301 · ufw status numbered · ในเครื่องเอง curl ได้ปกติ" ;;
  5) A="nginx ถูก stop — ไม่มีใครฟัง 80/443
หลักฐาน: connection refused ทันที · ss -tlnp ไม่มี :80 :443 · pixbin ที่ 127.0.0.1:8000 ยังปกติ" ;;
  6) A="pixbin ถูก SIGSTOP (แช่แข็ง) — process ยังอยู่ systemd ยังบอก active แต่ไม่ตอบ
หลักฐาน: 504 หลัง 5 วินาที · error log 'upstream timed out' · ps state = T · curl 127.0.0.1:8000 ค้าง" ;;
  7) A="certificate หมดอายุ (ออกด้วยวันที่ปี 2020)
หลักฐาน: curl (60) certificate has expired · openssl s_client ... | openssl x509 -noout -dates" ;;
  8) A="/var/lib/pixbin ถูก mount เป็น tmpfs ขนาด 64 KB — disk เต็มเมื่อไฟล์ใหญ่
หลักฐาน: upload ใหญ่ได้ 500 · journal 'no space left on device' · df -h /var/lib/pixbin · findmnt /var/lib/pixbin" ;;
esac
echo "Drill #$N (เริ่ม $WHEN)"
echo "$A"
