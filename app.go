package yt_downloader

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/BRUHItsABunny/bunnlog"
	"github.com/BRUHItsABunny/bunny-innertube-api/api"
	biac "github.com/BRUHItsABunny/bunny-innertube-api/client"
	"github.com/BRUHItsABunny/bunny-innertube-api/innertube"
	gokhttp "github.com/BRUHItsABunny/gOkHttp"
	"github.com/BRUHItsABunny/yt-downloader/utils"
	"github.com/BRUHItsABunny/ytcaps2srt"
	"github.com/dustin/go-humanize"
	"google.golang.org/protobuf/proto"
	"html"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

type App struct {
	Args           *utils.AppArgs
	YTClient       *biac.YTClient
	DownloadClient *gokhttp.HttpClient
	BLog           *bunnlog.BunnyLog
	UIChannel      chan *gokhttp.TrackerMessage
	Wg             *sync.WaitGroup
}

func (app *App) Run(ctx context.Context) error {
	rand.Seed(time.Now().UnixNano())
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
		err = app.DownloadVideo(ctx)
	} else if app.Args.DoList() {
		err = app.DownloadList(ctx)
	} else if app.Args.DoPlaylist() {
		err = app.DownloadPlaylist(ctx)
	} else if app.Args.DoChannel() {
		err = app.DownloadChannel(ctx)
	} else {
		// Nothing to do, TODO: maybe add a wipe cache function too execute here?
	}

	app.Wg.Wait()

	return err
}

func (app *App) DownloadVideo(ctx context.Context) error {
	var (
		videoID *innertube.InnerTubeResolvedURL
		err     error
	)
	app.BLog.Info("Trigger: Download a video")
	videoURL := utils.GetVideoURL(app.Args.GetVideo())
	videoID, err = app.YTClient.ResolveURL(ctx, videoURL, "com.whatsapp", false)
	if err == nil {
		err = app.processDownloadVideo(ctx, videoID.Result)
	}
	return err
}

