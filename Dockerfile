FROM golang:1.24@sha256:14fd8a55e59a560704e5fc44970b301d00d344e45d6b914dda228e09f359a088 AS builder

COPY . /build

RUN cd /build && \
    go build ./cmd/reconfy

FROM gcr.io/distroless/base-debian12:nonroot@sha256:06c713c675e983c5aea030592b1d635954218d29c4db2f8ec66912da1b87e228

COPY --from=builder /build/reconfy /app/reconfy

ENTRYPOINT [ "/app/reconfy" ]
