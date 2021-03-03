FROM alpine:latest AS builder

RUN apk update && apk add build-base go

ENV GOPATH=/build \
    CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64

ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /build/src

COPY . .

RUN go install -v

WORKDIR /dist

RUN cp /build/bin/casper casper

RUN ls -lah /lib

RUN ldd casper | tr -s '[:blank:]' '\n' | grep '^/' | \
    xargs -I % sh -c 'mkdir -p $(dirname ./%); cp % ./%;'
RUN mkdir -p lib && cp /lib/ld-musl-x86_64.so.1 lib/

FROM scratch

COPY --from=builder /dist /
COPY --from=builder /etc/ssl/cert.pem /etc/ssl/cert.pem

ENTRYPOINT ["/casper"]
