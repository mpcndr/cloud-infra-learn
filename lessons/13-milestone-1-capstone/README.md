# Lesson 13 — Milestone 1 Capstone: request หนึ่งตัว ตั้งแต่พิมพ์ URL จนถึง disk

Phase 0–3 รวม · Level 3–10 · Project: Pixbin Stage 1 เสร็จสมบูรณ์

## 1. ปัญหาที่เรากำลังแก้

12 บทที่ผ่านมาเราเรียนทีละชิ้น แต่ระบบจริงไม่พังทีละชิ้นแบบมีป้ายบอก
user ส่งมาแค่ว่า "เว็บใช้ไม่ได้" แล้วคุณต้องหาเองว่ามันพังที่ชั้นไหนใน 10 กว่าชั้น

บทนี้ไม่มีเนื้อหาใหม่ มีแต่การ **ต่อทุกชิ้นเข้าด้วยกัน** แล้วพิสูจน์ว่าคุณ:
1. อธิบายเส้นทางของ request ได้ทุกขั้น และรู้ว่าจะพิสูจน์แต่ละขั้นด้วยอะไร
2. รับมือ incident ที่ไม่รู้ล่วงหน้าได้อย่างเป็นระบบ ไม่ใช่เดาสุ่ม
3. สอนคนอื่นได้

ผ่านบทนี้ = ผ่าน **Milestone 1**

## 2. Mental Model: การวิ่งผลัด

request หนึ่งตัวคือการวิ่งผลัด แต่ละไม้มีผู้รับผิดชอบคนละคน ส่งต่อกันเป็นทอด ๆ
ถ้าปลายทางไม่ได้ไม้ ต้องหาว่า **ไม้หลุดที่ช่วงไหน** — และแต่ละช่วงมีวิธีตรวจเฉพาะของมัน

```
ชื่อ → IP → เส้นทาง → ประตู (firewall) → ท่อ (TCP) → ซองปิดผนึก (TLS) → จดหมาย (HTTP)
   → พนักงานต้อนรับ (nginx) → ท่อใน (TCP loopback) → app (process) → ไฟล์ (disk, permission)
```
หลักของการ debug: **อย่าเริ่มจากช่วงที่คุ้นเคย เริ่มจากหลักฐาน** แล้วตัดช่วงที่ยืนยันแล้วว่าปกติทิ้งไปทีละช่วง

## 3. Concept — แผนที่ของระบบที่คุณสร้าง

| ชิ้น | อยู่ที่ไหน | เรียนใน | ตรวจด้วย |
|---|---|---|---|
| ชื่อ `pixbin.test` | `/etc/hosts` ของ Mac | L10 | `dscacheutil -q host -a name pixbin.test` |
| เส้นทาง Mac → VM | route ของ Mac (bridge100) | L7 | `route -n get <IP VM>` · `ping` |
| firewall | ufw ใน VM: 22, 80, 443 | L8 | `sudo ufw status numbered` |
| TCP | kernel ทั้งสองฝั่ง | L1, L9 | `curl -w %{time_connect}` · `ss -tan` |
| TLS | nginx :443, `/etc/pixbin/tls/` | L12 | `openssl s_client` |
| HTTP routing | nginx server blocks | L11 | `curl -v` · `/var/log/nginx/pixbin.*.log` |
| upstream | nginx → 127.0.0.1:8000 | L1, L11 | `curl 127.0.0.1:8000/healthz` บน VM |
| process | pixbin, user `pixbin`, systemd | L3–5 | `systemctl status pixbin` · `journalctl -u pixbin` |
| resources | CPU / RAM / disk / fd | L2, L6 | `uptime` `free -m` `df -h` `ls /proc/PID/fd` |
| storage | `/var/lib/pixbin` (750, pixbin) | L4, L6 | `ls -ld` `df -h` `namei -l` |

## 4. สิ่งที่เกิดขึ้นข้างใน — `curl https://pixbin.test/upload`

