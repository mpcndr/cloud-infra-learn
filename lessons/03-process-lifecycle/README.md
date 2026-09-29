# Lesson 3 — Process เกิด อยู่ ตายอย่างไร (และทำไมปิด terminal แล้ว app ตาย)

Phase 1 · Linux · Level 1–3 · Project: Pixbin Stage 1

## 1. ปัญหาที่เรากำลังแก้

คุณ SSH เข้า server รัน `./pixbin` เว็บใช้ได้ ปิด laptop กลับบ้าน → เว็บล่ม
เช้ามา deploy version ใหม่ด้วย `kill` ตัวเก่าแล้วรันตัวใหม่ → user บางคนได้ error ตรงจังหวะนั้น

ทั้งสองอย่างเกิดจากเรื่องเดียว: เราไม่รู้ว่า process **เกิดจากใคร ผูกอยู่กับอะไร และถูกสั่งให้ตายอย่างไร**

บทนี้ยังเป็นจุดที่เราย้ายจาก Mac ไปทำงานบน **Linux จริง** เพราะ server ใน cloud เกือบทั้งหมดคือ Linux
และทุกอย่างตั้งแต่ Docker ถึง Kubernetes ถูกสร้างบน feature ของ Linux kernel

## 2. Mental Model: ต้นไม้ครอบครัว + ข้อความสั้น

- ทุก process **มีพ่อแม่** (parent) — process ไม่ได้เกิดขึ้นเอง มีคนสั่งให้เกิด
- ต้นตระกูลคือ **PID 1** (บน Ubuntu คือ `systemd`) ซึ่ง kernel สร้างตอน boot
- เวลาคุณพิมพ์ `./pixbin` ใน shell → shell (bash) คือพ่อของ pixbin
- shell เองก็มีพ่อ คือ `sshd` ที่รับ SSH connection ของคุณ

**Signal** = ข้อความสั้น ๆ ที่ส่งถึง process เช่น "ช่วยปิดตัวเองหน่อย" หรือ "ตายเดี๋ยวนี้"
process เลือกได้ว่าจะจัดการ signal ส่วนใหญ่อย่างไร — **ยกเว้น SIGKILL ที่ kernel ฆ่าให้ทันที ไม่ถามใคร**

เมื่อคุณปิด terminal: SSH connection ขาด → `sshd` บอก bash ว่า "สายหลุดแล้ว" (SIGHUP)
→ bash ส่ง SIGHUP ต่อให้ลูก ๆ ทุกตัวที่มันสร้าง → pixbin ไม่ได้เขียนให้รับมือ SIGHUP → ตาย

> ⚠️ ขอบเขตของ analogy: "ครอบครัว" ของจริงไม่ได้ตายตามกันอัตโนมัติ — พ่อตาย ลูกไม่ได้ตายด้วยเสมอ
> ลูกที่พ่อตายจะถูกย้ายไปให้ PID 1 รับเลี้ยง (orphan) ที่ app ตายตอนปิด terminal เพราะ **มีคนส่ง SIGHUP** ไม่ใช่เพราะพ่อตาย

## 3. Concept

**การเกิด: fork + exec**
- `fork()` = process คัดลอกตัวเองเป็นลูก · `exec()` = ลูกเปลี่ยนตัวเองเป็นอีก program
- shell ใช้สองขั้นนี้ทุกครั้งที่คุณรันคำสั่ง

**ข้อมูลที่ process ได้รับตอนเกิด (สืบทอดจากพ่อ)**
- **Environment variables** — `HOST`, `PORT`, `PATH`, ... ลูกได้สำเนาของพ่อ (แก้ของลูก พ่อไม่เปลี่ยน)
- **File descriptors 0, 1, 2** — ช่องที่เปิดไว้แล้ว:
  - `0` stdin (รับ input) · `1` stdout (output ปกติ) · `2` stderr (error และ log)
  - ถ้ารันใน terminal ทั้งสามต่ออยู่กับ terminal
- **User** ที่ใช้รัน (Lesson 4) และ **working directory**

**Signals ที่ต้องรู้**

| Signal | เลข | ความหมาย | process จัดการเองได้ไหม |
|---|---|---|---|
| SIGHUP | 1 | terminal/สายหลุด (หรือใช้บอกให้ reload config) | ได้ |
| SIGINT | 2 | Ctrl+C | ได้ |
| SIGKILL | 9 | ตายทันที | **ไม่ได้** kernel ทำเอง |
| SIGTERM | 15 | ขอให้ปิดตัวเองดี ๆ (default ของ `kill`) | ได้ |
| SIGSTOP / SIGCONT | 19 / 18 | แช่แข็ง / ปลุก (Lesson 1 ข้อ 8) | ไม่ได้ |

