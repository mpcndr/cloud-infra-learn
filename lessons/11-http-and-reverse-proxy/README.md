# Lesson 11 — HTTP และ Reverse Proxy

Phase 3 · Web · Level 1–5 · Project: Pixbin Stage 1 (เริ่มมี nginx อยู่หน้า pixbin)

## 1. ปัญหาที่เรากำลังแก้

ตอนนี้ pixbin ฟังอยู่ที่ `127.0.0.1:8000` (ปลอดภัย แต่ข้างนอกเข้าไม่ได้) และเราอยากให้คนเข้าได้ที่ `http://pixbin.test` (port 80)
ทางเลือกที่ผ่านมาล้วนมีปัญหา:
- ให้ pixbin bind `0.0.0.0:80` → ต้องรันเป็น root หรือให้ capability (Lesson 4–5) และ pixbin ต้องรับมือทุกอย่างของอินเทอร์เน็ตเอง
- อยากมีหลาย app บนเครื่องเดียว (pixbin, admin, api) แต่ port 80 มีได้แค่ตัวเดียว
- อยากจำกัดขนาด upload, เก็บ access log, ใส่ HTTPS (Lesson 12) โดยไม่ต้องเขียนเองในทุก app

คำตอบคือวางตัวกลางไว้หน้า app: **reverse proxy** — และจะเข้าใจมันได้ ต้องเข้าใจก่อนว่า **HTTP จริง ๆ คือข้อความหน้าตาแบบไหน**

## 2. Mental Model

**HTTP = จดหมายที่มีรูปแบบตายตัว** ส่งผ่านท่อ TCP (Lesson 9)
- บรรทัดแรก: ขออะไร (`GET /files HTTP/1.1`)
- หัวจดหมาย (headers): ข้อมูลประกอบ (`Host: pixbin.test`, `Content-Length: 1024`)
- บรรทัดว่าง แล้วตามด้วยเนื้อความ (body)
- คำตอบก็รูปแบบเดียวกัน แต่บรรทัดแรกเป็นผลลัพธ์ (`HTTP/1.1 200 OK`)

**Reverse proxy = พนักงานต้อนรับหน้าตึก**
- ลูกค้าคุยกับพนักงานต้อนรับเท่านั้น ไม่รู้ว่าข้างในมีกี่แผนก
- พนักงานต้อนรับดูว่าลูกค้าขออะไร (`Host`, path) → เดินไปถามแผนกที่ถูก → เอาคำตอบกลับมาให้
- ตรวจของก่อนเข้า (ขนาด, รูปแบบ), จดบันทึกทุกคนที่มา, และถ้าแผนกไม่ตอบก็บอกลูกค้าเองว่า "แผนกนั้นไม่ว่าง"

> ⚠️ ขอบเขตของ analogy: proxy เปิด **connection ใหม่** ไปหา app — จากมุมของ app ทุก request มาจาก proxy (`127.0.0.1`)
> ข้อมูลของลูกค้าจริงต้องถูก "แนบไปในจดหมาย" (header `X-Forwarded-For`) ไม่งั้น app ไม่มีทางรู้

## 3. Concept

**HTTP request / response**
```
GET /files?limit=10 HTTP/1.1          ← method  path  version
Host: pixbin.test                     ← บอกว่าต้องการเว็บไหน (หลายเว็บใช้ IP เดียวกันได้เพราะ header นี้)
User-Agent: curl/8.5.0
Accept: */*
                                      ← บรรทัดว่าง = headers จบ
(body — GET ปกติไม่มี)

HTTP/1.1 200 OK                       ← version  status code  ข้อความ
Content-Type: text/plain; charset=utf-8
Content-Length: 142                   ← body ยาวเท่าไร (ผู้รับจะได้รู้ว่าอ่านจบเมื่อไร)

hello from Pixbin ...
```

**Methods**: `GET` อ่าน · `POST` สร้าง/ส่งข้อมูล · `PUT`/`PATCH` แก้ · `DELETE` ลบ · `HEAD` เหมือน GET แต่ไม่เอา body

**Status code — ตัวแรกบอกว่าใครผิด**

