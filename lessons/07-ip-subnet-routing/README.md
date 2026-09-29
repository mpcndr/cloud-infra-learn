# Lesson 7 — IP, Subnet และ Routing: packet เดินทางอย่างไร

Phase 2 · Networking · Level 0–4 · Project: Pixbin Stage 1

## 1. ปัญหาที่เรากำลังแก้

ตอนนี้ Mac คุยกับ VM ได้ (`multipass shell` ใช้งานได้) และ VM ออกอินเทอร์เน็ตได้ (`apt install` ได้)
แต่ถ้าเพื่อนที่บ้านอื่นพยายามเข้า VM ของคุณ → ไม่มีทางเข้าได้เลย

ทำไม? packet รู้ได้อย่างไรว่าต้องไปทางไหน และทำไมบางเส้นทางไปได้ บางเส้นทางไปไม่ได้

อีกไม่กี่บทเราจะสร้าง **VPC** ใน AWS ซึ่งคือการออกแบบ network เอง: เลือกช่วง IP, แบ่ง subnet, เขียน route table
ถ้าไม่เข้าใจบทนี้ หน้า VPC ของ AWS จะเป็นแค่ช่องให้กรอกตัวเลขที่ไม่รู้ความหมาย

## 2. Mental Model: ที่อยู่ไปรษณีย์ + ป้ายบอกทาง

- **IP address** = ที่อยู่ของบ้าน เช่น `192.168.64.5`
- **Subnet** = ซอย — บ้านในซอยเดียวกันส่งของหากันเองได้ตรง ๆ ไม่ต้องผ่านไปรษณีย์
- **Subnet mask / CIDR** = กติกาว่า "ส่วนไหนของที่อยู่คือชื่อซอย ส่วนไหนคือเลขบ้าน"
- **MAC address** = ชื่อเจ้าของบ้านที่เขียนบนกล่องจดหมาย ใช้ได้เฉพาะในซอยเดียวกัน
- **Router / Default gateway** = ที่ทำการไปรษณีย์ปากซอย — อะไรที่ไม่ใช่ในซอย ส่งให้มันจัดการ
- **Routing table** = ป้ายบอกทาง: "ปลายทางแบบนี้ ไปทางนี้"

ทุก router ตัดสินใจแค่ **"hop ถัดไปคือใคร"** ไม่มีใครรู้เส้นทางทั้งหมด — packet ถูกส่งต่อเป็นทอด ๆ จนถึงปลายทาง

> ⚠️ ขอบเขตของ analogy: บ้านจริงมีที่อยู่เดียว แต่เครื่องหนึ่งมีหลาย IP ได้ (หนึ่งต่อ interface — Lesson 1)
> และ "ชื่อเจ้าของ" (MAC) ไม่ได้ติดตัวไปตลอดทาง — ถูกเปลี่ยนทุกครั้งที่ข้าม router (ข้อ 4)

## 3. Concept

**IPv4 address** = ตัวเลข 32 bit เขียนเป็น 4 ส่วน ส่วนละ 8 bit (0–255): `192.168.64.5`

**CIDR** — `/n` บอกว่า **n bit แรกคือส่วน network** ที่เหลือคือส่วน host

| CIDR | mask | จำนวน address | ตัวอย่าง |
|---|---|---|---|
| `/32` | 255.255.255.255 | 1 | IP เดียว (ใช้ใน firewall rule) |
| `/24` | 255.255.255.0 | 256 | `192.168.64.0/24` = .0 ถึง .255 |
| `/20` | 255.255.240.0 | 4,096 | |
| `/16` | 255.255.0.0 | 65,536 | VPC ขนาดทั่วไป `10.0.0.0/16` |
| `/0` | 0.0.0.0 | ทั้งหมด | `0.0.0.0/0` = "ทุกปลายทาง" (default route) |

สูตร: จำนวน address = 2^(32−n) · ใน subnet ปกติ address แรก (network) และสุดท้าย (broadcast) ใช้กับเครื่องไม่ได้
(AWS จองเพิ่มอีก 3 ตัวต่อ subnet — /24 จึงใช้ได้จริง 251)

**Private IP** (ใช้ภายในองค์กร/บ้าน ไม่มีบนอินเทอร์เน็ต — RFC 1918)
- `10.0.0.0/8` · `172.16.0.0/12` · `192.168.0.0/16`
- ต่อไปเจอ IP ที่ขึ้นต้นแบบนี้ ให้รู้ทันทีว่า "เข้าจากอินเทอร์เน็ตตรง ๆ ไม่ได้" (Lesson 8 จะอธิบายว่าแล้วออกไปได้อย่างไร)

