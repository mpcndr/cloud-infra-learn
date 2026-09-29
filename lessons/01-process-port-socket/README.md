# Lesson 1 — โปรแกรมรันอยู่ แต่ทำไมคนอื่นเข้าไม่ได้?

Phase 0 · Level 0–3 (Mental Model → Debugging) · Project: Pixbin Stage 1

## 1. ปัญหาที่เรากำลังแก้

คุณเขียน web app เสร็จ รัน `python3 app.py` เปิด browser ที่ `http://localhost:8000` เห็นหน้าเว็บ
ส่งลิงก์ให้เพื่อน → เพื่อนเปิดไม่ได้

ก่อนจะพูดถึง Cloud, DNS, Load Balancer อะไรทั้งนั้น ต้องตอบคำถามพื้นฐานที่สุดให้ได้ก่อน:

> "โปรแกรมที่รันอยู่" กับ "request ที่วิ่งมาจากเครื่องอื่น" มาเจอกันได้อย่างไร?

ทุกอย่างใน cloud (container, load balancer, kubernetes) คือการจัดการเรื่องนี้ในระดับที่ใหญ่ขึ้น
ถ้าบทนี้ไม่แน่น ทุกบทหลังจากนี้จะเป็นเวทมนตร์

## 2. Mental Model

นึกถึงตึกสำนักงาน:

- **ตึก** = เครื่อง (machine) — มี **ที่อยู่** = IP address
- **ห้องในตึก** = port (เลข 0–65535)
- **พนักงานในห้อง** = process (โปรแกรมที่กำลังรัน)
- **ช่องรับจดหมายที่พนักงานเปิดไว้** = socket
- **แผนกต้อนรับชั้นล่าง** = OS kernel — ทุกจดหมายต้องผ่านที่นี่ แล้ว kernel เป็นคนดูว่าจ่าหน้าถึงห้องไหน

จดหมายมาถึง "ตึก X ห้อง 8000":
- มีคนเปิดช่องรับไว้ → ส่งให้คนนั้น
- ไม่มีใครเปิดช่องห้องนั้น → แผนกต้อนรับตีกลับทันที ("ไม่มีคนนี้") = **connection refused**
- ยามหน้าตึกโยนจดหมายทิ้งเงียบ ๆ → คนส่งรอเก้อ = **timeout**

> ⚠️ ขอบเขตของ analogy: ของจริงไม่มี "ห้อง" ที่มีอยู่จริง port เป็นแค่ตัวเลขใน header ของ packet
> process หนึ่งตัวเปิดได้หลาย port, และเครื่องหนึ่งมี "ที่อยู่" ได้หลายอัน (หนึ่งอันต่อ network interface)
> ซึ่งข้อหลังนี่แหละที่เป็นต้นเหตุของปัญหาในบทนี้

## 3. Concept

**Program vs Process**
- Program = ไฟล์บน disk (`app.py`, `/usr/bin/python3`) — นิ่ง ไม่ทำอะไร
- Process = program ที่ OS โหลดเข้า memory แล้วกำลังรัน มีเลขประจำตัว **PID**, มี memory ของตัวเอง, มี owner (user)
- รัน program เดียวกัน 2 ครั้ง = 2 process, 2 PID

**Process แตะ network เองไม่ได้** — ต้องขอ kernel ผ่าน *system call* เสมอ
(process ถูกขังใน "user space" ส่วน hardware เช่น network card อยู่ในมือ kernel)

**Socket** = "handle" ที่ kernel ให้ process ใช้ส่ง/รับข้อมูลผ่าน network

**IP address** = ที่อยู่ของ *network interface* ไม่ใช่ของเครื่อง เครื่องปกติมีอย่างน้อย:
- `127.0.0.1` (loopback, `lo`) — interface ปลอมที่วนกลับเข้าเครื่องตัวเอง **ออกนอกเครื่องไม่ได้**
- `192.168.x.x` หรือเลขอื่น (เช่น `wlan0`, `eth0`) — interface ที่ต่อกับ network จริง

