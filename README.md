# cloud-infra-learn

บันทึกการเรียน Cloud / Infrastructure ตั้งแต่ 0 จนดูแลระบบ Production ได้เอง
เรียนจาก "ปัญหา" ไม่ใช่จาก "ชื่อ technology" และทุกบทต้องลงมือทำ + ทำให้มันพัง + debug เอง

## Project หลัก: Pixbin

เว็บแชร์รูปภาพ: สมัครสมาชิก → login → upload รูป → ระบบทำ thumbnail เบื้องหลัง → แสดง feed

เลือกโจทย์นี้เพราะมันบังคับให้เจอทุกปัญหาของระบบจริง:
auth (session/secret), database (ข้อมูล user/รูป), file storage (ไฟล์ใหญ่ ไม่ควรอยู่ใน DB),
background job (ย่อรูปช้า ไม่ควรให้ user รอ), cache (feed ถูกอ่านบ่อยกว่าเขียน), traffic ขึ้นลง

| Stage | สิ่งที่เพิ่ม | เรียนใน Phase |
|---|---|---|
| 1 | Simple web app รันบนเครื่องตัวเอง | 0–3 |
| 2 | Database | 3, 12 (พื้นฐาน) |
| 3 | Docker | 4–5 |
| 4 | Deploy ขึ้น Cloud (VM) | 6–7 |
| 5 | Domain + HTTPS | 3, 7 |
| 6–7 | Load Balancer + หลาย instance | 7, 14 |
| 8 | CI/CD | 8–9 |
| 9 | Monitoring + Logging | 10 |
| 10 | Auto Scaling | 14 |
| 11 | Caching | 14 |
| 12–13 | Queue + Background Worker | 15 |
| 14 | High Availability | 17 |
| 15 | Disaster Recovery | 18 |
| 16 | Security Hardening | 11 (+ ทุก stage) |
| 17 | Cost Optimization | 19 (+ ทุก stage) |
| 18 | Production Incident Simulation | 21 |

## Roadmap

| Phase | หัวข้อ | ทำไมอยู่ตรงนี้ |
|---|---|---|
| 0 | Computer & Internet Fundamentals | ทุกอย่างข้างบนคือ process + network; ถ้าไม่เข้าใจ 2 อย่างนี้ cloud คือเวทมนตร์ |
| 1 | Linux | server ใน cloud เกือบทั้งหมดคือ Linux; container คือ feature ของ Linux kernel |
| 2 | Networking | 70% ของปัญหา "เข้าไม่ได้" คือ network: IP, port, route, NAT, firewall |
| 3 | Web / HTTP / DNS / TLS | ประกอบ Linux + Network เป็นเส้นทางเต็มจาก browser → app |
| 4 | Virtualization | ต้องรู้ว่า "เครื่อง" ใน cloud คืออะไรจริง ๆ ก่อนเช่ามัน |
| 5 | Containers / Docker | ต่อยอดจาก process + Linux + VM เพื่อเข้าใจว่า container ต่างจาก VM ตรงไหน |
| 6 | Cloud Fundamentals | region/AZ, shared responsibility, pricing model — กติกาของเกม |
| 7 | AWS Core Services | ลงมือกับของจริง: EC2, VPC, S3, RDS, ALB, IAM |
| 8 | Infrastructure as Code | พอสร้างด้วยมือจนเจ็บแล้ว จะเห็นว่าทำไมต้องเขียนเป็น code |
| 9 | CI/CD | deploy ด้วยมือซ้ำ ๆ จะพัง → ทำให้อัตโนมัติ |
| 10 | Observability | ก่อนจะ scale ต้องมองเห็นระบบก่อน ไม่งั้น scale ไปก็ไม่รู้ว่าอะไรพัง |
| 11 | Security (เจาะลึก) | แทรกทุกบทอยู่แล้ว บทนี้รวบเป็นระบบ: IAM, secrets, network, supply chain |
| 12 | Databases & Storage | connection, index, replication, backup — จุดที่ระบบส่วนใหญ่ตันก่อน |
| 13 | Load Balancing / Scaling / Caching | *ย้ายขึ้นมาก่อน Distributed Systems* — ต้องมีหลายเครื่องจริงก่อน ปัญหา distributed ถึงจะเห็นเป็นรูปธรรม |
| 14 | Queues / Event-driven | งานที่ไม่ต้องตอบทันที แยกออกไปทำเบื้องหลัง |
| 15 | Distributed Systems | *ย้ายมาหลัง LB/Queue* — ตอนนี้จะเจอ consistency, retry, idempotency กับมือตัวเองแล้ว |
| 16 | Kubernetes | มาหลังสุดของ compute เพราะมันคือคำตอบรวมของปัญหา 5, 13, 14, 15 |
| 17 | High Availability | ออกแบบให้ส่วนหนึ่งพังแล้วระบบยังอยู่ |
| 18 | Disaster Recovery | ถ้าทั้ง region/ข้อมูลหาย จะกลับมาได้ไหม ภายในกี่นาที เสียข้อมูลกี่นาที |
| 19 | Cost Optimization | แทรกทุกบทอยู่แล้ว บทนี้ดูทั้งบิล |
| 20 | Production Architecture | รวมทุกอย่างเป็นระบบเดียว |
| 21 | Incident Response | ฝึกรับมือตอนระบบล่มจริง |
| 22 | Capstone | ออกแบบ + สร้างระบบ 1 ล้าน user เอง |

ลำดับนี้ไม่ตายตัว ถ้าเจอรูในพื้นฐานจะย้อนกลับไปอุดก่อน

## Milestones

- **M1 (จบ Phase 0–3):** อธิบายได้ทุกขั้นตั้งแต่พิมพ์ URL จนถึง app ตอบกลับ, รัน app บน Linux server เองได้, แยกได้ว่า "เข้าไม่ได้" เพราะ DNS / network / TLS / app
- **M2 (จบ Phase 4–5):** containerize app, อธิบายได้ว่า container คือ process ที่ถูกแยกด้วยอะไร, debug container ที่ crash loop ได้
- **M3 (จบ Phase 6–8):** deploy Pixbin บน AWS ใน VPC ที่ออกแบบเอง ด้วย Terraform, DB อยู่ใน private subnet
- **M4 (จบ Phase 9–11):** push code แล้ว deploy เอง, มี dashboard/alert, secrets ไม่อยู่ใน code
- **M5 (จบ Phase 12–15):** หลาย instance + cache + queue + worker และอธิบาย failure mode ของแต่ละส่วนได้
- **M6 (จบ Phase 16–19):** รันบน Kubernetes, ออกแบบ HA/DR พร้อมตัวเลข RTO/RPO, ประเมิน cost ได้
- **M7 (จบ Phase 20–22):** รับโจทย์ระบบ 1 ล้าน user แล้วออกแบบ–สร้าง–ดูแล–อธิบายเหตุผลได้เอง

## บทเรียน

- [Lesson 1 — โปรแกรมรันอยู่ แต่ทำไมคนอื่นเข้าไม่ได้? (Process, Port, Socket)](lessons/01-process-port-socket/README.md)

## สมุดเรียน (Infra Notebook)

หน้าเว็บรวมทุกบท มี lab, self-check และ quiz ที่ส่งให้ Claude ตรวจได้:
https://claude.ai/artifact/DiYo3dLeSk8N8ia6GP1Z3k
(ซอร์สของหน้าอยู่ที่ `notebook/index.html`)
