package utils

import (
	"os"
	"strings"
)

func GetVideoURL(videoStr string) string {
	if !strings.HasPrefix(videoStr, "https://") {
		videoStr = "https://www.youtube.com/watch?v=" + videoStr
	}
	return videoStr
}

func GetPlaylistURL(playlistStr string) string {
	if !strings.HasPrefix(playlistStr, "https://") {
		playlistStr = "https://www.youtube.com/playlist?list=" + playlistStr
	}
	return playlistStr
}

func FileExists(location string) bool {
	_, err := os.Stat(location)
	return err == nil
}

func CreateOrOpen(location string) (*os.File, error) {
	var f *os.File
	var fileErr error
	if _, err := os.Stat(location); os.IsNotExist(err) {
		f, fileErr = os.Create(location)
	} else {
		f, fileErr = os.Open(location)
	}
	return f, fileErr
}

func SanitizeFileName(fileName string) string {
	fileName = strings.ReplaceAll(fileName, `\`, ``)
	fileName = strings.ReplaceAll(fileName, `/`, ``)
	fileName = strings.ReplaceAll(fileName, `?`, ``)
	fileName = strings.ReplaceAll(fileName, `*`, ``)
	fileName = strings.ReplaceAll(fileName, `:`, ``)
	fileName = strings.ReplaceAll(fileName, `<`, ``)
	fileName = strings.ReplaceAll(fileName, `>`, ``)
	fileName = strings.ReplaceAll(fileName, `|`, ``)
	fileName = strings.ReplaceAll(fileName, `"`, ``)
	return fileName
}
