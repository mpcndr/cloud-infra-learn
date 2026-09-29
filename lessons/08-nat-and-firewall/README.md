# Lesson 8 — NAT และ Firewall: ออกได้ แต่เข้าไม่ได้

Phase 2 · Networking · Level 1–5 · Project: Pixbin Stage 1

## 1. ปัญหาที่เรากำลังแก้

จาก Lesson 7:
- VM มี IP `192.168.64.5` ซึ่งเป็น **private IP** — ไม่มีอยู่บนอินเทอร์เน็ต
- แต่ VM `apt install` ได้ ดาวน์โหลดของจากอินเทอร์เน็ตได้ — แล้ว server ปลายทางตอบกลับมาหา "192.168.64.5" ได้อย่างไร?
- ขณะเดียวกัน ไม่มีใครบนอินเทอร์เน็ตเริ่มคุยกับ VM ได้เลย

คำตอบคือ **NAT** ซึ่งทำให้ "ออกได้" และ **Firewall** ซึ่งเป็นตัวตัดสินอย่างตั้งใจว่า "ใครเข้าได้"
สองอย่างนี้คือหัวใจของ network security ใน cloud: Security Group, NAT Gateway, private subnet ล้วนเป็นเรื่องในบทนี้

## 2. Mental Model: พนักงานต้อนรับของตึก

**NAT** — ตึกสำนักงานที่มีเบอร์โทรภายนอกเบอร์เดียว
- พนักงานในตึก (private IP) โทรออกไปข้างนอก → โอเปอเรเตอร์โทรออกให้ด้วยเบอร์ของตึก และ **จดไว้ว่า "สายนี้เป็นของโต๊ะ 5"**
- ปลายทางโทรกลับเบอร์ตึก → โอเปอเรเตอร์ดูสมุด → ต่อไปที่โต๊ะ 5
- คนนอกที่โทรเข้ามาเฉย ๆ โดยไม่เคยมีใครโทรออกไปหา → ไม่มีในสมุด → **ไม่รู้จะต่อไปโต๊ะไหน** → สายหลุด

**Firewall** — ยามที่มีรายชื่อ: "เข้าได้เฉพาะคนแบบนี้ ไปห้องนี้"
- ยามที่ดีจำได้ว่า "คนนี้เป็นแขกที่พนักงานเชิญมา (ตอบกลับการคุยที่เราเริ่ม)" → ให้ผ่าน = **stateful**

> ⚠️ ขอบเขตของ analogy: NAT **ไม่ได้ถูกออกแบบมาเพื่อความปลอดภัย** มันถูกสร้างเพราะ IPv4 ไม่พอ
> การที่ "คนนอกเข้าไม่ได้" เป็นผลข้างเคียง อย่าใช้ NAT แทน firewall

## 3. Concept

**NAT (Network Address Translation)** — router เปลี่ยน IP/port ใน packet ระหว่างทาง
- **SNAT / Masquerade** (ขาออก): เปลี่ยน source IP ของเครื่องข้างในเป็น IP ของ router
  `192.168.64.5:51000 → 1.1.1.1:443` กลายเป็น `203.0.113.7:62001 → 1.1.1.1:443`
- **DNAT / Port forwarding** (ขาเข้า): "อะไรที่เข้ามาที่ port 8080 ของ router ส่งต่อไปเครื่อง 192.168.1.20:8000"
- router จำการแปลงแต่ละ connection ไว้ใน **connection tracking table (conntrack)**

**Public vs Private IP**
- public IP = มีเส้นทางจากอินเทอร์เน็ตมาถึงได้ · private IP = มีความหมายเฉพาะในวงของตัวเอง
- บ้านส่วนใหญ่มี public IP เดียว (หรือไม่มีเลย ถ้า ISP ทำ NAT ซ้อนอีกชั้น = **CGNAT**)

**Firewall** — กฎว่า packet แบบไหน ผ่าน/ไม่ผ่าน ดูจาก: ทิศทาง, protocol, source IP, destination port
- **Stateful**: จำ connection ได้ ตอบกลับของ connection ที่เราเริ่มผ่านอัตโนมัติ (ufw, AWS Security Group)
- **Stateless**: ดูทีละ packet ไม่จำอะไร ต้องเขียนกฎทั้งขาไปและขากลับ (AWS Network ACL)
- **DROP** = ทิ้งเงียบ ๆ → ปลายทางรอจน **timeout**
- **REJECT** = ตอบกลับว่าไม่รับ → ปลายทางเห็น **connection refused** ทันที

