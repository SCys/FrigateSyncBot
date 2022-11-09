all: arm64 x86 py
	echo done

arm64:
	GOARCH=arm64 go build -ldflags "-w -s" -o frigate_sync_arm64 .

x86:
	go build -ldflags "-w -s" -o frigate_sync_x86 .

py:
	scp frigate_sync.py jp:/data/scripts/