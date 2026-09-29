// Pixbin v0 (Go) — ทำงานเหมือน app.py ทุกอย่าง ใช้แค่ standard library
// รัน: go run main.go   หรือ   HOST=0.0.0.0 PORT=8001 go run main.go
package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	host := getenv("HOST", "127.0.0.1")
	port := getenv("PORT", "8000")
	addr := net.JoinHostPort(host, port)
	hostname, _ := os.Hostname()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "hello from Pixbin\n")
		fmt.Fprintf(w, "server pid   : %d\n", os.Getpid())
		fmt.Fprintf(w, "server host  : %s\n", hostname)
		fmt.Fprintf(w, "listening on : %s\n", addr)
		fmt.Fprintf(w, "you came from: %s\n", r.RemoteAddr)
	})

	// net.Listen = socket() + bind() + listen() ในบรรทัดเดียว
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err) // เช่น "bind: address already in use"
	}
	log.Printf("[pid %d] listening on %s", os.Getpid(), addr)
	// http.Serve วน accept() รับ connection ทีละตัว แล้วแยก goroutine ไปตอบแต่ละตัว
	log.Fatal(http.Serve(ln, nil))
}
