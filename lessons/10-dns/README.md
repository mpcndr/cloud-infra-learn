# Lesson 10 — DNS: ชื่อกลายเป็น IP ได้อย่างไร

Phase 3 · Web · Level 1–5 · Project: Pixbin Stage 1

## 1. ปัญหาที่เรากำลังแก้

ทุกอย่างที่เรียนมาใช้ IP: `192.168.64.5:8000` แต่ไม่มี user คนไหนพิมพ์ IP
และ IP ของ server เปลี่ยนได้ (ย้ายเครื่อง, ย้าย cloud, เพิ่มเครื่อง) — ถ้า user จำ IP ไว้ ทุกครั้งที่ย้ายก็เข้าไม่ได้

เราต้องการ "ชื่อ" ที่คงที่ แล้วให้มีระบบแปลงชื่อเป็น IP ปัจจุบัน — ระบบนั้นต้อง:
- ใหญ่ระดับโลก (ชื่อหลายร้อยล้านชื่อ) โดยไม่มีใครเป็นเจ้าของทั้งหมด
- เร็ว (ถูกถามก่อนทุก connection)
- ให้เจ้าของชื่อแก้ของตัวเองได้

นั่นคือ **DNS** — และเพราะมันอยู่หน้าทุกอย่าง เมื่อมันพัง ทุกอย่างพังพร้อมกัน วงการจึงมีมุกว่า **"It's always DNS"**

## 2. Mental Model: สมุดโทรศัพท์แบบกระจาย + ผู้ช่วยค้นหา

- ไม่มีสมุดเล่มเดียวของทั้งโลก แต่เป็นลำดับชั้น:
  - **Root** รู้แค่ว่า ".com อยู่ที่ใคร .th อยู่ที่ใคร"
  - **.com** รู้แค่ว่า "example.com ให้ถามเครื่องนี้"
  - **Authoritative server ของ example.com** รู้คำตอบจริง
- **Recursive resolver** (เช่น 1.1.1.1, 8.8.8.8, ของ ISP, ของ AWS) = ผู้ช่วยที่เดินถามแต่ละชั้นแทนคุณ แล้ว **จดคำตอบไว้ (cache)** ตามเวลาที่เจ้าของชื่อกำหนด (**TTL**)
- เครื่องคุณเองมีผู้ช่วยตัวเล็กอีกชั้น (**stub resolver**) + สมุดส่วนตัว `/etc/hosts`

> ⚠️ ขอบเขตของ analogy: สมุดโทรศัพท์ให้เบอร์เดียว แต่ DNS ตอบได้หลาย IP และตอบต่างกันตามคนถามได้ (ตามพื้นที่ — GeoDNS)
> และ "cache" ไม่ได้มีที่เดียว มีทุกชั้น: browser, OS, resolver — การแก้ DNS จึง "ค่อย ๆ มีผล" ไม่ใช่ทันที

## 3. Concept

**Record types ที่ใช้บ่อย**

| Type | ใช้ทำอะไร | ตัวอย่าง |
|---|---|---|
| A | ชื่อ → IPv4 | `example.com → 104.20.23.154` |
| AAAA | ชื่อ → IPv6 | |
| CNAME | ชื่อ → อีกชื่อหนึ่ง (alias) | `www.github.com → github.com` |
| NS | โซนนี้ใครเป็น authoritative | |
| MX | mail server ของโดเมน | |
| TXT | ข้อความ — ใช้ยืนยันความเป็นเจ้าของโดเมน, SPF, ออก certificate (Lesson 12) | |

**TTL** (วินาที) — resolver เก็บคำตอบได้นานเท่าไร
- TTL สูง (86400 = 1 วัน): เร็ว ลดภาระ — แต่เปลี่ยน IP แล้วคนเห็นของเก่านานสุด 1 วัน
- TTL ต่ำ (60): เปลี่ยนแล้วมีผลเร็ว — แต่ถามบ่อย ช้ากว่านิดหน่อย พึ่ง DNS server มากขึ้น
- "DNS propagation" ที่คนพูดถึง จริง ๆ คือ **cache ตาม TTL ค่อย ๆ หมดอายุ** ไม่ได้มีอะไรกระจายออกไป

**คำตอบที่เป็นความล้มเหลว** (ต้องแยกให้ออก)

