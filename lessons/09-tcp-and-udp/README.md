# Lesson 9 — TCP และ UDP: ข้างใน connection เกิดอะไรขึ้น

Phase 2 · Networking · Level 2–5 · Project: Pixbin Stage 1

## 1. ปัญหาที่เรากำลังแก้

Network จริงไม่น่าเชื่อถือ: packet หายได้, มาช้าได้, มาสลับลำดับได้, มาซ้ำได้
แต่เวลาคุณโหลดรูปจาก Pixbin รูปต้องมาครบทุก byte เรียงถูกทุก byte

ใครทำให้สิ่งที่ไม่น่าเชื่อถือกลายเป็นสิ่งที่เชื่อถือได้? → **TCP**
แล้วทำไมบางอย่าง (DNS, video call, เกม) ถึงเลือก **ไม่** ใช้ความน่าเชื่อถือนั้น? → **UDP**

อีกเหตุผลที่ต้องเข้าใจบทนี้: **latency ของระบบคุณไม่ได้เท่ากับระยะทาง แต่เท่ากับ "ระยะทาง × จำนวนรอบที่ต้องคุยกัน"**
ถ้าไม่รู้ว่ามีกี่รอบ จะไม่มีทางรู้ว่าทำไม API ข้ามทวีปช้าขนาดนั้น

## 2. Mental Model

**TCP = โทรศัพท์**
- ต้องโทรติดก่อน ("ฮัลโหล" – "ฮัลโหล ได้ยินครับ" – "โอเค") = **handshake**
- ทุกประโยคที่พูด อีกฝ่ายต้องตอบว่า "ได้ยินแล้ว" (ACK) ถ้าเงียบนาน ๆ ก็พูดซ้ำ
- ประโยคมาถึงตามลำดับเสมอ · จบแล้วต้องบอกลากัน (FIN)

**UDP = ส่งโปสการ์ด**
- ไม่ต้องนัด เขียนแล้วส่งเลย ไม่รู้ว่าถึงไหม ไม่รู้ว่าถึงก่อนหรือหลังใบอื่น
- เร็ว เบา ไม่มีภาระ — เหมาะกับข้อความสั้น ๆ หรือเรื่องที่ "ข้อมูลเก่า = ไร้ค่า" (ภาพ video เฟรมที่หายไปแล้ว ส่งซ้ำก็ไม่มีประโยชน์)

> ⚠️ ขอบเขตของ analogy: โทรศัพท์จริงจองสายไว้ตลอด แต่ TCP ไม่ได้จองอะไรใน network เลย
> "connection" มีอยู่แค่ใน memory ของเครื่องปลายทั้งสองฝั่ง (ใน kernel) — router ระหว่างทางไม่รู้จัก connection (ยกเว้น NAT/firewall ที่จด conntrack)

## 3. Concept

**TCP ให้อะไร**
- **Connection**: handshake 3 ขั้น `SYN → SYN-ACK → ACK` ก่อนส่งข้อมูล
- **Reliability**: ทุก byte มี sequence number ปลายทางตอบ ACK ถ้าไม่ได้ ACK ในเวลาที่กำหนด → ส่งซ้ำ (retransmit)
- **Ordering**: เรียงข้อมูลตาม sequence ก่อนส่งให้ app
- **Flow control**: ผู้รับบอกว่ายังรับได้อีกเท่าไร (window) ผู้ส่งไม่ส่งเกิน
- **Congestion control**: ถ้าเห็นว่า packet หาย สันนิษฐานว่า network แน่น → ลดความเร็วลง

**ราคาของ TCP**
- ต้องเสีย **1 RTT** (round-trip time = ไปกลับหนึ่งรอบ) สำหรับ handshake ก่อนส่ง byte แรก
- packet หายหนึ่งตัว ข้อมูลที่ตามมาต้องรอ (head-of-line blocking)
- ต้องเก็บ state ทุก connection

**UDP ให้อะไร** — แค่ส่ง datagram จาก port ไป port พร้อม checksum ไม่มีอะไรอีกเลย
- ใช้ใน: DNS (คำถามสั้น ๆ ส่งซ้ำเองได้), video/voice, เกม, และ **QUIC / HTTP/3** (สร้างความน่าเชื่อถือเองบน UDP เพื่อลดรอบ handshake)

**สถานะ connection ที่เจอบ่อยใน `ss -tan`**

