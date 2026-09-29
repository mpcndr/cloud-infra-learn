# Lesson 4 — Filesystem, Users และ Permissions: process รันในนามของใคร

Phase 1 · Linux · Level 1–4 · Project: Pixbin Stage 1

## 1. ปัญหาที่เรากำลังแก้

Pixbin รับ upload ไฟล์จาก user สมมติว่าวันหนึ่งมีคนเจอช่องโหว่ที่ทำให้สั่ง pixbin อ่าน/เขียนไฟล์ path อะไรก็ได้

- ถ้า pixbin รันเป็น **root** → คนร้ายอ่าน `/etc/shadow` (hash รหัสผ่าน), แก้ SSH key, ติดตั้ง backdoor ได้ทั้งเครื่อง
- ถ้า pixbin รันเป็น user ที่มีสิทธิ์ **แค่โฟลเดอร์ upload** → คนร้ายได้แค่โฟลเดอร์นั้น

ความต่างระหว่างสองกรณีนี้ไม่ได้อยู่ที่โค้ด แต่อยู่ที่ **process รันในนามของใคร และคนนั้นมีสิทธิ์อะไร**
หลักนี้เรียกว่า **Least Privilege** และจะตามเราไปทุก phase (IAM ใน AWS, ServiceAccount ใน K8s ก็คือเรื่องเดียวกัน)

## 2. Mental Model: บัตรพนักงาน + ป้ายหน้าห้อง

- **User** = บัตรพนักงาน ทุก process ถือบัตรใบหนึ่ง (ของคนที่รันมัน)
- **Group** = แผนก บัตรหนึ่งใบสังกัดได้หลายแผนก
- **Permission** = ป้ายหน้าห้อง บอกว่า "เจ้าของห้องทำอะไรได้ · คนในแผนกทำอะไรได้ · คนอื่นทำอะไรได้"
- **root** = บัตร master — kernel ไม่ตรวจป้ายเลย

ทุกครั้งที่ process เปิดไฟล์ kernel ถามแค่: "บัตรใบนี้ เจอป้ายนี้ ทำสิ่งที่ขอได้ไหม"

> ⚠️ ขอบเขตของ analogy: ของจริง "ห้อง" มีสองแบบ คือไฟล์กับโฟลเดอร์ และสิทธิ์บนโฟลเดอร์ความหมายไม่เหมือนสิทธิ์บนไฟล์
> (ข้อ 3) ตรงนี้แหละที่คนพลาดกันบ่อยที่สุด

## 3. Concept

**Filesystem: ทุกอย่างอยู่ใต้ `/`**

| Path | เก็บอะไร |
|---|---|
| `/etc` | config ของระบบ (`/etc/passwd`, `/etc/nginx/`, `/etc/systemd/`) |
| `/var` | ข้อมูลที่เปลี่ยนตลอด: `/var/log` (log), `/var/lib` (ข้อมูลของ service) |
| `/usr/bin`, `/usr/local/bin` | program · `/usr/local` = ของที่เราติดตั้งเอง |
| `/home/<user>` | ไฟล์ของ user แต่ละคน · `/root` = home ของ root |
| `/tmp` | ไฟล์ชั่วคราว ทุกคนเขียนได้ (มักหายเมื่อ reboot) |
| `/proc`, `/sys` | ไม่ใช่ไฟล์จริง kernel สร้างให้ดูสถานะระบบ (เจอแล้วใน Lesson 3) |
| `/dev` | อุปกรณ์ เช่น disk (`/dev/vda`), `/dev/null` (หลุมดำ) |

"Everything is a file" — disk, terminal, process info ถูกเปิดให้เห็นเป็นไฟล์ ทำให้ใช้เครื่องมือชุดเดียว (`cat`, `ls`) ดูได้ทุกอย่าง

**Users และ Groups**
- user จริง ๆ คือ **ตัวเลข UID** · ชื่อเป็นแค่ป้ายที่เก็บใน `/etc/passwd`
- `UID 0` = root เสมอ ไม่ว่าจะชื่ออะไร
- **system user** = user สำหรับ service ไม่มีรหัสผ่าน ไม่มี shell login ไม่ได้ (`nologin`)
- `id` บอกว่าตัวเองเป็นใคร อยู่กลุ่มไหน

**Permission bits** — `ls -l` แสดงแบบนี้:
```
-rwxr-x---  1  pixbin  pixbin  8.1M  pixbin
│└┬┘└┬┘└┬┘      owner   group
│ │  │  └─ others: ---  (ไม่มีสิทธิ์)
│ │  └──── group : r-x
│ └─────── owner : rwx
└───────── ชนิด: - ไฟล์, d โฟลเดอร์, l symlink
```
เขียนเป็นเลขฐาน 8: r=4, w=2, x=1 → `rwxr-x---` = 7 5 0 = `750`

