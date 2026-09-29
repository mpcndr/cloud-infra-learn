# Lesson 6 — เมื่อทรัพยากรหมด: Memory เต็ม, Disk เต็ม, CPU 100%, File descriptor หมด

Phase 1 · Linux · Level 3–5 · Project: Pixbin Stage 1

## 1. ปัญหาที่เรากำลังแก้

ระบบ production ส่วนใหญ่ไม่ได้พังเพราะ bug ซับซ้อน แต่พังเพราะ **ของหมด**:
- memory รั่วช้า ๆ 3 วันแล้ว process ถูกฆ่า
- log โตจน disk เต็ม ทุกอย่างที่ต้องเขียนไฟล์พังพร้อมกัน
- traffic ขึ้นจน CPU เต็ม ทุก request ช้า
- connection เยอะจนเปิดไฟล์เพิ่มไม่ได้

อาการที่ user เห็นมักเหมือนกันหมด ("เว็บช้า" / "เว็บ error") แต่ต้นเหตุต่างกันและแก้ต่างกัน
บทนี้คือการทำให้ของหมดทีละอย่างแบบตั้งใจ ดูว่า Linux ทำอะไร และฝึกวิธีหาต้นเหตุอย่างเป็นระบบ

## 2. Mental Model: ร้านอาหารช่วงเที่ยง

| ทรัพยากร | ในร้าน | เมื่อหมด |
|---|---|---|
| CPU | พ่อครัว | ออเดอร์ต่อคิวยาว ทุกจานช้า แต่ยังได้กิน |
| Memory | โต๊ะเตรียมของ | ผู้จัดการ **ไล่พนักงานที่ใช้โต๊ะเยอะที่สุดออก** (OOM killer) |
| Disk | ห้องเก็บของ | รับของใหม่ไม่ได้ สิ่งที่ต้องเก็บของ (log, upload, DB) ล้มหมด |
| File descriptor | จำนวนเบอร์คิวที่แจกได้ | ลูกค้าใหม่ต้องยืนรอหน้าร้าน ทั้งที่ข้างในอาจยังมีที่ |

สังเกตว่าแต่ละอย่าง "หมด" แล้วพฤติกรรมต่างกันมาก: CPU หมด = **ช้า** · memory หมด = **ตาย** · disk หมด = **error** · fd หมด = **รอ/ปฏิเสธ**

> ⚠️ ขอบเขตของ analogy: OOM killer ไม่ได้เลือก "ตัวที่ใช้เยอะสุด" เป๊ะ ๆ มันให้คะแนน (oom_score) จากหลายปัจจัย
> และถ้ามีการจำกัด memory ระดับ cgroup (ข้อ 3) มันจะเลือกฆ่าเฉพาะใน cgroup นั้น

## 3. Concept

**CPU**
- **Load average** (`uptime`) = จำนวน process ที่ "อยากใช้ CPU หรือรอ disk" เฉลี่ย 1, 5, 15 นาที
  - เทียบกับจำนวน core: load 4 บนเครื่อง 4 core = เต็มพอดี · load 8 = งานรอคิวอีกเท่าตัว
- `%CPU` ใน `top` ของ process: 100% = ใช้ 1 core เต็ม (มีหลาย core จึงเกิน 100% ได้)
- `vmstat 1`: คอลัมน์ `r` = งานรอ CPU · `us` = เวลาใน app · `sy` = ใน kernel · `wa` = CPU ว่างแต่รอ disk

**Memory**
- `free -m`: ดูที่ **available** ไม่ใช่ free — Linux เอา RAM ว่างไปทำ cache ของไฟล์ ซึ่งคืนได้ทันทีเมื่อมีคนต้องการ
- **Swap** = ใช้ disk แทน RAM ช่วยไม่ให้ตาย แต่ช้าลงเป็นพันเท่า (Lesson 2)
- **OOM killer**: เมื่อ memory หมดจริงและคืนไม่ได้แล้ว kernel เลือก process หนึ่งแล้ว **SIGKILL** มัน → exit code 137
- **cgroup memory limit** (`MemoryMax=` ใน systemd, `--memory` ใน Docker, `limits.memory` ใน K8s): จำกัดต่อกลุ่ม เกินเมื่อไร OOM killer ฆ่าในกลุ่มนั้น ไม่กระทบคนอื่น