| # | ขั้น | อะไรเกิดขึ้น | ถ้าพังจะเห็น |
|---|---|---|---|
| 1 | DNS | Mac อ่าน `/etc/hosts` → `192.168.64.5` | `Could not resolve host` |
| 2 | Routing | ปลายทางอยู่วง 192.168.64.0/24 → ส่งออก bridge100 ตรง ๆ (ARP หา MAC ของ VM) | `No route to host` / timeout |
| 3 | Firewall | ufw: 443 allow → ผ่าน | DROP = timeout |
| 4 | TCP | kernel ของ VM: มี socket LISTEN ที่ :443 (nginx) → handshake | ไม่มีคนฟัง = refused |
| 5 | TLS | SNI = pixbin.test → nginx เลือก cert → Mac ตรวจ chain/ชื่อ/วันหมดอายุ | curl (60) / (35) |
| 6 | HTTP | nginx อ่าน request · Host = pixbin.test · ขนาด body ≤ 10m | 413, 444 |
| 7 | Proxy | nginx เปิด TCP ใหม่ไป 127.0.0.1:8000 + ใส่ X-Forwarded-* | 502 (refused) / 504 (ช้า) |
| 8 | App | pixbin (UID pixbin) รับ request | 500 + log ใน journal |
| 9 | Kernel/FS | `open()` ใน `/var/lib/pixbin` → ตรวจสิทธิ์ → เขียน → ENOSPC? | 500: permission denied / no space left |
| 10 | กลับ | response ย้อนเส้นทางเดิม, nginx เขียน access log, TLS เข้ารหัส | |

**จำตารางนี้ให้ได้ — มันคือ mental model ทั้ง Milestone 1 ในหน้าเดียว**

## 5. Diagram

```
 ┌───────────────────────── Mac ─────────────────────────┐
 │ curl https://pixbin.test                               │
 │   /etc/hosts: pixbin.test → 192.168.64.5               │
 │   trust: pixbin-lab-ca.crt (--cacert)                  │
 │   route: 192.168.64.0/24 → bridge100 (192.168.64.1)    │
 └──────────────────────────┬─────────────────────────────┘
                            │ TCP 443 + TLS 1.3
 ┌──────────────────────────▼──── VM "lab" (Ubuntu) ──────────────────────────┐
 │ ufw: default deny · allow 22, 80, 443                                      │
 │                                                                            │
 │ nginx (systemd)  :80  → 301 https   :443 → TLS termination                 │
 │   cert /etc/pixbin/tls/  · limits: body 10m, timeout 5s                    │
 │   logs /var/log/nginx/pixbin.{access,error}.log                            │
 │            │ HTTP + X-Forwarded-For / -Proto                               │
 │            ▼ 127.0.0.1:8000                                                │
 │ pixbin (systemd: pixbin.service, User=pixbin, Restart=on-failure,          │
 │         sandbox: ProtectSystem=strict, ReadWritePaths=/var/lib/pixbin)     │
 │   config /etc/pixbin/pixbin.env (600) · logs → journald                    │
 │            │                                                               │
 │            ▼                                                               │
 │ /var/lib/pixbin  (pixbin:pixbin 750)                                       │
 └────────────────────────────────────────────────────────────────────────────┘
```

## 6. Example จริง — Runbook หน้าเดียวของ Pixbin

เมื่อมีคนแจ้งว่า "เว็บใช้ไม่ได้" ให้ทำตามลำดับนี้ **และจดผลทุกข้อ**:

```
0. อาการจริงคืออะไร? ขอ error ที่ user เห็น / ลองเองจากเครื่องของเรา
   curl -sv --cacert pixbin-lab-ca.crt https://pixbin.test/healthz
     ├─ Could not resolve host      → DNS (L10)
     ├─ timeout ก่อน connect         → route / firewall DROP (L7, L8)
     ├─ Connection refused           → ไม่มีใครฟัง 443 (nginx?) (L1, L11)
     ├─ SSL certificate problem (60) → cert (L12)
     ├─ 502 / 504                    → nginx ↔ pixbin (L11)
     ├─ 500                          → pixbin / disk / permission (L4, L6)
     └─ 200 แต่ฟีเจอร์เดียวพัง       → ลองฟีเจอร์นั้นตรง ๆ (upload?) + journal
1. บน VM: systemctl status nginx pixbin --no-pager
2. บน VM: sudo tail -20 /var/log/nginx/pixbin.error.log
3. บน VM: journalctl -u pixbin -n 50 --no-pager
4. บน VM: curl -v http://127.0.0.1:8000/healthz   (ข้าม nginx, ข้าม TLS)
5. บน VM: uptime; free -m; df -h; sudo ss -tlnp
6. สรุป: ชั้นไหน · หลักฐานอะไร · แก้อะไร · ยืนยันอย่างไร · กันไม่ให้เกิดซ้ำอย่างไร
```

