FROM golang:1.24@sha256:762bb9cb6d35eb03537551112a3519cf0e6bfc66891530ce7dc7d6169ea1eeb3 AS builder

COPY . /build

RUN cd /build && \
    go build ./cmd/reconfy

FROM gcr.io/distroless/base-debian12:nonroot@sha256:23fa4a8575bc94e586b94fb9b1dbce8a6d219ed97f805369079eebab54c2cb23

COPY --from=builder /build/reconfy /app/reconfy

ENTRYPOINT [ "/app/reconfy" ]
