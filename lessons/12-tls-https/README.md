# Lesson 12 — TLS และ HTTPS: เข้ารหัส และพิสูจน์ว่าคุยกับตัวจริง

Phase 3 · Web · Level 1–5 · Project: Pixbin Stage 1 (HTTPS)

## 1. ปัญหาที่เรากำลังแก้

ตอนนี้ทุกอย่างระหว่าง user กับ pixbin วิ่งเป็นข้อความธรรมดา (Lesson 11 ข้อ 1 เราอ่านได้ด้วย `nc`)
ใครก็ตามที่อยู่ระหว่างทาง — Wi-Fi ร้านกาแฟ, router, ISP — ทำได้ 3 อย่าง:
1. **แอบอ่าน**: เห็น password, cookie, รูปที่ upload
2. **แอบแก้**: ฉีด script/โฆษณาลงในหน้าเว็บ เปลี่ยนเลขบัญชี
3. **ปลอมตัว**: ตั้ง server ปลอมแล้วบอกว่า "ฉันคือ pixbin" (เช่นทำ DNS ปลอม — Lesson 10)

การเข้ารหัสอย่างเดียวแก้ข้อ 1–2 แต่ **ไม่แก้ข้อ 3** — ถ้าคุณเข้ารหัสอย่างดีกับคนร้าย ก็ไม่มีประโยชน์
TLS แก้ทั้งสามข้อ และข้อ 3 คือส่วนที่ทำให้มันซับซ้อน (certificate, CA) และคือที่มาของ incident "cert หมดอายุ" ที่ทำเว็บใหญ่ ๆ ล่มมาแล้วมากมาย

## 2. Mental Model: บัตรประชาชน + ตู้เซฟแบบกุญแจสองดอก

**กุญแจคู่ (asymmetric)** — ตู้เซฟที่มีกุญแจ 2 ดอกคู่กัน
- **public key**: แจกทุกคนได้ ใช้ "ล็อก" หรือ "ตรวจลายเซ็น"
- **private key**: เจ้าของเก็บคนเดียว ใช้ "ปลดล็อก" หรือ "เซ็นชื่อ"
- ใครเซ็นด้วย private key → ทุกคนตรวจได้ด้วย public key ว่าเจ้าของเซ็นจริง

**Certificate = บัตรประชาชนของ server**
- เขียนว่า "public key นี้เป็นของ pixbin.test" แล้ว **มีหน่วยงานที่น่าเชื่อถือ (CA) เซ็นรับรอง**
- browser มีรายชื่อ CA ที่เชื่อถืออยู่แล้วในเครื่อง (trust store) — เหมือนเรารู้จักตราของกรมการปกครอง
- server ต้องพิสูจน์ด้วยว่า **ถือ private key** ที่คู่กับ public key ในบัตร (ไม่งั้นแค่ขโมยสำเนาบัตรไปก็ปลอมได้)

**กุญแจชั่วคราว (symmetric)** — พอพิสูจน์ตัวตนเสร็จ สองฝั่งตกลงกุญแจลับร่วมกันหนึ่งดอก แล้วใช้มันเข้ารหัสข้อมูลจริง (เร็วกว่ากุญแจคู่มาก)

> ⚠️ ขอบเขตของ analogy: บัตรประชาชนแสดงตัวตนของ "คน" แต่ certificate ยืนยันแค่ว่า "ใครคุมชื่อโดเมนนี้"
> CA ทั่วไปตรวจแค่ว่าคุณคุมโดเมนได้ (DV) ไม่ได้ตรวจว่าเป็นบริษัทน่าเชื่อถือ — เว็บ phishing ก็มีกุญแจเขียวได้

## 3. Concept

**TLS ให้ 3 อย่าง**
- **Confidentiality** (อ่านไม่ได้) — symmetric encryption
- **Integrity** (แก้ไม่ได้โดยไม่ถูกจับ) — ทุก record มีรหัสตรวจสอบ
- **Authentication** (คุยกับตัวจริง) — certificate + CA