| กลุ่ม | ความหมาย | ที่เจอบ่อย |
|---|---|---|
| 2xx | สำเร็จ | 200 OK, 201 Created, 204 No Content |
| 3xx | ไปที่อื่น | 301 ย้ายถาวร, 302/307 ย้ายชั่วคราว, 304 ใช้ของใน cache ได้ |
| 4xx | **client ผิด** | 400 รูปแบบผิด, 401 ยังไม่ login, 403 ไม่มีสิทธิ์, 404 ไม่มี, 413 ใหญ่ไป, 429 ถี่ไป |
| 5xx | **server ผิด** | 500 app พัง, **502 proxy ต่อ app ไม่ได้**, 503 ไม่พร้อม, **504 app ตอบช้าเกิน** |

502 และ 504 เป็นของ **proxy** — มันกำลังบอกว่า "ฉันไม่เป็นไร แต่ตัวที่อยู่ข้างหลังฉันมีปัญหา"

**Reverse proxy ทำอะไรให้**
- ฟัง port 80/443 แทน app (app bind 127.0.0.1 ได้)
- route ตาม `Host` / path ไปหลาย app
- ใส่ header ข้อมูลของ client (`X-Forwarded-For`, `X-Forwarded-Proto`)
- จำกัดขนาด body, timeout, rate limit
- access log กลาง · TLS termination (Lesson 12) · load balancing (Phase 13)
- nginx, HAProxy, Caddy, Envoy, และ AWS ALB ล้วนเป็นสิ่งนี้

## 4. สิ่งที่เกิดขึ้นข้างใน

```
Mac: curl http://pixbin.test/
 1. DNS (Lesson 10): pixbin.test → 192.168.64.5 (จาก /etc/hosts)
 2. TCP handshake ไป 192.168.64.5:80 (Lesson 9) — ผ่าน ufw เพราะเรา allow 80
 3. ส่ง "GET / HTTP/1.1\r\nHost: pixbin.test\r\n..."
 4. nginx: Host = pixbin.test → เข้า server block ของ pixbin → location / → proxy_pass
 5. nginx เปิด TCP ใหม่ไป 127.0.0.1:8000 ส่ง request ต่อ พร้อมเพิ่ม
       X-Forwarded-For: 192.168.64.1     ← IP ของ Mac
       X-Real-IP: 192.168.64.1
 6. pixbin ตอบ → nginx ส่งต่อให้ Mac + เขียน access log
    ถ้าข้อ 5 connect ไม่ได้ → nginx ตอบ 502 เอง
    ถ้า pixbin ไม่ตอบภายใน proxy_read_timeout → nginx ตอบ 504 เอง
```

## 5. Diagram

```
                  ┌──────────────────────── VM ─────────────────────────┐
 Mac / user       │  ufw: allow 22, 80                                  │
  curl ──TCP:80──►│  nginx (0.0.0.0:80, user www-data)                  │
                  │    │  access.log / error.log                         │
                  │    │ proxy_pass (TCP ใหม่)                           │
                  │    ▼                                                 │
                  │  pixbin (127.0.0.1:8000, user pixbin, systemd)      │
                  └─────────────────────────────────────────────────────┘
  pixbin เห็น: RemoteAddr = 127.0.0.1 (nginx) · X-Forwarded-For = IP ของ user จริง
```

## 6. Example จริง — config ของ nginx

`/etc/nginx/sites-available/pixbin`
```nginx
server {
    listen 80;
    server_name pixbin.test;                 # ใช้ server block นี้เมื่อ Host = pixbin.test

    access_log /var/log/nginx/pixbin.access.log;
    error_log  /var/log/nginx/pixbin.error.log;

    location / {
        proxy_pass http://127.0.0.1:8000;    # ส่งต่อให้ pixbin
        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 5s;               # รอ pixbin ตอบไม่เกิน 5 วินาที (default 60s)
    }
}
```
ยังไม่ได้ตั้ง `client_max_body_size` — default ของ nginx คือ **1 MB** (Lab ข้อ 5 จะเจอ)

## 7. Hands-on Lab

VM: pixbin รันผ่าน systemd ที่ 127.0.0.1:8000 · Mac: มี `pixbin.test` ใน `/etc/hosts` (Lesson 10 ข้อ 6)

### ข้อ 1 — HTTP ดิบ ๆ ด้วยมือ
```bash
# VM
printf 'GET / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n' | nc 127.0.0.1 8000
```
- `\r\n` = สิ้นสุดบรรทัดของ HTTP · `\r\n\r\n` = บรรทัดว่าง (headers จบ)
Expected: เห็น status line, headers และ body แยกกันชัดเจน — นี่คือทั้งหมดที่ HTTP เป็น
ลองส่งแบบผิด: `printf 'HELLO\r\n\r\n' | nc 127.0.0.1 8000` ได้ status อะไร

