# Lesson 2 — ข้างในเครื่องมีอะไร: CPU, RAM, Disk และ Kernel

Phase 0 · Level 0–3 · Project: Pixbin Stage 1

## 1. ปัญหาที่เรากำลังแก้

มีคนบอกว่า "เว็บช้า" — คำนี้ไม่ได้บอกอะไรเลย ถ้าเราไม่รู้ว่า **ช้าเพราะรออะไร**

request หนึ่งอันใช้เวลา 2 วินาที อาจเป็นเพราะ:
- CPU คำนวณหนักจริง
- CPU ว่าง แต่ process กำลัง **รอ** database / disk / network
- RAM ไม่พอ เครื่องเลยเอา disk มาใช้แทน RAM (ช้ากว่าเป็นพันเท่า)
- มี request อื่นแย่ง CPU อยู่

แต่ละแบบแก้คนละวิธี ถ้าแยกไม่ออก จะ "เพิ่มเครื่อง" ไปเรื่อย ๆ แล้วไม่หาย (และจ่ายเงินฟรี)
บทนี้คือการรู้จัก "ทรัพยากร" ที่ process ทุกตัวต้องใช้ และรู้ว่า **อะไรช้ากว่าอะไรกี่เท่า**

## 2. Mental Model: ครัวร้านอาหาร

- **CPU core** = พ่อครัว (1 core = พ่อครัว 1 คน ทำได้ทีละอย่าง)
- **RAM** = โต๊ะเตรียมของ — หยิบเร็ว แต่พื้นที่จำกัด และ **ปิดร้าน (ดับไฟ) ของบนโต๊ะหายหมด**
- **Disk** = ห้องเก็บของหลังร้าน — ใหญ่ ของไม่หายเมื่อปิดร้าน แต่เดินไปหยิบช้ากว่ามาก
- **Network** = สั่งของจากร้านอื่น — ช้าที่สุด และบางทีของไม่มา
- **Kernel** = ผู้จัดการครัว — ตัดสินว่าพ่อครัวคนไหนทำออเดอร์ไหน ใครได้ใช้โต๊ะเท่าไร ห้ามใครเข้าห้องเก็บของเอง
- **Process** = ออเดอร์แต่ละใบ

ถ้าออเดอร์ "ช้า" อาจเป็นเพราะ: พ่อครัวไม่พอ (CPU), โต๊ะเต็มต้องวิ่งเข้าออกห้องเก็บของ (RAM → swap), รอของจากร้านอื่น (I/O)

> ⚠️ ขอบเขตของ analogy: พ่อครัวจริงทำอาหารจานเดียวจนเสร็จ แต่ CPU สลับงานไปมาหลายพันครั้งต่อวินาที (ข้อ 4)
> และ "รอของ" ใน analogy ดูเหมือนพ่อครัวยืนรอ — ของจริง kernel ให้ core ไปทำงานอื่นระหว่างรอ core ไม่ได้ว่าง

## 3. Concept

**CPU**
- core หนึ่งรันคำสั่งได้ทีละ thread ในแต่ละขณะ
- เครื่องมี 4 core → รันจริง ๆ พร้อมกันได้ 4 อย่าง ที่เหลือต้องต่อคิว
- **CPU-bound** = งานที่ช้าเพราะคำนวณ (ย่อรูป, เข้ารหัส, บีบอัด)
- **I/O-bound** = งานที่ช้าเพราะรอ (query DB, เรียก API, อ่านไฟล์) — web app ส่วนใหญ่เป็นแบบนี้

**RAM (memory)**
- ที่เก็บข้อมูลของ process ขณะรัน เร็ว แต่ **volatile** (ไฟดับ = หาย) และแพง
- แต่ละ process เห็น memory ของตัวเองเท่านั้น (virtual memory) — process A อ่าน memory ของ B ไม่ได้
- **RSS** (Resident Set Size) = RAM จริงที่ process ใช้อยู่ตอนนี้
- RAM เต็มแล้วเกิดอะไร → Lesson 6 (swap, OOM killer)

**Disk (storage)**
- ข้อมูลอยู่รอดหลังปิดเครื่อง (**persistent**)
- SSD เร็วกว่า HDD มาก แต่ยังช้ากว่า RAM หลายร้อยถึงพันเท่า
- ใน cloud "disk" มักอยู่อีกเครื่องหนึ่ง ต่อผ่าน network (เช่น AWS EBS) — ช้ากว่า disk ในเครื่องอีก