**Certificate มีอะไร**
- **Subject / SAN** (Subject Alternative Name): ชื่อที่ใช้ได้ — browser ดูที่ **SAN** เป็นหลัก
- **Issuer**: CA ที่เซ็น · **Validity**: `notBefore` – `notAfter`
- **Public key** ของ server · **ลายเซ็น** ของ issuer

**Chain of trust**
```
Root CA (อยู่ใน trust store ของ OS/browser — self-signed)
  └─ Intermediate CA (root เซ็นให้)
       └─ Server cert: pixbin.com (intermediate เซ็นให้)
```
server ต้องส่ง **server cert + intermediate** ให้ client · client ไล่ลายเซ็นขึ้นไปจนถึง root ที่ตัวเองเชื่อ
ส่ง intermediate ไม่ครบ = บางเครื่องใช้ได้ (มี cache) บางเครื่องไม่ได้ — bug ที่เจอบ่อยมาก

**SNI** (Server Name Indication): client บอกชื่อที่ต้องการตั้งแต่ข้อความแรกของ handshake
เพื่อให้ server หนึ่ง IP เลือก certificate ถูกใบได้ (คล้าย `Host` header ของ HTTP แต่มาก่อนการเข้ารหัส)

**ใครออก certificate จริง**: Let's Encrypt (ฟรี, อายุ 90 วัน, ต่ออัตโนมัติด้วย ACME), AWS Certificate Manager (ฟรีเมื่อใช้กับ ALB/CloudFront), CA เชิงพาณิชย์
ต้องพิสูจน์ว่าคุมโดเมนได้ (ตั้ง DNS TXT record หรือวางไฟล์บนเว็บ) — **pixbin.test เป็นชื่อสมมติ จึงขอจากใครไม่ได้** บทนี้เราจะตั้ง CA ของตัวเอง ซึ่งช่วยให้เห็นทุกขั้นชัดกว่าด้วย

## 4. สิ่งที่เกิดขึ้นข้างใน — TLS 1.3 handshake

```
Client                                              Server
  │── ClientHello ────────────────────────────────►│  "อยากคุยกับ pixbin.test (SNI), รองรับ cipher เหล่านี้,
  │                                                 │   นี่ครึ่งหนึ่งของกุญแจลับ (key share)"
  │◄──────────────────────────────── ServerHello ──│  "เลือก cipher นี้, นี่อีกครึ่งของกุญแจลับ"
  │◄────────────── Certificate + CertificateVerify ─│  "นี่บัตรของฉัน + ลายเซ็นที่พิสูจน์ว่าฉันถือ private key"
  │◄──────────────────────────────────── Finished ─│  (ตั้งแต่ตรงนี้เข้ารหัสแล้ว)
  │  client ตรวจ: chain ถึง root ที่เชื่อ? ชื่อตรง SAN? ยังไม่หมดอายุ? ลายเซ็นถูก?
  │── Finished + HTTP request (เข้ารหัส) ─────────►│
  │◄──────────────────────── HTTP response (เข้ารหัส)│
```
- TLS 1.3 ใช้ **1 RTT** · TLS 1.2 ใช้ 2 RTT — รวมกับ TCP (Lesson 9) connection ใหม่แบบ HTTPS = TCP 1 RTT + TLS 1 RTT + request 1 RTT
- ถ้าตรวจไม่ผ่านข้อไหน client **ตัด connection** — browser แสดงหน้าเตือนเต็มจอ, curl ได้ exit code 60

## 5. Diagram — Pixbin หลังบทนี้