**Disk**
- `df -h` = พื้นที่ของแต่ละ filesystem · `du -sh <dir>` = ขนาดของโฟลเดอร์
- disk เต็มแล้ว: `write` คืน error `ENOSPC: no space left on device` — process ไม่ตาย แต่ทำงานที่ต้องเขียนไม่ได้
- **ไฟล์ที่ถูกลบแต่ยังมี process เปิดอยู่ ยังกินพื้นที่** จนกว่า process จะปิดมัน → `df` บอกเต็ม แต่ `du` หาไม่เจอ
  หาได้ด้วย `lsof +L1` (ไฟล์ที่ link count = 0 แต่ยังเปิดอยู่)
- **inode หมด** (`df -i`) — มีไฟล์เล็ก ๆ ล้านไฟล์ พื้นที่เหลือแต่สร้างไฟล์ใหม่ไม่ได้

**File descriptors**
- ทุกไฟล์ที่เปิด และ **ทุก socket / connection** ใช้ fd หนึ่งตัว (Lesson 3: `/proc/<PID>/fd`)
- จำกัดต่อ process ด้วย `ulimit -n` / `LimitNOFILE=` ใน systemd
- หมดแล้ว: `accept` / `open` คืน `EMFILE: too many open files` — connection ใหม่ค้างอยู่ในคิวของ kernel

## 4. สิ่งที่เกิดขึ้นข้างใน — OOM

```
pixbin ขอ memory เพิ่ม (เขียนลงหน้า memory ใหม่)
   │
kernel: cgroup pixbin.service ใช้เกิน MemoryMax แล้ว?
   ├─ ยังไม่เกิน → ให้
   └─ เกิน → พยายามคืน memory (ทิ้ง cache ของไฟล์ในกลุ่ม) → ยังไม่พอ
             → OOM killer เลือก process ในกลุ่ม → SIGKILL
             → systemd เห็น status=9/KILL + result 'oom-kill' → Restart=on-failure → เริ่มใหม่
             → memory กลับมาเป็นศูนย์ (แต่ถ้า bug ยังอยู่ ก็จะรั่วใหม่ แล้วตายใหม่)
```
สิ่งที่ user เห็น: request ที่ค้างอยู่ทั้งหมดขาดกลางคัน ช่วง restart เข้าไม่ได้ แล้วกลับมาใช้ได้ — "เว็บกระตุกเป็นระยะ"

## 5. Diagram — ต้นไม้การหาต้นเหตุ

```
                         "เว็บช้า / error"
                               │
              ┌────────────────┼──────────────────┬──────────────────┐
              ▼                ▼                  ▼                  ▼
          uptime/top        free -m           df -h / df -i     log ของ app
          CPU เต็ม?         available ต่ำ?     100%?             too many open files?
              │             swap ขึ้น?          │                  │
      ┌───────┴──────┐     journalctl -k      lsof +L1          ls /proc/PID/fd | wc -l
      │              │     "Out of memory"    du หาตัวใหญ่       เทียบกับ LimitNOFILE
  process ไหน?   wa สูง?        │                  │                  │
  top -o %CPU   → disk ช้า   ตัวไหนโดนฆ่า?     อะไรโต? log? upload?  connection รั่ว?
```
หลักที่ใช้เรียกว่า **USE method**: สำหรับทุกทรัพยากร ดู **U**tilization (ใช้ไปเท่าไร), **S**aturation (มีงานรอคิวไหม), **E**rrors (มี error ไหม)

## 6. Example จริง

