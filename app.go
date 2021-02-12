package main

import (
	"bufio"
	"fmt"
	"github.com/BRUHItsABunny/bunnlog"
	bunny_innertube_api "github.com/BRUHItsABunny/bunny-innertube-api"
	gokhttp "github.com/BRUHItsABunny/gOkHttp"
	"github.com/BRUHItsABunny/yt-downloader/utils"
	"github.com/dustin/go-humanize"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type App struct {
	Args           *utils.AppArgs
	YTClient       *bunny_innertube_api.YTClient
	DownloadClient *gokhttp.HttpClient
	BLog           *bunnlog.BunnyLog
	UIChannel      chan *gokhttp.TrackerMessage
	Wg             *sync.WaitGroup
}

func (app *App) Run() error {
	var (
		output []byte
		err    error
	)
	// Verify ffmpeg exists
	cmd := exec.Command(app.Args.GetFFmpegPath(), "-version")
	output, err = cmd.Output()
	if err == nil {
		app.BLog.Debugln("FFmpeg -version output:\n\n", string(output))
	}

	// TODO: Run UI (live UI wanted, not hard just annoying)
	app.UIChannel = make(chan *gokhttp.TrackerMessage, 0)
	app.Wg = &sync.WaitGroup{}

	// Triggers
	if app.Args.DoVideo() {
		err = app.DownloadVideo()
	} else if app.Args.DoList() {
		err = app.DownloadList()
	} else if app.Args.DoPlaylist() {
		err = app.DownloadPlaylist()
	} else if app.Args.DoChannel() {
		err = app.DownloadChannel()
	} else {
		// Nothing to do, TODO: maybe add a wipe cache function too execute here?
	}

	app.Wg.Wait()

	return err
}

func (app *App) DownloadVideo() error {
	var (
		videoID string
		err     error
	)
	app.BLog.Info("Trigger: Download a video")
	videoURL := utils.GetVideoURL(app.Args.GetVideo())
	videoID, err = app.YTClient.GetVideoID(videoURL, "com.whatsapp")
	if err == nil {
		err = app.processDownloadVideo(videoID)
	}
	return err
}

func (app *App) DownloadList() error {
	var (
		file    *os.File
		videoID string
		err     error
	)
	app.BLog.Info("Trigger: Download videos in list file")
	if utils.FileExists(app.Args.GetList()) {
		file, err = os.Open(app.Args.GetList())
		if err == nil {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				if err == nil {
					videoURL := utils.GetVideoURL(scanner.Text())
					if len(videoURL) > 0 {
						// TODO: in lib, result is NIL if url is invalid
						videoID, err = app.YTClient.GetVideoID(videoURL, "com.whatsapp")
						if err == nil {
							err = app.processDownloadVideo(videoID)
						}
					}
				} else {
					// TODO: Break on error?, maybe not always?
					break
				}
			}
			if err == nil {
				err = scanner.Err()
			}
		}
	}
	return err
}

func (app *App) DownloadPlaylist() error { // TODO: untested
	var (
		playlistData *bunny_innertube_api.BunnyBrowsePlaylistResponse
		listID       string
		continuation string
		err          error
	)
	app.BLog.Info("Trigger: Download videos in playlist")
	listURL := utils.GetPlaylistURL(app.Args.GetPlaylist())
	listID, err = app.YTClient.GetPlaylistID(listURL, "com.whatsapp")
	counter := 0
	offsetCounter := 0
	resetCounter := 0
	for {
		app.BLog.Debug("We are inside the pagination loop, counter = ", counter)
		app.BLog.Debug("Continuation: ", continuation)
		if err == nil && counter <= app.Args.GetAmount() {
			playlistData, err = app.YTClient.GetPlaylistContent(listID, continuation)
			if err == nil {
				app.BLog.Debug("Page has ", len(playlistData.Results), " results")
				continuation = playlistData.NextContinuation
				for _, video := range playlistData.Results {
					if err == nil {
						if offsetCounter >= app.Args.GetOffset() {
							app.BLog.Info("Current tab (", counter%30, "/", len(playlistData.Results), " & Current Total(", counter, "/", app.Args.GetAmount(), ")")
							err = app.processDownloadVideo(video.VideoId)
							counter++
							resetCounter++
						} else {
							offsetCounter++
						}
					} else {
						app.BLog.Warn("Breaking because of err\n", err)
						break
					}
					if counter > app.Args.GetAmount() {
						app.BLog.Info("Breaking because counter reached amount as limit")
						break
					}
					if resetCounter == 10 {
						// Reset the device authentication for stability
						app.BLog.Info("Resetting authentication")
						_ = app.YTClient.RegisterDevice(nil)
						app.BLog.Debug("New visitor ID: ", app.YTClient.Device.VisitorID)
						resetCounter = 0
					}
				}
			}
		} else {
			if err == nil {
				app.BLog.Info("Breaking because counter reached amount as limit")
			} else {
				app.BLog.Warn("Breaking because of err\n", err)
			}
			break
		}
	}

	return err
}

