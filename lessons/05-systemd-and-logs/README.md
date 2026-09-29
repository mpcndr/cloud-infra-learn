# Lesson 5 — systemd, Service และ Logs: ให้ OS ดูแล app แทนเรา

Phase 1 · Linux · Level 2–5 · Project: Pixbin Stage 1

## 1. ปัญหาที่เรากำลังแก้

จาก Lesson 3–4 เรารัน pixbin ได้ถูก user แล้ว แต่ยังมีปัญหาครบชุด:
- ปิด terminal → ตาย
- crash ตอนตี 3 → ไม่มีใครรันใหม่ เว็บล่มจนเช้า
- reboot เครื่อง → ไม่มีใครรันให้
- log ขึ้นแค่หน้าจอ ปิดหน้าจอแล้วหายหมด หาย้อนหลังไม่ได้

ทุกข้อคือ "ต้องมีคนเฝ้า" — เราต้องการผู้เฝ้าที่ไม่หลับ ไม่ลืม และอยู่ตั้งแต่เครื่องเปิด
บน Linux ผู้เฝ้านั้นคือ **systemd** (PID 1 ที่เราเห็นใน Lesson 3)

## 2. Mental Model: ผู้จัดการกะที่มีสมุดคำสั่ง

- **systemd** = ผู้จัดการที่มาทำงานคนแรกตอนเปิดร้าน (PID 1) และอยู่จนปิดร้าน
- **unit file** = ใบคำสั่งงาน: "จ้างใคร (User) · ทำงานอะไร (ExecStart) · ถ้าเขาล้มให้ทำไง (Restart) · เริ่มหลังใคร (After)"
- **journald** = สมุดบันทึก เก็บทุกอย่างที่พนักงานพูด (stdout/stderr) พร้อมเวลาและชื่อ
- **systemctl** = วิธีที่เราคุยกับผู้จัดการ · **journalctl** = วิธีเปิดอ่านสมุด

> ⚠️ ขอบเขตของ analogy: systemd ไม่ได้ "ดู" ว่า app ทำงานถูกไหม มันรู้แค่ว่า process **ยังไม่ตาย**
> app ที่ค้าง (เหมือน `kill -STOP` ใน Lesson 1) systemd จะไม่รู้เลยและไม่ restart ให้ — ต้องมี health check จากภายนอก (Phase 10, 14)

## 3. Concept

**Unit** = สิ่งที่ systemd ดูแล มีหลายชนิด: `.service` (process), `.timer` (ตั้งเวลาแบบ cron), `.socket`, `.mount` ...
เราใช้ `.service` เป็นหลัก

**ส่วนสำคัญของ service unit**

| คำสั่ง | ทำอะไร |
|---|---|
| `ExecStart=` | program ที่จะรัน (path เต็ม) |
| `User=`, `Group=` | รันในนามของใคร (Lesson 4) |
| `Environment=`, `EnvironmentFile=` | env variables / ไฟล์ config |
| `Restart=on-failure` | restart เมื่อ exit code ไม่ใช่ 0 หรือโดนฆ่าด้วย signal ที่ "ไม่ปกติ" |
| `RestartSec=` | รอกี่วินาทีก่อน restart |
| `TimeoutStopSec=` | ตอน stop ส่ง SIGTERM แล้วรอกี่วินาทีก่อนส่ง SIGKILL |
| `After=network.target` | เริ่มหลังจาก network พร้อม |
| `WantedBy=multi-user.target` | ใช้กับ `enable` = ให้เริ่มตอน boot |

**คำว่า "failure" ของ systemd** — process ที่ออกด้วย exit 0 หรือตายด้วย SIGTERM, SIGINT, SIGHUP, SIGPIPE ถือว่า **จบปกติ** (ถือว่ามีคนตั้งใจสั่งหยุด)
`Restart=on-failure` จึง restart เฉพาะ exit code ≠ 0, โดน SIGKILL/SIGSEGV, หรือ timeout