| ผล | ความหมาย |
|---|---|
| `NXDOMAIN` | ชื่อนี้ **ไม่มีอยู่จริง** (ตอบชัดเจน) — และคำตอบนี้ก็ถูก cache ด้วย (negative cache) |
| `NOERROR` แต่ไม่มี answer | ชื่อมี แต่ไม่มี record ชนิดที่ถาม (เช่นมี A ไม่มี AAAA) |
| `SERVFAIL` | resolver หาคำตอบไม่ได้ (authoritative ล่ม, DNSSEC ผิด) |
| timeout | ติดต่อ DNS server ไม่ได้เลย |

**ลำดับการหาคำตอบบนเครื่อง Linux** (app ส่วนใหญ่ผ่าน `getaddrinfo`):
1. `/etc/hosts` 2. stub resolver (Ubuntu: `systemd-resolved` ที่ `127.0.0.53`) → resolver ที่ได้จาก DHCP
ลำดับนี้กำหนดใน `/etc/nsswitch.conf` (`hosts: files dns`)

**⚠️ `dig` ไม่ได้ถามแบบเดียวกับ app** — dig ถาม DNS server ตรง ๆ **ข้าม `/etc/hosts`**
ถ้าอยากรู้ว่า app จะได้ IP อะไร ใช้ `getent hosts <name>` (Linux) หรือ `dscacheutil -q host -a name <name>` (Mac)

## 4. สิ่งที่เกิดขึ้นข้างใน — ครั้งแรกที่ถาม `www.example.com`

```
app → getaddrinfo("www.example.com")
  1. /etc/hosts มีไหม? ไม่มี
  2. stub resolver (127.0.0.53) cache มีไหม? ไม่มี → ถาม recursive resolver (เช่น router บ้าน / 1.1.1.1)
  3. recursive resolver cache มีไหม? ไม่มี → เริ่มเดินถาม:
       → root server:        "www.example.com?"  → "ไม่รู้ แต่ .com ถาม a.gtld-servers.net"
       → .com server:        "www.example.com?"  → "ไม่รู้ แต่ example.com ถาม ns1.example-dns.net"
       → authoritative:      "www.example.com?"  → "A 104.20.23.154, TTL 300"
  4. resolver cache ไว้ 300 วินาที → ส่งคืน stub → stub cache → app
ทั้งหมดนี้ส่วนใหญ่เป็น UDP port 53 (Lesson 9: ไม่มี handshake)
ครั้งถัดไปภายใน 5 นาที → ได้จาก cache ใน ~0–1 ms
```

## 5. Diagram

```
  ┌───────┐  1. /etc/hosts
  │  app  │───────────────────► (สมุดส่วนตัว)
  └───┬───┘
      │ 2. ถาม stub resolver (127.0.0.53, มี cache)
      ▼
  ┌──────────────────┐   3. ถาม recursive resolver (cache)
  │ systemd-resolved │──────────────────────────┐
  └──────────────────┘                          ▼
                                   ┌──────────────────────┐
                                   │ Recursive resolver   │
                                   │ (1.1.1.1 / ISP / AWS)│
                                   └───┬─────┬─────┬──────┘
                          4a. root ◄───┘     │     └───► 4c. authoritative ของ example.com
                                    4b. .com ◄┘              "A 104.20.23.154 TTL 300"
```

## 6. Example จริง — อ่านผล dig

```
$ dig example.com

;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 45381     ← ผลลัพธ์: NOERROR / NXDOMAIN / SERVFAIL
;; QUESTION SECTION:
;example.com.                   IN      A                      ← ถามอะไร (ชนิด A)
;; ANSWER SECTION:
example.com.            300     IN      A       104.20.23.154  ← ชื่อ  TTL  class  type  ค่า
example.com.            300     IN      A       172.66.147.243 ← มีหลาย IP ได้
;; Query time: 12 msec
;; SERVER: 127.0.0.53#53(127.0.0.53) (UDP)                     ← ใครตอบ และผ่าน UDP
```

## 7. Hands-on Lab

ทำใน VM (มี `dig` แล้วจาก Lesson 9; ถ้าไม่มี `sudo apt install -y dnsutils`)

### ข้อ 1 — อ่านคำตอบ
```bash
dig example.com
dig example.com +short
```
ชี้ให้ได้: status, TTL, IP, ใครเป็นคนตอบ, ใช้เวลาเท่าไร