Pixbin ที่รันผ่าน systemd จาก Lesson 5 ใช้ endpoint:
- `/work/mem?mb=N` จอง memory ไม่คืน → จำลอง memory leak
- `/work/cpu?n=N` งานคำนวณ
- `/upload` เขียนไฟล์ → ใช้ทำ disk เต็ม
- `/work/sleep?ms=N` ถือ connection ค้างไว้ → ใช้ทำ fd หมด

ติดตั้งเครื่องมือก่อน:
```bash
sudo apt update && sudo apt install -y htop sysstat
```

## 7. Hands-on Lab

ทำใน VM, pixbin รันผ่าน systemd แล้ว (`systemctl status pixbin`)

### ข้อ 1 — ภาพรวมตอนปกติ (baseline)
```bash
uptime
nproc
free -m
df -h /
vmstat 1 5
```
จดค่าไว้ทั้งหมด **คุณจะรู้ว่าอะไรผิดปกติได้ ก็ต่อเมื่อรู้ว่าปกติเป็นอย่างไร**

### ข้อ 2 — CPU 100%
```bash
# V1
htop                  # หรือ top
# V2 — งานหนักต่อเนื่องประมาณ 30 วินาที
for i in $(seq 40); do curl -s "localhost:8000/work/cpu?n=5000" > /dev/null & done
# V3 ระหว่างนั้น
time curl localhost:8000/
uptime
vmstat 1 5
```
สังเกต: load average ขึ้นเกินจำนวน core, คอลัมน์ `r` ของ vmstat สูง, หน้า `/` ที่ปกติเร็วมากก็ช้าลง
**คำถาม:** ในสถานการณ์นี้ ถ้าคุณเป็น on-call คุณจะรู้ได้อย่างไรว่า "pixbin คือตัวที่กิน CPU" และ "เพราะ endpoint ไหน"

### ข้อ 3 — memory เต็ม (OOM)
จำกัด memory ของ pixbin:
```bash
sudo systemctl edit pixbin
```
```ini
[Service]
MemoryMax=300M
MemorySwapMax=0
```
```bash
sudo systemctl restart pixbin
systemctl status pixbin --no-pager | grep -i memory
# V1
journalctl -u pixbin -f
# V2
curl "localhost:8000/work/mem?mb=100"
curl "localhost:8000/work/mem?mb=100"
curl "localhost:8000/work/mem?mb=100"
```
Expected: ครั้งที่ 3 curl ได้ `Empty reply from server` และ journal มี
`A process of this unit has been killed by the OOM killer` · `Failed with result 'oom-kill'` · แล้ว restart
```bash
journalctl -k -n 20 --no-pager | grep -i -A2 "memory"
systemctl show -p NRestarts pixbin
```
- `journalctl -k` = log ของ kernel (เหมือน `dmesg`) — หลักฐานว่าใครถูกฆ่า ใช้ memory เท่าไร
- เชื่อมกับ Lesson 3: ถ้าไม่มี systemd คุณจะเห็นแค่ exit code 137

### ข้อ 4 — disk เต็ม
ทำ DATA_DIR ให้มีพื้นที่แค่ 20 MB (tmpfs = filesystem ใน RAM ใช้แทน disk เล็ก ๆ โดยไม่ต้องทำ disk ของ VM เต็มจริง):
```bash
sudo mount -t tmpfs -o size=20M,uid=$(id -u pixbin),gid=$(id -g pixbin),mode=0750 tmpfs /var/lib/pixbin
sudo systemctl restart pixbin     # ให้ pixbin เห็น mount ใหม่แน่ ๆ (sandbox จาก Lesson 5 สร้าง mount namespace แยก)
df -h /var/lib/pixbin
head -c 15M /dev/urandom > /tmp/15m.bin
curl -w ' %{http_code}\n' --data-binary @/tmp/15m.bin localhost:8000/upload
curl -w ' %{http_code}\n' --data-binary @/tmp/15m.bin localhost:8000/upload
curl -w ' %{http_code}\n' localhost:8000/healthz
df -h /var/lib/pixbin
sudo ls -l /var/lib/pixbin
journalctl -u pixbin -n 5 --no-pager
```
Expected: ครั้งแรก 200, ครั้งที่สอง 500 (`no space left on device`), healthz 503
**ดู `ls -l` ให้ดี** — มีไฟล์อะไรค้างอยู่ที่ไม่ควรอยู่? นั่นคือ bug ของ Pixbin (Challenge ข้อ 1)