**คำสั่งที่ใช้ทุกวัน**
```
systemctl status pixbin        สถานะ, PID, exit code ล่าสุด, log 10 บรรทัดท้าย
systemctl start|stop|restart pixbin
systemctl enable pixbin        ให้เริ่มตอน boot (ไม่ได้ start ตอนนี้) · enable --now = ทั้งสองอย่าง
systemctl daemon-reload        ต้องรันทุกครั้งที่แก้ unit file
journalctl -u pixbin -f        ตาม log แบบ real-time (เหมือน tail -f)
journalctl -u pixbin -n 50 --no-pager   50 บรรทัดล่าสุด
journalctl -u pixbin --since "10 min ago"
```

## 4. สิ่งที่เกิดขึ้นข้างใน

```
systemctl start pixbin
      │ (ส่งคำขอไปหา systemd ผ่าน D-Bus)
      ▼
systemd (PID 1)
  1. อ่าน /etc/pixbin/pixbin.env (ในฐานะ root — ไฟล์นี้จึงเป็นสิทธิ์ 600 ได้)
  2. fork → ตั้ง UID/GID เป็น pixbin → ตั้ง env → ต่อ stdout/stderr เข้า journald
  3. exec /usr/local/bin/pixbin
  4. ใส่ process ไว้ใน cgroup ชื่อ pixbin.service  ← จำคำนี้ไว้ Phase 5 (Docker) ใช้กลไกเดียวกัน
      │
      ▼
เฝ้ารอ: process ตาย → ดู exit code/signal → ตรงเงื่อนไข Restart? → รอ RestartSec → เริ่มใหม่
```

**cgroup** (control group) คือกลไกของ kernel ที่จัดกลุ่ม process และ **จำกัดทรัพยากร** ของกลุ่มได้ (CPU, memory)
systemd ใช้มันเพื่อรู้ว่า process ไหนเป็นของ service ไหน (แม้จะ fork ลูกออกไปกี่ตัวก็ตาม) และเพื่อฆ่าทั้งกลุ่มตอน stop
Lesson 6 เราจะใช้ cgroup จำกัด memory ของ pixbin — และ Docker ก็คือการเอา cgroup + namespace มาห่อให้ใช้ง่าย

## 5. Diagram

```
                boot
                 │
          ┌──────▼───────┐
          │ systemd PID 1│──── อ่าน /etc/systemd/system/pixbin.service
          └──────┬───────┘
                 │ fork + setuid(pixbin) + exec
          ┌──────▼───────────────┐   stdout/stderr    ┌──────────┐
          │ pixbin (cgroup:       │──────────────────►│ journald │──► journalctl -u pixbin
          │  pixbin.service)      │                    └──────────┘
          └──────┬───────────────┘
                 │ ตาย (exit 1 / SIGKILL)
                 ▼
          systemd: "failure" → รอ 2s → start ใหม่ (นับ NRestarts)
          ถ้า restart ถี่เกินกำหนด → หยุดพยายาม สถานะ failed
```

## 6. Example จริง

ไฟล์ทั้งหมดที่เราจะสร้าง:

**`/etc/pixbin/pixbin.env`** — config (และในอนาคต secret) แยกจาก unit file
```
HOST=127.0.0.1
PORT=8000
DATA_DIR=/var/lib/pixbin
```

**`/etc/systemd/system/pixbin.service`**
```ini
[Unit]
Description=Pixbin image sharing app
After=network.target

[Service]
User=pixbin
Group=pixbin
EnvironmentFile=/etc/pixbin/pixbin.env
ExecStart=/usr/local/bin/pixbin
Restart=on-failure
RestartSec=2
TimeoutStopSec=15

[Install]
WantedBy=multi-user.target
```
ทำไม `TimeoutStopSec=15`? เพราะ pixbin ให้เวลา request ที่ค้าง 10 วินาที (ใน main.go) — timeout ของ systemd ต้องยาวกว่าของ app
ไม่งั้น systemd จะ KILL ก่อนที่ app จะปิดตัวเองเสร็จ

## 7. Hands-on Lab