## 7. Hands-on Lab

### ส่วน A — ตรวจว่าระบบครบตามแผนที่
ทำให้ทุกข้อเป็นจริง (ย้อนกลับไปบทที่เกี่ยวข้องถ้ายังไม่ครบ):
```bash
# VM
systemctl is-active pixbin nginx
systemctl is-enabled pixbin nginx
sudo ss -tlnp | grep -E ':(80|443|8000)\s'      # nginx: 0.0.0.0:80,443 · pixbin: 127.0.0.1:8000
ps -o user= -p "$(systemctl show -p MainPID --value pixbin)"   # pixbin
ls -ld /var/lib/pixbin                          # drwxr-x--- pixbin pixbin
sudo ufw status | grep -E '^(22|80|443)'
# Mac
curl --cacert pixbin-lab-ca.crt https://pixbin.test/healthz
curl -I http://pixbin.test/                     # 301
```

### ส่วน B — ตามรอย request เดียวผ่านทุกชั้นพร้อมกัน
เปิด 4 หน้าต่างใน VM:
```bash
# V1: packet ที่ :443 จาก Mac (เข้ารหัส)
sudo tcpdump -i any -n -A 'tcp port 443' | head -60
# V2: packet ที่ nginx ส่งต่อให้ pixbin (ไม่เข้ารหัส)
sudo tcpdump -i lo -n -A 'tcp port 8000'
# V3: log ของ nginx
sudo tail -f /var/log/nginx/pixbin.access.log
# V4: log ของ pixbin
journalctl -u pixbin -f
```
```bash
# Mac
echo "hello capstone" | curl --cacert pixbin-lab-ca.crt --data-binary @- https://pixbin.test/upload
```
- `-A` = แสดงเนื้อ packet เป็นตัวอักษร
**ตอบให้ได้:**
- ใน V1 คุณอ่านคำว่า `hello capstone` ได้ไหม? ใน V2 ล่ะ? อธิบายด้วยคำว่า TLS termination
- ใน V2 หา header `X-Forwarded-For` แล้วบอกว่าค่ามาจากไหน
- request เดียวกันปรากฏใน V3 และ V4 — เวลาใน log สองที่ต่างกันเท่าไร หมายความว่าอะไร
- ไฟล์ที่ upload ไปอยู่ที่ไหน owner เป็นใคร (`sudo ls -l /var/lib/pixbin`)

### ส่วน C — Incident drills
ติดตั้ง script (อยู่ใน `lessons/13-milestone-1-capstone/drills/`):
```bash
cd ~/cloud-infra-learn/lessons/13-milestone-1-capstone/drills
sudo install -m 755 break.sh fix.sh reveal.sh /usr/local/sbin/
```
- `break.sh` = สร้าง incident แบบสุ่ม 1 ใน 8 แบบ และบอกแค่อาการที่ user แจ้ง
- `reveal.sh` = เฉลย · `fix.sh` = คืนสภาพ

**กติกา:**
1. `sudo break.sh` แล้ว **ห้ามอ่าน script** ก่อนจบ drill
2. จับเวลา เริ่ม debug จาก Mac ก่อน (ตำแหน่งของ user) แล้วค่อยเข้า VM
3. ทุกคำสั่งที่รัน ต้องเขียนก่อนว่า "รันเพื่อพิสูจน์อะไร"
4. เมื่อคิดว่าเจอต้นเหตุแล้ว **แก้เองก่อน** แล้วยืนยันว่าหายจากฝั่ง Mac
5. จากนั้นค่อย `sudo reveal.sh` เทียบ แล้ว `sudo fix.sh`
6. เขียน incident report (template ด้านล่าง)
7. ทำอย่างน้อย **5 drill** (ถ้าสุ่มได้ซ้ำ ให้ `sudo break.sh <เลข>` เลือกแบบที่ยังไม่เคยเจอ)