**Kernel กับ User space**
- process อยู่ใน **user space** แตะ hardware ตรง ๆ ไม่ได้
- ทุกครั้งที่อยากอ่านไฟล์, ส่ง network, ขอ memory → เรียก **system call** ให้ kernel ทำแทน
- ทำไม? ถ้าทุก process แตะ hardware ได้เอง: process หนึ่ง bug → เขียนทับข้อมูลของคนอื่น / ทำเครื่องล่มทั้งเครื่อง / ขโมยข้อมูลกันได้
- kernel จึงเป็นทั้ง **ผู้แบ่งทรัพยากร** และ **ผู้คุมความปลอดภัย**

## 4. สิ่งที่เกิดขึ้นข้างใน

**Scheduler — ทำไม 4 core รัน process ได้หลายร้อยตัว**

kernel แบ่งเวลา CPU เป็นช่วงสั้น ๆ (ระดับ millisecond) แล้วสลับ process ที่พร้อมทำงานเข้าออก core
เร็วจนดูเหมือนทุกตัวรันพร้อมกัน — เรียกว่า **time-slicing** / **context switch**

process ในเครื่องส่วนใหญ่ **ไม่ได้อยากใช้ CPU** — มันกำลังรออะไรบางอย่าง (รอ network, รอ user กด) kernel จึงไม่ให้ CPU กับตัวที่รออยู่

```
สถานะของ process (ที่ต้องรู้ตอนนี้)
  Running / Runnable (R)  กำลังรัน หรือพร้อมรันแต่รอคิว core
  Sleeping (S)            รอเหตุการณ์ (network, timer) — ไม่ใช้ CPU เลย
  Disk wait (D)           รอ disk แบบขัดจังหวะไม่ได้
```

**ตัวเลขที่ Engineer ทุกคนต้องจำ "ลำดับขนาด" ได้** (ค่าโดยประมาณ)

| การกระทำ | เวลา | ถ้า 1 ns = 1 วินาที |
|---|---|---|
| คำสั่ง CPU 1 คำสั่ง / อ่าน L1 cache | ~1 ns | 1 วินาที |
| อ่าน RAM | ~100 ns | ~2 นาที |
| อ่าน SSD (random) | ~100 µs | ~1 วัน |
| ส่ง packet ไปกลับในศูนย์ข้อมูลเดียวกัน | ~0.5 ms | ~6 วัน |
| อ่าน HDD (seek) | ~10 ms | ~4 เดือน |
| กรุงเทพ ↔ สิงคโปร์ ไปกลับ | ~30 ms | ~1 ปี |
| กรุงเทพ ↔ US ไปกลับ | ~200 ms | ~6 ปี |

ข้อสรุปจากตาราง: **network กับ disk ช้ากว่า CPU เป็นล้านเท่า** เว็บส่วนใหญ่จึง "ช้าเพราะรอ" ไม่ใช่ "ช้าเพราะคิด"
และนี่คือเหตุผลที่ cache (Phase 13) และการวาง server ใกล้ user (region, CDN) มีผลมหาศาล

## 5. Diagram

```
                 ┌────────────── user space ──────────────┐
                 │  pixbin (PID 812)   postgres   nginx    │
                 │  memory ของตัวเอง   ของตัวเอง   ของตัวเอง │
                 └────────┬──────────────┬───────────┬─────┘
                          │   system call (read, write, send, mmap ...)
 ────────────────────────────────────────────────────────────────
                 ┌────────▼──────── kernel ──────────▼─────┐
                 │ scheduler   memory manager   filesystem  │
                 │ network stack   device drivers           │
                 └───┬──────────┬──────────┬───────────┬────┘
                     ▼          ▼          ▼           ▼
                  CPU cores    RAM        Disk     Network card
                  (เร็วสุด)    (~100ns)  (~100µs)  (~0.5ms–200ms)
```

## 6. Example จริง

Pixbin (ในโฟลเดอร์ [`pixbin/`](../../pixbin/main.go)) มี endpoint ไว้จำลองงานแต่ละแบบ:

| Endpoint | จำลองอะไร |
|---|---|
| `/work/cpu?n=1000` | งาน CPU-bound — hash ซ้ำ n พันครั้ง (งานปริมาณคงที่) |
| `/work/sleep?ms=1000` | งาน I/O-bound — รอเฉย ๆ เหมือนรอ database |
| `/work/mem?mb=100` | จอง RAM เพิ่ม 100 MB แล้วไม่คืน (memory leak) |

Pixbin เขียน log ทุก request ออกหน้าจอ พร้อมระยะเวลา: `GET /work/cpu?n=1000 200 312ms`

## 7. Hands-on Lab

ทำบน Mac, terminal 2 หน้าต่าง (T1, T2)

### ข้อ 1 — build และดูว่าเครื่องมีกี่ core
```bash
cd cloud-infra-learn/pixbin
go build -o pixbin .
sysctl -n hw.ncpu              # จำนวน core (Linux: nproc)
sysctl -n hw.memsize           # RAM ทั้งหมด (byte)
```
จดตัวเลข core ไว้ เรียกว่า **C**

### ข้อ 2 — วัดงาน CPU หนึ่งชิ้น
```bash
# T1
./pixbin
# T2
curl "localhost:8000/work/cpu?n=3000"
```
Expected: `hashed 3000k times in 0.8s` (เวลาขึ้นกับเครื่อง) — จดไว้ เรียกว่า **T**

### ข้อ 3 — ยิงพร้อมกัน C ตัว แล้ว 2C ตัว
```bash
# แทน 8 ด้วยค่า C ของคุณ
time (for i in $(seq 8); do curl -s "localhost:8000/work/cpu?n=3000" & done; wait)
# แล้วลอง 2C
time (for i in $(seq 16); do curl -s "localhost:8000/work/cpu?n=3000" & done; wait)
```
- `for ... & done` = ยิง curl หลายตัวพร้อมกันเป็น background, `wait` = รอให้ครบ, `time` = จับเวลาทั้งก้อน
- **ก่อนรัน ให้ทำนายเวลาของทั้งสองแบบก่อน** แล้วค่อยเทียบกับของจริง
- ระหว่างรัน เปิด Activity Monitor → CPU หรือ `top -o cpu` ใน T3 ดู `%CPU` ของ pixbin (100% = 1 core เต็ม)

ตัวอย่างผลจริงบนเครื่อง 4 core: ยิง 4 ตัว ~1.2s, ยิง 8 ตัว ~1.8s (แต่ละ request ช้าลงเพราะแย่ง core กัน)

### ข้อ 4 — ยิงงานรอ 100 ตัวพร้อมกัน
```bash
time (for i in $(seq 100); do curl -s "localhost:8000/work/sleep?ms=1000" >/dev/null & done; wait)
```
ทำนายก่อน: 100 request × 1 วินาที จะใช้เวลาเท่าไร? แล้ว CPU ของ pixbin ขึ้นไหม?
Expected: เสร็จในประมาณ 1 วินาทีกว่า ๆ และ CPU แทบไม่ขยับ — ทำไม?

### ข้อ 5 — ดู memory ของ process
```bash
PID=$(pgrep pixbin)
ps -o pid,rss,command -p $PID          # rss หน่วย KB
curl "localhost:8000/work/mem?mb=200"
ps -o pid,rss,command -p $PID
curl "localhost:8000/work/mem?mb=200"
ps -o pid,rss,command -p $PID
```
Expected: RSS เพิ่มขึ้นทีละ ~200 MB และ **ไม่ลดลง** จนกว่า process จะตาย (ลอง Ctrl+C แล้วรันใหม่ ดู RSS อีกครั้ง)

### ข้อ 6 — memory หายเมื่อ process ตาย แต่ไฟล์ไม่หาย
```bash
echo "hello" | curl --data-binary @- localhost:8000/upload
curl localhost:8000/files
# Ctrl+C pixbin แล้วรันใหม่
curl localhost:8000/files
```
ไฟล์ยังอยู่ (อยู่บน disk ใน `./data`) แต่ memory 400 MB ที่จองไว้หายไปแล้ว — นี่คือความต่างของ RAM กับ disk ในทางปฏิบัติ