คืนสภาพ: `sudo systemctl stop pixbin && sudo umount /var/lib/pixbin && sudo systemctl start pixbin`

### ข้อ 5 — df บอกเต็ม แต่ du หาไม่เจอ
```bash
sudo mount -t tmpfs -o size=20M tmpfs /mnt
sudo sh -c 'exec 3>/mnt/big.log; head -c 12M /dev/zero >&3; rm /mnt/big.log; sleep 120' &
sleep 1
df -h /mnt              # ใช้ไป 12M
sudo du -sh /mnt        # 0 ?!
sudo lsof -nP +L1 | grep /mnt
```
- `exec 3>file` = เปิดไฟล์ไว้ที่ fd 3 แล้วเขียน จากนั้นลบไฟล์ทิ้งทั้งที่ยังเปิดอยู่
- นี่คือสิ่งที่เกิดจริงเมื่อมีคนลบ log file ที่ app ยังเขียนอยู่ เพื่อ "เคลียร์ disk" แล้วพื้นที่ไม่คืน
หาวิธีคืนพื้นที่ **โดยไม่ reboot** แล้ว `sudo umount /mnt`

### ข้อ 6 — file descriptors หมด
```bash
sudo systemctl edit pixbin
```
เพิ่มใต้ `[Service]`: `LimitNOFILE=20`
```bash
sudo systemctl restart pixbin
P=$(systemctl show -p MainPID --value pixbin)
sudo ls /proc/$P/fd | wc -l
grep "open files" /proc/$P/limits
for i in $(seq 30); do curl -s -m 3 -o /dev/null -w "%{http_code} " "localhost:8000/work/sleep?ms=2000" & done; wait; echo
journalctl -u pixbin -n 5 --no-pager
```
Expected: บาง request ได้ `000` (curl timeout 3 วินาที) และ journal มี `accept4: too many open files; retrying`
อธิบายว่าทำไม request ที่ไม่ผ่านถึงได้ timeout ไม่ใช่ refused (ย้อนไป Lesson 1: ใครทำ handshake?)

ลบ `LimitNOFILE=20` และ `MemoryMax`, `MemorySwapMax` ออก (`sudo systemctl edit pixbin`) แล้ว restart

### ตรวจว่าทำถูก
- [ ] บอกได้ว่าแต่ละทรัพยากรหมดแล้วอาการต่างกันอย่างไร (ช้า / ตาย / error / รอ)
- [ ] หาหลักฐาน OOM kill ใน journal ของ kernel ได้
- [ ] อธิบายข้อ 5 ได้ด้วยความรู้เรื่อง inode และ fd

### Common errors
| อาการ | สาเหตุ |
|---|---|
| ข้อ 3 ไม่ถูกฆ่า | VM มี swap หรือยังไม่ได้ restart หลังแก้ · เช็ค `systemctl show -p MemoryMax pixbin` |
| `mount: ... target is busy` ตอน umount | ยังมี process ใช้อยู่ (`sudo lsof +D /var/lib/pixbin`) |
| ข้อ 6 ไม่เห็น error | ตั้ง `LimitNOFILE` ไม่ติด — ดู `/proc/$P/limits` |

## 8. Debugging

**Scenario:** "เว็บ error เป็นช่วง ๆ ทุกประมาณ 6 ชั่วโมง แล้วกลับมาเอง"