func (app *App) DownloadChannel() error {
	var (
		tabContent              *bunny_innertube_api.BunnyBrowseChannelResponse
		channelTabs             map[string]string
		channelID, continuation string
		err                     error
	)
	app.BLog.Info("Trigger: Download videos from channel uploads")
	// Transform /channel/ID or user/username to channelID
	channelURL := app.Args.GetChannel()
	channelID, err = app.YTClient.GetChannelID(channelURL, "com.whatsapp")
	counter := 0
	offsetCounter := 0
	resetCounter := 0
	if err == nil {
		channelTabs, err = app.YTClient.GetChannelTabs(channelID)
		if err == nil {
			continuation = channelTabs["videos"]
			for {
				app.BLog.Debug("We are inside the pagination loop, counter = ", counter)
				app.BLog.Debug("Continuation: ", continuation)
				if err == nil && counter <= app.Args.GetAmount() {
					tabContent, err = app.YTClient.GetChannelTabContent(channelID, continuation)
					if err == nil {
						continuation = tabContent.NextContinuation
						app.BLog.Debug("Tab has ", len(tabContent.Results), " results")
						for _, video := range tabContent.Results {
							if err == nil {
								if offsetCounter >= app.Args.GetOffset() {
									app.BLog.Info("Current tab (", counter%30, "/", len(tabContent.Results), " & Current Total(", counter, "/", app.Args.GetAmount(), ")")
									err = app.processDownloadVideo(video.CompactVideoData.OnTap.InnerTubeCommand.WatchEndpoint.VideoId)
									counter++
									resetCounter++
								} else {
									offsetCounter++
								}
							} else {
								app.BLog.Warn("Breaking because of err\n", err)
								break
							}
							if counter > app.Args.GetAmount() {
								app.BLog.Info("Breaking because counter reached amount as limit")
								break
							}
							if resetCounter == 10 {
								// Reset the device authentication for stability
								app.BLog.Info("Resetting authentication")
								_ = app.YTClient.RegisterDevice(nil)
								app.BLog.Debug("New visitor ID: ", app.YTClient.Device.VisitorID)
								resetCounter = 0
							}
						}
					}
				} else {
					if err == nil {
						app.BLog.Info("Breaking because counter reached amount as limit")
					} else {
						app.BLog.Warn("Breaking because of err\n", err)
					}
					break
				}
			}
		}
	}

	return err
}