**Exit code** — ตัวเลขที่ process คืนให้พ่อตอนตาย
- `0` = สำเร็จ · `1–125` = error ที่ program กำหนดเอง
- `128 + N` = ตายเพราะ signal N ที่ไม่ได้จัดการ → `130` (INT), `137` (KILL), `143` (TERM), `129` (HUP)
- **ตัวเลข 137 คือสิ่งที่จะเจอบ่อยมากใน Docker/K8s** — แปลว่าโดน SIGKILL (มักมาจาก OOM killer, Lesson 6)

**Graceful shutdown** — เมื่อได้ SIGTERM app ที่ดีจะ:
1. หยุดรับ connection ใหม่
2. ทำ request ที่ค้างอยู่ให้เสร็จ
3. ปิด resource (DB connection, ไฟล์) แล้วค่อยออกด้วย exit code 0

Pixbin ทำแบบนี้ (ดู `signal.NotifyContext` ใน `main.go`) แต่ **ไม่ได้จัดการ SIGHUP** — จงใจไว้ให้ lab

## 4. สิ่งที่เกิดขึ้นข้างใน

```
ssh เข้า VM แล้วพิมพ์ ./pixbin

systemd (PID 1)
 └─ sshd (daemon รอรับ SSH)
     └─ sshd (session ของคุณ)
         └─ bash                    ← shell ของคุณ
             └─ pixbin              ← fork จาก bash แล้ว exec เป็น pixbin
                  fd 0,1,2 → terminal ของคุณ (/dev/pts/0)
```

kernel เก็บข้อมูลทุก process ไว้ และเปิดให้เราดูผ่าน **`/proc/<PID>/`** — ไฟล์ปลอมที่ kernel สร้างขึ้นตอนเราอ่าน:
- `/proc/<PID>/status` สถานะ, PPid, memory, จำนวน thread
- `/proc/<PID>/fd/` ไฟล์/socket ที่เปิดอยู่ (เห็น socket ที่ listen port ด้วย)
- `/proc/<PID>/environ` environment variables ที่ process ได้รับตอนเกิด

เมื่อลูกตาย kernel ยังเก็บ exit code ไว้จนกว่าพ่อจะมาอ่าน (`wait()`) ระหว่างนั้นลูกเป็น **zombie** (สถานะ `Z`)
ถ้าพ่อไม่เคยอ่าน zombie จะค้างเต็มตาราง process — เป็น bug ที่เจอในโปรแกรมที่สร้าง process ลูกเยอะ ๆ

## 5. Diagram — ทำไมปิด terminal แล้ว pixbin ตาย

```
 คุณปิดหน้าต่าง terminal
        │
        ▼
 SSH connection ขาด
        │
 sshd (session) รู้ว่าสายหลุด ──SIGHUP──► bash
                                         │ bash ส่งต่อ SIGHUP ให้ job ทุกตัวที่มันสร้าง
                                         ▼
                                      pixbin
                                         │ ไม่มีโค้ดจัดการ SIGHUP → ใช้ default = ตาย
                                         ▼
                                   exit 129 (128+1)  → เว็บล่ม
```

## 6. Example จริง — ตั้ง Linux VM

เราจะใช้ **Multipass** (ของ Canonical) สร้าง Ubuntu VM บน Mac — ทำไมต้องเป็น VM อธิบายละเอียดใน Phase 4
ตอนนี้ให้คิดว่า "มันคือเครื่อง Linux อีกเครื่องที่อยู่ในเครื่องเรา"

```bash
# บน Mac
brew install --cask multipass
multipass launch 24.04 --name lab --cpus 2 --memory 2G --disk 10G
multipass list                        # เห็น lab สถานะ Running พร้อม IP
multipass mount ~/cloud-infra-learn lab:/home/ubuntu/cloud-infra-learn
multipass shell lab                   # เข้าไปใน VM
```
(Windows ก็ติดตั้ง Multipass ได้ แต่ lab นี้เขียนตาม Mac)

**Build pixbin ให้ Linux** — binary ที่ build บน Mac รันบน Linux ไม่ได้ เพราะ OS และรูปแบบไฟล์ต่างกัน
Go ช่วยให้ build ข้าม OS ได้ (cross-compile):
```bash
# บน Mac, ในโฟลเดอร์ pixbin
multipass exec lab -- uname -m      # aarch64 = ARM (Mac M1–M4) / x86_64 = Intel
GOOS=linux GOARCH=arm64 go build -o pixbin-linux .   # Intel ใช้ GOARCH=amd64
file pixbin pixbin-linux             # เทียบ: Mach-O (macOS) vs ELF (Linux)
```

