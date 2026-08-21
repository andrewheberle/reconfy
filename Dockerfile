FROM golang:1.24@sha256:14fd8a55e59a560704e5fc44970b301d00d344e45d6b914dda228e09f359a088 AS builder

COPY . /build

RUN cd /build && \
    go build ./cmd/reconfy

FROM gcr.io/distroless/base-debian12:nonroot@sha256:7f0c72cd138b442ae0deeb69c08b1acf5525439ba251a49ad93c320a061567e5

COPY --from=builder /build/reconfy /app/reconfy

ENTRYPOINT [ "/app/reconfy" ]
