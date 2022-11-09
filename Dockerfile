FROM alpine AS builder
WORKDIR /data/
ADD . .
RUN apk add --no-cache go && \
    go build -ldflags "-w -s" -o frigate_sync .

FROM alpine
WORKDIR /app/
VOLUME /data/
VOLUME /app/main.ini
RUN apk add --no-cache ffmpeg
COPY builder:/data/frigate_sync /app/frigate_sync
RUN chmod +x /app/frigate_sync
CMD ['/app/frigate_sync']