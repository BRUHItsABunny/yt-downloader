package utils

import (
	"path/filepath"
	"runtime"
	"strings"
)

type AppArgs struct {
	Video          *string
	List           *string
	Playlist       *string
	Channel        *string
	FFmpegPath     *string
	StoreMetadata  *string
	MP4            *bool
	HEVC           *bool
	PrependVideoID *bool
	AudioOnly      *bool
	Subs           *bool
	MergeSubs      *bool
	Debug          *bool
	Threads        *int
	FFmpegThreads  *int
	Amount         *int
	Offset         *int
}

func (args *AppArgs) checkStrEmpty(attribute *string) (string, bool) {
	if attribute != nil {
		value := *attribute
		return value, len(value) > 0
	}
	return "", false
}

func (args *AppArgs) DoVideo() bool {
	_, ok := args.checkStrEmpty(args.Video)
	return ok
}

func (args *AppArgs) DoList() bool {
	_, ok := args.checkStrEmpty(args.List)
	return ok
}

func (args *AppArgs) DoPlaylist() bool {
	_, ok := args.checkStrEmpty(args.Playlist)
	return ok
}

func (args *AppArgs) DoChannel() bool {
	_, ok := args.checkStrEmpty(args.Channel)
	return ok
}

func (args *AppArgs) GetVideo() string {
	result, _ := args.checkStrEmpty(args.Video)
	return result
}

func (args *AppArgs) GetList() string {
	result, _ := args.checkStrEmpty(args.List)
	return result
}

func (args *AppArgs) GetStoreMetadata() string {
	result, _ := args.checkStrEmpty(args.StoreMetadata)
	return result
}

func (args *AppArgs) GetPlaylist() string {
	result, _ := args.checkStrEmpty(args.Playlist)
	return result
}

func (args *AppArgs) GetChannel() string {
	result, _ := args.checkStrEmpty(args.Channel)
	return result
}

func (args *AppArgs) GetFFmpegPath() string {
	result, _ := args.checkStrEmpty(args.FFmpegPath)
	return result
}

func (args *AppArgs) GetFFprobePath() string {
	result, _ := args.checkStrEmpty(args.FFmpegPath)
	if strings.Contains(result, "\\") {
		result = filepath.FromSlash(result)
	}
	fileName := filepath.Base(result)
	newFileName := "ffprobe"
	if strings.Contains(fileName, ".") {
		newFileName += strings.Split(fileName, ".")[1]
	}
	result = strings.Replace(result, fileName, newFileName, 1)
	return result
}

func (args *AppArgs) GetThreads() int {
	if *args.Threads < 1 {
		return 1
	} else if *args.Threads > 12 {
		return 12
	} else {
		return *args.Threads
	}
}

func (args *AppArgs) GetFFmpegThreads() int {
	cpuCores := runtime.NumCPU()
	if *args.FFmpegThreads < 1 {
		return 1
	} else if *args.FFmpegThreads > cpuCores {
		return cpuCores
	} else {
		return *args.Threads
	}
}

func (args *AppArgs) GetAmount() int {
	if *args.Amount < 1 {
		return 1
	} else {
		return *args.Amount
	}
}

func (args *AppArgs) GetOffset() int {
	if *args.Amount < 0 {
		return 0
	} else {
		return *args.Offset
	}
}
