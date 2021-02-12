# YTDownloader
A bunny-style YT downloader, prototype for BGLoader
Current stage: Alpha stage, tested for less than a month

## Usage
This program is freely available and so is most of its source code.

This program is used at own risk, due to copyright stuff (a download program is not illegal, but a person that downloads content to illegally distribute with said legal program is doing something illegal)

This program is aimed to be more stable than most other downloaders as it does not rely on HTML to work, but again using it is at your own risk.

Download the right binary from the bin folder in the repository:
* Windows 64 bit -> yt-downloader_windows_amd64.exe
* Windows 32 bit -> yt-downloader_windows_386.exe
* Linux 64 bit -> yt-downloader_linux_amd64
* Linux 32 bit -> yt-downloader_linux_386
* Android Termux -> yt-downloader_linux_arm

After downloading the appropriate executable you should rename it to `yt-downloader` and then install FFmpeg too ([windows guide](https://blog.gregzaal.com/how-to-install-ffmpeg-on-windows/)).

This downloader currently supports a few commands

Downloading a single video:

```yt-downloader --video="https://www.youtube.com/watch?v=8RwBOsDmXeQ"```

If you wanted audio only then just add ```audio_only=true``` for a total of:

```yt-downloader --video="https://www.youtube.com/watch?v=8RwBOsDmXeQ" --audio_only=true```

If you have faster internet than what YT allows to be streamed at, you can speed things up by adding the ```--threads=6``` option like this:

```yt-downloader --video="https://www.youtube.com/watch?v=8RwBOsDmXeQ" --threads=6```

All the commands below support these options, so you can just add it to the ones below to download only audios instead of videos.


Downloading from a list of URL's:

Imagine a file called ```urls.txt``` with the following content:

```
https://www.youtube.com/watch?v=8RwBOsDmXeQ
https://www.youtube.com/watch?v=8RwBOsDmXeQ
https://www.youtube.com/watch?v=8RwBOsDmXeQ
https://www.youtube.com/watch?v=8RwBOsDmXeQ
```
The command to download all the videos in this file would be:

```yt-downloader --list="./list.txt"```

To download 10 videos from a playlist (to download all just make the number equal to the size expected):

```yt-downloader --playlist="https://www.youtube.com/playlist?list=PL6OLQI9dX3d3gh-yK9apAgTIhjS8x5wqU" --amount=10```

To download 10 videos from a channel (to download all just make the number equal to the size expected):

```yt-downloader --channel="https://www.youtube.com/user/elithecomputerguy" --amount=10```


## Features
This program is capable of downloading videos by supplying the following information:
* A video URL
* A text file with multiple video URL's
* A playlist URL
* A channel URL (/channel/ID & /user/username & /c/channelname formats are all supported)

It automatically downloads the highest quality available, for video and for audio.

It seems to use about 10-20% CPU during spikes in activity, RAM memory is about 30mb (downloaded 500+ videos as a benchmark and it never got above 31mb), so perfect for low powered machines.

This program relies on FFmpeg (to merge audio and video into one file and to download livestreams) so please make sure to install FFmpeg and add it to your system PATH.
Alternatively you can just provide the program with the location of ffmpeg the executable by adding te option: 

```--ffmpeg_path=C:\Users\BUNNY\Downloads\ffmpeg\bin\ffmpeg.exe```

This was just an example, your location may be different

## Known bugs
There is a known bug in the way downloads are currently being tracked, small chance of a nil pointer error occurring

Program doesn't compile, this is due to a missing dependency because of a closed source library

Program hangs after stream is done, this is something between YouTube and FFmpeg and I am not sure how to work around this yet.

## TODO
This project is just a prototype for a future project, this is not going to be maintained daily, I can probably only dedicate 1-2 hours per week.
* Add a way to download private videos (as in, the private videos on your channel)
* Improve the terminal UI
* Fix the known bugs
* Add archiving support (can't always have the actual files in the working directory, so I need a better way to track archived videos)
* Actually test playlist download, currently untested

### Why bunny style?
Bunnies like long term stability and given the way this was coded it promotes said stability as it relies on an API that doesn't change much (if at all)

I refuse to open source the private library at this moment, if anything changes then I will update that here with a link to it. (don't count on it)