**Port** = เลข 16-bit ที่บอกว่า data ชิ้นนี้เป็นของ socket ไหนบนเครื่อง

**Bind address** — ตอน server เปิด socket ต้องบอกว่า "ฟังที่ IP ไหน":
- `127.0.0.1` → รับเฉพาะคนที่มาจากในเครื่องเดียวกัน
- `0.0.0.0` → รับจากทุก interface ของเครื่อง
- `192.168.1.20` → รับเฉพาะที่เข้ามาทาง interface นั้น

## 4. สิ่งที่เกิดขึ้นข้างใน

เมื่อ `app.py` เริ่มทำงาน Python เรียก system call ตามลำดับนี้ (ทุก web server ทุกภาษาทำเหมือนกัน):

```
socket()   ขอ kernel สร้าง socket (ได้เลข file descriptor กลับมา)
bind()     ผูก socket กับ (IP, port) เช่น (127.0.0.1, 8000)
           → ถ้ามี socket อื่นจองคู่นี้อยู่แล้ว: EADDRINUSE "Address already in use"
           → ถ้า port < 1024 และไม่ใช่ root: EACCES "Permission denied"
listen()   บอก kernel ว่า "socket นี้รอรับ connection"
           → kernel เริ่มรับ TCP handshake แทนเรา แล้วเอา connection ที่เสร็จแล้วไปต่อคิว (backlog)
accept()   process ดึง connection ออกจากคิวมาทีละอัน → ได้ socket ใหม่สำหรับ client คนนั้น
recv/send  อ่าน request, เขียน response
```

จุดที่ต้องเข้าใจให้ขาด: **TCP handshake ทำโดย kernel ไม่ใช่ app**
ดังนั้น "connect ได้" ไม่ได้แปลว่า "app ยังดีอยู่" (Lab ข้อ 8 จะพิสูจน์)

เมื่อ packet มาถึงเครื่อง kernel ดู (IP ปลายทาง, port ปลายทาง):
- ตรงกับ socket ที่ listen อยู่ → เข้าคิว
- ไม่ตรงกับอะไรเลย → kernel ส่ง TCP **RST** กลับ → client เห็น "connection refused" *ทันที*
- ถ้า firewall drop ทิ้งก่อนถึงขั้นนี้ → client ไม่ได้อะไรกลับเลย → รอจน **timeout**

แต่ละ connection ถูกระบุด้วย 4 ค่า: `(client IP, client port, server IP, server port)`
client port เป็นเลขสุ่มที่ kernel ฝั่ง client เลือกให้ (*ephemeral port*) — นี่คือเหตุผลที่ server port เดียวรับได้หลายพัน connection

## 5. Diagram

```
 เครื่องเพื่อน (192.168.1.35)                   เครื่องคุณ
 ┌──────────────┐                    ┌──────────────────────────────────────┐
 │ browser      │                    │  user space                          │
 │  connect to  │                    │   ┌───────────────────────────┐      │
 │  .1.20:8000  │                    │   │ process: python3 app.py   │      │
 └──────┬───────┘                    │   │ PID 547                   │      │
        │                            │   │ socket bound 127.0.0.1:8000│     │
        │                            │   └────────────▲──────────────┘      │
        │                            │  ──────────────┼──── system call ────│
        │                            │  kernel        │                     │
        │  packet                    │   socket table:                      │
        │  dst=192.168.1.20:8000     │   127.0.0.1:8000 LISTEN  → PID 547   │
        └───────────────────────────►│   192.168.1.20:8000 ???  → ไม่มี      │
                  ◄──── RST ─────────│   → "connection refused"            │
                                     │                                      │
                                     │   interfaces: lo=127.0.0.1           │
                                     │               wlan0=192.168.1.20     │
                                     └──────────────────────────────────────┘
```