**Special**: `127.0.0.0/8` loopback · `0.0.0.0` "ไม่ระบุ/ทุก interface" (ตอน bind) · `169.254.0.0/16` link-local (ได้ IP นี้มักแปลว่าขอ IP จาก DHCP ไม่สำเร็จ — และ AWS ใช้ `169.254.169.254` เป็น metadata service)

**Routing table** — kernel ใช้ตัดสินว่าจะส่ง packet ออก interface ไหน ไปให้ใคร
- เลือก route ที่ **เจาะจงที่สุด** (prefix ยาวที่สุด) ที่ครอบปลายทาง = *longest prefix match*
- `default` (= `0.0.0.0/0`) ครอบทุกอย่าง จึงถูกใช้เมื่อไม่มีอันไหนเจาะจงกว่า

**MAC address + ARP**
- MAC = เลข 48 bit ของ network interface ใช้ส่ง frame ใน **network วงเดียวกัน** (Layer 2)
- **ARP** = ตะโกนถามในซอย "ใครคือ 192.168.64.1 ขอ MAC หน่อย" แล้วจำคำตอบไว้ (ARP table)

## 4. สิ่งที่เกิดขึ้นข้างใน

VM (`192.168.64.5/24`) ส่ง packet ไป `1.1.1.1`:

```
1. kernel ดู routing table:
     192.168.64.0/24 dev enp0s1          ← 1.1.1.1 ไม่อยู่ในนี้
     default via 192.168.64.1            ← ใช้อันนี้: ส่งให้ gateway
2. ต้องรู้ MAC ของ 192.168.64.1 → ดู ARP table → ไม่มี → ARP ถาม → ได้ MAC ของ Mac (bridge100)
3. ส่ง frame: [MAC ปลายทาง = Mac] [IP ปลายทาง = 1.1.1.1]
                ^ เปลี่ยนทุก hop       ^ ไม่เปลี่ยนตลอดทาง (ยกเว้นผ่าน NAT — Lesson 8)
4. Mac รับ → ดู routing table ของตัวเอง → ส่งต่อให้ router ที่บ้าน → ISP → ... → 1.1.1.1
   ทุก router ลดค่า TTL ลง 1 ถ้าเหลือ 0 ทิ้ง packet แล้วแจ้งกลับ (traceroute ใช้กลไกนี้)
```

ถ้าปลายทางอยู่ **ในซอยเดียวกัน** (เช่น `192.168.64.1`) ข้ามข้อ gateway ไป ARP หา MAC ของปลายทางเองแล้วส่งตรง

## 5. Diagram — network ที่คุณมีอยู่ตอนนี้

```
   อินเทอร์เน็ต (1.1.1.1, google.com ...)
          ▲
          │  public IP ของบ้าน (ISP ให้)
   ┌──────┴────────┐
   │ Router ที่บ้าน  │  192.168.1.1
   └──────┬────────┘
          │  ซอย 192.168.1.0/24 (Wi-Fi)
   ┌──────┴──────────────────────────────┐        มือถือ 192.168.1.35
   │ Mac   en0 = 192.168.1.20            │
   │       bridge100 = 192.168.64.1  ◄───┼── Mac ทำตัวเป็น router ให้ VM
   └──────┬──────────────────────────────┘
          │  ซอย 192.168.64.0/24 (network ของ Multipass)
   ┌──────┴────────┐
   │ VM "lab"      │  enp0s1 = 192.168.64.5
   │ default via 192.168.64.1
   └───────────────┘
```
(ตัวเลขของคุณอาจต่างจากนี้ — Lab ข้อ 1–2 ให้หาของจริงแล้ววาดใหม่)

## 6. Example จริง — อ่าน routing table

```
$ ip route
default via 192.168.64.1 dev enp0s1 proto dhcp src 192.168.64.5 metric 100
192.168.64.0/24 dev enp0s1 proto kernel scope link src 192.168.64.5
```
- บรรทัด 1: ปลายทางอื่น ๆ ทั้งหมด → ส่งให้ `192.168.64.1` ผ่าน `enp0s1` (ได้มาจาก DHCP)
- บรรทัด 2: `192.168.64.0/24` อยู่บน `enp0s1` ตรง ๆ (`scope link` = ไม่ต้องผ่าน router)

```
$ ip route get 1.1.1.1
1.1.1.1 via 192.168.64.1 dev enp0s1 src 192.168.64.5
```
`ip route get` = ถาม kernel ตรง ๆ ว่า "ถ้าส่งไปที่นี่ จะไปทางไหน" — เครื่องมือ debug ที่ดีที่สุดของบทนี้

