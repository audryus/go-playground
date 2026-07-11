package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/trace"
	"sync/atomic"
	"time"
)

// --- Flight Recorder setup -------------------------------------------------

var fr *trace.FlightRecorder

func startFlightRecorder() {
	fr = trace.NewFlightRecorder(trace.FlightRecorderConfig{
		// MinAge ~ 2x a janela que você quer investigar.
		// Se seu SLO de latência é 1s, 2-3s já cobre o "antes e depois".
		MinAge:   3 * time.Second,
		MaxBytes: 5 << 20, // 5 MiB — limite de memória do buffer
	})
	if err := fr.Start(); err != nil {
		log.Fatalf("failed to start flight recorder: %v", err)
	}
}

// dumpSnapshot grava o conteúdo atual do buffer circular em um arquivo.
var snapshotSeq atomic.Int64

func dumpSnapshot(route string) {
	if !fr.Enabled() {
		return
	}
	id := snapshotSeq.Add(1)
	filename := fmt.Sprintf("snapshot-%d-%s.trace", id, time.Now().Format("150405"))

	f, err := os.Create(filename)
	if err != nil {
		log.Printf("failed to create snapshot file: %v", err)
		return
	}
	defer f.Close()

	n, err := fr.WriteTo(f)
	if err != nil {
		log.Printf("failed to write snapshot: %v", err)
		return
	}
	log.Printf("snapshot #%d written (%d bytes) — triggered by route %q", id, n, route)
}

// --- Middleware: dispara captura quando a requisição é lenta ---------------

const latencyThreshold = 200 * time.Millisecond

func tracingMiddleware(next http.Handler) http.Handler {
	// evita disparar N snapshots simultâneos sob carga
	var inFlight atomic.Bool

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		elapsed := time.Since(start)

		if elapsed > latencyThreshold && inFlight.CompareAndSwap(false, true) {
			go func() {
				defer inFlight.Store(false)
				dumpSnapshot(r.URL.Path)
			}()
		}
	})
}

// --- Endpoint administrativo: captura manual sob demanda --------------------

func adminSnapshotHandler(w http.ResponseWriter, r *http.Request) {
	dumpSnapshot("manual")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("snapshot requested\n"))
}

// --- Handlers de negócio ------------------------------------------------

func guessHandler(w http.ResponseWriter, r *http.Request) {
	// simula trabalho variável (ex.: contenção de lock, I/O externo)
	time.Sleep(time.Duration(150+time.Now().UnixNano()%300) * time.Millisecond)
	w.Write([]byte("ok\n"))
}

func main() {
	startFlightRecorder()
	defer fr.Stop()

	mux := http.NewServeMux()
	mux.HandleFunc("/guess", guessHandler)
	mux.HandleFunc("/debug/flight-recorder/snapshot", adminSnapshotHandler)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: tracingMiddleware(mux),
	}

	log.Println("listening on :8080")
	log.Fatal(srv.ListenAndServe())

	_ = context.Background()
}