**ความหมายต่างกันระหว่างไฟล์กับโฟลเดอร์** (สำคัญมาก)

| สิทธิ์ | บนไฟล์ | บนโฟลเดอร์ |
|---|---|---|
| r | อ่านเนื้อหา | ดูรายชื่อไฟล์ข้างใน (`ls`) |
| w | แก้เนื้อหา | **สร้าง / ลบ / เปลี่ยนชื่อ** ไฟล์ข้างใน |
| x | รันเป็น program | **เข้าไปผ่าน** (`cd`, เข้าถึงไฟล์ข้างในด้วยชื่อ) |

ผลที่ตามมา: ลบไฟล์ได้หรือไม่ ขึ้นกับสิทธิ์ **ของโฟลเดอร์** ไม่ใช่ของไฟล์
และไฟล์ที่มีอยู่แล้วแก้ได้ ถึงโฟลเดอร์จะห้ามสร้างไฟล์ใหม่ก็ตาม (Lab ข้อ 5)

**sudo** — "รันคำสั่งนี้ในนามของ user อื่น" (default = root) ตรวจสิทธิ์จาก `/etc/sudoers`
`sudo -u pixbin <cmd>` = รันในนามของ pixbin

## 4. สิ่งที่เกิดขึ้นข้างใน

เมื่อ process เรียก `open("/var/lib/pixbin/a.bin", write)`:

```
kernel ดู UID/GID ของ process (บัตร)
 ├─ UID = 0 (root)? → อนุญาต (ข้ามการตรวจเกือบทั้งหมด)
 ├─ ต้องมี x บนทุกโฟลเดอร์ในเส้นทาง: /  /var  /var/lib  /var/lib/pixbin
 ├─ ไฟล์ยังไม่มี → ต้องมี w บนโฟลเดอร์ /var/lib/pixbin
 └─ ไฟล์มีแล้ว  → ต้องมี w บนตัวไฟล์
    ใช้ชุดสิทธิ์ของ owner ถ้า UID ตรง · ของ group ถ้าอยู่กลุ่มนั้น · ไม่งั้นใช้ others
ไม่ผ่าน → errno EACCES → program เห็น "permission denied"
```

metadata ของไฟล์ (owner, สิทธิ์, ขนาด, ตำแหน่งข้อมูลบน disk) เก็บใน **inode** ส่วนชื่อไฟล์เก็บใน "โฟลเดอร์"
— นี่คือเหตุผลที่สิทธิ์ของโฟลเดอร์คุมการสร้าง/ลบ/เปลี่ยนชื่อ เพราะการกระทำพวกนั้นคือการแก้ "รายการชื่อ" ในโฟลเดอร์

## 5. Diagram — Pixbin แบบ least privilege

```
                    ┌────────────── VM ──────────────────────┐
  request ─────────►│ pixbin process  (UID=pixbin, ไม่ใช่ root)│
                    │     │                                   │
                    │     ├─ /var/lib/pixbin/   drwxr-x--- pixbin   ✅ เขียนได้
                    │     ├─ /usr/local/bin/pixbin -rwxr-xr-x root  ✅ รันได้ ❌ แก้ไม่ได้
                    │     ├─ /etc/shadow        -rw-r----- root     ❌
                    │     ├─ /home/ubuntu/      drwxr-x--- ubuntu   ❌
                    │     └─ port < 1024                            ❌ (ต้องมีสิทธิ์พิเศษ)
                    └────────────────────────────────────────────┘
  ถ้า pixbin โดนเจาะ → คนร้ายได้แค่สิ่งที่มี ✅
```

## 6. Example จริง

จัดวาง Pixbin แบบที่ server จริงทำ:

| สิ่งของ | ที่อยู่ | owner | สิทธิ์ | เหตุผล |
|---|---|---|---|---|
| program | `/usr/local/bin/pixbin` | root | 755 | ทุกคนรันได้ แต่ pixbin แก้ตัวเองไม่ได้ (โดนเจาะก็ฝัง backdoor ในตัว program ไม่ได้) |
| ข้อมูล upload | `/var/lib/pixbin` | pixbin | 750 | pixbin เขียนได้ คนอื่นนอกกลุ่มเข้าไม่ได้ |
| user ที่รัน | `pixbin` | — | — | system user ไม่มี shell ไม่มีรหัสผ่าน |

## 7. Hands-on Lab