## 7. Hands-on Lab

### ข้อ 1 — Mac ของคุณ
```bash
ifconfig en0 | grep "inet "          # IP และ netmask (เป็นเลขฐาน 16 เช่น 0xffffff00 = /24)
route -n get default | grep gateway  # default gateway
netstat -rn -f inet | head -15        # routing table
ifconfig bridge100 | grep "inet "    # ขา Mac ที่ต่อกับ VM
```

### ข้อ 2 — VM
```bash
multipass shell lab
ip -br addr                          # -br = แสดงแบบสั้น
ip route
ip -br link                          # MAC address ของแต่ละ interface
```
วาด diagram ข้อ 5 ใหม่ด้วยตัวเลขจริงของคุณ

### ข้อ 3 — ถาม kernel ว่าจะไปทางไหน
```bash
ip route get 1.1.1.1
ip route get 192.168.64.1             # ใช้ IP gateway ของคุณ
ip route get 127.0.0.1
```
เทียบผลทั้งสามบรรทัด: บรรทัดไหนมี `via`, บรรทัดไหนไม่มี, ทำไม

### ข้อ 4 — ARP
```bash
ip neigh                              # ARP table ของ VM
ping -c 2 192.168.64.1
ip neigh                              # มี MAC ของ gateway แล้ว
```
บน Mac:
```bash
arp -an | grep bridge100
```
หา MAC ของ VM ใน ARP table ของ Mac แล้วเทียบกับ `ip -br link` ใน VM — ตรงกันไหม

### ข้อ 5 — เดินตามเส้นทาง
```bash
sudo apt install -y traceroute
traceroute -n 1.1.1.1
```
- `-n` = ไม่แปลง IP เป็นชื่อ (เร็วกว่า และไม่พึ่ง DNS)
Expected: hop 1 = IP ของ Mac (bridge100), hop 2 = router ที่บ้าน, ถัดไป = ISP ...
hop ที่ขึ้น `* * *` = router ตัวนั้นไม่ตอบกลับ (ปิดไว้) ไม่ได้แปลว่าเส้นทางขาด

### ข้อ 6 — ทำ route หาย
```bash
sudo ip route del default
ip route
ping -c 2 1.1.1.1                     # ได้อะไร? เร็วแค่ไหน?
ping -c 2 192.168.64.1                # ยังได้ไหม?
```
Expected: `ping: connect: Network is unreachable` ทันที แต่ ping gateway ยังได้
(SSH ของคุณยังไม่หลุด เพราะ Mac อยู่ในซอยเดียวกัน ไม่ต้องใช้ default route — ลองอธิบายว่าทำไม)

คืนค่า:
```bash
sudo ip route add default via 192.168.64.1     # ใช้ IP gateway ของคุณ
ping -c 1 1.1.1.1
```

### ข้อ 7 — ส่งไปหาบ้านที่ไม่มีอยู่จริงในซอย
```bash
ping -c 3 192.168.64.250            # IP ในซอยเดียวกันที่ไม่มีเครื่องใช้
ip neigh | grep 192.168.64.250
```
Expected: `Destination Host Unreachable` และ ARP entry เป็น `FAILED` / `INCOMPLETE`
เทียบกับข้อ 6: "Network is unreachable" (ไม่รู้ทางไป) กับ "Host Unreachable" (รู้ทาง แต่หาเครื่องไม่เจอ) ต่างกันตรงไหน

### ข้อ 8 — คำนวณ CIDR (ทำในหัวก่อน แล้วตรวจด้วย Python)
```bash
python3 -c "
import ipaddress as ip
for c in ['10.0.0.0/16','10.0.1.0/24','10.0.1.0/28','172.20.5.9/20']:
    n = ip.ip_network(c, strict=False)
    print(f'{c:>16} -> {n}  first={n[0]} last={n[-1]} total={n.num_addresses}')
print(ip.ip_address('10.0.1.77') in ip.ip_network('10.0.1.64/26'))
"
```
ตอบก่อนรัน: `172.20.5.9/20` อยู่ใน network อะไร? `10.0.1.77` อยู่ใน `10.0.1.64/26` ไหม?

### ตรวจว่าทำถูก
- [ ] วาด network จริงของคุณ (Mac, VM, router บ้าน) พร้อม IP และ subnet ได้
- [ ] อธิบายได้ว่าทำไมข้อ 6 ping 1.1.1.1 ไม่ได้ แต่ SSH ยังอยู่
- [ ] คำนวณข้อ 8 ถูกโดยไม่ต้องพึ่ง Python