## 7. Hands-on Lab

ทุกข้อทำใน VM (`multipass shell lab`) เปิด 2 หน้าต่าง: **V1** และ **V2**

### ข้อ 1 — ต้นไม้ process
```bash
# V1
cd ~/cloud-infra-learn/pixbin
./pixbin-linux
# V2
ps -ef --forest | grep -B4 pixbin-linux
```
- `-e` ทุก process · `-f` แสดง PPID และคำสั่งเต็ม · `--forest` วาดเป็นต้นไม้
Expected: เห็นสาย `sshd → bash → pixbin-linux` จดว่า PPID ของ pixbin คือ PID ของใคร

### ข้อ 2 — แอบดูผ่าน /proc
```bash
# V2
P=$(pgrep -f pixbin-linux)
grep -E '^(State|PPid|Threads|VmRSS)' /proc/$P/status
ls -l /proc/$P/fd
tr '\0' '\n' < /proc/$P/environ | head
```
- `fd` เห็น 0, 1, 2 ชี้ไปที่ `/dev/pts/N` (terminal) และมี `socket:[...]` — นั่นคือ socket ที่ listen port 8000
- `environ` คั่นด้วย byte ศูนย์ `tr` เปลี่ยนเป็นขึ้นบรรทัดใหม่

### ข้อ 3 — stdout กับ stderr ไม่ใช่ช่องเดียวกัน
```bash
# V1 (Ctrl+C ตัวเดิมก่อน)
./pixbin-linux > out.log
```
ดู V1: log ยังขึ้นหน้าจอ! แล้ว `out.log` มีอะไร?
```bash
./pixbin-linux 2> err.log      # ลองอีกแบบ
```
Hypothesis ของคุณคือ? Pixbin เขียน log ไปที่ fd ไหน? พิสูจน์ด้วย `ls -l /proc/$P/fd`

### ข้อ 4 — signal แต่ละแบบให้ผลต่างกัน
รัน pixbin ใน V1 แล้วจาก V2 ส่ง signal ต่อไปนี้ทีละครั้ง (รัน pixbin ใหม่ทุกครั้ง) หลังมันตาย ใน V1 รัน `echo $?` ดู exit code

| คำสั่งใน V2 | V1 เห็น log อะไร | exit code |
|---|---|---|
| `kill -TERM $(pgrep -f pixbin-linux)` | ? | ? |
| `kill -INT $(pgrep -f pixbin-linux)` | ? | ? |
| `kill -HUP $(pgrep -f pixbin-linux)` | ? | ? |
| `kill -KILL $(pgrep -f pixbin-linux)` | ? | ? |

กรอกตารางให้ครบ ทำนายก่อนรันทุกแถว แล้วอธิบายว่าทำไม TERM ได้ 0 แต่ KILL ได้ 137

### ข้อ 5 — graceful shutdown ในจังหวะที่มี request ค้าง
```bash
# V2
curl "localhost:8000/work/sleep?ms=5000" &    # request ที่ใช้เวลา 5 วินาที
sleep 1; kill -TERM $(pgrep -f pixbin-linux)
```
Expected: curl ยังได้ `slept 5000ms` ครบ และ V1 เห็น `got shutdown signal ...` แล้วตามด้วย `bye`
ทำซ้ำด้วย `kill -KILL` — curl ได้อะไร? (`curl: (52) Empty reply from server` หรือ `(56) Connection reset by peer`) นี่คือสิ่งที่ user เจอตอน deploy แบบฆ่าทิ้ง

### ข้อ 6 — ปิด terminal แล้ว app ตาย
```bash
# บน Mac เปิด terminal หน้าต่างใหม่ (เรียกว่า M1)
multipass shell lab
cd ~/cloud-infra-learn/pixbin && ./pixbin-linux
```
จากนั้น **ปิดหน้าต่าง M1 ทิ้งไปเลย** (Cmd+W — ห้ามพิมพ์ `exit` ห้าม Ctrl+C) แล้วใน V2:
```bash
pgrep -fa pixbin-linux          # ยังอยู่ไหม?
```
ลองอีกรอบ: เปิดหน้าต่างใหม่ `multipass shell lab` แล้วรัน
```bash
cd ~/cloud-infra-learn/pixbin && nohup ./pixbin-linux > pixbin.log 2>&1 &
```
ปิดหน้าต่างอีกครั้ง แล้วใน V2 ดูว่ายังอยู่ไหม ถ้ายังอยู่ PPID ของมันเป็นใคร (`ps -o pid,ppid,cmd -p $(pgrep -f pixbin-linux)`)