**Default deny** — "ห้ามทุกอย่าง ยกเว้นที่อนุญาต" ดีกว่า "อนุญาตทุกอย่าง ยกเว้นที่ห้าม" เสมอ
เพราะอย่างแรก ถ้าลืมก็ปลอดภัย อย่างหลัง ถ้าลืมก็รั่ว

## 4. สิ่งที่เกิดขึ้นข้างใน

VM เรียก `curl https://ifconfig.me` — packet ผ่าน NAT สองชั้น:

```
VM 192.168.64.5:51000 ──► dst 34.x.x.x:443
   │
Mac (NAT ชั้นที่ 1): src 192.168.64.5:51000 → 192.168.1.20:40001    จดใน conntrack
   │
Router บ้าน (NAT ชั้นที่ 2): src 192.168.1.20:40001 → 203.0.113.7:62001  จดใน conntrack
   │
ifconfig.me เห็นว่าคนเรียกคือ 203.0.113.7 → ตอบกลับไปที่นั่น
   │
Router บ้าน: 203.0.113.7:62001 เป็นของใคร? ดูสมุด → 192.168.1.20:40001
Mac:         192.168.1.20:40001 เป็นของใคร? ดูสมุด → 192.168.64.5:51000
   │
VM ได้ response ✅
```

ถ้ามี packet เข้ามาที่ `203.0.113.7:9999` โดยไม่มีใครเริ่มก่อน → router ไม่มีในสมุด → ทิ้ง
→ นี่คือสาเหตุที่ "เพื่อนบนอินเทอร์เน็ตเข้า VM ไม่ได้" ถึงแม้จะไม่มี firewall เลย

**Firewall ใน Linux** = **netfilter** ใน kernel (ตัวเดียวกับที่ทำ NAT)
`ufw` เป็นแค่ตัวช่วยเขียนกฎให้ง่าย ข้างใต้คือ iptables/nftables → netfilter
Docker และ Kubernetes ก็เขียนกฎลง netfilter เพื่อทำ port mapping — จะเจออีกใน Phase 5 และ 16

## 5. Diagram — ใครคุยกับใครได้

```
                      อินเทอร์เน็ต
                           │  ❌ เริ่มเข้ามาเองไม่ได้ (NAT ไม่รู้จะส่งให้ใคร)
                           │  ✅ ตอบกลับสิ่งที่ข้างในเริ่ม
                    ┌──────┴───────┐
                    │ Router บ้าน   │ public 203.0.113.7  (NAT + firewall)
                    └──────┬───────┘
          192.168.1.0/24   │
      ┌─────────┬──────────┴───────────┐
   มือถือ .35     Mac .20 (NAT ให้ VM)
      │            │  bridge100 192.168.64.1
      │            │
      │  ❌ มือถือไม่มี route ไป 192.168.64.0/24    ✅ Mac ↔ VM (วงเดียวกัน)
      │            │
                  VM 192.168.64.5  (ufw: default deny incoming)
```

## 6. Example จริง — ufw

```bash
sudo ufw default deny incoming     # ขาเข้า: ห้ามทุกอย่าง
sudo ufw default allow outgoing    # ขาออก: อนุญาต (ตอบกลับผ่านได้เพราะ stateful)
sudo ufw allow OpenSSH             # port 22 — ต้องมาก่อน enable!
sudo ufw allow from 192.168.64.1 to any port 8001 proto tcp   # เฉพาะ Mac เข้า 8001 ได้
sudo ufw enable
sudo ufw status numbered
```

> 🚨 **ถ้า enable ufw โดยไม่ allow SSH คุณจะถูกล็อกออกจาก VM** (`multipass shell` ใช้ SSH)
> วิธีกู้คือลบ VM แล้วสร้างใหม่ — นี่คือบทเรียนราคาถูกของสิ่งที่ใน production เรียกว่า "ล็อกตัวเองออกจาก server"

## 7. Hands-on Lab