ทำใน VM · ต้องทำ Lesson 4 ข้อ 2–3 แล้ว (มี user `pixbin`, `/usr/local/bin/pixbin`, `/var/lib/pixbin`)
ปิด pixbin ที่รันค้างจากบทก่อนทั้งหมด: `sudo pkill -x pixbin; pkill -f pixbin-linux`

### ข้อ 1 — สร้าง config และ unit
```bash
sudo mkdir -p /etc/pixbin
sudo tee /etc/pixbin/pixbin.env > /dev/null <<'EOF'
HOST=127.0.0.1
PORT=8000
DATA_DIR=/var/lib/pixbin
EOF
sudo chmod 600 /etc/pixbin/pixbin.env

sudo tee /etc/systemd/system/pixbin.service > /dev/null <<'EOF'
[Unit]
Description=Pixbin image sharing app
After=network.target

[Service]
User=pixbin
Group=pixbin
EnvironmentFile=/etc/pixbin/pixbin.env
ExecStart=/usr/local/bin/pixbin
Restart=on-failure
RestartSec=2
TimeoutStopSec=15

[Install]
WantedBy=multi-user.target
EOF
sudo systemctl daemon-reload
sudo systemctl enable --now pixbin
systemctl status pixbin
```
- `tee` เขียนไฟล์ด้วยสิทธิ์ root ได้ (`sudo echo ... > file` ใช้ไม่ได้ เพราะ `>` ทำโดย shell ของคุณซึ่งไม่ใช่ root — ลองคิดว่าทำไม)
- `<<'EOF'` = heredoc ส่งข้อความหลายบรรทัดเข้า stdin
Expected: `Active: active (running)` และบรรทัด log `listening on 127.0.0.1:8000`

### ข้อ 2 — ตรวจว่าเป็นอย่างที่ออกแบบ
```bash
curl localhost:8000/
ps -o user,pid,ppid,cmd -p $(systemctl show -p MainPID --value pixbin)
systemd-cgls -u pixbin.service
```
- PPID ต้องเป็น **1** (systemd) — เทียบกับ Lesson 3 ที่ PPID เป็น bash
- `systemd-cgls` แสดง process ใน cgroup ของ service

### ข้อ 3 — ตาม log
```bash
# V1
journalctl -u pixbin -f
# V2
curl "localhost:8000/work/sleep?ms=300"; curl localhost:8000/nope
```
V1 เห็นทุก request พร้อมเวลา ไม่ว่าคุณจะปิด terminal ไปกี่รอบ log ก็ยังอยู่ (`journalctl -u pixbin --since "1 hour ago"`)

### ข้อ 4 — ตายแล้วฟื้น
```bash
sudo kill -KILL $(systemctl show -p MainPID --value pixbin)
sleep 3; systemctl status pixbin --no-pager | head -5
systemctl show -p NRestarts pixbin
```
Expected: PID ใหม่, `NRestarts=1`, และใน journal เห็น `Main process exited, code=killed, status=9/KILL` ตามด้วย `Scheduled restart job`

ทำซ้ำด้วย `kill -TERM` แล้วดูว่า systemd restart ให้ไหม **อธิบายผลที่ต่างกัน** (ย้อนดูตาราง "failure" ข้อ 3)
เสร็จแล้ว `sudo systemctl start pixbin`

### ข้อ 5 — stop อย่างสุภาพ และเมื่อความสุภาพหมดเวลา
```bash
curl "localhost:8000/work/sleep?ms=5000" & sleep 1; sudo systemctl stop pixbin; wait
journalctl -u pixbin -n 5 --no-pager
```
curl ยังได้คำตอบครบ (graceful shutdown ทำงาน) — ดูว่า `systemctl stop` ค้างนานแค่ไหน

แล้วลองให้ request ยาวกว่าที่ทุกคนยอมรอ:
```bash
sudo systemctl start pixbin
curl "localhost:8000/work/sleep?ms=30000" & sleep 1; time sudo systemctl stop pixbin; wait
journalctl -u pixbin -n 5 --no-pager
```
ใครยอมแพ้ก่อน — pixbin (10s) หรือ systemd (15s)? curl ได้อะไร?
จากนั้นแก้ unit เป็น `TimeoutStopSec=5` (`sudo systemctl edit --full pixbin`) ทำซ้ำ แล้วดู journal — ตอนนี้ใครฆ่าใคร?
คืนค่าเป็น 15 เมื่อทำเสร็จ

