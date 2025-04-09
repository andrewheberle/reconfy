FROM golang:1.24@sha256:fb224f950b0dfb889f8f1122bf3dfc21976735ccf983b73a1e6e014215a272f5 AS builder

COPY . /build

RUN cd /build && \
    go build ./cmd/reconfy

FROM gcr.io/distroless/base-debian12:nonroot@sha256:fa5f94fa433728f8df3f63363ffc8dec4adcfb57e4d8c18b44bceccfea095ebc

COPY --from=builder /build/reconfy /app/reconfy

ENTRYPOINT [ "/app/reconfy" ]