Request จาก browser บนเครื่องเดียวกัน (`localhost:8000`) วิ่งผ่าน `lo` → ตรงกับ socket → ได้
Request จากเพื่อนเข้าทาง `wlan0` → ไม่มี socket ไหนฟังที่ `192.168.1.20:8000` → ถูกตีกลับ

## 6. Example จริง

ดูไฟล์ [`app.py`](app.py) — web server ที่เล็กที่สุดที่ยังบอกเราได้ว่า:
PID ของตัวเอง, ฟังอยู่ที่ไหน, และ client มาจาก IP:port อะไร

```python
HOST = os.environ.get("HOST", "127.0.0.1")   # อ่านจาก environment variable
PORT = int(os.environ.get("PORT", "8000"))
HTTPServer((HOST, PORT), Handler).serve_forever()  # ← ข้างในคือ socket/bind/listen/accept
```

สังเกตว่า HOST/PORT มาจาก environment variable ไม่ได้ hardcode — เพราะอีกไม่กี่บท
เราจะรัน app ตัวเดียวกันใน container / cloud ที่ต้องเปลี่ยนค่าพวกนี้โดยไม่แก้ code

## 7. Hands-on Lab

**ต้องมี:** เครื่อง Linux / macOS / Windows+WSL2, Python 3, curl
(ถ้ามีมือถือต่อ Wi-Fi เดียวกันจะทำข้อ 7 ได้ครบ)

เปิด terminal 2 หน้าต่าง เรียกว่า **T1** และ **T2**

**จะใช้ Python หรือ Go ก็ได้** — `main.go` ทำงานเหมือน `app.py` ทุกอย่าง
ถ้าใช้ Go ให้ `go build -o pixbin main.go` แล้วรัน `./pixbin` (ใช้แทน `python3 app.py` ในทุกข้อ)
อย่าใช้ `go run` ใน lab นี้ เพราะ `go run` จะ compile แล้วสร้าง process ลูกขึ้นมาอีกตัว คุณจะได้ 2 PID ซ้อนกัน
(ลองดูด้วย `ps` ก็ได้ว่ามันเป็นแบบนั้นจริงไหม)

**คำสั่งตามแต่ละ OS**

| ทำอะไร | macOS | Windows (PowerShell) |
|---|---|---|
| ดู process | `ps -p <PID> -o pid,ppid,user,rss,etime,command` | `Get-Process -Id <PID>` |
| ใครฟัง port ไหน | `lsof -nP -iTCP -sTCP:LISTEN` | `Get-NetTCPConnection -State Listen -LocalPort 8000` |
| IP ใน LAN | `ipconfig getifaddr en0` | `ipconfig` (ดู IPv4 Address) |
| ฆ่า process | `kill <PID>` | `Stop-Process -Id <PID>` |
| แช่แข็ง / ปลุก | `kill -STOP` / `kill -CONT` | ไม่มี command ในตัว → ทำข้อ 8 บน Mac |
| curl | มีอยู่แล้ว | ใช้ `curl.exe` (คำว่า `curl` ใน PowerShell เป็น alias ของคำสั่งอื่น) |

แนะนำให้ทำ lab ชุดนี้บน **Mac** เป็นหลัก เพราะคำสั่งเกือบทั้งหมดเหมือนของ Linux
แต่ต้องรู้ไว้ว่า macOS **ไม่ใช่ Linux** (kernel เป็นตระกูล BSD) — ถึง Phase 1 เราจะใช้ Linux จริงบน VM

### ข้อ 1 — รัน app
```bash
# T1
cd lessons/01-process-port-socket
python3 app.py
```
Expected: `[pid 12345] listening on 127.0.0.1:8000` และ terminal ค้างอยู่ (process ยังรัน)