### ข้อ 6 — crash loop
```bash
sudo sed -i 's/^PORT=.*/PORT=80/' /etc/pixbin/pixbin.env
sudo systemctl restart pixbin
sleep 15; systemctl status pixbin --no-pager
journalctl -u pixbin -n 30 --no-pager
```
Expected: log `bind: permission denied` ซ้ำหลายรอบ แล้ว systemd เลิกพยายาม (`Start request repeated too quickly` / `Failed with result 'exit-code'`)
**ใช้ journal อย่างเดียวหาสาเหตุ** แล้วบอกว่าจะแก้ได้ 2 ทางอย่างไร

แก้กลับ:
```bash
sudo sed -i 's/^PORT=.*/PORT=8000/' /etc/pixbin/pixbin.env
sudo systemctl reset-failed pixbin && sudo systemctl start pixbin
```

### ข้อ 7 — รอด reboot
```bash
sudo reboot
# บน Mac รอ ~20 วินาที แล้ว
multipass shell lab
systemctl status pixbin --no-pager | head -3
curl localhost:8000/healthz
```

### ข้อ 8 — ลดสิทธิ์ลงอีกชั้น (systemd sandboxing)
```bash
systemd-analyze security pixbin | tail -1          # คะแนน exposure ก่อน
sudo systemctl edit pixbin
```
ใส่ใน editor (ส่วนนี้จะถูกเก็บเป็น drop-in `/etc/systemd/system/pixbin.service.d/override.conf`):
```ini
[Service]
NoNewPrivileges=yes
ProtectSystem=strict
ReadWritePaths=/var/lib/pixbin
ProtectHome=yes
PrivateTmp=yes
```
```bash
sudo systemctl restart pixbin
systemd-analyze security pixbin | tail -1          # คะแนนหลัง
echo hi | curl --data-binary @- localhost:8000/upload
```
- `ProtectSystem=strict` = ทั้ง filesystem อ่านได้อย่างเดียวสำหรับ service นี้ ยกเว้น path ใน `ReadWritePaths`
- `ProtectHome=yes` = มองไม่เห็น `/home` เลย · `NoNewPrivileges` = ห้ามยกระดับสิทธิ์ (เช่นผ่าน sudo)
ลองลบ `ReadWritePaths` ออก แล้ว upload — เกิดอะไรขึ้น? error ต่างจากกรณี chmod ใน Lesson 4 ไหม?

### ตรวจว่าทำถูก
- [ ] อธิบายได้ว่าทำไม `kill -KILL` ทำให้ restart แต่ `kill -TERM` ไม่
- [ ] ข้อ 5: บอกได้ว่าในแต่ละรอบใครเป็นคนจบ process และเห็นหลักฐานใน journal
- [ ] ข้อ 6: หาสาเหตุ crash loop จาก journal ได้โดยไม่เดา

### Common errors
| อาการ | สาเหตุ |
|---|---|
| แก้ unit แล้วไม่มีผล | ลืม `sudo systemctl daemon-reload` |
| `status=217/USER` | ไม่มี user ตามที่เขียนใน `User=` |
| `status=203/EXEC` | path ใน `ExecStart` ผิด หรือไฟล์ไม่มีสิทธิ์ x |
| `Failed to load environment files` | path ของ `EnvironmentFile` ผิด |
| `Start request repeated too quickly` | crash loop — ดูบรรทัด log ก่อนหน้านั้นเพื่อหาสาเหตุจริง แล้ว `reset-failed` |

## 8. Debugging

**Scenario:** "service ไม่ขึ้น"

