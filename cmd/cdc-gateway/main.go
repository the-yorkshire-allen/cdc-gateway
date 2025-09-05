
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/the-yorkshire-allen/cdc-gateway/internal/cass"
	"github.com/the-yorkshire-allen/cdc-gateway/internal/config"
	"github.com/the-yorkshire-allen/cdc-gateway/internal/delivery"
	handlerspkg "github.com/the-yorkshire-allen/cdc-gateway/internal/handlers"
)

func main() {
	cfg := config.LoadFromEnv()
	sess := cass.MustSession(cass.Config{
		Hosts:        cfg.CassHosts,
		Keyspace:     cfg.CdcKeyspace,
		Consistency:  cfg.Consistency,
		LocalDC:      cfg.LocalDC,
		Timeout:      cfg.Timeout,
	})
	defer sess.Close()

	if err := cass.EnsureSchema(sess, cfg.CdcKeyspace); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}

	mux := http.NewServeMux()
	h := handlerspkg.NewHandlers(sess, cfg)

	mux.HandleFunc("/healthz", h.Healthz)
	mux.HandleFunc("/sinks/register", h.RegisterSink)
	mux.HandleFunc("/ingest", h.Ingest)
	mux.HandleFunc("/debug/enqueue-last", h.DebugEnqueueLast)
	mux.HandleFunc("/debug/enqueue-unqueued", h.DebugEnqueueUnqueued)
	mux.Handle("/metrics", promhttp.Handler())

	// start workers
	delivery.Start(sess, cfg)

	go func() {
		addr := ":" + os.Getenv("METRICS_PORT")
		if addr == ":" { addr = ":9108" }
		log.Printf("metrics on %s", addr)
		if err := http.ListenAndServe(addr, mux); err != nil { log.Fatal(err) }
	}()

	addr := ":" + os.Getenv("PORT")
	if addr == ":" { addr = ":8088" }
	log.Printf("gateway on %s", addr)
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(server.ListenAndServe())
}