func (app *App) DownloadList(ctx context.Context) error {
	var (
		file    *os.File
		videoID *innertube.InnerTubeResolvedURL
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
						videoID, err = app.YTClient.ResolveURL(ctx, videoURL, "com.whatsapp", false)
						if err == nil {
							err = app.processDownloadVideo(ctx, videoID.Result)
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

func (app *App) RandomSleep(min, max int, dur time.Duration) {
	app.BLog.Debug("Generating random sleep min =", min, " max = ", max)
	sleepDuration := time.Duration(rand.Intn(max-min+1)+min) * dur
	app.BLog.Debug("RandomSleeping random duration: ", sleepDuration.String())
	time.Sleep(sleepDuration)
}

func (app *App) DownloadPlaylist(ctx context.Context) error { // TODO: untested
	var (
		playlistBrowse *innertube.InnerTubeBrowseResponse
		listID         *innertube.InnerTubeResolvedURL
		continuation   string
		err            error
	)
	app.BLog.Info("Trigger: Download videos in playlist")
	listURL := utils.GetPlaylistURL(app.Args.GetPlaylist())
	listID, err = app.YTClient.ResolveURL(ctx, listURL, "com.whatsapp", false)
	counter := 0
	offsetCounter := 0
	resetCounter := 0
	rsMin, rsMax := app.Args.GetRandomSleepMinMax()
	for {
		app.BLog.Debug("We are inside the pagination loop, counter = ", counter)
		app.BLog.Debug("Continuation: ", continuation)
		if err == nil && counter < app.Args.GetAmount() {
			playlistBrowse, err = app.YTClient.GetPlaylistContent(ctx, listID.Result, continuation, false)
			if err == nil {
				resetCounter++
				playlistData := playlistBrowse.Playlist
				app.BLog.Debug("Page has ", len(playlistData.Playlist.Videos), " results")
				continuation = playlistData.NextContinuation
				if err == nil {
					for _, video := range playlistData.Playlist.Videos {
						if offsetCounter >= app.Args.GetOffset() {
							app.BLog.Info("Current tab (", counter%30, "/", len(playlistData.Playlist.Videos), " & Current Total(", counter, "/", app.Args.GetAmount(), ")")
							err = app.processDownloadVideo(ctx, video.VideoID)
							counter++
							resetCounter++
							if err != nil {
								app.BLog.Warnf("Err occurred doing video: %s: %s", video.VideoID, err.Error())
								if !*app.Args.IgnoreErrorsForLoop {
									app.BLog.Warn("Breaking")
									break
								}
							}
							app.RandomSleep(rsMin, rsMax, time.Millisecond)
						} else {
							offsetCounter++
						}
						if (counter + 1) > app.Args.GetAmount() {
							app.BLog.Info("Breaking because counter reached amount as limit")
							break
						}
						if resetCounter > 0 && resetCounter%10 == 0 {
							// Reset the device authentication for stability
							app.BLog.Info("Resetting authentication")
							_ = app.YTClient.RegisterDevice(ctx, true)
							app.BLog.Debug("New visitor ID: ", app.YTClient.Device.VisitorID)
							resetCounter = 0
						}
					}
					err = nil
				} else {
					app.BLog.Warnf("Err occurred - breaking: %s", err.Error())
					break
				}

				// Rotate authentication faster when skipping through videos
				if resetCounter > 0 && resetCounter%5 == 0 {
					// Reset the device authentication for stability
					app.BLog.Info("Resetting authentication")
					_ = app.YTClient.RegisterDevice(ctx, true)
					app.BLog.Debug("New visitor ID: ", app.YTClient.Device.VisitorID)
					resetCounter = 0
				} else {
					app.RandomSleep(rsMin, rsMax, time.Millisecond)
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

func (app *App) DownloadChannel(ctx context.Context) error {
	var (
		tabContent    *innertube.InnerTubeBrowseResponse
		channelTabs   *innertube.InnerTubeBrowseResponse
		channelBrowse *innertube.InnerTubeResolvedURL
		continuation  string
		err           error
	)
	app.BLog.Info("Trigger: Download videos from channel uploads")
	// Transform /channel/ID or user/username to channelID
	channelURL := app.Args.GetChannel()
	channelBrowse, err = app.YTClient.ResolveURL(ctx, channelURL, "com.whatsapp", false)
	counter := 0
	offsetCounter := 0
	resetCounter := 0
	rsMin, rsMax := app.Args.GetRandomSleepMinMax()
	if err == nil {
		channelID := channelBrowse.Result
		channelTabs, err = app.YTClient.GetChannelTabs(ctx, channelID, false)
		if err == nil {
			continuation = channelTabs.Channel.ChannelTabs["videos"]
			for {
				app.BLog.Debug("We are inside the pagination loop, counter = ", counter)
				app.BLog.Debug("Continuation: ", continuation)
				if err == nil && counter < app.Args.GetAmount() {
					tabContent, err = app.YTClient.GetChannelTabContent(ctx, channelID, continuation, false)
					if err == nil {
						resetCounter++
						continuation = tabContent.Channel.NextContinuation
						app.BLog.Debug("Tab has ", len(tabContent.Channel.Videos), " results")
						for _, video := range tabContent.Channel.Videos {
							if err == nil {
								if offsetCounter >= app.Args.GetOffset() {
									app.BLog.Info("Current tab (", counter%30, "/", len(tabContent.Channel.Videos), " & Current Total(", counter, "/", app.Args.GetAmount(), ")")
									err = app.processDownloadVideo(ctx, video.VideoID)
									counter++
									resetCounter++
									app.RandomSleep(rsMin, rsMax, time.Millisecond)
								} else {
									offsetCounter++
								}
							} else {
								app.BLog.Warn("Breaking because of err\n", err)
								break
							}
							if (counter + 1) > app.Args.GetAmount() {
								app.BLog.Info("Breaking because counter reached amount as limit")
								break
							}
							if resetCounter > 0 && resetCounter%10 == 0 {
								// Reset the device authentication for stability
								app.BLog.Info("Resetting authentication")
								_ = app.YTClient.RegisterDevice(ctx, true)
								app.BLog.Debug("New visitor ID: ", app.YTClient.Device.VisitorID)
								resetCounter = 0
							}
						}

						// Rotate authentication faster when skipping through videos
						if resetCounter > 0 && resetCounter%5 == 0 {
							// Reset the device authentication for stability
							app.BLog.Info("Resetting authentication")
							_ = app.YTClient.RegisterDevice(ctx, true)
							app.BLog.Debug("New visitor ID: ", app.YTClient.Device.VisitorID)
							resetCounter = 0
						} else {
							app.RandomSleep(rsMin, rsMax, time.Millisecond)
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

func (app *App) processDownloadVideo(ctx context.Context, videoID string) error {
	// TODO: Check if video exists before downloading parts
	var (
		videoData   *innertube.InnerTubeVideo
		probeResult *utils.FFprobeResult
		err         error
	)
	app.BLog.Debug("Enter: processingDownloadVideo(\"", videoID, "\")")
	oldVersion := app.YTClient.Device.Context.Client.ClientVersion
	if *app.Args.BypassRestrictions {
		// Switching clients can cause more than just restriction bypass, only do it for getPlayer
		app.BLog.Info("Setting up restriction bypass context")
		// Credits: https://github.com/TeamNewPipe/NewPipe/issues/8102#issuecomment-1081085801
		app.YTClient.Device.Context.Client.ClientName = innertube.InnerTubeClient_TVHTML5_SIMPLY_EMBEDDED_PLAYER
		app.YTClient.Device.Context.Client.ClientVersion = "2.0"
		app.YTClient.Device.Context.Client.Platform = innertube.PlatformType_TV
	}
	videoData, err = app.YTClient.GetPlayer(ctx, videoID, *app.Args.BypassRestrictions)
	app.DownloadClient.Headers["user-agent"] = api.GetHeaders(app.YTClient.Device, false, false)["User-Agent"][0]
	app.YTClient.Device.Context.Client.ClientName = innertube.InnerTubeClient_ANDROID
	app.YTClient.Device.Context.Client.ClientVersion = oldVersion
	app.YTClient.Device.Context.Client.Platform = innertube.PlatformType_MOBILE
	videoMeta, errMeta := app.YTClient.Next(ctx, &api.NextArgs{
		VideoID:      videoID,
		Continuation: "",
	}, false)

	if err == nil {
		if videoData.IsLiveStream {
			app.BLog.Info("URL is live stream")
			// TODO: Download stream, pipe ffmpeg to UI
			probeResult, err = utils.ProbeContent(app.Args.GetFFprobePath(), videoData.ManifestHLS)
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
				err = utils.DownloadLiveContentWithMaps(app.Args.GetFFmpegPath(), videoData.ManifestHLS, fileName+".ts", audioStream.Index.String(), videoStream.Index.String(), true)

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
			videoFilesList, audioFilesList := videoData.GetFormatsV2(!*app.Args.MP4)
			videoFiles, audioFiles := videoFilesList.Formats, audioFilesList.Formats

			var extension, tmpExtension string
			if *app.Args.AudioOnly {
				extension = ".m4a"
				if !*app.Args.MP4 {
					extension = ".opus"
				}
			} else {
				extension = ".mp4"
				tmpExtension = "_temp.mp4"
				if !*app.Args.MP4 {
					extension = ".webm"
					tmpExtension = "_temp.webm"
				}
			}

			baseFilename := utils.SanitizeFileName(videoData.ChannelName + " - " + videoData.Title)
			if *app.Args.PrependVideoID {
				baseFilename = "[" + videoData.VideoID + "] " + baseFilename
			}
			videoFilename := baseFilename + "_video" + tmpExtension
			audioFilename := baseFilename + "_audio" + tmpExtension
			app.BLog.Debug("Checking for existence of: \"", baseFilename+extension, "\"")
			if !utils.FileExists(baseFilename + extension) {
				app.BLog.Info("Going to download \"", baseFilename+extension, "\" with ", app.Args.GetThreads(), " threads")
				app.Wg.Add(1)
				go app.RenderUI()

				if videoData.FormatsAreAdaptive {
					bestVideo := videoFiles[len(videoFiles)-1]
					bestAudio := audioFiles[len(audioFiles)-1]

					if !*app.Args.AudioOnly {
						app.BLog.Debug("Checking for existence of: \"", audioFilename, "\"")
						if !utils.FileExists(audioFilename) {
							app.BLog.Debug("Going to download \"", audioFilename, "\"")
							app.BLog.Debug(bestAudio.Url)
							go app.DownloadFile(app.Args.GetThreads(), bestAudio.Url, audioFilename)
						}

						app.BLog.Debug("Checking for existence of: \"", videoFilename, "\"")
						app.BLog.Debug(bestVideo.Url)
						if !utils.FileExists(videoFilename) {
							app.BLog.Debug("Going to download \"", videoFilename, "\"")
							go app.DownloadFile(app.Args.GetThreads(), bestVideo.Url, videoFilename)
						}

						var (
							subs           = make(map[string]string, 0)
							errCaption     error
							req            *http.Request
							res            *gokhttp.HttpResponse
							capBytes       []byte
							timedTxt       *ytcaps2srt.TimedText
							realTxt        []*ytcaps2srt.Text
							captionSrtBody string
						)

						if *app.Args.Subs || *app.Args.MergeSubs {
							for _, caption := range videoData.Captions {
								subFilename := baseFilename + "_" + strings.ReplaceAll(caption.Name, " ", "") + ".srt"
								req, _ = http.NewRequest("GET", caption.BaseURL, nil)
								res, errCaption = app.DownloadClient.Do(req)
								if errCaption == nil {
									capBytes, errCaption = res.Bytes()
									if errCaption == nil {
										timedTxt, errCaption = ytcaps2srt.ParseTimedText(capBytes)
										if errCaption == nil {
											realTxt, errCaption = timedTxt.Beautify()
											if errCaption == nil {
												captionSrtBody, errCaption = ytcaps2srt.ConvertToSRT(realTxt)
												if errCaption == nil {
													errCaption = ioutil.WriteFile(subFilename, []byte(html.UnescapeString(captionSrtBody)), 0777)
													if errCaption == nil {
														if *app.Args.MergeSubs {
															subs[caption.Name] = subFilename
														}
													} else {
														app.BLog.Error("Error wrtiting to file: ", errCaption)
													}
												} else {
													app.BLog.Error("Error : ", errCaption)
												}
											} else {
												app.BLog.Error("Error converting to SRT: ", errCaption)
											}
										} else {
											app.BLog.Error("Error parsing TimedText: ", errCaption)
										}
									} else {
										app.BLog.Error("Error reading TimedText bytes: ", errCaption)
									}
								} else {
									app.BLog.Error("Error : requesting TimedText URL", errCaption)
								}
							}
						}

						if errMeta == nil {
							app.BLog.Debug("Abl to write metadata...")
							var metaBytes []byte
							// Clone object, add and remove some data
							videoClone := proto.Clone(videoData).(*innertube.InnerTubeVideo)
							videoClone.Formats = nil
							videoClone.FormatsAreAdaptive = false
							videoClone.ExtraMetaData = videoMeta // Day accuracy of upload, is great for OLD videos
							switch app.Args.GetStoreMetadata() {
							case "j":
								fallthrough
							case "json":
								// Store meta as JSON
								app.BLog.Info("Going to write metadata as JSON")
								metaBytes, err = json.Marshal(videoClone)
								if err == nil {
									err = ioutil.WriteFile(baseFilename+"_metadata.json", metaBytes, 0777)
									if err != nil {
										app.BLog.Error("Error storing metadata: " + err.Error())
									}
								} else {
									app.BLog.Error("Error marshalling [JSON] metadata: " + err.Error())
								}
								break
							case "p":
								fallthrough
							case "pb":
								fallthrough
							case "proto":
								fallthrough
							case "protobuf":
								app.BLog.Info("Going to write metadata as PB")
								metaBytes, err = proto.Marshal(videoClone)
								if err == nil {
									err = ioutil.WriteFile(baseFilename+"_metadata.pb", metaBytes, 0777)
									if err != nil {
										app.BLog.Error("Error storing metadata: " + err.Error())
									}
								} else {
									app.BLog.Error("Error marshalling [PB] metadata: " + err.Error())
								}
								break
							default:
								app.BLog.Info("Not going to write metadata at all")
							}
						} else {
							app.BLog.Error("Error getting metadata: " + errMeta.Error())
						}
						app.BLog.Debug("Going to wait for threads to finish")
						app.Wg.Wait()
						app.BLog.Debug("Going to merge video and audio")
						err = utils.MergeVideoAndAudio(app.Args.GetFFmpegPath(), videoFilename, audioFilename, baseFilename+extension, subs, true)
						if err == nil {
							_ = os.Remove(videoFilename)
							_ = os.Remove(audioFilename)
							if *app.Args.MP4 {
								if *app.Args.HEVC {
									// Save file size using HEVC
									err = utils.ConvertToHEVCMP4(app.Args.GetFFmpegPath(), baseFilename+tmpExtension, baseFilename+extension, runtime.NumCPU()-1, true)
									if err == nil {
										_ = os.Remove(baseFilename + tmpExtension)
									}
								} else {
									_ = os.Rename(baseFilename+tmpExtension, baseFilename+extension)
								}
							}
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
				// Filename: 100mb._bin
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
