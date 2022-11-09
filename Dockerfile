FROM alpine AS builder
WORKDIR /data/
ADD . .
RUN apk add --no-cache go && \
    go build -o frigate_sync .
#    go build -ldflags "-w -s" -o frigate_sync .

FROM alpine
WORKDIR /app/
VOLUME /data/
VOLUME /app/main.ini
#RUN apk add --no-cache ffmpeg
COPY --from=builder /data/frigate_sync /app/frigate_sync
ENTRYPOINT /app/frigate_sync