### ข้อ 2 — เรียกมัน
```bash
# T2
curl http://127.0.0.1:8000
curl http://127.0.0.1:8000
```
Expected: `you came from: 127.0.0.1:5xxxx` — **จดเลข port ทั้งสองครั้ง** มันเหมือนกันไหม? เพราะอะไร?

### ข้อ 3 — ดู process
```bash
# T2 (แทน 12345 ด้วย PID จริง)
ps -p 12345 -o pid,ppid,user,rss,etime,command
```
- `pid` เลขประจำตัว · `ppid` process แม่ (ใครสั่งให้มันเกิด — ลองเดาก่อนดูว่าเป็นอะไร)
- `user` รันในสิทธิ์ใคร · `rss` ใช้ RAM จริงกี่ KB · `etime` รันมานานเท่าไร

### ข้อ 4 — ใครฟังอยู่ที่ port ไหน
```bash
# Linux
ss -tlnp            # t=TCP l=listening n=แสดงตัวเลขไม่แปลงเป็นชื่อ p=บอก process
# macOS (หรือ Linux ที่ไม่มี ss)
lsof -nP -iTCP -sTCP:LISTEN
```
Expected: เห็นบรรทัด `127.0.0.1:8000` พร้อม `python3` และ PID เดียวกับข้อ 1
command นี้คือ **เครื่องมือตัวแรกที่ต้องใช้เสมอ** เวลามีคนบอกว่า "เข้า service ไม่ได้"

### ข้อ 5 — แย่ง port
```bash
# T2
python3 app.py
```
Expected: `OSError: [Errno 98] Address already in use` (macOS: Errno 48)
ก่อนอ่านต่อ ให้เขียน hypothesis เองว่าเกิดจากอะไร แล้วพิสูจน์ด้วย command จากข้อ 4

จากนั้นลอง: `PORT=8001 python3 app.py` → ได้ไหม? ตอนนี้มีกี่ process, กี่ socket? (ตรวจด้วยข้อ 4) แล้ว Ctrl+C ตัวที่ 8001

### ข้อ 6 — process ตาย
```bash
# T2
kill 12345          # ส่ง signal SIGTERM ไปบอก process ให้จบตัวเอง
curl -m 3 http://127.0.0.1:8000
```
Expected: `Failed to connect ... Couldn't connect to server` (= connection refused) และมัน **ขึ้นทันที**
เพราะ kernel ตอบ RST กลับเองว่าไม่มีใครฟัง port นี้

### ข้อ 7 — ให้เครื่องอื่นเข้า
```bash
# หา IP ของเครื่องใน LAN
ip -4 addr          # Linux  (ดู inet ของ wlan0/eth0)
ipconfig getifaddr en0   # macOS
```
รัน app ใหม่ (T1: `python3 app.py`) แล้วจากมือถือเปิด `http://<IP ที่ได้>:8000`
หรือจากเครื่องตัวเอง: `curl -m 3 http://<IP ที่ได้>:8000`
→ **เข้าไม่ได้** (อธิบายด้วย diagram ในข้อ 5 ของบทเรียน)

Ctrl+C แล้วรันใหม่:
```bash
HOST=0.0.0.0 python3 app.py
```
ตรวจด้วย `ss -tlnp` ว่าตอนนี้ฟังที่ไหน แล้วลองจากมือถืออีกครั้ง
ดูบรรทัด `you came from:` — ตอนนี้เป็น IP ของมือถือ

ถ้ายังเข้าไม่ได้: host firewall (macOS จะเด้งถาม, Linux ดู `sudo ufw status`), หรือ Wi-Fi สาธารณะที่ตั้ง *client isolation* ไว้ — ให้สังเกตว่าอาการเป็น refused หรือ timeout

