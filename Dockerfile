FROM alpine AS builder
WORKDIR /data/
ADD . .
RUN apk add --no-cache go tzdata && \
    go build -o frigate_sync .

# runner
FROM alpine
WORKDIR /app/
VOLUME /data/
VOLUME /app/main.ini
COPY --from=builder /data/frigate_sync /app/frigate_sync
ENTRYPOINT /app/frigate_sync