| State | ความหมาย |
|---|---|
| `LISTEN` | server รอรับ (Lesson 1) |
| `SYN-SENT` | client ส่ง SYN แล้ว รอคำตอบ — ค้างอยู่นาน = ปลายทาง DROP (firewall) |
| `ESTAB` | connection ใช้งานอยู่ |
| `TIME-WAIT` | ฝั่งที่ปิดก่อนรอสักพัก (~60 วินาที บน Linux) เผื่อ packet หลงมา |
| `CLOSE-WAIT` | อีกฝั่งปิดแล้ว แต่ app ฝั่งนี้ **ยังไม่ปิด** — ถ้าค้างเยอะ = bug ใน app (ลืม close) |

## 4. สิ่งที่เกิดขึ้นข้างใน — request หนึ่งครั้ง

`curl http://localhost:8000/` ดูด้วย tcpdump (ผลจริง):

```
client.45094 > server.8000: Flags [S],  seq 3003848400                  ← SYN: ขอเชื่อมต่อ
server.8000 > client.45094: Flags [S.], seq 2651792700, ack 3003848401  ← SYN-ACK: ตกลง
client.45094 > server.8000: Flags [.],  ack 1                           ← ACK: handshake เสร็จ (1 RTT)
client.45094 > server.8000: Flags [P.], seq 1:78, length 77             ← HTTP request 77 bytes
server.8000 > client.45094: Flags [.],  ack 78                          ← "ได้รับถึง byte 78 แล้ว"
server.8000 > client.45094: Flags [P.], seq 1:236, length 235           ← HTTP response
client.45094 > server.8000: Flags [.],  ack 236
client.45094 > server.8000: Flags [F.]                                  ← client ขอปิด
server.8000 > client.45094: Flags [F.]                                  ← server ปิดด้วย
client.45094 > server.8000: Flags [.]                                   ← จบ → client เข้า TIME-WAIT
```
Flags: `S` = SYN · `.` = ACK · `P` = PUSH (มีข้อมูล) · `F` = FIN · `R` = RST (ตัดทิ้ง — ที่มาของ "connection refused" ใน Lesson 1)

**สูตรคิด latency แบบหยาบ** (HTTP ธรรมดา ไม่มี TLS):
```
เวลาถึง byte แรกของ response ≈ 1 RTT (handshake) + 1 RTT (request→response) + เวลาที่ server ประมวลผล
```
กรุงเทพ ↔ US (RTT ~200ms) → อย่างน้อย 400ms ก่อนเห็นอะไร แม้ server ตอบใน 1ms
เพิ่ม HTTPS อีกอย่างน้อย 1 RTT (Lesson 12) → 600ms · นี่คือเหตุผลที่ **connection reuse (keep-alive)** และ **CDN** สำคัญมาก

## 5. Diagram

```
 Client                                   Server
   │ ── SYN ──────────────────────────────► │  ┐
   │ ◄─────────────────────────── SYN-ACK ─ │  │ 1 RTT (ยังไม่มีข้อมูลเลย)
   │ ── ACK + HTTP request ───────────────► │  ┘
   │                                        │  server ประมวลผล
   │ ◄──────────────────── HTTP response ── │    1 RTT
   │ ── FIN ─────────────────────────────►  │
   │ ◄─────────────────────────────── FIN ─ │
   │ ── ACK ─────────────────────────────►  │
 TIME-WAIT (~60s)                         CLOSED

 UDP:
 Client ── datagram ──────────────────────► Server   (จบ ไม่มีอะไรตอบ ถ้า app ไม่ตอบเอง)
```

## 6. Example จริง — วัดเวลาแต่ละช่วงด้วย curl

```bash
curl -s -o /dev/null -w "connect=%{time_connect} ttfb=%{time_starttransfer} total=%{time_total}\n" http://localhost:8000/
```
- `time_connect` = เวลาจนถึง handshake เสร็จ (≈ 1 RTT)
- `time_starttransfer` (TTFB) = เวลาจนได้ byte แรกของ response
- `total` = ทั้งหมด
คำสั่งนี้คือเครื่องมือแรกที่ใช้ตอบคำถาม "ช้าเพราะ network หรือเพราะ server" — ถ้า connect เร็วแต่ TTFB ช้า = server คิดนาน

## 7. Hands-on Lab

ทำใน VM · pixbin (systemd) ฟังที่ 127.0.0.1:8000
```bash
sudo apt install -y tcpdump netcat-openbsd
```