### ข้อ 8 — process ยังอยู่ แต่ไม่ตอบ (ข้อสำคัญที่สุด)
```bash
# T2
kill -STOP <PID>                   # แช่แข็ง process (ยังไม่ตาย ยังเป็นเจ้าของ socket)
curl -m 3 http://127.0.0.1:8000    # -m 3 = ยอมรอสูงสุด 3 วินาที
```
Expected: `Operation timed out after 3000 milliseconds with 0 bytes received`
สังเกต: ไม่ใช่ "couldn't connect" — มัน **connect ได้** แต่ไม่ได้ข้อมูลกลับเลย

```bash
kill -CONT <PID>                   # ปลุก
```
แล้วดู T1 — เกิดอะไรขึ้นกับ request ที่ค้างไว้?

### ตรวจว่าทำถูก
- [ ] บอกได้ว่าทำไม client port เปลี่ยนทุกครั้ง
- [ ] ชี้ใน output ของ `ss`/`lsof` ได้ว่า socket ไหนเป็นของ PID ไหน และ bind ที่ IP อะไร
- [ ] อธิบายได้ว่าทำไมข้อ 6 error ทันที แต่ข้อ 8 รอจนหมดเวลา

### Common errors
| อาการ | สาเหตุที่เป็นไปได้ |
|---|---|
| `python3: command not found` | ยังไม่ติดตั้ง / ชื่อเป็น `python` |
| `ss: command not found` | ใช้ `lsof` แทน (macOS ไม่มี `ss`) |
| `lsof` ไม่แสดง process | process เป็นของ user อื่น → ใส่ `sudo` |
| `kill: No such process` | PID ผิด หรือ process ตายไปแล้ว |

## 8. Debugging

**Methodology** (ใช้ตลอดทั้งคอร์ส):
Observe → Hypothesis → Measure → Eliminate → Root cause → Fix → Verify → Prevent

**Scenario:** "เพื่อนบอกเข้า `http://192.168.1.20:8000` ไม่ได้"

อย่าเริ่มจากเดา ให้ตั้งคำถามทีละชั้นจากล่างขึ้นบน แต่ละ command ใช้พิสูจน์ hypothesis หนึ่งข้อ:

| # | Hypothesis | วิธีพิสูจน์ | ถ้าผลคือ... |
|---|---|---|---|
| 1 | process ไม่ได้รันอยู่ | `ps aux \| grep app.py` | ไม่เจอ → start มัน / ดูว่าทำไมมันตาย |
| 2 | รันอยู่แต่ไม่ได้ listen port นั้น | `ss -tlnp \| grep 8000` | ไม่เจอ → ดู log ตอน start (port ชน? config ผิด?) |
| 3 | listen แต่ bind ผิด address | ดูคอลัมน์ Local Address | `127.0.0.1` → ต้องเป็น `0.0.0.0` หรือ IP ของ LAN |
| 4 | app ในเครื่องเองยังตอบไม่ได้ | `curl -m 3 localhost:8000` บน server | timeout → app ค้าง (ไม่ใช่ network) |
| 5 | จากนอกเครื่องโดนกั้น | `curl -m 3 192.168.1.20:8000` จากเครื่องอื่น | refused = ถึงเครื่องแต่ไม่มีคนฟัง / timeout = มีอะไร drop ระหว่างทาง (firewall) |

หลักที่ต้องจำ: **แยก "ถึงเครื่องไหม" ออกจาก "app ตอบไหม" ก่อนเสมอ**
refused กับ timeout บอกข้อมูลคนละเรื่องกัน — อย่าเรียกรวมว่า "เข้าไม่ได้"

## 9. Failure Scenarios