### ตรวจว่าทำถูก
- [ ] อธิบายได้ว่าทำไมข้อ 3 ยิง 2C ตัวแล้วใช้เวลามากขึ้น แต่ข้อ 4 ยิง 100 ตัวยังเร็ว
- [ ] บอกได้ว่า `/work/sleep` ตอนรอ process อยู่ในสถานะไหน
- [ ] อธิบายได้ว่าทำไม RSS ไม่ลดลงหลัง request จบ

### Common errors
| อาการ | สาเหตุ |
|---|---|
| `go: command not found` | ยังไม่ติดตั้ง Go (`brew install go`) |
| `address already in use` | pixbin จาก Lesson 1 ยังรันอยู่ (ใช้วิธีจาก Lesson 1 หาแล้ว kill) |
| ข้อ 3 เวลาไม่เปลี่ยนเลย | เครื่องมี core มากกว่าที่ยิง — เพิ่มจำนวน curl |

## 8. Debugging

**Scenario:** "หน้า feed ช้า 3 วินาที"

| # | Hypothesis | วิธีพิสูจน์ | ถ้าผลคือ... |
|---|---|---|---|
| 1 | CPU ของเครื่องเต็ม | `top` ดู %CPU รวม และของ process | ~100% ทุก core → CPU-bound หรือมีตัวอื่นแย่ง |
| 2 | CPU ว่าง แต่ process รออะไรอยู่ | %CPU ต่ำ แต่ request ช้า | I/O-bound → ต้องหาว่ารออะไร (DB? API ภายนอก?) |
| 3 | memory ไม่พอ ใช้ swap | Activity Monitor → Memory Pressure / Linux `free -m`, `vmstat` | swap ขึ้นเรื่อย ๆ → RAM ไม่พอ |
| 4 | ช้าเฉพาะบาง request | ดู log ระยะเวลาของแต่ละ request | ช้าเป็นช่วง ๆ → มีงานหนักอื่นมาแย่ง |

หลัก: **"CPU ต่ำ + ช้า" แปลว่ารอ** — เพิ่ม CPU ไม่ช่วย ต้องหาว่ารออะไร

## 9. Failure Scenarios

| เกิดอะไร | อาการ | ข้างใน |
|---|---|---|
| request CPU-bound เข้ามาเยอะ | ทุก request ช้าลงพร้อมกัน แม้แต่หน้าเบา ๆ | core ไม่พอ ทุกงานต่อคิว scheduler |
| database ช้า | request ค้างนาน แต่ CPU ของ app ต่ำ | process หลับรอ response (S) |
| memory leak | ใช้ RAM เพิ่มทุกชั่วโมง สุดท้าย process ถูกฆ่า | Lesson 6: OOM killer |
| server restart | ข้อมูลที่เก็บไว้ใน memory (เช่น session) หายหมด | RAM เป็น volatile |
| disk ช้า (cloud disk ถูกจำกัด IOPS) | ทุกอย่างที่เขียน log/ไฟล์ช้าลง | process ติดสถานะ D |

## 10. Trade-offs

**เก็บข้อมูลไว้ใน memory vs disk**
- memory: เร็วมาก แต่หายเมื่อ restart และแต่ละเครื่องมีของตัวเอง (ถ้ามี 2 เครื่อง ข้อมูลไม่ตรงกัน)
- disk: อยู่รอด แต่ช้ากว่า — และ disk ของเครื่องเดียวก็ยังหายได้ถ้าเครื่องพัง
- นี่คือเหตุผลที่ระบบจริงเก็บข้อมูลสำคัญใน database แยก และใช้ memory เป็น cache (Phase 12–13)

**เพิ่ม core (Vertical scaling) ช่วยเมื่อไหร่**
- ช่วย: งาน CPU-bound ที่ทำพร้อมกันได้
- ไม่ช่วย: งานที่รอ DB หรือ network — เพิ่ม core แล้วก็ยังรอเท่าเดิม แต่จ่ายแพงขึ้น

## 11. Production Considerations

- **Cost:** ใน cloud ค่า compute คิดตามจำนวน vCPU + RAM ต่อชั่วโมง ถ้า app ของคุณ I/O-bound แล้วซื้อเครื่อง CPU เยอะ = จ่ายทิ้ง
  ถ้า RAM ไม่พอแล้วซื้อเครื่อง CPU มากขึ้นเพื่อให้ได้ RAM มากขึ้น ก็จ่ายทิ้งเช่นกัน — ต้องรู้ว่าตัวไหนตึงก่อน