ทำใน VM (`multipass shell lab`) ถ้ายังไม่ได้ build: บน Mac `GOOS=linux GOARCH=arm64 go build -o pixbin-linux .` ในโฟลเดอร์ `pixbin`

### ข้อ 1 — ตัวเองคือใคร
```bash
id
ls -l /etc/passwd /etc/shadow
grep -E '^(root|ubuntu)' /etc/passwd
cat /etc/shadow
```
- `/etc/passwd` ทุกคนอ่านได้ แต่ `/etc/shadow` (hash รหัสผ่าน) อ่านได้แค่ root กับกลุ่ม shadow
- ดูคอลัมน์ใน `/etc/passwd`: `ชื่อ:x:UID:GID:คำอธิบาย:home:shell`

### ข้อ 2 — สร้าง system user และติดตั้ง program
```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin pixbin
id pixbin
sudo install -m 755 ~/cloud-infra-learn/pixbin/pixbin-linux /usr/local/bin/pixbin
ls -l /usr/local/bin/pixbin
```
- `--system` = UID ช่วงของ service (ต่ำกว่า 1000) · `--shell /usr/sbin/nologin` = login เข้ามาไม่ได้
- `install -m 755` = copy + ตั้งสิทธิ์ในคำสั่งเดียว ไฟล์ที่ได้เป็นของ root (เพราะใช้ sudo)

### ข้อ 3 — เตรียมโฟลเดอร์ข้อมูล
```bash
sudo mkdir -p /var/lib/pixbin
sudo chown pixbin:pixbin /var/lib/pixbin
sudo chmod 750 /var/lib/pixbin
ls -ld /var/lib/pixbin
ls /var/lib/pixbin          # ในฐานะ ubuntu เข้าได้ไหม? ทำไม
```

### ข้อ 4 — รัน pixbin ในนามของ pixbin
```bash
# V1
sudo -u pixbin env DATA_DIR=/var/lib/pixbin /usr/local/bin/pixbin
# V2
ps -o user,pid,cmd -p $(pgrep -x pixbin)
curl localhost:8000/healthz
echo "hello" | curl --data-binary @- localhost:8000/upload
sudo ls -l /var/lib/pixbin
```
- `env DATA_DIR=...` ใส่ env variable ให้ process ที่ sudo สร้าง (sudo ล้าง env ส่วนใหญ่ทิ้งเพื่อความปลอดภัย)
Expected: `ps` แสดง USER = pixbin และไฟล์ที่ upload เป็นของ `pixbin`

### ข้อ 5 — health check โกหก
```bash
sudo chmod 550 /var/lib/pixbin        # ถอด w ของ owner ออก
curl -w ' %{http_code}\n' localhost:8000/healthz
echo "hello" | curl -w ' %{http_code}\n' --data-binary @- localhost:8000/upload
```
Expected: `/healthz` ยังตอบ `ok 200` แต่ upload ได้ `500`
ดู log ใน V1 → `permission denied`
**ก่อนอ่านต่อ ให้อธิบายเองว่าทำไม healthz ยังผ่าน** (ใบ้: ดูโค้ด `/healthz` ใน main.go แล้วดูตาราง "ไฟล์ vs โฟลเดอร์")

คืนสิทธิ์: `sudo chmod 750 /var/lib/pixbin`

### ข้อ 6 — ลองเป็นคนร้าย
สมมติคุณยึด process pixbin ได้แล้ว ลองทำสิ่งที่คนร้ายอยากทำ:
```bash
sudo -u pixbin cat /etc/shadow
sudo -u pixbin ls /home/ubuntu
sudo -u pixbin sh -c 'echo evil > /usr/local/bin/pixbin'
sudo -u pixbin cat /proc/1/environ
```
ทุกข้อต้องได้ `Permission denied` — อธิบายว่าแต่ละข้อถูกกั้นด้วยสิทธิ์ตรงไหน

แล้วเทียบกับ:
```bash
sudo cat /etc/shadow | head -3        # ถ้าเป็น root
```

### ข้อ 7 — port ต่ำกว่า 1024
```bash
# Ctrl+C ตัวเดิมก่อน
sudo -u pixbin env DATA_DIR=/var/lib/pixbin PORT=80 /usr/local/bin/pixbin
cat /proc/sys/net/ipv4/ip_unprivileged_port_start
```
Expected: `bind: permission denied` และค่า 1024 — kernel ยอมให้ user ธรรมดา bind ได้ตั้งแต่ 1024 ขึ้นไปเท่านั้น
(คำตอบของ Challenge ข้อ 4 ใน Lesson 1 เริ่มจากตรงนี้ วิธีแก้แบบถูกต้องจะเห็นใน Lesson 5 และ 11)

