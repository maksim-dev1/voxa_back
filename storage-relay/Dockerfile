# voxa-storage-relay — runs on nexus. Needs write access to the
# recordings volume (mounted at /data/voxa/recordings, see compose).
FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/voxa-storage-relay .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && \
    adduser -D -u 10001 voxa
USER voxa

COPY --from=builder /out/voxa-storage-relay /usr/local/bin/voxa-storage-relay

EXPOSE 8100
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://localhost:8100/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/voxa-storage-relay"]