### ข้อ 1 — ดู handshake ด้วยตาตัวเอง
```bash
# V1
sudo tcpdump -i lo -n 'tcp port 8000'
# V2
curl localhost:8000/
```
- `-i lo` = ดักที่ loopback · `-n` = ไม่แปลงชื่อ · `'tcp port 8000'` = filter เฉพาะที่สนใจ
จับคู่แต่ละบรรทัดกับ diagram ข้อ 5 หา handshake, request, response, การปิด

### ข้อ 2 — connection refused ในระดับ packet
```bash
# V1 (ยังดักอยู่) เปลี่ยน filter
sudo tcpdump -i lo -n 'tcp port 8009'
# V2
curl localhost:8009/
```
Expected: `[S]` ตามด้วย `[R.]` — นี่คือ RST ที่ kernel ส่งกลับ (Lesson 1 ข้อ 4)

### ข้อ 3 — dropped: SYN-SENT ค้าง
ใส่กฎ firewall ชั่วคราวให้ทิ้ง packet ที่เข้า port 8010 เงียบ ๆ (DROP):
```bash
sudo iptables -I INPUT -p tcp --dport 8010 -j DROP
# V1
sudo tcpdump -i lo -n 'tcp port 8010'
# V2
curl -m 10 http://localhost:8010/ &
sleep 2; ss -tan state syn-sent
wait
sudo iptables -D INPUT -p tcp --dport 8010 -j DROP     # ลบกฎทิ้ง
```
- `-I INPUT` = แทรกกฎไว้บนสุดของ chain ขาเข้า · `-D` = ลบกฎที่ตรงกันเป๊ะ
Expected: `ss` เห็น connection ในสถานะ `SYN-SENT` และ curl ได้ `Connection timed out after 10000 milliseconds`
ใน V1 เห็น `[S]` หลายครั้ง — kernel ส่ง SYN ซ้ำเอง สังเกตระยะห่างระหว่างแต่ละครั้ง (1s, 2s, 4s ...) นี่คือ exponential backoff

### ข้อ 4 — TIME-WAIT
```bash
for i in $(seq 20); do curl -s localhost:8000/ > /dev/null; done
ss -tan state time-wait '( dport = :8000 or sport = :8000 )'
```
Expected: TIME-WAIT ประมาณ 20 รายการ — ฝั่งไหนเป็นคนปิดก่อน? ดูว่า port 8000 อยู่คอลัมน์ไหน

### ข้อ 5 — ทำให้ network ช้า แล้วดูว่ามันถูกคูณ
```bash
curl -s -o /dev/null -w "connect=%{time_connect} ttfb=%{time_starttransfer}\n" localhost:8000/
sudo tc qdisc add dev lo root netem delay 50ms
curl -s -o /dev/null -w "connect=%{time_connect} ttfb=%{time_starttransfer}\n" localhost:8000/
```
- `tc qdisc ... netem delay 50ms` = ให้ kernel หน่วงทุก packet ที่ออกทาง `lo` 50ms
  บน loopback ทั้งขาไปและขากลับผ่าน `lo` จึงได้ **RTT = 100ms**
**ทำนายก่อนรัน**: connect จะประมาณเท่าไร? TTFB เท่าไร?
Expected: connect ≈ 0.1s, TTFB ≈ 0.2s — ทั้งที่ pixbin ตอบใน 1ms

ลอง keep-alive: ขอ 2 ครั้งใน connection เดียว
```bash
curl -s -o /dev/null -o /dev/null -w "connects=%{num_connects} ttfb=%{time_starttransfer}\n" localhost:8000/ localhost:8000/
```
ครั้งที่ 2 ใช้ connection เดิม (`connects=0`) — เร็วขึ้นเท่าไร?

**ลบ delay ทิ้ง** (สำคัญ ไม่งั้นทุกอย่างใน VM จะช้า): `sudo tc qdisc del dev lo root`

### ข้อ 6 — UDP ไม่มีใครบอกว่าไม่มีคนรับ
```bash
# TCP ไปที่ port ที่ไม่มีคนฟัง
nc -z -w 2 127.0.0.1 9999; echo "tcp exit=$?"
# UDP ไปที่ port ที่ไม่มีคนฟัง
echo hi | nc -u -w 1 127.0.0.1 9999; echo "udp exit=$?"
```
Expected: TCP ล้มเหลว (exit 1) · UDP "สำเร็จ" (exit 0) ทั้งที่ไม่มีใครได้รับ