`> pixbin.log 2>&1` = ส่ง stdout ไปไฟล์ แล้วส่ง stderr (2) ไปที่เดียวกับ stdout (1)
ปิด pixbin ตัวนี้ด้วย `pkill -f pixbin-linux` ก่อนทำข้อต่อไป

> `nohup` ทำให้รอดจาก SIGHUP ได้ แต่ **ไม่ใช่วิธีรัน production**: ถ้ามันตาย ไม่มีใครรันใหม่ ถ้าเครื่อง reboot ก็ไม่มีใครรันให้
> วิธีที่ถูกคือให้ systemd ดูแล → Lesson 5

### ตรวจว่าทำถูก
- [ ] วาดต้นไม้ process ของ pixbin จาก PID 1 ได้
- [ ] ตารางข้อ 4 ครบ และอธิบายได้ว่าเลข 129, 137 มาจากไหน
- [ ] อธิบายได้ว่าทำไมปิด terminal แล้ว pixbin ตาย แต่ `nohup` รอด

### Common errors
| อาการ | สาเหตุ |
|---|---|
| `cannot execute binary file: Exec format error` | build ผิด OS/arch (ลืม GOOS=linux หรือ GOARCH ไม่ตรงกับ `uname -m`) |
| `Permission denied` ตอนรัน | ไฟล์ไม่มีสิทธิ์ execute (`chmod +x pixbin-linux`) — Lesson 4 |
| `multipass mount` error | เปิด Privacy & Security → Full Disk Access ให้ multipassd หรือใช้ `multipass transfer pixbin-linux lab:` แทน |
| `pgrep` เจอหลายตัว | มี pixbin ค้างจากข้อก่อน `pkill -f pixbin-linux` |

## 8. Debugging

**Scenario:** "app ตายเองตอนตี 3 ไม่มีใครแตะ"

| # | Hypothesis | วิธีพิสูจน์ |
|---|---|---|
| 1 | crash เพราะ bug (panic) | ดู stderr/log ช่วงนั้น — Go panic จะพิมพ์ stack trace, exit code 2 |
| 2 | โดน SIGKILL จาก OOM killer | exit code 137 + `dmesg` / `journalctl -k` มีคำว่า `Out of memory: Killed process` (Lesson 6) |
| 3 | ผูกกับ session ที่หลุด | รันด้วย SSH ค้างไว้? exit code 129 |
| 4 | มีคน/script ส่ง SIGTERM | exit 0 หรือ 143 + log shutdown · ดู cron / deploy script |

**หลัก: exit code คือหลักฐานชิ้นแรก** เก็บให้ได้เสมอ — process manager ที่ดี (systemd, Docker, K8s) บันทึกให้

## 9. Failure Scenarios

| เกิดอะไร | อาการ | ข้างใน |
|---|---|---|
| deploy ด้วย `kill -9` | user บางคนได้ connection reset | request ที่ค้างถูกตัดกลางคัน |
| app ไม่จัดการ SIGTERM | shutdown ช้าไปจนหมดเวลา แล้วโดน KILL | Docker/K8s ส่ง TERM แล้วรอ (default 10s/30s) ก่อนส่ง KILL |
| log เขียนลงไฟล์ด้วย `>` แต่ app เขียน stderr | "ไม่มี log เลย" | redirect ผิด fd |
| process ลูกไม่ถูก `wait()` | zombie เต็ม | พ่อไม่เก็บ exit code ของลูก |
| container ที่ app เป็น PID 1 | กด Ctrl+C / docker stop แล้วไม่ตาย | PID 1 ไม่มี default signal handler (จะเจอใน Phase 5) |

## 10. Trade-offs

**Graceful shutdown timeout ยาว vs สั้น**
- ยาว (เช่น 60s): request ยาว ๆ ทำเสร็จทัน แต่ deploy ช้า และถ้า app ค้างจริงต้องรอนาน
- สั้น (เช่น 5s): deploy เร็ว แต่ request ยาวโดนตัด
- เลือกจาก request ที่ยาวที่สุดที่ยอมรับได้ งานยาวกว่านั้นไม่ควรทำใน request → ย้ายไป background worker (Phase 14)

