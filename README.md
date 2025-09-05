
# CDC Gateway 

Cassandra CDC gateway

## Build
```
# build binary (default)
make

# override cmd path
make CMD_PATH=cmd/cdc-gateway

# tidy, format, vet, test
make tidy fmt vet test

# build image and push
make docker-build
REGISTRY=ghcr.io/the-yorkshire-allen TAG=v0.1.0 make docker-push

# bring up stack defined in deploy/docker-compose.yml
make up
make logs
make down
```

## Gateway env
```
CASS_HOSTS=cassandra:9042
CASS_DC=datacenter1
CDC_KEYSPACE=cdcgw
PAYLOAD_INLINE_MAX=262144
PAYLOAD_CHUNK_MAX=2097152
PAYLOAD_HARD_MAX=67108864
COMPRESSION=gzip
DELIVERY_WORKERS=4
DELIVERY_BATCH=128
DELIVERY_TICK=1s
```

## Endpoints
- POST `/sinks/register` → `{ "name": "...", "url": "http://example-sink:9208", "retention_seconds": 600 }`
- POST `/ingest` → JSON event body (gateway compresses/chunks and enqueues)
- GET `/debug/enqueue-last` / `/debug/enqueue-unqueued`
- GET `/healthz` (204), `/metrics` (Prometheus)