```
 Mac (มี ca.crt ของเรา)
   │  https://pixbin.test  (TCP 443, TLS)
   ▼
 ┌──────────── VM ────────────────────────────────────────────┐
 │ ufw: 22, 80, 443                                           │
 │ nginx :80  → 301 ไป https://                                │
 │ nginx :443 → TLS termination                               │
 │     cert: /etc/pixbin/tls/pixbin.crt                       │
 │     key : /etc/pixbin/tls/pixbin.key  (600, root)           │
 │     │  HTTP ธรรมดา (ในเครื่อง, X-Forwarded-Proto: https)     │
 │     ▼                                                      │
 │ pixbin 127.0.0.1:8000                                      │
 └────────────────────────────────────────────────────────────┘
 CA ของ lab: ~/pki/ca.key (ต้องปกป้องที่สุด), ~/pki/ca.crt (แจกได้)
```
**TLS termination** = proxy ถอดรหัสแล้วส่งต่อเป็น HTTP ธรรมดาให้ app — app ไม่ต้องรู้เรื่อง certificate

## 6. Example จริง — สร้าง CA และ certificate

```bash
# 1) CA: key + certificate ที่เซ็นตัวเอง (root)
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out ca.key
openssl req -x509 -new -key ca.key -days 30 -subj "/CN=Pixbin Lab CA" -out ca.crt

# 2) server: key + CSR (คำขอให้เซ็น: "นี่ public key ของฉัน ชื่อ pixbin.test")
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out pixbin.key
openssl req -new -key pixbin.key -subj "/CN=pixbin.test" -out pixbin.csr

# 3) CA เซ็น CSR พร้อมใส่ SAN
printf "subjectAltName=DNS:pixbin.test\nextendedKeyUsage=serverAuth\n" > san.ext
openssl x509 -req -in pixbin.csr -CA ca.crt -CAkey ca.key -CAcreateserial -days 7 -extfile san.ext -out pixbin.crt
```
- `genpkey -algorithm EC ... P-256` = สร้าง private key แบบ elliptic curve (เล็กและเร็วกว่า RSA ที่ความปลอดภัยเท่ากัน)
- `req -x509` = ออก certificate ที่เซ็นตัวเองทันที (ใช้เป็น root) · `req -new` = สร้าง CSR
- `x509 -req ... -CA -CAkey` = ใช้ CA เซ็น CSR · `-days 7` = อายุ 7 วัน
- ไม่ใส่ `-extfile san.ext` = certificate ไม่มี SAN → client สมัยใหม่ปฏิเสธ (ลองใน Challenge)

## 7. Hands-on Lab

ทำใน VM ต่อจาก Lesson 11 (nginx + pixbin ทำงาน)

### ข้อ 1 — ดู TLS ของเว็บจริง
```bash
curl -sv -o /dev/null https://example.com 2>&1 | grep -E "SSL connection|subject:|issuer:|expire date|subjectAltName"
echo | openssl s_client -connect example.com:443 -servername example.com 2>/dev/null | grep -E "^ *[0-9] s:|^ *i:|Verify return code|Protocol"
```
- `s_client` = TLS client ดิบ ๆ แสดง chain (`0 s:` = server cert, `1 s:` = intermediate) และผลตรวจ
หา: ใครออก cert ให้ example.com, หมดอายุเมื่อไหร่, chain ยาวกี่ชั้น

### ข้อ 2 — สร้าง CA และ cert ของ pixbin
```bash
mkdir -p ~/pki && cd ~/pki
# รันทั้ง 3 ขั้นจากข้อ 6 ของบทเรียน
chmod 600 ca.key pixbin.key
openssl x509 -in pixbin.crt -noout -subject -issuer -dates -ext subjectAltName
openssl verify -CAfile ca.crt pixbin.crt
```
Expected: `pixbin.crt: OK` · issuer = `Pixbin Lab CA` · SAN = `DNS:pixbin.test`