1. **Observe:** error ช่วงไหน? ตรงกับอะไรไหม (deploy? cron? traffic?)
2. **Hypothesis:** รูปแบบ "พังแล้วหายเอง เป็นรอบ" → process ถูก restart เป็นรอบ?
3. **Measure:** `systemctl show -p NRestarts pixbin` · `journalctl -u pixbin | grep -E "oom-kill|Main process exited"`
4. **Eliminate:** มี `oom-kill` → memory · ไม่มี แต่ exit code 1 → app crash · ไม่มี restart เลย → ไม่ใช่ process ตาย ไปดูที่อื่น (disk? dependency?)
5. **Root cause:** ถ้า OOM → memory โตตามเวลา (leak) หรือโตตาม traffic (limit ต่ำไป)? ดูกราฟ memory ตามเวลา (Phase 10)
6. **Fix:** leak → แก้โค้ด · limit ต่ำ → ปรับ limit / เพิ่มเครื่อง
7. **Verify:** memory ไม่โตต่อเนื่องแล้ว, NRestarts ไม่เพิ่ม
8. **Prevent:** alert เมื่อ memory ใกล้ limit และเมื่อ restart

**อย่าเพิ่ม memory ก่อนรู้ว่ามันโตเพราะอะไร** — ถ้าเป็น leak การเพิ่ม memory แค่ยืดเวลาก่อนตาย

## 9. Failure Scenarios

| เกิดอะไร | user เห็น | ต้นเหตุที่เจอบ่อย |
|---|---|---|
| CPU เต็ม | ทุกหน้าช้า | traffic เกินกำลัง, loop ผิดพลาด, regex ช้า, งานหนักอยู่ใน request |
| OOM kill | error เป็นช่วง + ช่วงเข้าไม่ได้ | memory leak, โหลดไฟล์ทั้งก้อนเข้า memory, cache ไม่มีขนาดจำกัด |
| disk เต็ม | upload/บันทึกข้อมูล error, DB หยุดรับ write | log ไม่ rotate, upload ไม่มี quota, backup เก่าไม่ลบ, ไฟล์ค้าง |
| ลบ log แล้ว disk ไม่คืน | "ลบแล้วแต่ยังเต็ม" | process ยังเปิดไฟล์ที่ถูกลบ |
| fd หมด | request ใหม่ค้าง/timeout ทั้งที่ CPU/memory ว่าง | connection ไม่ถูกปิด (leak), limit ต่ำ |

## 10. Trade-offs

**ตั้ง memory limit vs ไม่ตั้ง**
- ไม่ตั้ง: app ใช้ได้เต็มเครื่อง — แต่ถ้ารั่ว อาจลาก service อื่น (sshd, DB) ตายไปด้วย เพราะ OOM killer ระดับเครื่องอาจเลือกผิดตัว
- ตั้ง: ความเสียหายอยู่ในกรอบ — แต่ตั้งต่ำไปจะโดนฆ่าตอน traffic สูงแม้ไม่มี leak
- หลัก: ตั้งเสมอ โดยวัดการใช้จริงตอน peak แล้วเผื่อ

**Swap เปิด vs ปิด**
- เปิด: ไม่ตายทันที มีเวลาให้คนมาแก้ — แต่ระบบอาจช้าจนใช้ไม่ได้ ("ช้าแบบตายไม่ตาย" ซึ่งบางทีแย่กว่าตาย)
- ปิด: ตายเร็ว ชัดเจน restart ได้ — ระบบที่มีหลายเครื่อง + auto restart มักเลือกแบบนี้ (K8s default ไม่ใช้ swap)

## 11. Production Considerations

- **ตั้ง limit ให้ทุกอย่าง**: memory ของ service, ขนาด upload, ขนาด log (`SystemMaxUse=` ใน `/etc/systemd/journald.conf`), จำนวน connection
- **Monitor 4 ตัวนี้ทุกเครื่อง** และ alert **ก่อน** เต็ม (เช่น disk > 80%) ไม่ใช่ตอนเต็ม (Phase 10)
- **แยก disk ของข้อมูลออกจาก disk ของระบบ** disk ข้อมูลเต็มจะได้ไม่ทำให้ OS พัง
- **Cost:** memory คือตัวกำหนดขนาดเครื่องที่พบบ่อยที่สุด app ที่รั่วทำให้ต้องซื้อเครื่องใหญ่เกินจำเป็น · disk ใน cloud คิดตาม GB ที่ **จอง** ไม่ใช่ที่ใช้
- **Security:** disk/memory หมดเป็นช่องทางโจมตีได้ (upload ไฟล์ใหญ่ ๆ ซ้ำ ๆ = DoS) → ต้องมี limit ขนาดและจำนวน