func (app *App) processDownloadVideo(videoID string) error {
	// TODO: Check if video exists before downloading parts
	var (
		videoData   *bunny_innertube_api.BunnyVideoResponse
		probeResult *utils.FFprobeResult
		err         error
	)
	app.BLog.Debug("Enter: processingDownloadVideo(\"", videoID, "\")")
	videoData, err = app.YTClient.GetVideoData(videoID)

	if err == nil {
		if videoData.IsLiveStream {
			app.BLog.Info("URL is live stream")
			// TODO: Download stream, pipe ffmpeg to UI
			probeResult, err = utils.ProbeContent(app.Args.GetFFprobePath(), videoData.HLSManifest)
			if err == nil {
				decentResult := probeResult.Sort()
				audioStream := decentResult["audio"][len(decentResult["audio"])-1]
				videoStream := decentResult["video"][len(decentResult["video"])-1]

				// Meta collection and parse the file name
				resolution := videoStream.Height.String()
				framerate := strings.Split(videoStream.RFrameRate, "/")[0]
				fileName := fmt.Sprintf("%s - %s [%sp%s]", videoData.ChannelName, videoData.Title, resolution, framerate)
				fileName = utils.SanitizeFileName(fileName)
				app.BLog.Info("Going to download the stream into " + fileName + ".ts")

				// Setup the exec call
				// TODO: route ctrl+C to this
				err = utils.DownloadLiveContentWithMaps(app.Args.GetFFmpegPath(), videoData.HLSManifest, fileName+".ts", audioStream.Index.String(), videoStream.Index.String(), true)

				if err == nil {
					if *app.Args.MP4 {
						// Convert to mp4 (HEVC)
						err = utils.ConvertToHEVCMP4(app.Args.GetFFmpegPath(), fileName+".ts", fileName+".mp4", app.Args.GetFFmpegThreads(), true)
					} else {
						// Convert to webm (VP9)
						err = utils.ConvertToWEBM(app.Args.GetFFmpegPath(), fileName+".ts", fileName+".webm", app.Args.GetFFmpegThreads(), true)
					}
					_ = os.Remove(fileName + ".ts")
				}
			}
		} else {
			videoFiles, audioFiles := videoData.GetFormats(!*app.Args.MP4)

			var extension string
			if *app.Args.AudioOnly {
				extension = ".m4a"
				if !*app.Args.MP4 {
					extension = ".opus"
				}
			} else {
				extension = "_temp.mp4"
				if !*app.Args.MP4 {
					extension = ".webm"
				}
			}

			baseFilename := utils.SanitizeFileName(videoData.ChannelName + " - " + videoData.Title)
			videoFilename := baseFilename + "_video" + extension
			audioFilename := baseFilename + "_audio" + extension
			app.BLog.Debug("Checking for existence of: \"", baseFilename+extension, "\"")
			if !utils.FileExists(baseFilename + extension) {
				app.BLog.Info("Going to download \"", baseFilename+extension, "\" with ", app.Args.GetThreads(), " threads")
				app.Wg.Add(1)
				go app.RenderUI()

				if videoData.FormatsAreAdaptive {
					bestVideo := videoFiles[len(videoFiles)-1]
					bestAudio := audioFiles[len(audioFiles)-1]

					if !*app.Args.AudioOnly {
						go app.DownloadFile(app.Args.GetThreads(), bestAudio.Url, audioFilename)
						go app.DownloadFile(app.Args.GetThreads(), bestVideo.Url, videoFilename)
						app.Wg.Wait()
						app.BLog.Debug("Going to merge video and audio")
						err = utils.MergeVideoAndAudio(app.Args.GetFFmpegPath(), videoFilename, audioFilename, baseFilename+extension, true)
						if err == nil {
							_ = os.Remove(videoFilename)
							_ = os.Remove(audioFilename)
							// if *app.Args.MP4 {
							// 	// Save file size using HEVC
							// 	err = utils.ConvertToHEVCMP4(app.Args.GetFFmpegPath(), baseFilename + extension, baseFilename + ".mp4")
							// 	if err == nil {
							// 		_ = os.Remove(baseFilename + extension)
							// 	}
							//	}
						}
					} else {
						go app.DownloadFile(app.Args.GetThreads(), bestAudio.Url, baseFilename+extension)
						app.Wg.Wait()
					}
				} else {
					app.BLog.Debug("Video has no adaptive formats, falling back on regular MP4)")
					bestVideo := videoFiles[len(videoFiles)-1]
					go app.DownloadFile(app.Args.GetThreads(), bestVideo.Url, baseFilename+".mp4")
					app.Wg.Wait()

					if !*app.Args.MP4 {
						app.BLog.Debug("Going to convert to WEBM (VP9 + opus)")
						// err = utils.ConvertToWEBM(app.Args.GetFFmpegPath(), baseFilename + ".mp4", baseFilename + ".webm", app.Args.GetFFmpegThreads(),  true)
						// if err == nil {
						// 	_ = os.Remove(baseFilename + ".mp4")
						// }
					}

					if *app.Args.AudioOnly && err == nil {
						app.BLog.Debug("Going to extract audio")
						if !*app.Args.MP4 {
							err = utils.ExtractAudio(app.Args.GetFFmpegPath(), baseFilename+".webm", baseFilename+extension, true)
							if err == nil {
								_ = os.Remove(baseFilename + ".webm")
							}
						} else {
							err = utils.ExtractAudio(app.Args.GetFFmpegPath(), baseFilename+".mp4", baseFilename+extension, true)
							if err == nil {
								_ = os.Remove(baseFilename + ".mp4")
							}
						}
					}
				}
			}
		}
	}

	return err
}

func (app *App) DownloadFile(threadCount int, url, fileName string) error {
	var err error

	task := gokhttp.Task{
		Id:              rand.Intn(1000),
		Name:            fileName,
		Threads:         threadCount,
		URL:             url,
		ProgressChannel: app.UIChannel,
	}
	err = app.DownloadClient.DownloadFile(&task, nil, map[string]string{})

	return err
}