- **cloud vCPU ไม่เท่ากับ core จริงเสมอไป** — บาง instance type (เช่น AWS t3) ใช้ CPU เต็มได้แค่ช่วงสั้น ๆ (CPU credits) หมดแล้วจะช้าลงมาก
- **อย่าเก็บ state สำคัญไว้ใน memory ของ app** ถ้าอยากให้ scale ได้และ restart ได้โดยไม่เสียข้อมูล
- **Security:** memory isolation ระหว่าง process คือเส้นกั้นความปลอดภัยพื้นฐาน — แต่ process ที่รันเป็น root หรือ user เดียวกัน ยังอ่าน memory กันได้ผ่านเครื่องมือ debug (Lesson 4)

## 12. Quiz

1. **(เข้าใจ)** ทำไม process ถึงแตะ disk หรือ network card ตรง ๆ ไม่ได้ ต้องผ่าน kernel? ให้ตอบทั้งเหตุผลด้าน "แบ่งทรัพยากร" และด้าน "ความปลอดภัย"
2. **(ประยุกต์)** เครื่อง 2 core รับ request ที่แต่ละตัวใช้ CPU 200ms ถ้ามี 20 request เข้ามาพร้อมกัน request สุดท้ายจะเสร็จเมื่อไหร่โดยประมาณ? ถ้าเปลี่ยนเป็น request ที่รอ DB 200ms (ใช้ CPU น้อยมาก) คำตอบเปลี่ยนอย่างไร?
3. **(debugging)** API ช้า 5 วินาที `top` แสดงว่า CPU ของ app ใช้แค่ 3% คุณจะไม่ทำอะไร และจะตรวจอะไรต่อ?
4. **(architecture)** ทีมเก็บ session ของ user ไว้ใน memory ของ app (map ใน Go) ระบบทำงานดี จนวันหนึ่งเพิ่ม server เป็น 2 เครื่อง แล้ว user โดน logout แบบสุ่ม อธิบายว่าเกิดอะไรขึ้น
5. **(trade-off)** งานย่อรูป (CPU-bound) กับ API ที่รอ database (I/O-bound) ควรรันอยู่บน server ชุดเดียวกันไหม? คิดจากสิ่งที่เห็นใน lab ข้อ 3

## 13. Challenge

1. หาจุดที่ pixbin "อิ่มตัว": เพิ่มจำนวน request CPU พร้อมกันทีละขั้น (1, C, 2C, 4C) บันทึกเวลาเฉลี่ยต่อ request ลงตาราง แล้วอธิบายรูปแบบที่เห็น
2. ยิง `/work/cpu` 2C ตัวพร้อมกับ `curl localhost:8000/` ตัวเดียว — หน้าแรกช้าลงไหม? ทำไม? นี่บอกอะไรเกี่ยวกับการให้ request หนักกับเบาใช้ server ร่วมกัน
3. ประมาณ: ถ้า request ปกติของ Pixbin ใช้ CPU 20ms และ server มี 2 vCPU รองรับได้สูงสุดกี่ request/วินาที (คิดแบบหยาบ ๆ) แล้วบอกว่าตัวเลขจริงจะน้อยกว่าเพราะอะไรบ้าง

## 14. สรุปสิ่งที่ต้องจำ

- CPU = คำนวณ · RAM = เร็วแต่หายเมื่อดับ · Disk = ช้าแต่อยู่รอด · Network = ช้าที่สุดและไม่แน่นอน
- kernel แบ่ง CPU ด้วยการสลับงานเร็ว ๆ — core มีเท่าไร ทำงานคำนวณจริงพร้อมกันได้เท่านั้น
- งานที่ **รอ** ไม่ใช้ CPU — web app ส่วนใหญ่ช้าเพราะรอ
- "CPU ต่ำ + ช้า" = กำลังรออะไรอยู่ · "CPU เต็ม + ช้า" = คำนวณไม่ทัน
- ลำดับขนาด: RAM ~100ns, SSD ~100µs, network ใน DC ~0.5ms, ข้ามทวีป ~200ms

## 15. เรียนต่อ

**Lesson 3 — Linux: Process เกิด อยู่ ตายอย่างไร** — เราจะย้ายไปทำงานบน Linux จริง (VM บน Mac)
และตอบคำถามว่าทำไมปิด terminal แล้ว app ตาย