### Common errors
| อาการ | สาเหตุ |
|---|---|
| ไม่มี `bridge100` บน Mac | VM ไม่ได้รันอยู่ หรือชื่อ interface ต่าง (`ifconfig \| grep -B3 192.168.64`) |
| `traceroute` ขึ้น `*` ทุก hop | network ปิด UDP/ICMP ที่ traceroute ใช้ ลอง `traceroute -I -n 1.1.1.1` (ใช้ ICMP) |
| คืน default route ไม่ได้ | ใส่ IP gateway ผิด · หรือ `sudo netplan apply` / reboot VM เพื่อให้ DHCP ตั้งให้ใหม่ |

## 8. Debugging

**Scenario:** "server ใหม่ ping อินเทอร์เน็ตไม่ได้"

ถามทีละชั้น จากใกล้ไปไกล:

| # | คำถาม | เครื่องมือ | ผลที่บอกอะไร |
|---|---|---|---|
| 1 | มี IP ไหม? ถูกวงไหม? | `ip -br addr` | ไม่มี / ได้ 169.254.x.x → DHCP พัง |
| 2 | มี default route ไหม? | `ip route` · `ip route get 1.1.1.1` | `Network is unreachable` → ไม่มี route |
| 3 | คุยกับ gateway ได้ไหม? | `ping <gateway>` · `ip neigh` | ARP FAILED → สาย/VLAN/security ของ network ชั้นล่าง |
| 4 | ผ่าน gateway ไปได้ถึงไหน? | `traceroute -n 1.1.1.1` | ขาดที่ hop ไหน → ปัญหาอยู่แถวนั้น |
| 5 | IP ได้ แต่ชื่อไม่ได้? | `ping 1.1.1.1` ได้ แต่ `ping google.com` ไม่ได้ | ปัญหา DNS (Lesson 10) ไม่ใช่ routing |

**หลัก: แยก "ไปไม่ถูก" (routing) ออกจาก "ไปถึงแต่โดนกั้น" (firewall, Lesson 8) ออกจาก "หาชื่อไม่เจอ" (DNS)**

## 9. Failure Scenarios

| เกิดอะไร | อาการ | ข้างใน |
|---|---|---|
| ไม่มี default route | ออกนอกวงไม่ได้ แต่ในวงได้ | ไม่มีทางไป |
| subnet ซ้อนกัน (เช่น VPN ใช้ 192.168.1.0/24 ซ้ำกับบ้าน) | บางเครื่องเข้าไม่ได้แบบงง ๆ | kernel เลือก route ผิดวง |
| IP ชนกันสองเครื่อง | เชื่อมต่อหลุด ๆ ติด ๆ | ARP ได้ MAC สลับไปมา |
| ออกแบบ VPC ด้วย /24 ตั้งแต่แรก | ต่อมาเพิ่ม server/subnet ไม่ได้ และเชื่อมกับ network อื่นที่ใช้วงซ้ำไม่ได้ | CIDR เปลี่ยนภายหลังยากมาก |
| DHCP ไม่ตอบ | ได้ 169.254.x.x | ไม่มี IP ที่ใช้ได้จริง |

## 10. Trade-offs

**Subnet ใหญ่ vs เล็ก**
- ใหญ่ (/16): ใส่เครื่องได้มาก ไม่ต้องวางแผนละเอียด — แต่เปลือง address, แบ่งกลุ่มเพื่อความปลอดภัยยาก, ชนกับ network อื่นง่าย
- เล็ก (/26): แยกกลุ่มชัด — แต่ address หมดเร็ว
- หลักใน cloud: VPC ใหญ่พอเผื่ออนาคต (`/16`) แบ่ง subnet ตามหน้าที่และ AZ (`/20`–`/24`) และ **ไม่ใช้ช่วงที่ซ้ำกับ network อื่นที่อาจต้องเชื่อมในอนาคต**

**IPv4 vs IPv6**
- IPv4: ทุกเครื่องมือรองรับ เข้าใจง่าย — แต่ address หมดโลกแล้ว ต้องพึ่ง NAT และ public IPv4 ใน cloud มีค่าใช้จ่าย
- IPv6 (128 bit): address พอให้ทุกอุปกรณ์มี public IP — แต่เครื่องมือ/ทีมยังไม่คุ้นเท่า
- คอร์สนี้ใช้ IPv4 เป็นหลัก แต่หลักการ (prefix, routing, longest match) เหมือนกัน

## 11. Production Considerations

- **Security:** subnet คือหน่วยพื้นฐานของการแยก network — Phase 7 จะวาง web ไว้ใน public subnet และ database ไว้ใน **private subnet ที่ไม่มี route จากอินเทอร์เน็ตเข้ามาเลย**
  นั่นคือการใช้ routing table เป็นกำแพงชั้นแรก