func (app *App) RenderUI() {
	t := time.NewTicker(time.Second)
	dbProgress := map[int]gokhttp.Download{}
	dbDone := map[int]gokhttp.Download{}
	shouldStop := false
	for {
		select {
		case msg := <-app.UIChannel:
			var download gokhttp.Download
			switch msg.Status {
			case gokhttp.StatusDone:
				if msg.IsFragment {
					download = dbProgress[msg.Id].Fragments[msg.Name]
					download.Status = msg.Status
					download.Finished = time.Now()
					dbProgress[msg.Id].Fragments[msg.Name] = download
				} else {
					// Main is done, if threaded merging is also done
					download = dbProgress[msg.Id]
					download.Status = msg.Status
					download.Finished = time.Now()
					dbDone[msg.Id] = download
					delete(dbProgress, msg.Id)
				}
			case gokhttp.StatusStart:
				// Build the DB!
				if msg.IsFragment {
					// Fragment Started
					download = gokhttp.Download{}
					download.Status = msg.Status
					download.FileName = msg.Name
					download.Started = time.Now()
					download.Size = int(msg.Expected)
					download.Progress = int(msg.Total)
					dbProgress[msg.Id].Fragments[msg.Name] = download
					// Make sure main object knows it is threaded
					download = dbProgress[msg.Id]
					download.Progress += int(msg.Total)
					download.Threaded = true
					dbProgress[msg.Id] = download
				} else {
					download = gokhttp.Download{}
					download.Status = msg.Status
					download.FileName = msg.Name
					download.Started = time.Now()
					download.Size = int(msg.Expected)
					download.Progress = int(msg.Total)
					download.Fragments = map[string]gokhttp.Download{}
					dbProgress[msg.Id] = download
				}
			case gokhttp.StatusProgress:
				if msg.IsFragment {
					// Progress fragment
					download = dbProgress[msg.Id].Fragments[msg.Name]
					download.Delta += msg.Delta
					download.Progress += msg.Delta
					download.Status = msg.Status
					dbProgress[msg.Id].Fragments[msg.Name] = download
					// Also update the main
					download = dbProgress[msg.Id]
					download.Delta += msg.Delta
					download.Progress += msg.Delta
					download.Status = msg.Status
					dbProgress[msg.Id] = download
				} else {
					// Progress main
					download = dbProgress[msg.Id]
					download.Delta += msg.Delta
					download.Progress += msg.Delta
					download.Status = msg.Status
					dbProgress[msg.Id] = download
				}
			case gokhttp.StatusError:
				// ERROR
				fmt.Println(msg.Err)
			case gokhttp.StatusMerging:
				// Merging started
				download = dbProgress[msg.Id]
				download.Status = msg.Status
				dbProgress[msg.Id] = download
			}
			break
		case <-t.C:
			var amount int
			if len(dbProgress) == 0 {
				shouldStop = true
			}
			for id, download := range dbProgress {
				// Filename: 100mb.bin
				fmt.Println("Filename: " + download.FileName)
				status := gokhttp.DownloadStatusString(download.Status)
				amount = int(math.Round(float64(download.Progress) / float64(download.Size) * float64(100) / float64(2)))
				if download.Status == gokhttp.StatusProgress {
					// Status: Downloading, Speed: 1.1mb/s, ETA: 00:01:52 [50%]
					eta := time.Duration((download.Size-download.Progress)/(download.Delta+1)) * time.Second
					percentage := int(math.Round(float64(download.Progress) / float64(download.Size) * float64(100)))
					speed := humanize.Bytes(uint64(download.Delta))
					download.Delta = 0
					fmt.Println(fmt.Sprintf("Status: %s, Speed: %s/s, ETA: %s [%d%%]", status, speed, eta.String(), percentage))
				} else {
					fmt.Println("Status: " + status)
				}
				barStr := ""
				if download.Threaded {
					// [XXXXXXXXXXX][XXXXXXXXXXX][XXXXXXXXXXX][XXXXXXXXXXXX]
					// Also wipe fragment deltas
					for name, thread := range download.Fragments {
						amount = int(math.Round(float64(thread.Progress) / float64(thread.Size) * float64(100) / float64(10)))
						barStr += "["
						for i := 0; i < 10; i++ {
							if i < amount {
								barStr += "X"
							} else {
								barStr += "="
							}
						}
						barStr += "]"
						thread.Delta = 0
						download.Fragments[name] = thread
					}
				} else {
					// [XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX]
					barStr += "["
					for i := 0; i < 50; i++ {
						if i < amount {
							barStr += "X"
						} else {
							barStr += "="
						}
					}
					barStr += "]"
				}
				fmt.Println(barStr)
				// Commit delta wipes
				dbProgress[id] = download
			}
			break
		}
		if shouldStop {
			break
		}
	}
	app.Wg.Done()
}