## 12. Quiz

1. **(เข้าใจ)** ทำไม `free -m` บนเครื่องที่ทำงานมานาน ๆ ถึงแสดง "free" เหลือน้อยมาก ทั้งที่ระบบไม่มีปัญหาอะไร
2. **(ประยุกต์)** เครื่อง 4 core มี load average `12.0, 11.5, 11.8` แต่ `top` แสดง CPU รวมแค่ 20% อะไรน่าจะเกิดขึ้น (ใบ้: load นับอะไรบ้าง)
3. **(debugging)** `df -h` บอก `/` เต็ม 100% แต่ `du -sh /*` รวมกันได้แค่ 60% ของ disk ตั้ง hypothesis อย่างน้อย 2 ข้อ และวิธีพิสูจน์
4. **(debugging)** container ตายด้วย exit code 137 ซ้ำทุก ~2 ชั่วโมง ระหว่างนั้นทำงานปกติ คุณจะเก็บข้อมูลอะไรเพื่อแยกว่าเป็น memory leak หรือ traffic peak
5. **(trade-off)** pixbin รับ upload ได้ไม่จำกัดขนาด ควรจำกัดที่ชั้นไหน (app, reverse proxy, filesystem quota) และทำไมควรมีมากกว่าหนึ่งชั้น

## 13. Challenge

1. แก้ `/upload` ของ Pixbin ให้ **ลบไฟล์ที่เขียนไม่ครบ** เมื่อเกิด error และจำกัดขนาด upload ไว้ที่ 10 MB (ใบ้: `http.MaxBytesReader`) พิสูจน์ด้วยข้อ 4 ว่าไม่มีไฟล์ค้างแล้ว และไฟล์ 11 MB ถูกปฏิเสธด้วย status ที่เหมาะสม
2. เขียน script `triage.sh` ที่พิมพ์สรุปสถานะเครื่องใน 1 หน้าจอ: load เทียบ core, memory available, disk ทุก mount ที่เกิน 80%, inode, process ที่กิน CPU/memory สูงสุด 3 ตัว, และ OOM kill ล่าสุดจาก kernel log — ใช้กับข้อ 2–6 แล้วดูว่ามันจับได้ทุกกรณีไหม
3. ตั้ง journald ให้ใช้ disk ไม่เกิน 100 MB พิสูจน์ว่าค่าถูกใช้งาน (`journalctl --disk-usage`) และอธิบายว่าเมื่อถึง limit journald ทำอะไร

## 14. สรุปสิ่งที่ต้องจำ

- CPU หมด = ช้า · memory หมด = ถูกฆ่า (137) · disk หมด = error ENOSPC · fd หมด = connection ใหม่ค้าง
- ดู **available** ใน `free` ไม่ใช่ free · load average เทียบกับจำนวน core
- หลักฐาน OOM อยู่ใน kernel log (`journalctl -k`)
- `df` ≠ `du` → หาไฟล์ที่ถูกลบแต่ยังเปิดอยู่ด้วย `lsof +L1`
- USE: Utilization, Saturation, Errors — ถามครบสามข้อกับทุกทรัพยากร
- ตั้ง limit ทุกอย่าง แล้ว alert ก่อนถึง limit

## 15. เรียนต่อ

**Phase 2 — Networking · Lesson 7 — IP, Subnet และ Routing:** packet ออกจากเครื่องเราไปถึงเครื่องอื่นได้อย่างไร
และทำไม Mac ของคุณคุยกับ VM ได้ แต่เพื่อนบนอินเทอร์เน็ตคุยกับ VM ไม่ได้