### ข้อ 2 — curl แบบเห็นทุกอย่าง
```bash
curl -v http://localhost:8000/files
```
- บรรทัด `>` = สิ่งที่ส่งไป · `<` = สิ่งที่ได้กลับ · `*` = ข้อมูลของ curl เอง (connect, ฯลฯ)

### ข้อ 3 — ติดตั้ง nginx และวาง config
```bash
sudo apt install -y nginx
sudo tee /etc/nginx/sites-available/pixbin > /dev/null <<'EOF'
server {
    listen 80;
    server_name pixbin.test;

    access_log /var/log/nginx/pixbin.access.log;
    error_log  /var/log/nginx/pixbin.error.log;

    location / {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 5s;
    }
}
EOF
sudo ln -s /etc/nginx/sites-available/pixbin /etc/nginx/sites-enabled/pixbin
sudo rm /etc/nginx/sites-enabled/default
sudo nginx -t                         # ตรวจ syntax ก่อนเสมอ
sudo systemctl reload nginx
sudo ufw allow 80/tcp
sudo ss -tlnp | grep -E ':80|:8000'
```
- `sites-available` = config ทั้งหมด · `sites-enabled` = symlink ของตัวที่เปิดใช้ (ปิดได้โดยลบ link)
- `nginx -t` ก่อน `reload` — config ผิดแล้ว reload จะถูกปฏิเสธ แต่ถ้า **restart** ด้วย config ผิด nginx จะไม่ขึ้นเลย
- `ss` ต้องเห็น nginx ที่ `0.0.0.0:80` และ pixbin ที่ `127.0.0.1:8000`

### ข้อ 4 — เข้าจาก Mac
```bash
# Mac
curl http://pixbin.test/
curl -I http://pixbin.test/           # -I = HEAD ดูแค่ headers
```
Expected: `you came from: 127.0.0.1:xxxxx` และ `forwarded for: 192.168.64.1`
อธิบายว่าทำไม `you came from` ไม่ใช่ IP ของ Mac แล้วดู header `Server:` — ใครตอบ?
```bash
# VM
sudo tail -3 /var/log/nginx/pixbin.access.log
```

### ข้อ 5 — 413: body ใหญ่เกิน
```bash
# Mac
head -c 2000000 /dev/urandom > /tmp/2m.bin
curl -w ' %{http_code}\n' --data-binary @/tmp/2m.bin http://pixbin.test/upload
# VM
sudo tail -1 /var/log/nginx/pixbin.error.log
journalctl -u pixbin -n 3 --no-pager
```
Expected: `413` และ error log `client intended to send too large body` — **pixbin ไม่เห็น request นี้เลย** (ดู journal)
แก้: เพิ่ม `client_max_body_size 10m;` ใต้ `server_name` → `nginx -t` → `reload` → ลองใหม่

### ข้อ 6 — 504: app ช้าเกิน
```bash
# Mac
curl -w ' %{http_code} %{time_total}s\n' "http://pixbin.test/work/sleep?ms=8000"
```
Expected: `504` หลัง 5 วินาที · error log: `upstream timed out`
คำถาม: หลัง nginx ตอบ 504 ไปแล้ว pixbin ยังทำ request นั้นต่อไหม? ดู journal ของ pixbin ประกอบ

### ข้อ 7 — 502: app ไม่อยู่
```bash
# VM
sudo systemctl stop pixbin
# Mac
curl -w ' %{http_code}\n' http://pixbin.test/
# VM
sudo tail -1 /var/log/nginx/pixbin.error.log
sudo systemctl start pixbin
```
Expected: `502` · error log: `connect() failed (111: Connection refused) while connecting to upstream`
เทียบกับ Lesson 1: refused ตอนนี้เกิดระหว่างใครกับใคร? user เห็นอะไร?

### ข้อ 8 — Host header เลือก server block
```bash
# Mac
curl -H "Host: other.test" http://pixbin.test/
curl http://<IP ของ VM>/
```
ทั้งสองได้หน้า pixbin ทั้งที่ Host ไม่ตรง — เพราะเมื่อไม่มี block ไหนตรง nginx ใช้ **default server** (block แรกของ port นั้น)
เพิ่ม block นี้ใน `/etc/nginx/sites-available/pixbin` (ไว้บนสุด) แล้ว reload และลองใหม่:
```nginx
server {
    listen 80 default_server;
    return 444;        # ปิด connection ทันทีโดยไม่ตอบอะไร
}
```

