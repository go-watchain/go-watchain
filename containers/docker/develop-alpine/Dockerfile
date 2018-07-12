FROM alpine:3.7

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 https://github.com/watchain/go-watchain && \
  (cd go-watereum && make gwat) && \
  cp go-watereum/build/bin/gwat /gwat && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /go-watereum && rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/gwat"]