ลำดับที่ใช้จริงทุกครั้ง:
1. `systemctl status pixbin` → ดู `Active:` และ exit code/`status=` (บอกว่าตายแบบไหน)
2. `journalctl -u pixbin -n 50 --no-pager` → หาบรรทัด error แรก **ไม่ใช่บรรทัดสุดท้าย** (บรรทัดท้ายมักเป็นผลตาม เช่น "repeated too quickly")
3. ถ้า log ไม่มีอะไรเลย → process อาจตายก่อนพิมพ์อะไรได้: ตรวจ `ExecStart`, สิทธิ์, user
4. ลองรันคำสั่งเดียวกับที่ systemd รัน ด้วยมือ ในนามของ user เดียวกัน (`systemctl stop` ก่อน ไม่งั้น port ชน):
   `sudo -u pixbin env $(sudo cat /etc/pixbin/pixbin.env | xargs) /usr/local/bin/pixbin`
   ถ้ามือรันได้แต่ systemd รันไม่ได้ → ความต่างอยู่ที่ environment หรือ sandboxing

| status ที่เห็น | แปลว่า |
|---|---|
| `status=1/FAILURE` | app ออกเองด้วย error — อ่าน log ของ app |
| `status=9/KILL` | โดน SIGKILL (OOM? systemd timeout?) |
| `status=203/EXEC`, `217/USER`, `226/NAMESPACE` | systemd เตรียม process ไม่สำเร็จ — app ยังไม่ได้เริ่มเลย |

## 9. Failure Scenarios

| เกิดอะไร | อาการ | ข้างใน |
|---|---|---|
| app ค้างแต่ไม่ตาย | เว็บ timeout แต่ `systemctl status` บอก running | systemd ดูแค่ว่ามี process อยู่ |
| crash ทันทีหลัง start ทุกครั้ง | restart ไม่กี่รอบแล้ว failed ถาวร | start rate limit ป้องกัน loop ไม่รู้จบ |
| journal โตจน disk เต็ม | เครื่องทั้งเครื่องมีปัญหา | ต้องตั้ง `SystemMaxUse=` ใน journald.conf (Lesson 6) |
| แก้ config ใน env file แล้วลืม restart | ค่าเก่ายังอยู่ | env อ่านตอน start เท่านั้น (Lesson 3: env สืบทอดตอนเกิด) |
| TimeoutStopSec สั้นกว่า shutdown ของ app | deploy แล้ว request โดนตัด | SIGKILL มาก่อน app ปิดเสร็จ |

## 10. Trade-offs

**systemd vs Docker vs Kubernetes ในบทบาท "ผู้เฝ้า process"**
- systemd: มีอยู่แล้วทุกเครื่อง เบา เข้าใจง่าย — แต่ดูแลได้แค่เครื่องเดียว และ deploy = copy binary เอง
- Docker: แพ็ก app + dependency เป็น image เดียว รันที่ไหนก็เหมือนกัน — เพิ่ม layer ที่ต้องเรียนรู้
- Kubernetes: เฝ้าหลายเครื่องพร้อมกัน ย้าย app เมื่อเครื่องตาย — ซับซ้อนกว่ามาก
- ทั้งสามใช้หลักเดียวกัน: ExecStart ≈ CMD ≈ container spec · Restart ≈ restartPolicy · TimeoutStopSec ≈ terminationGracePeriodSeconds
  เข้าใจ systemd ให้ดี แล้วอีกสองตัวจะดูคุ้นเคย

**`Restart=on-failure` vs `Restart=always`**
- on-failure: เคารพการหยุดโดยตั้งใจ (TERM) — เหมาะกับ web service ส่วนใหญ่
- always: restart แม้ exit 0 — เหมาะกับ worker ที่ควรรันตลอดไม่ว่าจะจบด้วยเหตุผลอะไร

## 11. Production Considerations

