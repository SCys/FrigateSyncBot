# Telegram Frigate Sync Bot

upload frigate videos/photos to telegram chat(group/private chat/channel)

## Depend

1. golang 1.7(up)
1. python 3.9(up)

## Quick Start

1. rename main.ini.sample to main.ini
1. setup main.ini
1. run make
1. add frigate_sync.py to crontab like "10 * * * * cd /data/scripts.d && python3 frigate_sync.py"
1. run frigate_sync in background