- **วางแผน CIDR ก่อนสร้าง** — การย้าย IP range ของ VPC ที่ใช้งานอยู่แทบเท่ากับสร้างใหม่
- **Cost:** traffic ภายใน subnet/AZ เดียวกันมักฟรี · ข้าม AZ หรือออกอินเทอร์เน็ตคิดเงิน — วิธีวาง subnet จึงมีผลกับบิล (Phase 19)
- AWS คิดเงินค่า public IPv4 ทุก address — เครื่องที่ไม่จำเป็นต้องมี public IP ไม่ควรมี

## 12. Quiz

1. **(เข้าใจ)** ทำไม MAC address ถึงเปลี่ยนทุกครั้งที่ packet ข้าม router แต่ IP ปลายทางไม่เปลี่ยน? ถ้าใช้แค่ MAC ส่งข้ามทั้งโลกจะมีปัญหาอะไร
2. **(ประยุกต์)** routing table มี `10.0.0.0/16 via A`, `10.0.5.0/24 via B`, `default via C` packet ไป `10.0.5.9`, `10.0.9.1`, `8.8.8.8` จะไปทางไหนตามลำดับ
3. **(debugging)** server `ping 10.0.2.15` (อีกเครื่องใน VPC) ได้ แต่ `ping 1.1.1.1` ได้ `Network is unreachable` ทันที ส่วนอีกเครื่อง `ping 1.1.1.1` แล้วค้างจน timeout สองเครื่องนี้น่าจะมีปัญหาต่างกันอย่างไร
4. **(architecture)** ออกแบบ CIDR สำหรับ VPC ของ Pixbin: ต้องมี subnet สำหรับ web (เข้าจากอินเทอร์เน็ตได้) และ database (ห้ามเข้าจากอินเทอร์เน็ต) อย่างละ 2 ชุด (เผื่อ 2 AZ) และเผื่อขยายในอนาคต ให้ตัวเลขทุก subnet
5. **(trade-off)** บริษัทอื่นที่จะ connect VPN เข้ามาใช้ `10.0.0.0/16` เหมือน VPC ของคุณ จะเกิดปัญหาอะไร และถ้าย้อนเวลาได้ คุณจะป้องกันอย่างไร

## 13. Challenge

1. เพิ่ม route เจาะจง `sudo ip route add 1.1.1.1/32 via 192.168.64.99` (IP ที่ไม่มีเครื่อง) แล้วทำนายผลของ `ping 1.1.1.1` และ `ping 1.0.0.1` ก่อนรัน อธิบายด้วย longest prefix match แล้วลบ route ทิ้ง
2. หา MTU ของ interface (`ip link`) แล้วใช้ `ping -M do -s <size> 1.1.1.1` หาขนาด packet ใหญ่สุดที่ส่งได้โดยไม่ต้องแบ่ง อธิบายว่า MTU ไม่ตรงกันระหว่างทางทำให้เกิดอาการอะไร (ใบ้: "เว็บเล็ก ๆ เปิดได้ แต่เว็บใหญ่ค้าง")
3. เขียนคำอธิบายสั้น ๆ (เหมือนอธิบายให้ dev ใหม่ในทีม) ว่า "ping 1.1.1.1 จาก VM ไปถึงปลายทางได้อย่างไร" ครอบคลุม route, ARP, gateway, TTL — ไม่เกิน 15 บรรทัด

## 14. สรุปสิ่งที่ต้องจำ

- IP = ที่อยู่ของ interface · CIDR `/n` = n bit แรกคือ network · address = 2^(32−n)
- private: `10/8`, `172.16/12`, `192.168/16` — เข้าจากอินเทอร์เน็ตตรง ๆ ไม่ได้
- ในวงเดียวกัน → ARP หา MAC แล้วส่งตรง · ต่างวง → ส่งให้ gateway
- routing ใช้ longest prefix match · `default` = `0.0.0.0/0`
- `ip route get <IP>` ถาม kernel ตรง ๆ · `traceroute -n` เดินตามเส้นทาง
- "Network is unreachable" = ไม่มีทาง · "Host Unreachable" = มีทางแต่หาเครื่องไม่เจอ · timeout = ไปแล้วเงียบ

## 15. เรียนต่อ

**Lesson 8 — NAT และ Firewall:** VM (private IP) ออกอินเทอร์เน็ตได้อย่างไร ทำไมข้างนอกเข้ามาไม่ได้ และจะเปิด/ปิดประตูอย่างมีสติได้อย่างไร