### ข้อ 3 — ให้ nginx ใช้ TLS
```bash
sudo mkdir -p /etc/pixbin/tls
sudo install -m 644 ~/pki/pixbin.crt /etc/pixbin/tls/pixbin.crt
sudo install -m 600 ~/pki/pixbin.key /etc/pixbin/tls/pixbin.key
sudo tee /etc/nginx/sites-available/pixbin > /dev/null <<'EOF'
server {
    listen 80 default_server;
    listen 443 ssl default_server;
    ssl_reject_handshake on;          # ชื่อที่ไม่รู้จัก: ปฏิเสธตั้งแต่ TLS handshake
    return 444;
}

server {
    listen 80;
    server_name pixbin.test;
    return 301 https://$host$request_uri;   # HTTP → HTTPS
}

server {
    listen 443 ssl;
    server_name pixbin.test;

    ssl_certificate     /etc/pixbin/tls/pixbin.crt;
    ssl_certificate_key /etc/pixbin/tls/pixbin.key;
    ssl_protocols       TLSv1.2 TLSv1.3;

    client_max_body_size 10m;
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
sudo nginx -t && sudo systemctl reload nginx
sudo ufw allow 443/tcp
```
- nginx อ่าน key ตอนเริ่ม (ในฐานะ root) แล้ว worker ค่อยลดสิทธิ์ — key จึงเป็น 600 ของ root ได้ (เหมือน EnvironmentFile ใน Lesson 5)

### ข้อ 4 — Mac ยังไม่เชื่อ CA ของเรา
```bash
# Mac
curl -I http://pixbin.test/                # 301 ไป https
curl https://pixbin.test/
```
Expected: `curl: (60) SSL certificate problem: unable to get local issuer certificate`
**นี่คือ TLS ทำงานถูกต้อง** — Mac ไม่รู้จัก "Pixbin Lab CA" จึงไม่เชื่อ

```bash
# Mac — เอา ca.crt (ของที่แจกได้) มา แล้วบอก curl ให้เชื่อ
multipass transfer lab:/home/ubuntu/pki/ca.crt ./pixbin-lab-ca.crt
curl --cacert pixbin-lab-ca.crt https://pixbin.test/
curl --cacert pixbin-lab-ca.crt -sv -o /dev/null https://pixbin.test/ 2>&1 | grep -E "SSL connection|subject:|issuer:|subjectAltName"
```
Expected: หน้า pixbin + `SSL connection using TLSv1.3` + `host "pixbin.test" matched cert's "pixbin.test"`

### ข้อ 5 — ต้นทุนของ HTTPS
```bash
# Mac
curl -so /dev/null -w "connect=%{time_connect} tls=%{time_appconnect} ttfb=%{time_starttransfer}\n" http://pixbin.test/        # ได้ 301 จาก nginx (ไม่มี TLS)
curl --cacert pixbin-lab-ca.crt -so /dev/null -w "connect=%{time_connect} tls=%{time_appconnect} ttfb=%{time_starttransfer}\n" https://pixbin.test/files
```
`time_appconnect` = เวลาจนถึง TLS handshake เสร็จ · ใน LAN ต่างกันน้อยมาก — ลองคำนวณว่าถ้า RTT = 200ms จะต่างกันเท่าไร (Lesson 9)

### ข้อ 6 — ชื่อไม่ตรง
```bash
# VM: ให้ nginx รับอีกชื่อหนึ่งด้วย แต่ cert ยังมีชื่อเดียว
sudo sed -i 's/server_name pixbin.test;/server_name pixbin.test www.pixbin.test;/' /etc/nginx/sites-available/pixbin
sudo nginx -t && sudo systemctl reload nginx
# Mac
echo "<IP ของ VM> www.pixbin.test" | sudo tee -a /etc/hosts
curl --cacert pixbin-lab-ca.crt https://www.pixbin.test/
curl --cacert pixbin-lab-ca.crt https://<IP ของ VM>/
```
Expected:
- `www`: `SSL: no alternative certificate subject name matches target host name 'www.pixbin.test'`
- ด้วย IP: `tlsv1 unrecognized name` — เข้า default server ที่ `ssl_reject_handshake`
อธิบายความต่างของสอง error: อันไหน **client** ปฏิเสธ อันไหน **server** ปฏิเสธ

