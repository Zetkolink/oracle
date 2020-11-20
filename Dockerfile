FROM golang:1.13-alpine3.10 AS builder

RUN apk add --no-cache build-base

WORKDIR /go/src/git.aqq.me/go/oracle
COPY . .

RUN go install

FROM alpine:3.10

RUN apk add --no-cache ca-certificates && \
  adduser -D oracle

COPY --from=builder /go/bin /usr/bin/
COPY etc/config.yml /etc/oracle.yml

ADD https://github.com/golang/go/raw/master/lib/time/zoneinfo.zip /home/oracle/zoneinfo.zip
ENV ZONEINFO /home/oracle/zoneinfo.zip
RUN chown -R oracle:oracle /home/oracle/zoneinfo.zip

USER oracle
EXPOSE 8080
ENV ORACLE_CONFPATH=/etc/oracle.yml

CMD [ "/usr/bin/oracle" ]