| เกิดอะไร | user เห็นอะไร | ข้างในเกิดอะไร |
|---|---|---|
| process crash | refused ทันที | ไม่มี socket listen → kernel ส่ง RST |
| process ค้าง (deadlock, ติด loop, รอ DB) | หมุนนานแล้ว timeout | kernel ยังรับ handshake ให้ แต่ app ไม่ accept/ไม่ตอบ |
| deploy version ใหม่แล้ว port ชน | version ใหม่ start ไม่ขึ้น | `EADDRINUSE` ใน log — ตัวเก่ายังไม่ปล่อย |
| bind 127.0.0.1 บน server | "ผมรัน curl บน server ได้นะ" แต่ข้างนอกเข้าไม่ได้ | ข้อ 7 ของ lab |
| คิว backlog เต็ม (traffic เยอะ, app ช้า) | บางคนเข้าได้ บางคน timeout | kernel ไม่รับ connection ใหม่จนกว่าคิวจะว่าง |
| ปิด terminal ที่รัน app | เว็บล่ม | process ได้ signal SIGHUP แล้วตาย (บทถัดไป) |

## 10. Trade-offs

**Bind `127.0.0.1` vs `0.0.0.0`**
- `127.0.0.1`: ปลอดภัยโดย default — ต่อให้ลืมตั้ง firewall คนข้างนอกก็เข้าไม่ได้ แต่ถ้าต้องให้คนนอกเข้าก็ใช้ไม่ได้
- `0.0.0.0`: เข้าได้จากทุกที่ที่ network ยอม — ความปลอดภัยไปฝากไว้กับ firewall แทน
- ใน production จริงมักใช้แบบผสม: app bind `127.0.0.1` แล้วมี reverse proxy (nginx) bind `0.0.0.0:443` อยู่ข้างหน้า
  → คนนอกแตะได้แค่ proxy (Phase 3) · แต่ใน container มักต้อง bind `0.0.0.0` (Phase 5 จะเห็นว่าทำไม)
- Database (Postgres, Redis) ควร bind ที่ไหน? — ข่าว "ข้อมูลรั่วหลายล้าน record" จำนวนมากเริ่มจาก Redis/MongoDB ที่ bind `0.0.0.0` บน server ที่มี public IP โดยไม่มี password

**Port ต่ำกว่า 1024 (เช่น 80, 443)** ต้องใช้สิทธิ์ root
- รัน app ทั้งตัวเป็น root: ง่าย แต่ถ้า app มีช่องโหว่ คนร้ายได้ root ทั้งเครื่อง
- ทางเลือก: รัน app port สูงหลัง reverse proxy / ให้ capability `CAP_NET_BIND_SERVICE` / ให้ load balancer ทำหน้าที่แทน

## 11. Production Considerations

- **อย่าเชื่อว่า "port เปิดอยู่" = "service ดี"** (lab ข้อ 8) → health check ต้องส่ง request จริงและตรวจ response (Phase 10, 14)
- **ตั้ง timeout ทุกครั้งที่เรียก service อื่น** — ไม่งั้น process ที่ค้างตัวหนึ่งจะลากทุกคนที่เรียกมันค้างตาม (ต้นเหตุ outage ใหญ่ ๆ หลายครั้ง)
- **Config จาก environment variable** ไม่ใช่ hardcode (หลัก 12-factor app)
- **process ที่รันจาก terminal ไม่ใช่ production** — ต้องมี process manager ที่คอย start ตอน boot และ restart ตอนตาย (systemd, Lesson ถัด ๆ ไป)
- **Security:** ทุก port ที่ listen บน `0.0.0.0` คือประตูที่ต้องตอบได้ว่า "ใครควรเข้าได้" — ตรวจ server จริงด้วย `ss -tlnp` เป็นนิสัย

**Cost (มองไปข้างหน้า):** ใน cloud คุณจ่ายเงินค่า "process ที่รันอยู่" (compute ต่อชั่วโมง) ไม่ว่าจะมีคนเรียกหรือไม่
process ที่ค้าง (ข้อ 8) = จ่ายเงินแต่ไม่ได้งาน และทำให้ต้องเพิ่มเครื่องโดยไม่จำเป็น

## 12. Quiz

ตอบด้วยภาษาตัวเอง อธิบาย *เหตุผล* ไม่ใช่แค่คำตอบ

