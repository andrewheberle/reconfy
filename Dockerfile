FROM golang:1.23@sha256:adfbe17b774398cb090ad257afd692f2b7e0e7aaa8ef0110a48f0a775e3964f4 AS builder

COPY . /build

RUN cd /build && \
    go build ./cmd/reconfy

FROM gcr.io/distroless/base-debian12:nonroot@sha256:97d15218016debb9b6700a8c1c26893d3291a469852ace8d8f7d15b2f156920f

COPY --from=builder /build/reconfy /app/reconfy

ENTRYPOINT [ "/app/reconfy" ]