### ข้อ 1 — public IP กับ private IP
```bash
# Mac
ifconfig en0 | grep "inet "
curl -s https://ifconfig.me; echo
# VM
ip -br addr
curl -s https://ifconfig.me; echo
```
Expected: Mac กับ VM มี private IP ต่างกัน แต่ **public IP ที่โลกเห็นเป็นตัวเดียวกัน**
อธิบายด้วยข้อ 4 ว่าเพราะอะไร

### ข้อ 2 — ใครเห็นใคร
เปิด pixbin ใน VM ให้ฟังทุก interface ที่ port 8001 (ตัว systemd ที่ 8000 ปล่อยไว้)
```bash
# VM, V1
DATA_DIR=/tmp/px HOST=0.0.0.0 PORT=8001 ~/cloud-infra-learn/pixbin/pixbin-linux
# Mac
curl http://<IP ของ VM>:8001/
```
ดูบรรทัด `you came from:` — เป็น IP อะไร? ทำไมไม่ใช่ `192.168.1.20` (IP Wi-Fi ของ Mac)

จากมือถือ (Wi-Fi เดียวกัน) เปิด `http://<IP ของ VM>:8001` → เข้าได้ไหม? อาการเป็น refused หรือ timeout?
ใช้ Lesson 7 อธิบายว่ามือถือส่ง packet ไปที่ไหน

### ข้อ 3 — เปิด firewall (อ่านคำเตือนในข้อ 6 ก่อน!)
```bash
# VM, V2
sudo ufw status verbose
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow OpenSSH
sudo ufw enable
sudo ufw status verbose
```
```bash
# Mac
curl -m 5 http://<IP ของ VM>:8001/
```
Expected: รอ 5 วินาทีแล้ว timeout — ufw default คือ **DROP**
และ VM ยัง `curl -s https://ifconfig.me` ได้ปกติ — ทั้งที่ห้ามขาเข้าทั้งหมด ทำไม response กลับเข้ามาได้?

### ข้อ 4 — DROP vs REJECT
```bash
# VM
sudo ufw reject 8001/tcp
# Mac
time curl -m 5 http://<IP ของ VM>:8001/
```
Expected: `Connection refused` ทันที
เทียบกับข้อ 3 แล้วเทียบกับ Lesson 1: ตอนนี้ refused มาจากใคร? (ใบ้: pixbin ยังรันอยู่ที่ port นี้)

### ข้อ 5 — เปิดเฉพาะคนที่ควรเข้า
```bash
# VM
sudo ufw delete reject 8001/tcp
sudo ufw allow from 192.168.64.1 to any port 8001 proto tcp
sudo ufw status numbered
# Mac
curl -m 5 http://<IP ของ VM>:8001/
```
Expected: Mac เข้าได้ — ลองเปลี่ยนกฎเป็น IP อื่น (`192.168.64.99`) แล้วลองใหม่ เข้าได้ไหม

### ข้อ 6 — ดูสมุด conntrack
```bash
sudo apt install -y conntrack
curl -s https://ifconfig.me > /dev/null &
sudo conntrack -L 2>/dev/null | grep -E "dport=(443|8001)" | head
```
แต่ละบรรทัดมีสองชุด: ขาไป (src → dst) และขากลับที่คาดไว้ — นี่คือสิ่งที่ทำให้ firewall "จำได้"

### ข้อ 7 — เก็บกวาด
```bash
# V1: Ctrl+C pixbin 8001
sudo ufw delete allow from 192.168.64.1 to any port 8001 proto tcp
sudo ufw status numbered            # ควรเหลือแค่ OpenSSH
```
**ปล่อย ufw เปิดไว้** (default deny + SSH) — บทถัด ๆ ไปเราจะเปิดเฉพาะ port ที่จำเป็น

### ตรวจว่าทำถูก
- [ ] อธิบายได้ว่าทำไม public IP ของ Mac กับ VM เหมือนกัน
- [ ] อธิบายได้ว่าทำไมขาเข้าถูกห้ามทั้งหมด แต่ VM ยังโหลดเว็บได้
- [ ] แยกได้ว่า timeout ในข้อ 3 และ refused ในข้อ 4 เกิดจากกฎแบบไหน

