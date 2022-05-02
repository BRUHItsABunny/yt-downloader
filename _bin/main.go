package main

import (
	"context"
	"flag"
	"github.com/BRUHItsABunny/bunnlog"
	"github.com/BRUHItsABunny/bunny-innertube-api"
	gokhttp "github.com/BRUHItsABunny/gOkHttp"
	"github.com/BRUHItsABunny/gOkHttp/cookies"
	. "github.com/BRUHItsABunny/yt-downloader"
	"github.com/BRUHItsABunny/yt-downloader/utils"
	"golang.org/x/net/publicsuffix"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"
)

func main() {
	// Parsing app arguments
	// go run _bin/main.go -mp4 -debug -pid -threads 8 -channel "https://www.youtube.com/channel/UC0aanx5rpr7D1M7KCFYzrLQ" -meta "p" -subs -msubs -bres -ierr -amount 15
	// go run _bin/main.go -debug -mp4 -video "https://www.youtube.com/watch?v=m9ThyFE0V1Q" -threads 8 -bres
	// yt_downloader --mp4=false --debug=true --threads=8 --channel=https://www.youtube.com/c/TheThingoftheName/ --amount=57
	// TODO: Actually check FFmpeg existence (not really necessary)
	// TODO: Version check for updates (postponed, bunny-github-api planned)
	// TODO: Automatically download subtitles and convert to .srt, optionally automatically bundle in the mp4 as a subtitle stream?
	// TODO: BUG: If all fragments were downloaded but not yet merged into one file, downloader gets stuck!
	appArgs := utils.AppArgs{
		Video:               flag.String("video", "", "URL containing videoId or videoId"),
		List:                flag.String("list", "", "path to text file with video URL's (THIS IS NOT PLAYLIST)"),
		Channel:             flag.String("channel", "", "URL to channel, can be /channel/ID or /user/username format"),
		Playlist:            flag.String("playlist", "", "URL to playlist or playlistId"),
		FFmpegPath:          flag.String("ffmpeg_path", "ffmpeg", "path to FFmpeg executable"),
		MP4:                 flag.Bool("mp4", true, "true = mp4 videos (or m4a audio) will be downloaded, false = webm videos (or vorbis audio)"),
		HEVC:                flag.Bool("hevc", false, "Compress MP4 with HEVC"),
		PrependVideoID:      flag.Bool("pid", false, "Prepends [yyyy-mm-dd HH:MM:SS] to filenames"),
		Subs:                flag.Bool("subs", false, "This will store .srt files if possible containing YT captions"),
		MergeSubs:           flag.Bool("msubs", false, "This will merge the stored .srt files into the video file"),
		StoreMetadata:       flag.String("meta", "", "Adds a file that contains the metadata belonging to the video (upload date, description, etc) [json, proto]"),
		AudioOnly:           flag.Bool("audio_only", false, "true = only audio will be downloaded, false = video + audio will be downloaded"),
		Debug:               flag.Bool("debug", false, "if true this will write a debug logfile"),
		BypassRestrictions:  flag.Bool("bres", false, "This will try to bypass restrictions by emulating a different client"),
		IgnoreErrorsForLoop: flag.Bool("ierr", false, "This will prevent the main loop from breaking from an error of one video"),
		Threads:             flag.Int("threads", 1, "Simultaneous threads for downloading (min 1, max 12)"),
		FFmpegThreads:       flag.Int("ffmpeg_threads", 1, "Simultaneous threads for encoding FFmpeg (min 1, max "+strconv.Itoa(runtime.NumCPU())+")"),
		Amount:              flag.Int("amount", 1, "Amount of videos to download from channel or video (starting from latest)"),
		Offset:              flag.Int("offset", 0, "Offset of videos to ignore before downloading videos from channel or video (starting from latest)"),
		RSleepMin:           flag.Int("rsmin", 500, "Randomized sleep minimum (in MILLISECONDS)"),
		RSleepMax:           flag.Int("rsmax", 600, "Randomized sleep maximum (in MILLISECONDS)"),
	}
	flag.Parse()

	// YT client and HTTP client initialization
	httpTimeout := time.Second * time.Duration(3)
	gOkHttpOptions := gokhttp.HttpClientOptions{
		JarOptions: &cookies.JarOptions{PublicSuffixList: publicsuffix.List, NoPersist: true, Filename: ".cookies", EncryptionPassword: ""},
		Transport: &http.Transport{
			TLSHandshakeTimeout: httpTimeout,
			DisableCompression:  false,
			DisableKeepAlives:   false,
		},
		RefererOptions: &gokhttp.RefererOptions{Update: false, Use: false},
		Timeout:        &httpTimeout,
	}
	httpClient := gokhttp.GetHTTPClient(&gOkHttpOptions)
	// _ = httpClient.SetProxy("http://127.0.0.1:8888")
	client := bunny_innertube_api.GetYTClient(httpClient.Client, nil)
	_ = client.RegisterDevice(nil)

	// Setup logger
	logFile, err := os.Create("yt-downloader.log")
	if err != nil {
		panic(err)
	}
	var bLog bunnlog.BunnyLog
	if *appArgs.Debug {
		bLog = bunnlog.GetBunnLog(true, bunnlog.VerbosityDEBUG, log.Ldate|log.Ltime)
	} else {
		bLog = bunnlog.GetBunnLog(false, bunnlog.VerbosityWARNING, log.Ldate|log.Ltime)
	}
	bLog.SetOutputFile(logFile)

	app := App{
		Args:           &appArgs,
		YTClient:       client,
		DownloadClient: &httpClient,
		BLog:           &bLog,
	}

	// Run app
	app.BLog.Debug("Going to run app")
	err = app.Run(context.Background())
	if err != nil {

	}
}
