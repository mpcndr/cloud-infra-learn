// Pixbin — app ตัวหลักของคอร์ส จะโตขึ้นเรื่อย ๆ ทุก stage
// ตอนนี้ (Milestone 1) มี endpoint สำหรับทดลองให้ระบบทำงานหนัก/พัง เพื่อฝึก debug
//
// Config (environment variables):
//
//	HOST      ฟังที่ IP ไหน        (default 127.0.0.1)
//	PORT      ฟังที่ port ไหน      (default 8000)
//	DATA_DIR  เก็บไฟล์ upload ที่ไหน (default ./data)
package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var (
	leakMu sync.Mutex
	leak   [][]byte // หน่วยความจำที่ /work/mem จองไว้แล้ว "ไม่คืน" (จำลอง memory leak)
)

func main() {
	host := getenv("HOST", "127.0.0.1")
	port := getenv("PORT", "8000")
	dataDir := getenv("DATA_DIR", "./data")
	addr := net.JoinHostPort(host, port)
	hostname, _ := os.Hostname()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello from Pixbin\n")
		fmt.Fprintf(w, "server pid   : %d\n", os.Getpid())
		fmt.Fprintf(w, "server host  : %s\n", hostname)
		fmt.Fprintf(w, "listening on : %s\n", addr)
		fmt.Fprintf(w, "you came from: %s\n", r.RemoteAddr)
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			fmt.Fprintf(w, "forwarded for: %s\n", xff) // มีเมื่อมี reverse proxy อยู่ข้างหน้า
		}
	})

	// health check ที่ตรวจของจริง: เขียนไฟล์ลง DATA_DIR ได้ไหม
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		probe := filepath.Join(dataDir, ".healthz")
		if err := os.WriteFile(probe, []byte("ok"), 0o600); err != nil {
			http.Error(w, "unhealthy: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		fmt.Fprintln(w, "ok")
	})

	// งานคำนวณจำนวนคงที่: hash ซ้ำ n พันครั้ง (ยิ่งแย่ง CPU กันมาก ยิ่งเสร็จช้า)
	mux.HandleFunc("/work/cpu", func(w http.ResponseWriter, r *http.Request) {
		n := intParam(r, "n", 1000)
		start := time.Now()
		sum := sha256.Sum256([]byte("pixbin"))
		for i := 0; i < n*1000; i++ {
			sum = sha256.Sum256(sum[:])
		}
		fmt.Fprintf(w, "hashed %dk times in %s (%x)\n", n, time.Since(start).Round(time.Millisecond), sum[:4])
	})

	// รอเฉย ๆ ไม่ใช้ CPU (จำลองการรอ database / network)
	mux.HandleFunc("/work/sleep", func(w http.ResponseWriter, r *http.Request) {
		ms := intParam(r, "ms", 500)
		time.Sleep(time.Duration(ms) * time.Millisecond)
		fmt.Fprintf(w, "slept %dms\n", ms)
	})

	// จอง memory เพิ่ม mb เมกะไบต์ แล้วไม่คืน
	mux.HandleFunc("/work/mem", func(w http.ResponseWriter, r *http.Request) {
		mb := intParam(r, "mb", 100)
		b := make([]byte, mb<<20)
		for i := range b { // เขียนจริงทุก byte ให้ OS ต้องให้ RAM จริง
			b[i] = 1
		}
		leakMu.Lock()
		leak = append(leak, b)
		total := 0
		for _, x := range leak {
			total += len(x)
		}
		leakMu.Unlock()
		fmt.Fprintf(w, "holding %d MB\n", total>>20)
	})

	// POST body แล้วเก็บเป็นไฟล์ใน DATA_DIR
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "use POST", http.StatusMethodNotAllowed)
			return
		}
		name := randomName() + ".bin"
		f, err := os.Create(filepath.Join(dataDir, name))
		if err != nil {
			log.Printf("upload failed: %v", err)
			http.Error(w, "cannot save file", http.StatusInternalServerError)
			return
		}
		defer f.Close()
		n, err := io.Copy(f, r.Body)
		if err != nil {
			log.Printf("upload failed after %d bytes: %v", n, err)
			http.Error(w, "cannot save file", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "saved %s (%d bytes)\n", name, n)
	})

	mux.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
		entries, err := os.ReadDir(dataDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, e := range entries {
			if info, err := e.Info(); err == nil && !e.IsDir() && e.Name() != ".healthz" {
				fmt.Fprintf(w, "%s\t%d\n", e.Name(), info.Size())
			}
		}
	})

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatalf("cannot create DATA_DIR %s: %v", dataDir, err)
	}

	srv := &http.Server{Addr: addr, Handler: logRequests(mux)}

	// SIGTERM / SIGINT → หยุดรับงานใหม่ ทำงานที่ค้างให้จบ แล้วค่อยออก (graceful shutdown)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		log.Printf("got shutdown signal, finishing in-flight requests (max 10s)")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("forced shutdown: %v", err)
		}
	}()

	log.Printf("[pid %d] listening on %s, DATA_DIR=%s", os.Getpid(), addr, dataDir)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	// ListenAndServe คืนค่าทันทีที่เริ่ม Shutdown ต้องรอให้ request ที่ค้างอยู่ทำเสร็จก่อนค่อยออก
	<-shutdownDone
	log.Printf("bye")
}

func intParam(r *http.Request, name string, def int) int {
	if v, err := strconv.Atoi(r.URL.Query().Get(name)); err == nil && v > 0 {
		return v
	}
	return def
}

func randomName() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// log ทุก request ออก stdout: method path status duration
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s %d %s from=%s", r.Method, r.URL.RequestURI(), rec.status, time.Since(start).Round(time.Millisecond), r.RemoteAddr)
	})
}
