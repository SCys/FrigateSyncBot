import sys
import os
from datetime import datetime, timedelta

import ffmpeg
from telethon.sync import TelegramClient

from configparser import ConfigParser

config = ConfigParser()
with open("config.ini", "r") as f:
    config.read_file(f)

app_id = config["telegram"]["app_id"]
app_hash = config["telegram"]["app_hash"]
chat_id = int(config["telegram"]["chat_id"])

path_video = config["ffmpeg"]["path_video"]
path_thumbnail = config["ffmpeg"]["path_thumbnail"]
path_list = config["ffmpeg"]["path_list"]

path_prefix = config["frigate"]["path_prefix"]
cameras = [i.strip() for i in config["frigate"]["cameras"].split(",")]


if len(sys.argv) > 1:
    # parse sys first arg like rfc3339 to ts_addition
    ts_addition = datetime.strptime(sys.argv[1], "%Y-%m-%dT%H")
else:
    ts_addition = datetime.now() - timedelta(hours=1)

client = TelegramClient("frigate_camera", app_id, app_hash)
client.start()


# for dialog in client.iter_dialogs():
#     if dialog.name == 'Cameras':
#         print(dialog.id, dialog.name)

# client.send_file(chat_id, path_video, caption="")

channel = client.get_entity(chat_id)

for i in cameras:
    # prepare path
    path_camera = path_prefix + "/" + ts_addition.strftime("%Y-%m/%d/%H/") + i
    name = i.replace("-", "_")
    caption = "#Hourly #" + name + "\n" + ts_addition.strftime("%Y-%m-%d %H")

    # get files
    try:
        # get all files's name from path_camera
        files = [i for i in os.listdir(path_camera) if os.path.isfile(os.path.join(path_camera, i))]
        # sort by filename
        files.sort()
    except FileNotFoundError:
        continue

    # build file list
    with open(path_list, "w+") as f:
        f.write("\n".join([f"file {path_camera}/{i}" for i in files]))

    # concat videos
    ffmpeg.input(path_list, format="concat", safe=0).output(path_video, c="copy").global_args(
        "-hide_banner", "-loglevel", "error"
    ).run(overwrite_output=True)

    # get thumbnail at 1s
    ffmpeg.input(path_video, ss=1).output(path_thumbnail, vframes=1).global_args("-hide_banner", "-loglevel", "error").run(
        overwrite_output=True
    )

    print(f"uploading video {i} {ts_addition.isoformat()}")

    # send file with caption i + tp_addition(rfc3339)
    client.send_file(
        channel,
        path_video,
        caption=caption,
        video_note=True,
        supports_streaming=True,
        thumb=path_thumbnail,
        silent=True,
    )

    print(f"video {i} {ts_addition.isoformat()} is uploaded")

try:
    os.unlink(path_video)
    os.unlink(path_list)
    os.unlink(path_thumbnail)
except FileNotFoundError:
    pass