### ข้อ 2 — cache และ TTL
```bash
dig github.com | grep -E "IN\s+A|Query time"
dig github.com | grep -E "IN\s+A|Query time"
sleep 10
dig github.com | grep -E "IN\s+A|Query time"
```
Expected: ครั้งที่ 2 เร็วกว่ามาก (มาจาก cache ของ `127.0.0.53`) และ TTL ลดลงตามเวลาที่ผ่านไป
คำถาม: cache นี้อยู่ที่ไหน? (ดู `SERVER:`)

### ข้อ 3 — เดินตามลำดับชั้นเอง
```bash
dig +trace example.com
```
Expected: 4 ช่วง — root (`.`) → `com.` → NS ของ example.com → คำตอบ A
(ถ้าติดอยู่ที่ root แปลว่า network ปิดการถาม root server ตรง ๆ — ข้ามข้อนี้ได้)

### ข้อ 4 — record หลายชนิด
```bash
dig www.github.com +noall +answer      # CNAME แล้วค่อย A
dig NS github.com +short
dig MX gmail.com +short
dig AAAA google.com +short
dig TXT google.com +short | head -3
```

### ข้อ 5 — dig โกหกเรื่อง /etc/hosts
```bash
echo "127.0.0.1 pixbin.test" | sudo tee -a /etc/hosts
dig pixbin.test +short                # ได้อะไร?
getent hosts pixbin.test              # ได้อะไร?
curl pixbin.test:8000/
```
Expected: `dig` ไม่ได้อะไร (ถาม DNS server ซึ่งไม่รู้จักชื่อนี้) แต่ `getent` และ `curl` ได้ 127.0.0.1
**นี่คือกับดักที่ทำให้คนเสียเวลา debug เป็นชั่วโมง** — จำไว้ว่าเครื่องมือไหนเห็นอะไร

> ใช้ `.test` เพราะเป็นโดเมนที่สงวนไว้สำหรับทดสอบ (ไม่มีใครจดได้) ห้ามใช้ `.local` บน Mac เพราะจะไปถาม Bonjour (mDNS) ก่อนและช้ามาก

### ข้อ 6 — ตั้งชื่อ pixbin บน Mac (ใช้ต่อใน Lesson 11–13)
```bash
# Mac
multipass list                                    # IP ของ lab
echo "<IP ของ VM> pixbin.test" | sudo tee -a /etc/hosts
dscacheutil -q host -a name pixbin.test
ping -c 1 pixbin.test
```

### ข้อ 7 — ความล้มเหลว 3 แบบ
```bash
# VM
dig this-name-does-not-exist-xyz123.com | grep status
dig dnssec-failed.org @1.1.1.1 | grep status       # โดเมนทดสอบที่ตั้งใจเซ็น DNSSEC ผิด
dig example.com @192.0.2.53 +time=2 +tries=1        # DNS server ที่ไม่มีอยู่จริง
```
Expected: `NXDOMAIN` · `SERVFAIL` · `connection timed out; no servers could be reached`
เขียนอธิบายว่าแต่ละแบบบอกอะไร และถ้า user เห็นใน browser จะเป็นข้อความแบบไหน

### ข้อ 8 — ทำ DNS ของ VM พัง
```bash
resolvectl status | grep -A2 "Current DNS"
sudo resolvectl dns enp0s1 192.0.2.53              # ชี้ไปที่ DNS server ที่ไม่มีอยู่จริง
sudo resolvectl flush-caches
curl -m 5 https://example.com -o /dev/null -w "%{http_code}\n"
curl -m 5 https://1.1.1.1 -o /dev/null -w "%{http_code}\n"
sudo apt update 2>&1 | tail -2
```
Expected: ใช้ชื่อไม่ได้ (`Could not resolve host`, `Temporary failure resolving`) แต่ใช้ IP ได้ — นี่คือหน้าตาของ "It's always DNS"
คืนค่า: `sudo resolvectl revert enp0s1` แล้วลอง `curl` อีกครั้ง

### ตรวจว่าทำถูก
- [ ] อธิบายขั้นตอนใน `dig +trace` ได้ว่าแต่ละชั้นตอบอะไร
- [ ] อธิบายข้อ 5 ได้ว่าทำไม dig กับ curl เห็นไม่เหมือนกัน
- [ ] แยก NXDOMAIN / SERVFAIL / timeout ได้