**Incident report template**
```
Drill #: ___   เวลาที่ใช้: ___ นาที
อาการที่ได้รับแจ้ง:
อาการที่ฉันเห็นเอง (error เป๊ะ ๆ):
Hypotheses (เรียงตามที่ตรวจ):  1) ...  2) ...  3) ...
คำสั่ง → ผล → สรุป:
ต้นเหตุ (ชั้นไหน):
หลักฐานที่ยืนยัน:
วิธีแก้ + ยืนยันว่าหาย:
ป้องกันไม่ให้เกิดอีก / ตรวจจับให้เร็วขึ้น:
สิ่งที่ทำให้เสียเวลา:
```

### ส่วน D — Baseline revisited
ตอบคำถาม 5 ข้อชุด Baseline ("พิมพ์ google.com แล้วเกิดอะไรขึ้น") **ใหม่ทั้งหมด** โดยไม่ดูคำตอบเก่า
แล้วเทียบกับคำตอบเดิม: อะไรที่เมื่อก่อนไม่รู้ อะไรที่เมื่อก่อนเข้าใจผิด

### ส่วน E — สอนคนอื่น
เขียนเอกสารไม่เกิน 1 หน้า ชื่อ **"เกิดอะไรขึ้นเมื่อพิมพ์ https://pixbin.test"** สำหรับ developer ใหม่ในทีมที่ไม่รู้เรื่อง infra
ต้องมี: diagram, ทุกขั้นในตารางข้อ 4, และ "ถ้าเห็น error X ให้ดูที่ Y"

### ตรวจว่าผ่าน Milestone 1
- [ ] ส่วน A ครบทุกข้อ
- [ ] ส่วน B ตอบได้ครบ 4 คำถาม
- [ ] drill 5 ครั้ง: หาต้นเหตุถูกอย่างน้อย 4 ครั้ง โดยมี reasoning ที่ตรวจสอบได้ (ไม่ใช่เดาถูก)
- [ ] Baseline ใหม่ อธิบายครบ DNS → TCP → TLS → HTTP → server
- [ ] เอกสารส่วน E อ่านแล้วคนอื่นเข้าใจ

## 8. Debugging — หลักรวมของ Milestone 1

**1. แยกให้ออกว่าอาการคือแบบไหน** (นี่คือข้อมูลที่มีค่าที่สุด)

| อาการ | บอกอะไร |
|---|---|
| resolve ไม่ได้ | ยังไม่ถึง network เลย — DNS |
| timeout ตอน connect | packet หาย/ถูก DROP — route, firewall |
| refused ทันที | ถึงเครื่องแล้ว ไม่มีคนฟัง (หรือ REJECT) |
| TLS error | ถึง server แล้ว แต่ตรวจตัวตนไม่ผ่าน |
| 4xx | server ตอบแล้ว ปฏิเสธ request |
| 502 / 504 | proxy ปกติ ข้างหลังมีปัญหา |
| 500 | app ทำงาน แต่ล้มระหว่างทำ |
| ช้า | ดู `curl -w` ว่าเวลาหายไปช่วงไหน |

**2. ตัดครึ่ง (bisect) เส้นทาง** — ทดสอบจากจุดกลางของ pipeline: `curl 127.0.0.1:8000` บน VM
ผ่าน → ปัญหาอยู่ก่อนหน้า (nginx, TLS, network) · ไม่ผ่าน → ปัญหาอยู่ที่ app หรือหลังจากนั้น

**3. เชื่อหลักฐาน ไม่เชื่อสถานะ** — `systemctl status` เขียว ≠ ใช้งานได้ (drill 2, 3, 6, 7)

## 9. Failure Scenarios — ทั้ง Milestone ในตารางเดียว