- **ทุก service ต้องมี unit file / manifest** อยู่ใน version control (ตอนนี้เราพิมพ์ด้วยมือ Phase 8 จะเขียนเป็น code)
- **Restart ไม่ใช่การแก้ปัญหา** — มันซื้อเวลา ต้องมี alert เมื่อ NRestarts เพิ่ม (Phase 10)
- **Config แยกจากโค้ด** ใน env file สิทธิ์ 600 — แต่ระวัง: ค่าใน env ยังเห็นได้ใน `/proc/<PID>/environ` โดย root และ user เดียวกัน
- **Sandboxing ราคาถูกมาก** เพิ่ม 5 บรรทัดลดความเสียหายได้เยอะ ควรเป็น default
- **Cost:** journald เก็บ log บน disk ของเครื่อง — ถ้าเครื่องหาย log หายด้วย ระบบจริงส่ง log ออกไปที่อื่น (Phase 10) ซึ่งมีค่าใช้จ่ายตามปริมาณ log

## 12. Quiz

1. **(เข้าใจ)** systemd รู้ได้อย่างไรว่า process ไหนเป็นของ pixbin.service ถึงแม้ pixbin จะ fork process ลูกออกไปหลายตัว?
2. **(ประยุกต์)** unit ตั้ง `Restart=on-failure` app ตายด้วย exit code 0 เพราะ bug ที่ทำให้ main function จบก่อนเวลา systemd จะ restart ไหม? ควรตั้งค่าอย่างไร
3. **(debugging)** `systemctl status` แสดง `code=exited, status=203/EXEC` ต่างจาก `status=1/FAILURE` อย่างไร และจะตรวจอะไร
4. **(debugging)** เว็บ timeout แต่ `systemctl status pixbin` บอก `active (running)` และ `NRestarts=0` อธิบายว่าเป็นไปได้อย่างไร และ systemd จะช่วยอะไรได้ไหม
5. **(trade-off)** ถ้าจะให้ pixbin ฟัง port 80 ตรง ๆ มีทางเลือก: รันเป็น root, `AmbientCapabilities=CAP_NET_BIND_SERVICE`, หรือวาง reverse proxy ข้างหน้า เปรียบเทียบข้อดีข้อเสียของแต่ละทาง

## 13. Challenge

1. เพิ่ม `AmbientCapabilities=CAP_NET_BIND_SERVICE` ให้ pixbin ฟัง port 80 โดยไม่เป็น root พิสูจน์ด้วย `ps` และ `curl localhost:80` จากนั้นอธิบายว่าทำไมใน Lesson 11 เราจะไม่ใช้วิธีนี้ (แล้วคืนค่า port 8000)
2. สร้าง `pixbin-cleanup.service` + `pixbin-cleanup.timer` ที่ลบไฟล์ใน `/var/lib/pixbin` ที่เก่ากว่า 1 วัน ทุก 10 นาที (ใบ้: `find ... -mtime +1 -delete`) รันในนามของ user ที่เหมาะสม อธิบายว่าเลือก user นั้นเพราะอะไร
3. ทำให้ `systemd-analyze security pixbin` ได้คะแนนต่ำกว่า 5.0 โดยที่ pixbin ยังทำงานได้ครบ บันทึกว่าเพิ่มอะไรไปบ้าง และตัวไหนทำให้พังระหว่างทาง

## 14. สรุปสิ่งที่ต้องจำ

- systemd = PID 1 ผู้เฝ้า process: start ตอน boot, restart เมื่อพัง, เก็บ log, จำกัดทรัพยากรด้วย cgroup
- unit file: `ExecStart`, `User`, `EnvironmentFile`, `Restart`, `TimeoutStopSec`, `WantedBy`
- แก้ unit → `daemon-reload` · ดูสถานะ → `status` · ดู log → `journalctl -u <name>`
- `on-failure` ไม่ restart เมื่อ exit 0 หรือโดน TERM/INT/HUP
- timeout ของ app < `TimeoutStopSec` < สิ่งที่ user ยอมรอ
- systemd รู้แค่ว่า process "ยังไม่ตาย" ไม่ได้รู้ว่า "ยังทำงานถูก"

## 15. เรียนต่อ

**Lesson 6 — เมื่อทรัพยากรหมด:** memory เต็ม, disk เต็ม, CPU 100% — Linux ทำอะไร และเราตามหาต้นเหตุอย่างไร