### Common errors
| อาการ | สาเหตุ |
|---|---|
| Mac `ping pixbin.test` ไม่ได้ | แก้ `/etc/hosts` ผิดไฟล์/ลืม sudo · ล้าง cache `sudo dscacheutil -flushcache; sudo killall -HUP mDNSResponder` |
| `resolvectl dns enp0s1` error | ชื่อ interface ต่าง (ดู `ip -br link`) |
| ข้อ 8 ยังใช้ชื่อได้ | ยังมี cache ใน resolved — `flush-caches` แล้วลองชื่อใหม่ที่ไม่เคยถาม |

## 8. Debugging

**Scenario:** "เว็บเข้าไม่ได้ บางคนเข้าได้ บางคนเข้าไม่ได้ หลังจากเพิ่งย้าย server"

| # | Hypothesis | วิธีพิสูจน์ |
|---|---|---|
| 1 | record ใหม่ยังไม่ถูกตั้ง / ตั้งผิด | ถาม authoritative ตรง ๆ: `dig NS <domain> +short` แล้ว `dig <name> @<ns>` |
| 2 | บาง resolver ยังจำ IP เก่า (TTL ยังไม่หมด) | `dig <name> @1.1.1.1` vs `@8.8.8.8` vs resolver ของ user · ดู TTL ของ record เดิม |
| 3 | เครื่อง user cache / มี /etc/hosts เก่า | `getent hosts` / `dscacheutil` บนเครื่องที่มีปัญหา |
| 4 | DNS ถูกแล้ว แต่ server ใหม่มีปัญหา | `curl --resolve name:443:<ip ใหม่> https://name` บังคับ IP แล้วทดสอบ server ตรง ๆ |

หลัก: **ถาม authoritative ก่อน** (ความจริง) แล้วค่อยไล่ดู cache แต่ละชั้น (สิ่งที่คนเห็น)

**Checklist ก่อนย้าย IP:** ลด TTL เหลือ 60 วินาที **ล่วงหน้าอย่างน้อยเท่ากับ TTL เดิม** → ย้าย → ตรวจ → คืน TTL

## 9. Failure Scenarios

| เกิดอะไร | user เห็น | ข้างใน |
|---|---|---|
| DNS provider ล่ม | ทุกเว็บของบริษัทเข้าไม่ได้ ทั้งที่ server ปกติ | resolver หาคำตอบไม่ได้ → SERVFAIL |
| โดเมนหมดอายุ (ลืมต่อ) | เว็บหาย email หาย | registrar ถอด NS |
| TTL สูง + ย้าย server | บางคนเข้าได้ บางคนไม่ได้ นานเป็นวัน | cache เก่าค้าง |
| resolver ของ server ช้า/ล่ม | app เรียก DB/API ด้วยชื่อแล้วช้าหรือพัง | ทุก connection ใหม่ต้องถาม DNS ก่อน |
| พิมพ์ record ผิด แล้วแก้ | ยังพังต่ออีกพักหนึ่ง | NXDOMAIN ถูก negative cache ไว้ด้วย |

## 10. Trade-offs

**TTL สูง vs ต่ำ** (ข้อ 3) — เลือกต่ำเมื่อมีแผนเปลี่ยนหรือใช้ DNS ทำ failover · สูงเมื่อค่าคงที่และอยากลดภาระ

**DNS load balancing (หลาย A record) vs Load balancer**
- หลาย A record: ง่าย ไม่มีค่าใช้จ่าย — แต่ client เลือกเอง, ไม่รู้ว่า server ไหนตาย, cache ทำให้เอาออกช้า
- Load balancer: ตรวจสุขภาพ เอาตัวตายออกทันที — แต่เป็นอีกชิ้นที่ต้องจ่ายและดูแล (Phase 13)

**Managed DNS (Route 53, Cloudflare) vs DNS server ของตัวเอง**
- managed: กระจายทั่วโลก ทนการโจมตี มี API — จ่ายตามจำนวน query และพึ่ง provider
- ของตัวเอง: ควบคุมได้หมด — แต่ถ้าล่ม ทุกอย่างล่ม และต้องทำให้ทนเองทั้งหมด

## 11. Production Considerations