### ตรวจว่าทำถูก
- [ ] เขียน HTTP request ด้วยมือผ่าน `nc` ได้
- [ ] อธิบายได้ว่า 413, 502, 504 ในข้อ 5–7 ใครเป็นคนตอบ และ pixbin รู้เรื่องไหม
- [ ] อธิบายได้ว่าทำไม pixbin เห็น 127.0.0.1 และจะรู้ IP จริงได้อย่างไร

### Common errors
| อาการ | สาเหตุ |
|---|---|
| Mac ได้หน้า "Welcome to nginx" | ยังไม่ได้ลบ `sites-enabled/default` หรือยังไม่ reload |
| Mac timeout | ufw ยังไม่ allow 80 (timeout = DROP — Lesson 8) |
| `nginx -t` error `unknown directive` | พิมพ์ผิด / ลืม `;` ท้ายบรรทัด |
| `ln: File exists` | ทำข้อ 3 ซ้ำ ข้ามได้ |

## 8. Debugging

**Scenario:** "เว็บขึ้น 502"

1. **502 มาจาก proxy** → ดู error log ของ proxy ก่อน (ไม่ใช่ log ของ app) — มันบอกสาเหตุตรง ๆ:
   - `Connection refused` → app ไม่ได้ฟังที่ upstream address (ตาย / port ผิด / bind ผิด)
   - `Connection timed out` while **connecting** → firewall/route ระหว่าง proxy กับ app
   - `upstream prematurely closed connection` → app ตายกลางคัน (OOM? panic?)
2. ยืนยันจากฝั่ง proxy: `curl -v http://127.0.0.1:8000/` บนเครื่อง proxy (ทดสอบ hop เดียวกันกับที่ proxy ใช้)
3. ดู app: `systemctl status`, journal, `ss -tlnp`

**ตาราง status → ชั้นที่ต้องดู**

| เห็น | ดูที่ |
|---|---|
| timeout / refused ที่ browser | DNS, network, firewall, proxy ไม่ได้ฟัง (Lesson 7–10) |
| 4xx จาก proxy (413, 444) | config ของ proxy |
| 502 / 504 | ระหว่าง proxy กับ app — error log ของ proxy |
| 500 | app — log ของ app |

## 9. Failure Scenarios

| เกิดอะไร | user เห็น | ข้างใน |
|---|---|---|
| app ตาย/restart | 502 ช่วงสั้น ๆ | proxy ต่อ upstream ไม่ได้ |
| app ช้า (DB ช้า) | 504 | เกิน `proxy_read_timeout` — แต่ app อาจยังทำงานนั้นต่อ (เปลือง resource) |
| upload ใหญ่ | 413 | limit ของ proxy |
| reload nginx ด้วย config ผิด | ไม่มีผลกระทบ (ถูกปฏิเสธ) | แต่ `restart` ด้วย config ผิด = เว็บล่มทั้งหมด |
| app อ่าน IP จาก RemoteAddr | rate limit/ban โดนทุกคนพร้อมกัน | ทุก request ดูเหมือนมาจาก 127.0.0.1 |
| app เชื่อ `X-Forwarded-For` จากใครก็ได้ | คนร้ายปลอม IP ได้ | header นี้ client ใส่เองได้ ต้องเชื่อเฉพาะที่ proxy ของเราใส่ |

## 10. Trade-offs

**มี reverse proxy vs ให้ app รับตรง**
- มี proxy: app เรียบง่าย ปลอดภัยขึ้น ได้ log/limit/TLS/routing ฟรี — แต่เพิ่มหนึ่ง hop (latency เล็กน้อย) และอีกหนึ่งชิ้นที่ต้องดูแล/ตั้งค่าให้ถูก
- ตรง: ชิ้นน้อย — แต่ทุก app ต้องทำเรื่องพวกนี้เอง
- ใน cloud มักใช้ managed proxy (AWS ALB) แทนการดูแล nginx เอง — แลกค่าใช้จ่ายกับการไม่ต้องดูแล

**Timeout สั้น vs ยาว ที่ proxy**
- สั้น: ไม่ปล่อยให้ connection ค้างกินทรัพยากร user รู้ผลเร็ว — แต่ request ที่ช้าโดยธรรมชาติ (report, upload ใหญ่) โดนตัด
- ยาว: งานช้าผ่าน — แต่ถ้า app ค้าง connection สะสมเต็ม (Lesson 6: fd)
- ทางออก: timeout สั้นเป็นค่าหลัก แยก location ที่ต้องการนานเป็นพิเศษ หรือย้ายงานยาวไป background (Phase 14)