| ชั้น | failure ที่เรียนมา | บท |
|---|---|---|
| process | crash, ถูก KILL, ค้าง (STOP), ตายตอนปิด terminal, crash loop | L1, L3, L5 |
| resources | CPU เต็ม, OOM, disk เต็ม, ไฟล์ลบแต่ยังเปิด, fd หมด | L2, L6 |
| permission | เขียนไม่ได้, health check โกหก, port < 1024 | L4 |
| network | ไม่มี route, host unreachable, firewall DROP/REJECT, NAT | L7, L8 |
| transport | SYN-SENT ค้าง, RST, CLOSE-WAIT, latency × RTT | L9 |
| naming | NXDOMAIN, SERVFAIL, cache เก่า, /etc/hosts vs dig | L10 |
| HTTP/proxy | 413, 502, 504, Host ผิด, IP จริงหาย | L11 |
| TLS | CA ไม่รู้จัก, ชื่อไม่ตรง, หมดอายุ, chain ไม่ครบ | L12 |

## 10. Trade-offs — สถาปัตยกรรมเครื่องเดียวนี้แพ้ตรงไหน

ระบบที่คุณสร้างทำงานได้จริง และสำหรับ side project ที่มีคนใช้ไม่กี่ร้อยคนก็เพียงพอ แต่:

| ข้อจำกัด | ผล | จะแก้ใน |
|---|---|---|
| เครื่องเดียว | เครื่องตาย = ทุกอย่างตาย, deploy = downtime | Phase 13, 17 |
| ข้อมูลอยู่บน disk ของเครื่อง | disk พัง = รูปหายหมด ไม่มี backup | Phase 12, 18 |
| ไม่มี database | ไม่มี user, ไม่มี metadata | Stage 2, Phase 12 |
| ตั้งค่าด้วยมือทั้งหมด | สร้างซ้ำไม่ได้แน่นอน ลืมขั้นตอน | Phase 5, 8 |
| deploy ด้วย copy binary | ไม่มีประวัติ ย้อนกลับยาก | Phase 9 |
| ดู log ทีละเครื่อง ไม่มี alert | รู้ว่าพังเมื่อ user บอก | Phase 10 |
| cert จาก CA ของเราเอง | ใช้ได้แค่เครื่องที่เชื่อ CA เรา | Phase 7 (ACM / Let's Encrypt) |

**Cost ถ้าย้ายขึ้น cloud แบบนี้ตรง ๆ** (ประมาณการคร่าว ๆ ตรวจราคาจริงก่อนใช้)
- VM เล็ก 2 vCPU / 2–4 GB ≈ $15–35 ต่อเดือน + disk 20 GB ≈ $2 + public IPv4 ≈ $3.6
- 1,000 user ที่ upload รูปวันละ 2 รูป รูปละ 2 MB = ~120 GB ต่อเดือน → **disk และ data transfer ขาออก (คนดูรูป) จะแพงกว่าค่า compute** เร็วมาก
- นี่คือเหตุผลที่ Phase 7 เราจะย้ายไฟล์ไป object storage (S3) และใช้ CDN

## 11. Production Considerations

สิ่งที่ระบบนี้ **ขาด** ถ้าจะให้คนจริงใช้ — จดไว้เป็น backlog ของ Milestone ถัดไป:
- backup ของ `/var/lib/pixbin` และทดสอบ restore
- monitoring + alert (disk > 80%, cert เหลือ < 14 วัน, 5xx rate, restart count)
- deploy ที่ไม่มี downtime และย้อนกลับได้
- config และ infrastructure เป็น code (สร้างใหม่ได้ใน 10 นาที)
- SSH: ปิด password, เปิดเฉพาะ IP ของทีม หรือไม่เปิดเลย
- rate limit และ upload quota ต่อ user
- log ส่งออกนอกเครื่อง

## 12. Quiz

1. **(เข้าใจ)** จากตารางข้อ 4 ขั้นไหนบ้างที่เกิดใน **kernel** และขั้นไหนเกิดใน **user space** ทำไมความต่างนี้ถึงสำคัญตอน debug
2. **(ประยุกต์)** ถ้าเปลี่ยน nginx ให้ส่งต่อไปที่ pixbin **อีกเครื่องหนึ่ง** (192.168.64.6:8000) แทน 127.0.0.1 ต้องเปลี่ยนอะไรบ้างใน pixbin, firewall และมีความเสี่ยงอะไรเพิ่ม
3. **(debugging)** user บอกว่า "upload รูปได้บ้างไม่ได้บ้าง ไม่มี pattern" ให้ตั้ง hypothesis อย่างน้อย 4 ข้อจากคนละชั้น และบอกลำดับที่จะตรวจพร้อมเหตุผล
4. **(architecture)** ถ้ามี 2 VM รัน pixbin และอยากให้ user เข้าได้ทั้งสองเครื่อง โดยถ้าเครื่องหนึ่งตาย user ไม่รู้สึก จะต้องเพิ่มอะไร และปัญหาใหม่อะไรจะเกิดกับไฟล์ที่ upload (ตอบจากสิ่งที่รู้ตอนนี้ — นี่คือโจทย์ของ Milestone ถัดไป)
5. **(trade-off)** เพื่อนร่วมทีมเสนอให้รวม nginx เข้าไปใน pixbin (ใช้ Go ทำ TLS และ limit เอง) เพื่อลดชิ้นส่วน คุณเห็นด้วยไหม ในสถานการณ์ไหนที่ข้อเสนอนี้สมเหตุสมผล

## 13. Challenge

1. **Drill #9 ของคุณเอง:** เพิ่มกรณีใหม่ใน `break.sh` / `reveal.sh` / `fix.sh` ที่ไม่ซ้ำ 8 แบบเดิม (เช่น DNS ของ VM พัง, journald เต็ม, pixbin crash loop, fd limit) แล้วให้คนอื่น (หรือตัวคุณในอีก 1 สัปดาห์) ลองทำ
2. **Rebuild from scratch:** ลบ VM (`multipass delete lab && multipass purge`) แล้วสร้างระบบทั้งหมดใหม่จาก incident report และความจำของคุณ **โดยไม่เปิดบทเรียน** จับเวลาไว้ จดทุกจุดที่ต้องกลับไปเปิดดู — ความเจ็บปวดตรงนี้คือเหตุผลของ Infrastructure as Code (Phase 8)
3. **Healthcheck ที่ดีกว่า:** เขียน `/healthz` ใหม่ให้ตรวจสิ่งที่ user ต้องใช้จริง (สร้างไฟล์ใหม่ได้, พื้นที่เหลือพอ) และเขียน script บน Mac ที่เรียกมันทุก 10 วินาทีผ่าน https แล้วแจ้งเตือนเมื่อพัง — รัน drill ทั้ง 8 แบบ ดูว่า script ของคุณจับได้กี่แบบ แบบที่จับไม่ได้เพราะอะไร

## 14. สรุปสิ่งที่ต้องจำ

- request = ชื่อ → route → firewall → TCP → TLS → HTTP → proxy → app → filesystem และย้อนกลับ
- อาการแรกบอกชั้น: resolve / timeout / refused / TLS / 4xx / 502-504 / 500 / ช้า
- ตัดครึ่ง pipeline ด้วยการทดสอบจากจุดกลาง
- สถานะเขียว ≠ ใช้งานได้ เชื่อการทดสอบแบบที่ user ใช้จริง
- ทุก incident จบด้วย: ต้นเหตุ + หลักฐาน + วิธีกันไม่ให้เกิดซ้ำ

## 15. เรียนต่อ — Milestone 2

**Phase 4 — Virtualization:** VM ที่คุณใช้มา 10 บท จริง ๆ แล้วคืออะไร — CPU, memory, disk และ network card ของมันมาจากไหน
แล้ว **Phase 5 — Containers:** ทำไม container ถึงเบากว่า VM (ใบ้: คุณรู้จักส่วนประกอบของมันแล้ว — cgroup จาก Lesson 5–6, process และ filesystem จาก Lesson 3–4, network จาก Lesson 7–8)