### ตรวจว่าทำถูก
- [ ] อ่าน `ls -l` แล้วแปลงเป็นเลข 3 หลักได้ และบอกได้ว่า user หนึ่ง ๆ ทำอะไรกับไฟล์นั้นได้
- [ ] อธิบายข้อ 5 ได้ด้วยหลัก "สิทธิ์บนโฟลเดอร์ vs บนไฟล์"
- [ ] บอกได้ว่าข้อ 6 แต่ละคำสั่งถูกกั้นที่ไหน

### Common errors
| อาการ | สาเหตุ |
|---|---|
| `sudo: unable to execute ... Permission denied` | program ไม่มี x สำหรับ others หรือโฟลเดอร์ทางผ่านไม่มี x |
| `cannot create DATA_DIR` ตอน start | pixbin ไม่มีสิทธิ์สร้างโฟลเดอร์ใน `/var/lib` — ต้องสร้างให้ก่อนด้วย root (ข้อ 3) |
| `useradd: user 'pixbin' already exists` | ทำข้อ 2 ซ้ำ ข้ามได้ |
| รัน `sudo -u pixbin ~/cloud-infra-learn/...` ไม่ได้ | pixbin เข้า `/home/ubuntu` ไม่ได้ (สิทธิ์ 750) — ต้อง install ไปที่ `/usr/local/bin` |

## 8. Debugging

**Scenario:** "upload ไม่ได้ ได้ 500 แต่ health check เขียว"

| # | Hypothesis | วิธีพิสูจน์ |
|---|---|---|
| 1 | app ไม่มีสิทธิ์เขียนโฟลเดอร์ | log ของ app มี `permission denied` · `ls -ld` โฟลเดอร์ |
| 2 | app รันเป็น user ผิดตัว | `ps -o user,pid,cmd -p <PID>` |
| 3 | โฟลเดอร์ทางผ่านไม่มี x | `namei -l /var/lib/pixbin/x` แสดงสิทธิ์ทุกชั้นของ path |
| 4 | disk เต็ม | `df -h` (Lesson 6) — error จะเป็น `no space left on device` ไม่ใช่ permission |

`namei -l <path>` เป็นเครื่องมือที่ดีที่สุดสำหรับ "permission denied ที่หาไม่เจอ" เพราะแสดงทุกชั้นในบรรทัดเดียว

## 9. Failure Scenarios

| เกิดอะไร | อาการ | ข้างใน |
|---|---|---|
| deploy แล้วลืม chown โฟลเดอร์ใหม่ | upload พังหลัง deploy | โฟลเดอร์เป็นของ root |
| app รันเป็น root แล้วโดนเจาะ | เครื่องถูกยึดทั้งเครื่อง | root ข้ามการตรวจสิทธิ์ |
| `chmod -R 777` เพื่อแก้ปัญหาเร็ว ๆ | ใช้ได้ แต่ทุก user/process ในเครื่องแก้ข้อมูลได้ | ปิดปัญหาด้วยการเปิดช่องโหว่ |
| ไฟล์ config ที่มี secret สิทธิ์ 644 | user อื่นในเครื่องอ่าน password ได้ | others มี r |
| health check ตรวจไม่ตรงกับงานจริง | ระบบบอกว่าปกติ แต่ user ใช้ไม่ได้ | Lab ข้อ 5 |

## 10. Trade-offs

**รันเป็น root vs user เฉพาะ**
- root: ง่าย ไม่มีปัญหาสิทธิ์เลย — แต่ bug หนึ่งตัว = เสียทั้งเครื่อง
- user เฉพาะ: ต้องคิดเรื่องสิทธิ์ทุกไฟล์ ปัญหา "permission denied" มากขึ้น — แต่ความเสียหายถูกจำกัด
- ใน production เลือกอย่างหลังเสมอ ความยุ่งยากที่เพิ่มขึ้นคือราคาที่คุ้ม

**Unix permission ธรรมดา vs เครื่องมือละเอียดกว่า**
- rwx 3 ชุด: เข้าใจง่าย ครอบคลุมเกือบทุกงาน แต่หยาบ (ให้สิทธิ์ user คนที่ 2 แบบเจาะจงไม่ได้)
- ACL, capabilities, AppArmor/SELinux, systemd sandboxing: ละเอียดกว่า แต่ซับซ้อนและ debug ยาก
- เริ่มจาก rwx ให้ถูกก่อน แล้วค่อยเพิ่มชั้นเมื่อจำเป็น (Lesson 5 จะเพิ่ม systemd sandboxing)

