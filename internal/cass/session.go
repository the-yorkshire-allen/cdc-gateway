
package cass

import (
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
)

type Config struct {
	Hosts string
	Keyspace string
	Consistency string
	LocalDC string
	Timeout time.Duration
}

func MustSession(cfg Config) *gocql.Session {
	cluster := gocql.NewCluster(strings.Split(cfg.Hosts, ",")...)
	cluster.ProtoVersion = 4
	cluster.Timeout = cfg.Timeout
	cluster.Consistency = gocql.LocalQuorum
	if strings.ToUpper(cfg.Consistency) == "QUORUM" {
		cluster.Consistency = gocql.Quorum
	}
	if cfg.LocalDC != "" { cluster.PoolConfig.HostSelectionPolicy = gocql.DCAwareRoundRobinPolicy(cfg.LocalDC) }
	if cfg.Keyspace != "" { cluster.Keyspace = cfg.Keyspace }
	sess, err := cluster.CreateSession()
	if err != nil { panic(err) }
	return sess
}

func EnsureSchema(sess *gocql.Session, ks string) error {
	stmts := []string{
		fmt.Sprintf(`CREATE KEYSPACE IF NOT EXISTS %s WITH replication = {'class':'SimpleStrategy','replication_factor':1}`, ks),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.sinks (name text PRIMARY KEY, url text, retention_seconds int, status text)`, ks),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.outbox_events (id uuid PRIMARY KEY, ks text, tbl text, ts_us bigint, payload blob, encoding text, original_bytes int, inline_bytes int, parts int, external_uri text, checksum text)`, ks),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.outbox_event_parts (id uuid, seq int, data blob, PRIMARY KEY (id, seq))`, ks),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.outbox_delivery (sink text, next_attempt_at timestamp, id uuid, state text, tries int, expiry timestamp, PRIMARY KEY ((sink), next_attempt_at, id)) WITH CLUSTERING ORDER BY (next_attempt_at ASC, id ASC)`, ks),
	}
	for _, s := range stmts {
		if err := sess.Query(s).Exec(); err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "already exists") {
				return fmt.Errorf("schema step failed: %w", err)
			}
		}
	}
	return nil
}
