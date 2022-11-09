FROM alpine

WORKDIR /app/
VOLUME /data/

RUN apk add --no-cache ffmpeg

ADD frigate_sync frigate_sync

ENTRYPOINT ['/app/frigate_sync']