### Common errors
| อาการ | สาเหตุ |
|---|---|
| `multipass shell` ค้างหลัง enable ufw | ลืม allow SSH — ต้องลบ VM สร้างใหม่ (`multipass delete lab && multipass purge`) แล้วทำ Lesson 3–5 ซ้ำ |
| ข้อ 2 Mac เข้าไม่ได้ตั้งแต่แรก | pixbin bind 127.0.0.1 (ลืม `HOST=0.0.0.0`) — ตรวจด้วย `ss -tlnp` |
| `curl ifconfig.me` ไม่ได้ | ปัญหา DNS หรือ network ของ Mac — ลอง `curl -s https://1.1.1.1` |

## 8. Debugging

**Scenario:** "เปิด port แล้วแต่ยังเข้าไม่ได้"

ตรวจจากในออกนอก — ทุกชั้นต้องผ่าน:

| ชั้น | คำถาม | ตรวจด้วย |
|---|---|---|
| app | ฟังอยู่จริงไหม ที่ address ไหน | `ss -tlnp` (Lesson 1) |
| host firewall | ufw/iptables ยอมไหม | `sudo ufw status numbered` · ลอง curl จากในเครื่องเอง |
| routing | ผู้เรียกมีทางมาถึงไหม | `ip route get` ฝั่งผู้เรียก, `traceroute` |
| NAT / network firewall | router / Security Group / NACL ยอมไหม | config ของ cloud · มี port forward ไหม |

**อาการบอกชั้น:**
- refused → มาถึงเครื่องแล้ว แต่ไม่มีใครฟัง หรือ firewall REJECT
- timeout → ถูก DROP ระหว่างทาง หรือไม่มี route กลับ หรือ app ค้าง
- ทดสอบแยกชั้น: `curl` จากในเครื่อง → จากเครื่องในวงเดียวกัน → จากข้างนอก ชั้นที่เริ่มพังคือชั้นที่มีปัญหา

## 9. Failure Scenarios

| เกิดอะไร | อาการ | ข้างใน |
|---|---|---|
| enable firewall โดยไม่เปิด SSH | ล็อกตัวเองออก | default deny ทำงานตามที่สั่ง |
| conntrack table เต็ม (traffic สูงมาก) | connection ใหม่หลุดแบบสุ่ม, kernel log `nf_conntrack: table full` | router/เครื่องจำ connection ไม่ไหว |
| เปิด DB port ต่ออินเทอร์เน็ต | โดนสแกนภายในไม่กี่นาที โดน brute force/ขโมยข้อมูล | อินเทอร์เน็ตมีบอทสแกนทุก IP ตลอดเวลา |
| กฎ allow ขาออกถูกปิดโดยไม่ตั้งใจ | server เรียก API ภายนอก/DNS ไม่ได้ | egress ก็ต้องอนุญาต |
| NAT Gateway ใน cloud ล่ม/ถูกลบ | private subnet ออกอินเทอร์เน็ตไม่ได้ (update, เรียก API) แต่ขาเข้ายังปกติ | ทางออกเดียวหาย |

## 10. Trade-offs

**Host firewall (ufw) vs Network firewall (Security Group ใน cloud)**
- host firewall: อยู่ในเครื่อง ละเอียด — แต่ถ้าเครื่องโดนยึด คนร้ายปิดได้ และถ้าตั้งผิดก็ล็อกตัวเองออก
- network firewall: อยู่นอกเครื่อง คนในเครื่องแก้ไม่ได้ แก้ได้จาก console แม้ตั้งผิด — แต่ต้องพึ่ง cloud provider
- production: ใช้ network firewall เป็นหลัก host firewall เป็นชั้นเสริม (**defense in depth**)

**Stateful vs Stateless**
- stateful: เขียนกฎง่าย (คิดแค่ขาเข้า) — แต่ต้องใช้ memory จำ connection (conntrack เต็มได้)
- stateless: ไม่มี state ให้เต็ม เร็ว — แต่ต้องเขียนกฎขากลับเอง รวมถึง ephemeral port ของ client (Lesson 1) ซึ่งพลาดง่าย

**Private IP + NAT vs Public IP ทุกเครื่อง**
- private + NAT: ประหยัด public IP, เครื่องไม่ถูกสแกนตรง ๆ — แต่ NAT เป็นจุดที่ต้องดูแล (และใน AWS คิดเงินต่อ GB)
- public ทุกเครื่อง: ตรงไปตรงมา — แต่ทุกเครื่องเป็นเป้า ต้องพึ่ง firewall ล้วน ๆ และเสียค่า IPv4