### ข้อ 7 — certificate หมดอายุ
```bash
# VM
sudo apt install -y faketime
cd ~/pki
faketime '2020-01-01 00:00:00' openssl x509 -req -in pixbin.csr -CA ca.crt -CAkey ca.key -CAcreateserial -days 1 -extfile san.ext -out expired.crt
openssl x509 -in expired.crt -noout -dates
sudo install -m 644 expired.crt /etc/pixbin/tls/pixbin.crt
sudo systemctl reload nginx
# Mac
curl --cacert pixbin-lab-ca.crt https://pixbin.test/
echo | openssl s_client -connect pixbin.test:443 -servername pixbin.test -CAfile pixbin-lab-ca.crt 2>/dev/null | grep "Verify return code"
```
- `faketime` หลอก program ว่าตอนนี้เป็นปี 2020 → cert ที่ออกมีอายุ 1 วันในปี 2020 = หมดอายุแล้ว
Expected: `certificate has expired` / `Verify return code: 10 (certificate has expired)`
สังเกต: nginx reload ผ่านไม่มีปัญหา, pixbin ปกติ, `systemctl status` เขียวหมด — **แต่ user ทุกคนเข้าไม่ได้**

คืนค่า: `sudo install -m 644 ~/pki/pixbin.crt /etc/pixbin/tls/pixbin.crt && sudo systemctl reload nginx`

### ข้อ 8 (ทางเลือก) — ให้ browser ของ Mac เชื่อ CA
```bash
# Mac — เพิ่ม
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain pixbin-lab-ca.crt
# เปิด https://pixbin.test ใน Safari/Chrome
# ลบออกเมื่อทำเสร็จ
sudo security delete-certificate -c "Pixbin Lab CA" /Library/Keychains/System.keychain
```
> ⚠️ การเชื่อ CA = ให้ใครก็ตามที่ถือ `ca.key` ออก cert ของ **ทุกเว็บ** (รวมธนาคาร) ที่ Mac จะเชื่อ
> นี่คือเหตุผลที่ต้องลบทิ้งหลังทดลอง และทำไม private key ของ CA จึงเป็นของที่ต้องปกป้องที่สุด

### ตรวจว่าทำถูก
- [ ] อธิบายได้ว่าทำไมข้อ 4 ครั้งแรกล้มเหลว และ `--cacert` แก้ด้วยหลักอะไร
- [ ] แยก error ข้อ 6 และ 7 ได้ว่าตรวจไม่ผ่านขั้นไหน
- [ ] อธิบายได้ว่าทำไม cert หมดอายุทำให้ระบบ "ล่ม" ทั้งที่ทุก service ยังเขียว

### Common errors
| อาการ | สาเหตุ |
|---|---|
| `nginx -t`: `cannot load certificate key` | path ผิด หรือ key ไม่คู่กับ cert (`openssl x509 -noout -pubkey -in crt` เทียบกับ `openssl pkey -pubout -in key`) |
| Mac timeout ที่ 443 | ลืม `ufw allow 443/tcp` |
| `unable to get local issuer certificate` แม้ใส่ `--cacert` | ใส่ไฟล์ผิด (ใส่ pixbin.crt แทน ca.crt) |
| faketime ไม่มีผล | ใช้กับ binary ที่ link แบบ static ไม่ได้ — openssl ของ Ubuntu ใช้ได้ |

## 8. Debugging

**Scenario:** "user บางคนเข้าเว็บไม่ได้ ขึ้น Your connection is not private"

| # | Hypothesis | วิธีพิสูจน์ |
|---|---|---|
| 1 | cert หมดอายุ | `echo \| openssl s_client -connect host:443 -servername host 2>/dev/null \| openssl x509 -noout -dates` |
| 2 | ชื่อไม่ตรง SAN (เช่น เข้าด้วย www หรือไม่มี www) | `... \| openssl x509 -noout -ext subjectAltName` |
| 3 | chain ไม่ครบ (ส่ง intermediate ไม่ครบ) | `s_client` ดูว่ามี `1 s:` ไหม และ `Verify return code` |
| 4 | นาฬิกาของ user/server เพี้ยน | cert "ยังไม่เริ่มใช้" หรือ "หมดอายุ" ในมุมมองของเครื่องที่นาฬิกาผิด |
| 5 | บาง server หลัง load balancer ยังใช้ cert เก่า | ทดสอบทีละ server ด้วย `--resolve` |

