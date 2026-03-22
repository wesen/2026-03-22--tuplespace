FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY 2026-03-22--tuplespace /src/tuplespace
COPY corporate-headquarters/glazed /src/glazed

WORKDIR /src/tuplespace

RUN go mod edit -replace github.com/go-go-golems/glazed=/src/glazed
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/tuplespaced ./cmd/tuplespaced
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/tuplespacectl ./cmd/tuplespacectl

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/tuplespaced /usr/local/bin/tuplespaced
COPY --from=builder /out/tuplespacectl /usr/local/bin/tuplespacectl
COPY --from=builder /src/tuplespace/migrations /app/migrations

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/tuplespaced"]