## 11. Production Considerations

- **app bind 127.0.0.1 (หรือ private IP) เสมอเมื่อมี proxy** — คนนอกจะได้ผ่าน proxy เท่านั้น
- **ตั้ง limit ที่ proxy**: body size, timeout, rate limit — เป็นแนวป้องกันแรกก่อนถึง app
- **ส่ง IP จริงของ client** ผ่าน `X-Forwarded-For` และให้ app เชื่อเฉพาะเมื่อมาจาก proxy ที่รู้จัก
- **Default server ที่ปฏิเสธ** Host ที่ไม่รู้จัก — กันบอทที่สแกนด้วย IP และกันการแอบอ้างชื่อ
- **ตรวจ config ก่อน reload ทุกครั้ง** (`nginx -t`) และทำเป็นขั้นตอนอัตโนมัติใน deploy (Phase 9)
- **Cost:** access log ทุก request = ปริมาณ log มหาศาลเมื่อ traffic สูง — ค่าเก็บ log ใน cloud คิดตาม GB (Phase 10)

## 12. Quiz

1. **(เข้าใจ)** header `Host` แก้ปัญหาอะไร ก่อนที่จะมีมัน การมีหลายเว็บบน IP เดียวทำได้ไหม
2. **(ประยุกต์)** user บอกว่าเห็น 504 ตอนกดสร้าง report ขณะที่ log ของ app บอกว่า report สร้างเสร็จใน 70 วินาทีและ "สำเร็จ" อธิบายว่าเกิดอะไรขึ้น และ user ควรกดใหม่ไหม (ผลข้างเคียงคืออะไร)
3. **(debugging)** เว็บได้ 502 error log ของ nginx บอก `connect() failed (111: Connection refused)` แต่ `systemctl status pixbin` บอก running ตั้ง hypothesis และบอกคำสั่งที่พิสูจน์
4. **(architecture)** เครื่องเดียวต้องรัน pixbin (pixbin.test) และ admin panel (admin.pixbin.test) โดย admin ต้องเข้าได้จาก IP ของออฟฟิศเท่านั้น ออกแบบ config ของ nginx (เขียนเป็นโครงก็พอ)
5. **(trade-off)** ทำไมเราไม่ให้ app อ่าน IP ของ user จาก `X-Forwarded-For` ทุกกรณี ถ้า app อยู่หลัง proxy สองชั้น (CDN → nginx → app) ต้องระวังอะไร

## 13. Challenge

1. เพิ่ม location `/admin` ใน nginx ที่อนุญาตเฉพาะ IP ของ Mac (`allow` / `deny`) และตอบ 403 กับคนอื่น พิสูจน์ด้วย curl จาก Mac และจาก VM เอง
2. เพิ่ม rate limit (`limit_req_zone`) ให้ `/upload` ได้ไม่เกิน 2 request/วินาทีต่อ IP ยิงทดสอบด้วย loop แล้วนับจำนวน 503/429 ที่ได้
3. เปลี่ยนรูปแบบ access log ให้มีเวลาตอบของ upstream (`$upstream_response_time`) และเวลารวม (`$request_time`) แล้วใช้มันพิสูจน์ว่าใน request `/work/sleep?ms=300` เวลาส่วนใหญ่หายไปที่ไหน

## 14. สรุปสิ่งที่ต้องจำ

- HTTP = ข้อความ: request line + headers + บรรทัดว่าง + body · response ขึ้นต้นด้วย status
- `Host` ทำให้หลายเว็บใช้ IP เดียวกันได้
- 4xx = client ผิด · 5xx = server ผิด · **502/504 = proxy ต่อ app ไม่ได้ / app ช้า**
- reverse proxy: รับแทน app, ส่ง IP จริงผ่าน `X-Forwarded-For`, limit, log, (TLS)
- เจอ 502/504 → ดู error log ของ proxy ก่อน
- `nginx -t` ก่อน reload เสมอ

## 15. เรียนต่อ

**Lesson 12 — TLS และ HTTPS:** ตอนนี้ทุกอย่างวิ่งเป็นข้อความธรรมดา ใครอยู่ระหว่างทางอ่านและแก้ได้หมด
เราจะเข้ารหัส สร้าง certificate เอง และทำให้ `https://pixbin.test` ใช้งานได้
