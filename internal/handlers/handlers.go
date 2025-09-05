package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	gjson "github.com/goccy/go-json"
	"github.com/gocql/gocql"
	"github.com/the-yorkshire-allen/cdc-gateway/internal/config"
	"github.com/the-yorkshire-allen/cdc-gateway/internal/util"
	"github.com/the-yorkshire-allen/cdc-gateway/internal/validate"
)

type Handlers struct {
	sess *gocql.Session
	cfg  config.Config
}

func NewHandlers(sess *gocql.Session, cfg config.Config) *Handlers {
	return &Handlers{sess: sess, cfg: cfg}
}

func (h *Handlers) Healthz(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }

type sinkReg struct {
	Name, URL        string
	RetentionSeconds int
}

func (h *Handlers) RegisterSink(w http.ResponseWriter, r *http.Request) {
	var s sinkReg
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	if s.Name == "" || s.URL == "" {
		http.Error(w, "name and url required", 400)
		return
	}
	if s.RetentionSeconds <= 0 {
		s.RetentionSeconds = 86400
	}
	if err := h.sess.Query(`INSERT INTO `+h.cfg.CdcKeyspace+`.sinks (name,url,retention_seconds,status) VALUES (?,?,?,'active')`,
		s.Name, s.URL, s.RetentionSeconds).Exec(); err != nil {
		log.Printf("register sink failed: %v", err)
		http.Error(w, "persist failed", 500)
		return
	}
	w.WriteHeader(201)
}

func (h *Handlers) Ingest(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	_ = r.Body.Close()
	if len(b) == 0 {
		http.Error(w, "empty", 400)
		return
	}
	if err := validate.MaxSize(b, h.cfg.HardMax); err != nil {
		http.Error(w, err.Error(), 413)
		return
	}

	// Parse minimal fields leniently
	var tmp map[string]any
	_ = gjson.Unmarshal(b, &tmp)
	ks, _ := tmp["keyspace"].(string)
	tbl, _ := tmp["table"].(string)

	id := gocql.TimeUUID()
	ts := time.Now().UnixMicro()

	body := b
	encPtr := (*string)(nil)

	if h.cfg.Compression == "gzip" && len(body) >= h.cfg.InlineMax {
		if gz, err := util.Gzip(body); err == nil {
			body = gz
			v := "gzip"
			encPtr = &v
		}
	}

	parts := 0
	if len(body) > h.cfg.ChunkMax {
		chunks := util.Chunk(body, 1<<20) // 1MiB parts
		for i, c := range chunks {
			if err := h.sess.Query(`INSERT INTO `+h.cfg.CdcKeyspace+`.outbox_event_parts (id, seq, data) VALUES (?,?,?)`, id, i, c).Exec(); err != nil {
				log.Printf("ingest: parts insert err: %v", err)
				http.Error(w, "persist failed", 500)
				return
			}
		}
		parts = len(chunks)
		body = nil
	}

	sum := sha256.Sum256(b)
	if err := h.sess.Query(`INSERT INTO `+h.cfg.CdcKeyspace+`.outbox_events (id, ks, tbl, ts_us, payload, encoding, original_bytes, inline_bytes, parts, checksum)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		id, ks, tbl, ts, body, encPtr, len(b), len(body), parts, hex.EncodeToString(sum[:])).Exec(); err != nil {
		log.Printf("ingest: outbox_events insert failed: %v", err)
		http.Error(w, "persist failed", 500)
		return
	}

	// enqueue for all active sinks
	it := h.sess.Query(`SELECT name, retention_seconds FROM ` + h.cfg.CdcKeyspace + `.sinks`).Iter()
	var name string
	var ret int
	enq := 0
	now := time.Now()
	for it.Scan(&name, &ret) {
		if ret <= 0 {
			ret = 86400
		}
		exp := now.Add(time.Duration(ret) * time.Second)
		if err := h.sess.Query(`INSERT INTO `+h.cfg.CdcKeyspace+`.outbox_delivery (sink, next_attempt_at, id, state, tries, expiry) VALUES (?, toTimestamp(now()), ?, 'queued', 0, ?)`,
			name, id, exp).Exec(); err != nil {
			log.Printf("ingest: enqueue sink=%s err=%v", name, err)
			continue
		}
		enq++
	}
	_ = it.Close()

	log.Printf("ingest: event %s queued to %d sink(s)", id, enq)
	w.WriteHeader(202)
}

func (h *Handlers) DebugEnqueueLast(w http.ResponseWriter, r *http.Request) {
	var id gocql.UUID
	iter := h.sess.Query(`SELECT id FROM ` + h.cfg.CdcKeyspace + `.outbox_events`).PageSize(1).Iter()
	if !iter.Scan(&id) {
		_ = iter.Close()
		http.Error(w, "no events", 404)
		return
	}
	_ = iter.Close()
	enq := h.enqueueAll(id)
	_ = json.NewEncoder(w).Encode(map[string]any{"id": id.String(), "enqueued": enq})
}

func (h *Handlers) DebugEnqueueUnqueued(w http.ResponseWriter, r *http.Request) {
	it := h.sess.Query(`SELECT id FROM ` + h.cfg.CdcKeyspace + `.outbox_events`).Iter()
	var id gocql.UUID
	total := 0
	for it.Scan(&id) {
		total += h.enqueueAll(id)
	}
	_ = it.Close()
	_ = json.NewEncoder(w).Encode(map[string]any{"enqueued": total})
}

func (h *Handlers) enqueueAll(id gocql.UUID) int {
	it := h.sess.Query(`SELECT name, retention_seconds FROM ` + h.cfg.CdcKeyspace + `.sinks`).Iter()
	var name string
	var ret int
	enq := 0
	now := time.Now()
	for it.Scan(&name, &ret) {
		if ret <= 0 {
			ret = 86400
		}
		exp := now.Add(time.Duration(ret) * time.Second)
		if err := h.sess.Query(`INSERT INTO `+h.cfg.CdcKeyspace+`.outbox_delivery (sink, next_attempt_at, id, state, tries, expiry) VALUES (?, toTimestamp(now()), ?, 'queued', 0, ?)`,
			name, id, exp).Exec(); err == nil {
			enq++
		}
	}
	_ = it.Close()
	return enq
}