- **DNS คือ single point of failure ที่คนลืม** — ใช้ provider ที่เชื่อถือได้ ตั้ง auto-renew โดเมน และ monitor ว่าชื่อสำคัญยัง resolve ได้
- **Security:** ใครแก้ DNS ได้ = ยึดเว็บและ email ของคุณได้ → บัญชี registrar/DNS ต้องมี MFA และสิทธิ์น้อยที่สุด
  record ที่ชี้ไปหา resource ที่ลบไปแล้ว (เช่น CNAME ไป bucket ที่ไม่มี) อาจถูกคนอื่นยึด (subdomain takeover)
- **ในระบบภายใน** service เรียกกันด้วยชื่อ (DB, cache) — DNS ภายใน (AWS Route 53 private zone, K8s CoreDNS) จึงสำคัญพอ ๆ กับ DNS สาธารณะ
- **Cost:** managed DNS คิดตามจำนวนโซนและ query — TTL ต่ำมากกับ traffic สูงทำให้ query เพิ่มขึ้นเป็นเงาตามตัว

## 12. Quiz

1. **(เข้าใจ)** ทำไม DNS ถึงออกแบบเป็นลำดับชั้น (root → TLD → authoritative) แทนที่จะมี database กลางที่เดียว
2. **(ประยุกต์)** record ของ pixbin.com มี TTL 3600 คุณเปลี่ยน IP ตอน 10:00 ถาม: ภายในเวลาไหนที่ทุกคนจะเห็น IP ใหม่แน่นอน (คิดแบบ worst case) และควรทำอะไรก่อนหน้านั้นถ้าอยากให้ย้ายเสร็จใน 1 นาที
3. **(debugging)** dev บอกว่า "ผม `dig api.internal` แล้วไม่มีผล แต่ app ใน server เดียวกันเรียก api.internal ได้" อธิบายว่าเป็นไปได้อย่างไร และจะใช้คำสั่งอะไรดูสิ่งที่ app เห็นจริง
4. **(debugging)** แยกให้ออก: user เห็น `DNS_PROBE_FINISHED_NXDOMAIN` กับ user เห็น "This site can't be reached — took too long to respond" ปัญหาอยู่คนละชั้นอย่างไร
5. **(trade-off)** ทีมอยากทำ failover ด้วยการเปลี่ยน DNS ไปที่ server สำรองเมื่อตัวหลักตาย มีข้อจำกัดอะไรบ้าง และ load balancer แก้ข้อจำกัดไหนได้

## 13. Challenge

1. เขียน script `dnscheck.sh <name>` ที่ถามชื่อเดียวกันจาก resolver 3 ตัว (1.1.1.1, 8.8.8.8, ของระบบ) และจาก authoritative ของโดเมนนั้นโดยตรง แสดง IP และ TTL เคียงกัน — ใช้กับ `github.com` และอธิบายความต่างที่เห็น
2. หาเวลาของ DNS ในการโหลดเว็บจริง: `curl -w "dns=%{time_namelookup} connect=%{time_connect}\n" -o /dev/null -s https://<เว็บที่ไม่เคยเข้า>` รันสองครั้ง เทียบ แล้วอธิบายว่าทำไม `time_connect` ของ curl รวม `time_namelookup` ไว้ด้วย
3. ใช้ `sudo tcpdump -i any -n port 53` ขณะรัน `curl https://github.com` ครั้งแรก — นับว่ามีการถาม DNS กี่ครั้ง ถามชนิดอะไรบ้าง (A? AAAA?) และทำไมถึงถามมากกว่าหนึ่ง

## 14. สรุปสิ่งที่ต้องจำ

- DNS = ลำดับชั้น root → TLD → authoritative · recursive resolver เดินถามแทนและ cache ตาม TTL
- A / AAAA / CNAME / NS / MX / TXT
- "propagation" = cache หมดอายุตาม TTL · ลด TTL **ก่อน** ย้าย
- NXDOMAIN = ไม่มีชื่อนี้ · SERVFAIL = หาคำตอบไม่ได้ · timeout = ติดต่อ DNS ไม่ได้
- `dig` ข้าม `/etc/hosts` · อยากรู้ว่า app เห็นอะไร ใช้ `getent hosts`
- IP ได้ ชื่อไม่ได้ → ปัญหา DNS

## 15. เรียนต่อ

**Lesson 11 — HTTP และ Reverse Proxy:** มองข้อความ HTTP ที่วิ่งบน TCP จริง ๆ แล้ววาง nginx ไว้หน้า Pixbin
ให้คนภายนอกเข้าได้ผ่าน port 80 โดยที่ pixbin ยังฟังแค่ 127.0.0.1