**Log ไป stdout/stderr vs เขียนไฟล์เอง**
- stdout/stderr: app ไม่ต้องรู้เรื่อง rotation, disk, path — ให้ systemd/Docker เก็บให้ (หลัก 12-factor)
- ไฟล์เอง: ควบคุมได้มาก แต่ต้องจัดการ disk เต็ม, rotate, permission เอง
- สมัยนี้นิยมแบบแรก Pixbin จึงเขียน log ออก stderr

## 11. Production Considerations

- **ห้ามรัน service ด้วย SSH session / nohup / screen** — ใช้ process manager (systemd, Docker, K8s)
- **ทุก app ต้องจัดการ SIGTERM** และ timeout ของ app ต้องสั้นกว่า timeout ที่ process manager รอ
- **Environment variables คือที่ใส่ config** — แต่ระวัง: `/proc/<PID>/environ` อ่านได้โดย user เดียวกันและ root
  secret ใน env จึงไม่ได้ "ซ่อน" (Phase 11 จะพูดถึงทางเลือก)
- **Cost:** process ที่ตายแล้วไม่มีใครรันใหม่ = downtime ซึ่งแพงกว่าค่า server เสมอ

## 12. Quiz

1. **(เข้าใจ)** ทำไม SIGKILL ถึงไม่ยอมให้ process จัดการเอง ถ้า process ปฏิเสธ SIGKILL ได้ จะเกิดปัญหาอะไร?
2. **(ประยุกต์)** container ตายด้วย exit code 137 และอีกตัวตายด้วย 143 สองแบบนี้บอกอะไรต่างกัน?
3. **(debugging)** เพื่อนรัน `./app > app.log` บน server แล้วบอกว่า "app.log ว่างเปล่า แต่ app ทำงานอยู่" คุณจะตั้ง hypothesis อะไร และพิสูจน์อย่างไรโดยไม่ต้องหยุด app?
4. **(architecture)** ระหว่าง deploy เราต้องหยุดตัวเก่าแล้วเริ่มตัวใหม่ ออกแบบลำดับขั้นตอนที่ทำให้ user ไม่เห็น error เลย (ใบ้: ช่วงเวลาที่ไม่มีใครฟัง port เกิดตอนไหน)
5. **(trade-off)** ทำไม `nohup` ถึงไม่พอสำหรับ production ถึงแม้ app จะไม่ตายตอนปิด terminal แล้ว

## 13. Challenge

1. แก้ Pixbin (ในเครื่องคุณ ไม่ต้อง commit) ให้เมื่อได้ **SIGHUP** แค่พิมพ์ log `reloading config` แล้วทำงานต่อ ไม่ตาย พิสูจน์ด้วยข้อ 6
2. เขียน Go program สั้น ๆ ที่สร้าง zombie process ได้ แล้วให้เห็นสถานะ `Z` ใน `ps` (ใบ้: `exec.Command(...).Start()` แล้วไม่เรียก `Wait()`) จากนั้นทำให้ zombie หายไปโดยไม่แก้โค้ด
3. อธิบายด้วยต้นไม้ process: เมื่อ `nohup` แล้วปิดหน้าต่าง ทำไม PPID ของ pixbin ถึงเปลี่ยน และเปลี่ยนเป็นใคร
4. ทำข้อ 6 ซ้ำ แต่รัน `./pixbin-linux &` (ไม่มี nohup) แล้วพิมพ์ `exit` แทนการปิดหน้าต่าง ผลต่างจากการปิดหน้าต่างไหม? หาคำตอบว่าทำไม (ใบ้: `shopt huponexit` ใน bash)

## 14. สรุปสิ่งที่ต้องจำ

- process ทุกตัวมีพ่อ สืบทอด env, fd 0/1/2, user และ working directory
- signal: TERM = ขอให้ปิดดี ๆ · KILL = ตายทันที (จัดการไม่ได้) · HUP = สายหลุด · INT = Ctrl+C
- exit code 128+N = ตายเพราะ signal N → **137 = KILL** (มักเป็น OOM), 143 = TERM
- ปิด terminal → SIGHUP → app ที่ไม่จัดการก็ตาย
- app ที่ดีต้อง graceful shutdown เมื่อได้ SIGTERM
- `/proc/<PID>/` คือหน้าต่างเข้าไปดู process จากมุมของ kernel

## 15. เรียนต่อ

**Lesson 4 — Filesystem, Users และ Permissions:** process รันในนามของ "ใคร" และทำไมไม่ควรรันเป็น root
