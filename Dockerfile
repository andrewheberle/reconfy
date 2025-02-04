FROM golang:1.23@sha256:e213430692e5c31aba27473cdc84cfff2896d0c097e984bef67b6a44c75a8181 AS builder

COPY . /build

RUN cd /build && \
    go build ./cmd/reconfy

FROM gcr.io/distroless/base-debian12:nonroot@sha256:97d15218016debb9b6700a8c1c26893d3291a469852ace8d8f7d15b2f156920f

COPY --from=builder /build/reconfy /app/reconfy

ENTRYPOINT [ "/app/reconfy" ]