คำสั่งเดียวที่ควรจำ: `openssl s_client -connect <host>:443 -servername <host>` — มันบอก chain, วันหมดอายุ, protocol และผลตรวจครบในที่เดียว

## 9. Failure Scenarios

| เกิดอะไร | user เห็น | ข้างใน |
|---|---|---|
| cert หมดอายุ | หน้าเตือนเต็มจอ/app มือถือเชื่อมต่อไม่ได้ ทุกคนพร้อมกัน | ไม่มีใครต่ออายุ/automation ต่ออายุพัง |
| private key หลุด | ไม่เห็นอะไรเลย — จนกว่าจะโดนปลอม | คนร้ายปลอมเป็น server ได้จนกว่า cert จะถูก revoke/หมดอายุ |
| ต่อ cert ใหม่แต่ลืม reload | ยังหมดอายุ ทั้งที่ไฟล์ใหม่แล้ว | nginx โหลด cert ตอนเริ่ม/reload เท่านั้น |
| mixed content | บางรูป/script ไม่โหลด | หน้า HTTPS โหลดของผ่าน HTTP — browser บล็อก |
| app สร้างลิงก์เป็น http:// | redirect loop หรือลิงก์ผิด | app ไม่รู้ว่า user ใช้ https (ต้องอ่าน `X-Forwarded-Proto`) |

## 10. Trade-offs

**TLS termination ที่ proxy vs ที่ app vs ตลอดทาง**
- ที่ proxy (แบบที่เราทำ): app ง่าย จัดการ cert ที่เดียว — แต่ระหว่าง proxy กับ app เป็นข้อความธรรมดา (ในเครื่องเดียวกันรับได้ ข้ามเครื่องต้องคิด)
- ที่ app: เข้ารหัสจนถึง app — แต่ทุก app ต้องจัดการ cert
- ตลอดทาง (end-to-end / re-encrypt / mTLS): ปลอดภัยที่สุด — ซับซ้อนที่สุด (service mesh ทำเรื่องนี้ใน Phase 16)

