package main

import (
	"flag"
	"github.com/BRUHItsABunny/bunnlog"
	"github.com/BRUHItsABunny/bunny-innertube-api"
	gokhttp "github.com/BRUHItsABunny/gOkHttp"
	"github.com/BRUHItsABunny/gOkHttp/cookies"
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
	// yt-downloader --mp4=false --debug=true --threads=8 --channel=https://www.youtube.com/c/TheThingoftheName/ --amount=57
	// TODO: Actually check FFmpeg existence (not really necessary)
	// TODO: Version check for updates (postponed, bunny-github-api planned)
	appArgs := utils.AppArgs{
		Video:         flag.String("video", "", "URL containing videoId or videoId"),
		List:          flag.String("list", "", "path to text file with video URL's (THIS IS NOT PLAYLIST)"),
		Channel:       flag.String("channel", "", "URL to channel, can be /channel/ID or /user/username format"),
		Playlist:      flag.String("playlist", "", "URL to playlist or playlistId"),
		FFmpegPath:    flag.String("ffmpeg_path", "ffmpeg", "path to FFmpeg executable"),
		MP4:           flag.Bool("mp4", true, "true = mp4 videos (or m4a audio) will be downloaded, false = webm videos (or vorbis audio)"),
		AudioOnly:     flag.Bool("audio_only", false, "true = only audio will be downloaded, false = video + audio will be downloaded"),
		Debug:         flag.Bool("debug", false, "if true this will write a debug logfile"),
		Threads:       flag.Int("threads", 1, "Simultaneous threads for downloading (min 1, max 12)"),
		FFmpegThreads: flag.Int("ffmpeg_threads", 1, "Simultaneous threads for encoding FFmpeg (min 1, max "+strconv.Itoa(runtime.NumCPU())+")"),
		Amount:        flag.Int("amount", 1, "Amount of videos to download from channel or video (starting from latest)"),
		Offset:        flag.Int("offset", 0, "Offset of videos to ignore before downloading videos from channel or video (starting from latest)"),
	}
	flag.Parse()

	// YT client and HTTP client initialization
	client := bunny_innertube_api.GetYTClient(false)
	_ = client.RegisterDevice(nil)
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
	err = app.Run()
	if err != nil {

	}
}
