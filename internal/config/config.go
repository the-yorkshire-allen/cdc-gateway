
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	CassHosts string
	LocalDC   string
	Consistency string
	Timeout   time.Duration
	CdcKeyspace string

	InlineMax int // bytes
	ChunkMax  int // bytes
	SpillMax  int // bytes
	HardMax   int // bytes
	Compression string // gzip|none

	Workers int
	Batch   int
	Tick    time.Duration
}

func getenv(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }
func geti(k string, d int) int { if v := os.Getenv(k); v != "" { if x,err := strconv.Atoi(v); err==nil { return x } }; return d }
func getdur(k string, d time.Duration) time.Duration { if v:=os.Getenv(k); v!="" { if x,err:=time.ParseDuration(v); err==nil { return x } }; return d }

func LoadFromEnv() Config {
	return Config{
		CassHosts: getenv("CASS_HOSTS", "cassandra:9042"),
		LocalDC: getenv("CASS_DC", "datacenter1"),
		Consistency: getenv("CASS_CONSISTENCY", "LOCAL_QUORUM"),
		Timeout: getdur("CASS_TIMEOUT", 10*time.Second),
		CdcKeyspace: getenv("CDC_KEYSPACE", "cdcgw"),
		InlineMax: geti("PAYLOAD_INLINE_MAX", 256*1024),
		ChunkMax:  geti("PAYLOAD_CHUNK_MAX", 2*1024*1024),
		SpillMax:  geti("PAYLOAD_SPILL_MAX", 16*1024*1024),
		HardMax:   geti("PAYLOAD_HARD_MAX", 64*1024*1024),
		Compression: getenv("COMPRESSION", "gzip"),
		Workers: geti("DELIVERY_WORKERS", 4),
		Batch:   geti("DELIVERY_BATCH", 128),
		Tick:    getdur("DELIVERY_TICK", time.Second),
	}
}
