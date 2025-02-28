FROM golang:1.23-alpine as builder

RUN apk update \
    && apk add --no-cache git ca-certificates make bash yarn nodejs

RUN go env -w GO111MODULE=on && \
    go env -w GOPROXY=https://goproxy.cn,direct

WORKDIR /app

RUN git clone https://github.com/gaggad/goscheduler.git \
    && cd goscheduler \
    && yarn config set ignore-engines true \
    && make install-vue \
    && make build-vue \
    && make statik \
    && CGO_ENABLED=0 make goscheduler

FROM alpine:3.12

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S -g app app

RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

WORKDIR /app

COPY --from=builder /app/goscheduler/bin/goscheduler .

RUN chown -R app:app ./

EXPOSE 5920

USER app

ENTRYPOINT ["/app/goscheduler", "web"]
