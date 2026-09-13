# voxa-api — deployed on Pi/Dokploy. Stateless, no GPU, no local storage.
FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/voxa-api .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && \
    adduser -D -u 10001 voxa
USER voxa

COPY --from=builder /out/voxa-api /usr/local/bin/voxa-api

EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://localhost:8000/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/voxa-api"]
