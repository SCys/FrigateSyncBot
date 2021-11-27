all: go py
	echo done

go:
	GOARCH=arm64 go build -ldflags "-w -s" -o frigate_sync .
	scp frigate_sync jp:/data/scripts/
	rm frigate_sync

py:
	scp frigate_sync.py jp:/data/scripts/