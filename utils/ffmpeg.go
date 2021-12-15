package utils

import (
	"encoding/json"
	"github.com/shopspring/decimal"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

type FFprobeResult struct {
	Streams []FFStream `json:"streams"`
}

type FFStream struct {
	Index              decimal.Decimal            `json:"index"`
	CodecName          string                     `json:"codec_name"`
	CodecLongName      string                     `json:"codec_long_name"`
	Profile            string                     `json:"profile"`
	CodecType          string                     `json:"codec_type"`
	CodecTimeBase      string                     `json:"codec_time_base"`
	CodecTagString     string                     `json:"codec_tag_string"`
	CodecTag           string                     `json:"codec_tag"`
	SampleFmt          *string                    `json:"sample_fmt,omitempty"`
	SampleRate         *string                    `json:"sample_rate,omitempty"`
	Channels           *decimal.Decimal           `json:"channels,omitempty"`
	ChannelLayout      *string                    `json:"channel_layout,omitempty"`
	BitsPerSample      *decimal.Decimal           `json:"bits_per_sample,omitempty"`
	RFrameRate         string                     `json:"r_frame_rate"`
	AvgFrameRate       string                     `json:"avg_frame_rate"`
	TimeBase           string                     `json:"time_base"`
	StartPts           decimal.Decimal            `json:"start_pts"`
	StartTime          string                     `json:"start_time"`
	Disposition        map[string]decimal.Decimal `json:"disposition"`
	Tags               FFTags                     `json:"tags"`
	Width              *decimal.Decimal           `json:"width,omitempty"`
	Height             *decimal.Decimal           `json:"height,omitempty"`
	CodedWidth         *decimal.Decimal           `json:"coded_width,omitempty"`
	CodedHeight        *decimal.Decimal           `json:"coded_height,omitempty"`
	HasBFrames         *decimal.Decimal           `json:"has_b_frames,omitempty"`
	SampleAspectRatio  *string                    `json:"sample_aspect_ratio,omitempty"`
	DisplayAspectRatio *string                    `json:"display_aspect_ratio,omitempty"`
	PixFmt             *string                    `json:"pix_fmt,omitempty"`
	Level              *decimal.Decimal           `json:"level,omitempty"`
	ColorRange         *string                    `json:"color_range,omitempty"`
	ColorSpace         *string                    `json:"color_space,omitempty"`
	ColorTransfer      *string                    `json:"color_transfer,omitempty"`
	ColorPrimaries     *string                    `json:"color_primaries,omitempty"`
	ChromaLocation     *string                    `json:"chroma_location,omitempty"`
	Refs               *decimal.Decimal           `json:"refs,omitempty"`
	IsAVC              *string                    `json:"is_avc,omitempty"`
	NalLengthSize      *string                    `json:"nal_length_size,omitempty"`
	BitsPerRawSample   *string                    `json:"bits_per_raw_sample,omitempty"`
}

type FFTags struct {
	VariantBitrate string `json:"variant_bitrate"`
}

func (r *FFprobeResult) Sort() map[string][]FFStream {
	var (
		result    = make(map[string][]FFStream)
		listVideo = make([]FFStream, 0)
		listAudio = make([]FFStream, 0)
	)

	// Separate the streams
	for _, stream := range r.Streams {
		switch stream.CodecType {
		case "audio":
			listAudio = append(listAudio, stream)
			break
		case "video":
			listVideo = append(listVideo, stream)
			break
		}
	}

	// Sort
	sort.Slice(listAudio, func(i, j int) bool {
		iDecimal, _ := decimal.NewFromString(listAudio[i].Tags.VariantBitrate)
		jDecimal, _ := decimal.NewFromString(listAudio[j].Tags.VariantBitrate)
		return iDecimal.LessThan(jDecimal)
	})
	sort.Slice(listVideo, func(i, j int) bool {
		iDecimal, _ := decimal.NewFromString(listVideo[i].Tags.VariantBitrate)
		jDecimal, _ := decimal.NewFromString(listVideo[j].Tags.VariantBitrate)
		return iDecimal.LessThan(jDecimal)
	})

	// Prepare and return
	result["audio"] = listAudio
	result["video"] = listVideo

	return result
}

func ProbeContent(ffprobePath, location string) (*FFprobeResult, error) {
	// ffprobe -v quiet -print_format json -show_format -show_streams -i url/location
	result := new(FFprobeResult)
	out, err := exec.Command(ffprobePath, "-v", "quiet", "-print_format", "json", "-show_streams", "-i", location).Output()
	if err == nil {
		err = json.Unmarshal(out, result)
	}
	return result, err
}

// func MergeVideoAndAudio(ffmpegPath, videoName, audioName, fileName string, pipeTerminal bool) error {
// 	// ffmpeg -i videoplaybacknew.webm -i videoplaybacknew.weba -c:v copy -c:a copy output.webm
// 	// Just make sure to always download compatible encodings, like webm + opus and mp4 + m4a
// 	cmd := exec.Command(ffmpegPath, "-i", videoName, "-i", audioName, "-c:v", "copy", "-c:a", "copy", "-movflags", " faststart", fileName)
// 	if pipeTerminal {
// 		cmd.Stdout = os.Stdout
// 		cmd.Stderr = os.Stderr
// 	}
// 	done := make(chan error, 1)
// 	go func() {
// 		done <- cmd.Run()
// 	}()
// 	return <-done
// }

func MergeVideoAndAudio(ffmpegPath, videoName, audioName, fileName string, subs map[string]string, pipeTerminal bool) error {
	// ffmpeg -i videoplaybacknew.webm -i videoplaybacknew.weba -c:v copy -c:a copy output.webm
	// Just make sure to always download compatible encodings, like webm + opus and mp4 + m4a

	cmdStr := []string{"-i", videoName, "-i", audioName}
	cmdMetaData := []string{}
	if len(subs) > 0 {
		i := 0
		for key, val := range subs {
			cmdStr = append(cmdStr, "-i", val)
			cmdMetaData = append(cmdMetaData, "-metadata:s:s:"+strconv.Itoa(i), "language="+strings.ReplaceAll(key, " ", ""))
			i++
		}
		cmdStr = append(cmdStr, "-c:s", "mov_text")
	}
	cmdStr = append(cmdStr, "-c:v", "copy", "-c:a", "copy")
	cmdStr = append(cmdStr, cmdMetaData...)
	cmdStr = append(cmdStr, "-movflags", "faststart", fileName)

	cmd := exec.Command(ffmpegPath, cmdStr...)
	if pipeTerminal {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()
	return <-done
}

func ConvertToWEBM(ffmpegPath, downloadName, resultName string, threads int, pipeTerminal bool) error {
	// ffmpeg -i download.mp4 -c:v libvpx-vp9 -c:a libopus download.webm // TODO: add -speed 3 maybe? this is horribly slow...,
	cmd := exec.Command(ffmpegPath, "-i", downloadName, "-c:v", "libvpx-vp9", "-c:a", "libopus", "-c:s", "copy", "-threads", strconv.Itoa(threads), "-row-mt", "1", resultName)
	if pipeTerminal {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()
	return <-done
}

func ConvertToHEVCMP4(ffmpegPath, downloadName, resultName string, threads int, pipeTerminal bool) error {
	// ffmpeg -i INPUT -c:v libx265 -c:a copy -x265-params crf=25 OUT.mp4
	cmd := exec.Command(ffmpegPath, "-i", downloadName, "-c:v", "libx265", "-c:a", "copy", "-c:s", "copy", "-threads", strconv.Itoa(threads), "-movflags", " faststart", resultName)
	if pipeTerminal {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()
	return <-done
}

func ConvertToAV1MP4(ffmpegPath, downloadName, resultName string, threads int, pipeTerminal bool) error { // The Daily Blob - TWITCH HACKED or, amazon tech company got spanked_temp.mp4
	// ffmpeg -i input.mp4 -c:v libaom-av1 -crf 30 -b:v 0 av1_test.mkv
	cmd := exec.Command(ffmpegPath, "-i", downloadName, "-c:v", "libaom-av1", "-crf", "17", "-b:v", "0", "-c:a", "copy", "-c:s", "copy", "-threads", strconv.Itoa(threads), "-movflags", " faststart", resultName)
	if pipeTerminal {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()
	return <-done
}

func ExtractAudio(ffmpegPath, sourceName, destinationName string, pipeTerminal bool) error {
	// ffmpeg -i input.mov -map 0:a -c copy output.mov
	cmd := exec.Command(ffmpegPath, "-i", sourceName, "-map", "0:a", "-c", "copy", destinationName)
	if pipeTerminal {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()
	return <-done
}

func DownloadLiveContent(ffmpegPath, hlsURL, fileName string) *exec.Cmd {
	// ffmpeg -i url -acodec copy -vcodec copy mutastreamaf.mp4
	// The above is NOT FAULT TOLERANT because of the MP4 container, use MPEGTS instead
	// ffmpeg -i url -acodec copy -vcodec copy -f mpegts mutastreamaf.ts
	return exec.Command(ffmpegPath, "-i", hlsURL, "-acodec", "copy", "-vcodec", "copy", "-f", "mpegts", fileName)
}

func DownloadLiveContentWithMaps(ffmpegPath, hlsURL, fileName, audioStream, videoStream string, pipeTerminal bool) error {
	// ffmpeg -i url -acodec copy -vcodec copy mutastreamaf.mp4
	// The above is NOT FAULT TOLERANT because of the MP4 container, use MPEGTS instead
	// ffmpeg -i url -acodec copy -vcodec copy -f mpegts mutastreamaf.ts
	// ffmpeg -i https://manifest.googlevideo.com/api/manifest/hls_variant/expire/1611392055/ei/148LYIbYG4L8jQTV8oOYCA/ip/71.81.159.64/id/8G5w-Qr71WY.3/source/yt_live_broadcast/requiressl/yes/hfr/1/playlist_duration/30/manifest_duration/30/maudio/1/vprv/1/go/1/nvgoi/1/keepalive/yes/dover/11/itag/0/playlist_type/DVR/sparams/expire%2Cei%2Cip%2Cid%2Csource%2Crequiressl%2Chfr%2Cplaylist_duration%2Cmanifest_duration%2Cmaudio%2Cvprv%2Cgo%2Citag%2Cplaylist_type/sig/AOq0QJ8wRgIhAI2N_AVxRBFncOselFyAGR2Cny-APUIbBtcVice5VG9zAiEA9qcUj3P_zbKXtAe9ss741D_X19hTemqfQdXww1ryLH4%3D/file/index.m3u8 -acodec copy -vcodec copy -f mpegts mutastreamaf.ts
	cmd := exec.Command(ffmpegPath, "-i", hlsURL, "-map", "0:"+videoStream, "-map", "0:"+audioStream, "-acodec", "copy", "-vcodec", "copy", "-f", "mpegts", fileName)
	if pipeTerminal {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()
	return <-done
}
