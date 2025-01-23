FROM golang:1.22@sha256:d5b17d684180648e16ea974bea677498945e8b619f7b26325958d8d99e97f9ea AS builder

COPY . /build

RUN cd /build && \
    go build ./cmd/reconfy

FROM gcr.io/distroless/base-debian12:nonroot@sha256:c3584d9160af7bbc6a0a6089dc954d0938bb7f755465bb4ef4265aad0221343e

COPY --from=builder /build/reconfy /app/reconfy

ENTRYPOINT [ "/app/reconfy" ]