แล้วลองแบบมีคนรับ:
```bash
# V1
nc -u -l 9999
# V2
echo "hello over udp" | nc -u -w 1 127.0.0.1 9999
```

### ข้อ 7 — DNS ใช้ UDP
```bash
# V1
sudo tcpdump -i any -n 'udp port 53'
# V2
dig example.com @1.1.1.1
```
Expected: คำถามหนึ่ง packet คำตอบหนึ่ง packet — ไม่มี handshake เลย (เร็วกว่า TCP อย่างน้อย 1 RTT)

### ตรวจว่าทำถูก
- [ ] ชี้ใน tcpdump ได้ว่าบรรทัดไหนคือ handshake, ข้อมูล, การปิด, RST
- [ ] อธิบายตัวเลขในข้อ 5 ด้วยจำนวน RTT
- [ ] อธิบายได้ว่าทำไมข้อ 6 UDP ถึง "สำเร็จ"

### Common errors
| อาการ | สาเหตุ |
|---|---|
| `Error: Specified qdisc kind is unknown` | kernel ไม่มี module netem → `sudo apt install -y linux-modules-extra-$(uname -r)` แล้วลองใหม่ |
| tcpdump ไม่เห็นอะไร | ดักผิด interface (`lo` vs `enp0s1`) หรือ filter port ผิด |
| ทุกอย่างใน VM ช้าผิดปกติ | ลืมลบ netem: `sudo tc qdisc del dev lo root` |

## 8. Debugging

**Scenario:** "API ช้า 800ms แต่ log ของ server บอกว่าตอบใน 20ms"

1. **Hypothesis:** เวลาหายไประหว่างทาง ไม่ใช่ใน server
2. **Measure:** จากฝั่ง client: `curl -w "dns=%{time_namelookup} connect=%{time_connect} tls=%{time_appconnect} ttfb=%{time_starttransfer}"`
3. **อ่านผล:**
   - `dns` สูง → DNS ช้า (Lesson 10)
   - `connect` สูง → RTT สูง (ไกล) หรือ SYN หาย ต้องส่งซ้ำ (ดู retransmit)
   - `tls − connect` สูง → TLS handshake (Lesson 12)
   - `ttfb − tls` สูงกว่าที่ server log บอก → คิวก่อนถึง app / proxy ช้า
4. **ตรวจ packet loss:** `ss -ti` ดู `retrans` ของ connection · `mtr <host>` ดู loss ต่อ hop
5. **Fix ตามต้นเหตุ:** ใช้ connection ซ้ำ, ย้าย server ใกล้ user, CDN, ลดจำนวนรอบการเรียก

## 9. Failure Scenarios

| เกิดอะไร | อาการ | ข้างใน |
|---|---|---|
| packet loss 1–2% | throughput ตก, latency กระโดดเป็นช่วง | retransmit + congestion control ลดความเร็ว |
| CLOSE-WAIT สะสมหลายพัน | fd หมด (Lesson 6) | app ไม่ปิด connection ที่อีกฝั่งปิดแล้ว |
| TIME-WAIT เต็ม ephemeral port | client ที่เปิด connection ใหม่ถี่มาก ๆ เชื่อมต่อไม่ได้ | ไม่ใช้ keep-alive / connection pool |
| service A เรียก B ทีละ connection ใหม่ทุก request | latency สูงเกินจำเป็น + CPU ไปกับ handshake | ไม่มี connection reuse |
| firewall ตัด connection ที่เงียบนาน | connection "ตาย" แบบไม่รู้ตัว request แรกหลังเงียบค้าง | conntrack หมดอายุ — ต้องมี TCP keepalive |

## 10. Trade-offs

**TCP vs UDP**
- TCP: ครบ ถูกต้อง เรียงลำดับ — แต่มี handshake, head-of-line blocking, state ต่อ connection
- UDP: เร็ว เบา ส่ง broadcast/multicast ได้ — แต่ความน่าเชื่อถือต้องทำเองใน app
- เลือก UDP เมื่อ: ข้อมูลเก่าไร้ค่า (real-time), ข้อความเล็กมาก (DNS), หรือต้องการควบคุม protocol เอง (QUIC)

**Connection ใหม่ทุกครั้ง vs Connection pool**
- ใหม่ทุกครั้ง: ง่าย ไม่มี state ค้าง — แต่จ่าย handshake ทุกครั้ง (+TLS อีก) และ TIME-WAIT สะสม
- pool: เร็ว ประหยัด — แต่ต้องจัดการ connection ที่ตายเงียบ ๆ, จำนวนสูงสุด, และ database จำกัดจำนวน connection (Phase 12)