## 11. Production Considerations

- **Least privilege ทุกชั้น:** 1 service = 1 user · program เป็นของ root แต่ service แก้ไม่ได้ · ข้อมูลเป็นของ service
- **Secret files:** สิทธิ์ 600 หรือ 640 และเป็นของ root/กลุ่มของ service เท่านั้น
- **ห้าม `chmod 777`** — ถ้ารู้สึกอยากใช้ แปลว่ายังหาต้นเหตุไม่เจอ
- **Health check ต้องตรวจสิ่งเดียวกับงานจริง** ถ้า app ต้องสร้างไฟล์ใหม่ health check ก็ควรสร้างไฟล์ใหม่ (ปรับ Pixbin ใน Challenge)
- **Cloud:** แนวคิดนี้เหมือนกันทุกประการกับ IAM Role ของ AWS — เครื่อง/service ได้สิทธิ์เท่าที่ต้องใช้ ไม่ใช้ key ของ admin

## 12. Quiz

1. **(เข้าใจ)** ทำไม program `/usr/local/bin/pixbin` ควรเป็นของ root ทั้งที่รันในนามของ pixbin? ถ้าให้เป็นของ pixbin จะเสี่ยงอะไร
2. **(ประยุกต์)** ไฟล์ `/data/report.txt` มีสิทธิ์ `rw-rw-rw-` (666) แต่โฟลเดอร์ `/data` มีสิทธิ์ `r-xr-xr-x` (555) user ธรรมดา แก้เนื้อหาไฟล์นี้ได้ไหม? ลบได้ไหม? เพราะอะไร
3. **(debugging)** app ได้ `permission denied` ตอนเปิด `/srv/app/data/x.db` คุณเช็ค `ls -l x.db` แล้วสิทธิ์ถูกต้องทุกอย่าง จะตรวจอะไรต่อ
4. **(architecture)** มี 2 service บนเครื่องเดียว: pixbin (web) และ thumbnailer (ย่อรูป) thumbnailer ต้องอ่านไฟล์ที่ pixbin upload และเขียน thumbnail ลงอีกโฟลเดอร์ ออกแบบ user, group, owner และสิทธิ์ของทุกโฟลเดอร์ ให้แต่ละตัวมีสิทธิ์น้อยที่สุด
5. **(trade-off)** ทีมบอกว่า "รัน app เป็น root ใน container ไม่เป็นไร เพราะ container แยกจากเครื่องอยู่แล้ว" คุณจะตอบอย่างไร (ตอบจากสิ่งที่รู้ตอนนี้ แล้วเก็บคำถามนี้ไว้ทบทวนอีกครั้งใน Phase 5)

## 13. Challenge

1. แก้ `/healthz` ของ Pixbin (ในเครื่องคุณ) ให้จับปัญหาในข้อ 5 ได้ แล้วพิสูจน์ว่าตอนนี้มันตอบ 503 เมื่อ upload ไม่ได้ อธิบายว่าการแก้ของคุณยังพลาดกรณีไหนได้อีก
2. สร้างโครงสร้างสำหรับ Quiz ข้อ 4 จริงใน VM (สร้าง user `thumb`, group, โฟลเดอร์) แล้วพิสูจน์ด้วย `sudo -u` ว่าแต่ละ user ทำได้/ไม่ได้ตามที่ออกแบบ
3. ใช้ `namei -l` หาว่าทำไม pixbin (user) เข้าถึง `/home/ubuntu/cloud-infra-learn/pixbin/pixbin-linux` ไม่ได้ แล้วเสนอ 2 วิธีแก้ พร้อมบอกว่าวิธีไหนดีกว่าและทำไม

## 14. สรุปสิ่งที่ต้องจำ

- ทุก process ถือ UID/GID และ kernel ตรวจสิทธิ์ทุกครั้งที่เปิดไฟล์ · root ข้ามการตรวจ
- `rwx` × owner/group/others · r=4 w=2 x=1
- **บนโฟลเดอร์:** w = สร้าง/ลบ/เปลี่ยนชื่อ · x = เข้าผ่าน
- ลบไฟล์ได้ไหมขึ้นกับโฟลเดอร์ ไม่ใช่ไฟล์
- Least privilege: 1 service = 1 system user, program เป็นของ root, ข้อมูลเป็นของ service
- `namei -l` สำหรับหาว่า path ติดสิทธิ์ที่ชั้นไหน

## 15. เรียนต่อ

**Lesson 5 — systemd, Service และ Logs:** ให้ OS ดูแล pixbin แทนเรา — start ตอน boot, restart เมื่อตาย, เก็บ log ให้