## 11. Production Considerations

- **Least privilege ของ network:** เปิดเฉพาะ port ที่ต้องใช้ ให้เฉพาะ source ที่ต้องเข้า (web → ทุกคนที่ 443 · SSH → เฉพาะ IP ของทีม หรือไม่เปิดเลยแล้วใช้ช่องทางอื่น · DB → เฉพาะ app server)
- **Database ต้องไม่มี route จากอินเทอร์เน็ต** (private subnet) และไม่มี public IP — ไม่ใช่แค่ "ปิดด้วย firewall"
- **Cost:** AWS NAT Gateway คิดทั้งรายชั่วโมงและต่อ GB ที่ผ่าน — ระบบที่ดาวน์โหลดของเยอะจาก private subnet อาจเสียค่า NAT มากกว่าค่า server
- **เขียนกฎ firewall เป็น code** (Phase 8) จะได้ review ได้ ย้อนดูได้ว่าใครเปิดอะไรเมื่อไหร่

## 12. Quiz

1. **(เข้าใจ)** ทำไม NAT ถึงทำให้คนข้างนอก "เริ่ม" คุยกับเครื่องข้างในไม่ได้ ทั้งที่ไม่มีกฎ firewall ใด ๆ
2. **(ประยุกต์)** ตั้ง stateless firewall ให้ server ที่รับ HTTPS (443) ต้องเขียนกฎขาออกอย่างไร ให้ response กลับไปถึง client ได้ (ใบ้: client ใช้ port อะไร — Lesson 1)
3. **(debugging)** เพื่อนเปิด port 5432 ใน Security Group แล้ว แต่ยัง connect DB ไม่ได้ อาการคือ refused ทันที คุณจะไม่ตรวจอะไร และจะตรวจอะไรเป็นอย่างแรก
4. **(architecture)** Pixbin มี web server, database และ admin (SSH) ให้ออกแบบกฎ firewall ของแต่ละเครื่อง (ขาเข้า: port, source) โดยใช้หลัก default deny
5. **(trade-off)** "ใช้ DROP ดีกว่า REJECT เพราะคนร้ายจะไม่รู้ว่ามีเครื่องอยู่" — เห็นด้วยไหม มีข้อเสียอะไรกับทีมตัวเองบ้าง

## 13. Challenge

1. ทำให้ **มือถือ** เข้า pixbin ใน VM ได้ **โดยไม่เปลี่ยน network ของ multipass** (ใบ้: Mac อยู่ทั้งสองวง — ใช้ Mac เป็นตัวกลาง เช่น `ssh -L` หรือ port forwarding) อธิบายว่า packet เดินอย่างไร และทำไมใน production เราไม่ทำแบบนี้
2. ใช้ `sudo nft list ruleset | less` (หรือ `sudo iptables -S`) หากฎที่ ufw สร้างให้ สำหรับ OpenSSH และ default deny แล้วอธิบายว่า ufw แปลงคำสั่งสั้น ๆ เป็นกฎอะไร
3. หา public IP ของบ้านคุณจาก ifconfig.me แล้วเทียบกับ WAN IP ในหน้า admin ของ router บ้าน ถ้าไม่ตรงกัน แปลว่าอะไร (CGNAT) และส่งผลอย่างไรถ้าอยากเปิด server ที่บ้าน

## 14. สรุปสิ่งที่ต้องจำ

- private IP ออกอินเทอร์เน็ตได้ผ่าน **SNAT** ซึ่งจำ connection ไว้ใน conntrack
- คนนอก "เริ่ม" เข้ามาไม่ได้เพราะ NAT ไม่รู้จะส่งให้ใคร — แต่ NAT ไม่ใช่ security
- firewall: default deny · stateful จำ connection · DROP = timeout · REJECT = refused
- เปิด SSH ก่อน enable firewall เสมอ
- ตรวจ "เข้าไม่ได้" ทีละชั้น: app → host firewall → routing → network firewall/NAT

## 15. เรียนต่อ

**Lesson 9 — TCP และ UDP:** มองเข้าไปใน connection จริง ๆ ด้วย tcpdump — handshake, การส่งซ้ำ และทำไม latency ถึงถูกคูณ