## 11. Production Considerations

- **Latency budget:** นับจำนวน RTT ในทุก request เสมอ — DNS + TCP + TLS + request = อย่างน้อย 3–4 RTT สำหรับ connection ใหม่
- **ใช้ connection reuse ทุกที่:** HTTP keep-alive, DB connection pool, gRPC/HTTP2
- **Timeout ต้องมีทุกชั้น** (connect timeout, read timeout) — ค่า default ของหลาย library คือ "รอตลอดไป"
- **Monitor:** retransmit rate, จำนวน connection แต่ละ state (CLOSE-WAIT พุ่ง = bug)
- **Cost:** ใน cloud คุณจ่ายตามข้อมูลที่ส่งออก ไม่ใช่ตามจำนวน packet — แต่ handshake/TLS ที่ไม่จำเป็นกิน CPU ของ load balancer ที่คิดเงินตามปริมาณงาน (Phase 13)

## 12. Quiz

1. **(เข้าใจ)** ทำไม TCP ต้องมี handshake ก่อนส่งข้อมูล ถ้าข้ามไปส่ง request ใน packet แรกเลยจะเสียอะไร
2. **(ประยุกต์)** user ที่ลอนดอนเรียก API ที่ server อยู่กรุงเทพ RTT 250ms ใช้ HTTPS แบบ connection ใหม่ทุกครั้ง server ตอบใน 10ms ประมาณเวลาที่ user รอก่อนได้ byte แรก ถ้าใช้ keep-alive ครั้งที่สองจะเหลือเท่าไร (TLS 1.3 = 1 RTT)
3. **(debugging)** `ss -tan` บน app server เห็น `CLOSE-WAIT` 3,000 รายการ และเพิ่มขึ้นเรื่อย ๆ บอกอะไรเกี่ยวกับ app และจะเกิดอะไรต่อถ้าไม่แก้
4. **(debugging)** client เห็น connection ค้างที่ `SYN-SENT` ต่างจากได้ `RST` กลับมาอย่างไร แต่ละแบบชี้ไปที่ปัญหาอะไร
5. **(trade-off)** ทำไม HTTP/3 ถึงเลือกสร้างบน UDP แทนที่จะปรับปรุง TCP ต่อ

## 13. Challenge

1. เขียน Go program สั้น ๆ ที่เปิด connection 100 เส้นไป pixbin **แล้วไม่ปิด** ดูผลใน `ss -tan` ทั้งฝั่ง client และ server แล้วฆ่า program — state เปลี่ยนเป็นอะไร ฝั่งไหน
2. ใช้ `tc qdisc add dev lo root netem loss 10%` (packet หาย 10%) แล้ววัดเวลา `curl` 20 ครั้ง เก็บค่า min / median / max อธิบายว่าทำไมค่า max ถึงกระโดดสูงมาก (ใบ้: retransmission timeout) — ลบกฎเมื่อเสร็จ
3. อธิบาย (ไม่เกิน 10 บรรทัด) ให้ dev ในทีมฟังว่าทำไม "เรียก API ภายนอก 5 ตัวแบบต่อกันทีละตัว" ถึงช้ากว่า "เรียกพร้อมกัน" มาก ใช้ตัวเลข RTT ประกอบ

## 14. สรุปสิ่งที่ต้องจำ

- TCP: handshake (SYN, SYN-ACK, ACK) → ส่ง/ACK/ส่งซ้ำ → FIN · UDP: ส่งแล้วจบ
- RST = ปฏิเสธทันที (refused) · SYN ไม่มีคนตอบ = SYN-SENT ค้าง (timeout)
- เวลาถึง byte แรก ≈ (จำนวนรอบ × RTT) + เวลา server — ลดจำนวนรอบสำคัญพอ ๆ กับทำ server ให้เร็ว
- CLOSE-WAIT สะสม = app ลืมปิด connection
- ใช้ keep-alive / pool, ตั้ง timeout ทุกชั้น
- `curl -w` แยกเวลาแต่ละช่วง · `tcpdump` ดูของจริง · `ss -tan` ดู state

## 15. เรียนต่อ

**Phase 3 — Web · Lesson 10 — DNS:** browser รู้ได้อย่างไรว่า google.com คือ IP อะไร — และทำไม "It's always DNS"
