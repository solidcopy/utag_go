package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/solidcopy/utag/internal/model"
	"github.com/solidcopy/utag/internal/tags_file"
)

func ExecuteRename(dir string) {
	fmt.Println("リネーム処理を開始します。")

	filePaths, err := FindAudioFiles(dir)
	if err != nil {
		fmt.Println(err)
		return
	}

	tracks, err := tags_file.ReadTagsFile(dir)
	if err != nil {
		fmt.Println(err)
		return
	}

	if len(filePaths) != len(tracks) {
		fmt.Println("オーディオファイルとtagsのトラック情報の数が一致しません。")
		return
	}

	for i, track := range tracks {
		filePath := filePaths[i]

		newBaseName := determineNewBaseName(track)
		ext := filepath.Ext(filePath)
		newFileName := newBaseName + ext

		os.Rename(filePath, filepath.Join(dir, newFileName))
	}

	fmt.Println("リネーム処理を終了します。")
}

var charReplacingMap map[string]string = map[string]string{
	"*":  "-",
	"\\": "",
	"|":  "",
	":":  "",
	"\"": "",
	"<":  "(",
	">":  ")",
	"/":  "",
	"?":  "",
}

func determineNewBaseName(track *model.Track) string {
	newBaseName := new(strings.Builder)

	if track.TotalDiscs > 1 {
		length := len(strconv.Itoa(track.TotalDiscs))
		format := "%0" + strconv.Itoa(length) + "d"
		discNumber := fmt.Sprintf(format, track.DiscNumber)
		newBaseName.WriteString(discNumber)
		newBaseName.WriteRune('.')
	}

	length := len(strconv.Itoa(track.TotalTracks))
	format := "%0" + strconv.Itoa(length) + "d"
	trackNumber := fmt.Sprintf(format, track.TrackNumber)
	newBaseName.WriteString(trackNumber)
	newBaseName.WriteRune('.')

	title := track.Title
	for from, to := range charReplacingMap {
		title = strings.ReplaceAll(title, from, to)
	}
	newBaseName.WriteString(title)

	return newBaseName.String()
}