**อายุ cert สั้น vs ยาว**
- สั้น (90 วัน, Let's Encrypt): key หลุดก็เสียหายไม่นาน บังคับให้ทำ automation — แต่ถ้า automation พังก็หมดอายุเร็ว
- ยาว (1 ปี): ต่อไม่บ่อย — แต่คนลืมวิธีต่อ และมักต่อด้วยมือ ซึ่งคือสาเหตุอันดับหนึ่งของ incident cert หมดอายุ
- ทิศทางของวงการ: สั้นลงเรื่อย ๆ + automation 100%

**Managed cert (ACM) vs จัดการเอง**
- managed: ต่ออายุอัตโนมัติ ไม่มี key ให้หลุด (AWS เก็บ) — ใช้ได้เฉพาะกับบริการของ AWS (ALB, CloudFront)
- จัดการเอง: ใช้ที่ไหนก็ได้ — ต้องดูแลการต่อ อายุ และการเก็บ key เอง

## 11. Production Considerations

- **Automate การต่ออายุ และ monitor วันหมดอายุแยกจากกัน** — alert เมื่อเหลือ < 14 วัน ไม่ว่า automation จะบอกว่าทำงานอยู่หรือไม่
- **ปกป้อง private key**: สิทธิ์ 600, ไม่อยู่ใน git, ไม่ส่งทาง chat — หลุดแล้วต้องออกใหม่และ revoke
- **HTTP → HTTPS redirect** และพิจารณา HSTS (`Strict-Transport-Security`) ให้ browser ไม่ลองใช้ HTTP อีกเลย — ระวัง: ตั้งแล้วถอนยาก
- **ปิด TLS เวอร์ชันเก่า** (1.0, 1.1) · ใช้ TLS 1.2+ เท่านั้น
- **Cost:** cert จาก Let's Encrypt และ ACM ฟรี · TLS ใช้ CPU ตอน handshake — ที่ traffic สูง connection reuse (keep-alive) ช่วยลดทั้ง latency และ CPU
- **Cloud:** ใน AWS มักทำ TLS termination ที่ ALB ด้วย cert จาก ACM แล้วส่ง HTTP (หรือ HTTPS อีกชั้น) เข้า private subnet (Phase 7)

## 12. Quiz

1. **(เข้าใจ)** ถ้าเข้ารหัสแต่ไม่มีการตรวจ certificate (เช่นใช้ `curl -k`) ยังเสี่ยงอะไร ยกตัวอย่างการโจมตีที่ทำได้
2. **(ประยุกต์)** user ที่ Tokyo เรียก `https://pixbin.com` ครั้งแรก RTT 80ms ประมาณเวลาจนได้ byte แรก แยกเป็น DNS (สมมติ 20ms), TCP, TLS 1.3, HTTP และบอกว่าถ้าเป็น TLS 1.2 จะเพิ่มเท่าไร
3. **(debugging)** เว็บใช้ได้บน Chrome ในคอม แต่ app มือถือ Android รุ่นเก่าเชื่อมต่อไม่ได้ ด้วย error เรื่อง certificate ตั้ง hypothesis 2 ข้อ และวิธีพิสูจน์
4. **(architecture)** Pixbin จะมี 3 ชื่อ: `pixbin.com`, `www.pixbin.com`, `api.pixbin.com` ต้องการ cert กี่ใบ ใส่ SAN อะไร และข้อดีข้อเสียของ wildcard `*.pixbin.com`
5. **(trade-off)** ทีมเสนอให้ออก cert อายุ 5 ปีจะได้ไม่ต้องยุ่ง คุณจะแย้งอย่างไร (อย่างน้อย 2 เหตุผล)

## 13. Challenge

1. ออก cert ใหม่ที่ใช้ได้ทั้ง `pixbin.test` และ `www.pixbin.test` แล้วทำให้ `www` redirect ไปที่ `pixbin.test` (301) พิสูจน์ด้วย curl ว่าไม่มี error เรื่องชื่ออีก
2. ออก cert **ไม่ใส่ SAN** (มีแค่ CN) ติดตั้ง แล้วบันทึก error ที่ curl ให้ อธิบายว่าทำไมมาตรฐานปัจจุบันไม่ดู CN แล้ว
3. เขียน `certcheck.sh <host>` ที่พิมพ์จำนวนวันก่อนหมดอายุ และ exit code ไม่เป็น 0 เมื่อเหลือน้อยกว่า 14 วัน (ใบ้: `openssl x509 -checkend <seconds>`) ทดสอบกับ pixbin.test (ทั้ง cert ปกติและ expired) และเว็บจริงสัก 2 เว็บ

## 14. สรุปสิ่งที่ต้องจำ

- TLS = อ่านไม่ได้ + แก้ไม่ได้ + **คุยกับตัวจริง** · ข้อสุดท้ายมาจาก certificate + CA
- client ตรวจ 4 อย่าง: chain ถึง root ที่เชื่อ · ชื่อตรง SAN · ยังไม่หมดอายุ · server ถือ private key จริง
- ส่ง intermediate ให้ครบ · ใช้ SNI เลือก cert
- TLS 1.3 = 1 RTT เพิ่มจาก TCP
- cert หมดอายุ = ล่มทั้งระบบ ทั้งที่ทุก service เขียว → automate + monitor แยกกัน
- `openssl s_client -connect host:443 -servername host` คือเครื่องมือหลัก

## 15. เรียนต่อ

**Lesson 13 — Milestone 1 Capstone:** ต่อทุกชิ้นตั้งแต่ Lesson 1–12 เข้าด้วยกัน เดินตาม request หนึ่งตัวตั้งแต่พิมพ์ URL
แล้วรับมือกับ incident จำลองที่คุณไม่รู้ล่วงหน้าว่าพังตรงไหน
