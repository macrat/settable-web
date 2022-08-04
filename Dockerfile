FROM golang:latest AS builder

COPY . /app
WORKDIR /app

RUN CGO_ENABLED=0 go build


FROM debian:stable-slim

RUN ln -f -s /usr/share/zoneinfo/Asia/Tokyo /etc/localtime
COPY --from=builder /app/settable-web /usr/local/bin/

USER 48:48
EXPOSE 8080

CMD ["/usr/local/bin/settable-web"]
