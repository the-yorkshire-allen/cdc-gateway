package delivery

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gocql/gocql"
	"github.com/the-yorkshire-allen/cdc-gateway/internal/config"
	"github.com/the-yorkshire-allen/cdc-gateway/internal/metrics"
	"github.com/the-yorkshire-allen/cdc-gateway/internal/util"
)

type row struct {
	sink  string
	t     time.Time
	id    gocql.UUID
	tries int
}

func Start(sess *gocql.Session, cfg config.Config) {
	for i := 0; i < cfg.Workers; i++ {
		go worker(sess, cfg, i)
	}
}

func worker(sess *gocql.Session, cfg config.Config, wid int) {
	client := &http.Client{Timeout: 10 * time.Second}
	ks := cfg.CdcKeyspace
	for {
		// list sinks
		itS := sess.Query(`SELECT name, url FROM ` + ks + `.sinks`).Iter()
		var name, url string
		sinks := map[string]string{}
		for itS.Scan(&name, &url) {
			sinks[name] = url
		}
		_ = itS.Close()

		now := time.Now()
		for s, u := range sinks {
			q := sess.Query(`SELECT sink, next_attempt_at, id, tries FROM `+ks+`.outbox_delivery WHERE sink=? AND next_attempt_at <= ? LIMIT ?`, s, now, cfg.Batch).PageSize(cfg.Batch)
			it := q.Iter()
			var r row
			for it.Scan(&r.sink, &r.t, &r.id, &r.tries) {
				body, err := loadBody(sess, ks, r.id)
				if err != nil {
					log.Printf("deliver[%d]: load %s err=%v", wid, r.id, err)
					retry(sess, ks, r)
					continue
				}
				req, _ := http.NewRequest("POST", u+"/events", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				resp, err := client.Do(req)
				if err == nil && resp.StatusCode < 500 {
					if resp != nil && resp.Body != nil {
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
					}
					_ = sess.Query(`DELETE FROM `+ks+`.outbox_delivery WHERE sink=? AND next_attempt_at=? AND id=?`, r.sink, r.t, r.id).Exec()
					metrics.DeliveriesSent.WithLabelValues(r.sink).Inc()
				} else {
					if resp != nil && resp.Body != nil {
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
					}
					retry(sess, ks, r)
					metrics.DeliveriesFail.WithLabelValues(r.sink).Inc()
				}
			}
			_ = it.Close()
		}
		time.Sleep(cfg.Tick)
	}
}

func retry(sess *gocql.Session, ks string, r row) {
	next := time.Now().Add(backoff(r.tries + 1))
	_ = sess.Query(`UPDATE `+ks+`.outbox_delivery SET tries=?, next_attempt_at=? WHERE sink=? AND next_attempt_at=? AND id=?`,
		r.tries+1, next, r.sink, r.t, r.id).Exec()
}

func backoff(n int) time.Duration {
	d := time.Second << n
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

func loadBody(sess *gocql.Session, ks string, id gocql.UUID) ([]byte, error) {
	var payload []byte
	var encoding *string
	var parts int
	var uri *string
	if err := sess.Query(`SELECT payload, encoding, parts, external_uri FROM `+ks+`.outbox_events WHERE id=?`, id).
		Scan(&payload, &encoding, &parts, &uri); err != nil {
		return nil, err
	}

	if uri != nil && *uri != "" {
		// TODO: stream from object store
		return nil, io.EOF
	}
	if parts > 0 {
		it := sess.Query(`SELECT data FROM `+ks+`.outbox_event_parts WHERE id=?`, id).Iter()
		var b bytes.Buffer
		var part []byte
		for it.Scan(&part) {
			b.Write(part)
		}
		_ = it.Close()
		payload = b.Bytes()
	}
	if encoding != nil && *encoding == "gzip" {
		return util.Gunzip(payload)
	}
	return payload, nil
}