1. **(เข้าใจ)** ทำไม process ถึงเปิด network เองตรง ๆ ไม่ได้ ต้องผ่าน kernel? ถ้าให้ทุก process แตะ network card ได้ตรง ๆ จะเกิดปัญหาอะไร?
2. **(ประยุกต์)** server หนึ่งเครื่อง app ฟัง port 8000 อยู่ ทำไมรับ user พร้อมกัน 1,000 คนได้ ทั้งที่ port มีอันเดียว? kernel แยก response ของแต่ละคนออกจากกันได้อย่างไร?
3. **(debugging)** user รายงาน 2 แบบ: A "กดแล้ว error ทันที" และ B "หมุน 30 วินาทีแล้ว error" สำหรับแต่ละแบบ คุณสงสัยอะไร และจะตรวจอะไรเป็นอย่างแรก?
4. **(debugging)** คุณ SSH เข้า server รัน `curl localhost:8000` ได้ผลปกติ แต่ user ข้างนอกเข้าไม่ได้ ระบุ hypothesis อย่างน้อย 3 ข้อ เรียงตามที่ควรตรวจก่อน-หลัง พร้อมบอกว่าตรวจด้วยอะไร
5. **(trade-off)** ทีมเสนอว่า "bind ทุก service เป็น `0.0.0.0` ไปเลย จะได้ไม่มีปัญหาเข้าไม่ได้" คุณเห็นด้วยไหม? ถ้าไม่ จะเสนออะไรแทน?

## 13. Challenge

ห้ามดูเฉลย ออกแบบและทำเอง:

1. รัน Pixbin **2 instance** พร้อมกันบนเครื่องเดียว โดยที่เรียกได้ทั้งคู่
2. instance หนึ่งต้องเข้าได้จากเครื่องอื่นใน LAN อีกตัวต้องเข้าได้ **เฉพาะจากในเครื่อง** — พิสูจน์ด้วย output
3. เขียน shell script `check.sh <host> <port>` ที่พิมพ์ผลออกมาเป็นหนึ่งใน 3 แบบ: `UP`, `REFUSED`, `TIMEOUT` (ใบ้: ดู exit code ของ `curl`)
4. ใช้ script ข้อ 3 พิสูจน์ทั้ง 3 สถานะกับ app จริง (ต้องทำให้เกิดทั้ง 3 แบบเอง)
5. พยายามรัน app ที่ port 80 แบบ user ปกติ บันทึก error แล้วอธิบายว่าทำไม — และเสนอวิธีที่ **ไม่** ต้องรัน app เป็น root

ส่งกลับมา: command ที่ใช้, output, และคำอธิบาย reasoning ของแต่ละข้อ

## 14. สรุปสิ่งที่ต้องจำ

- Program = ไฟล์ · Process = program ที่กำลังรัน มี PID
- Process คุยกับ network ผ่าน kernel ด้วย socket: `socket → bind → listen → accept`
- IP เป็นของ *interface* — `127.0.0.1` ออกนอกเครื่องไม่ได้, `0.0.0.0` = ทุก interface
- Connection = (client IP, client port, server IP, server port)
- **Refused** = ถึงเครื่องแล้ว แต่ไม่มีใครฟัง · **Timeout** = มีอะไรกั้น หรือ app ค้าง
- TCP handshake ทำโดย kernel → "connect ได้" ≠ "app ดี"
- เครื่องมือแรก: `ps`, `ss -tlnp` / `lsof`, `curl -m`

## 15. เรียนต่อ

**Lesson 2 — Process ใช้ชีวิตอย่างไร:** ทำไมปิด terminal แล้ว app ตาย?
signals (SIGTERM / SIGKILL / SIGHUP), parent/child process, stdout/stderr ไปไหน,
exit code, และการทำให้ app เป็น service ที่ OS ดูแลด้วย systemd
