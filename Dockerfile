FROM alpine

WORKDIR /app/
VOLUME /data/
VOLUME /app/main.ini

RUN apk add --no-cache ffmpeg

ADD frigate_sync /app/frigate_sync

RUN chmod +x /app/frigate_sync

ENTRYPOINT ['/app/frigate_sync']