all:
	GOARCH=arm64 go build -ldflags "-w -s" -o frigate_events_worker .
	scp frigate_events_worker jp:/data/scripts/
	rm frigate_events_worker