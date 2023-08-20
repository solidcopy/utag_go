package service

import (
	"fmt"

	"github.com/solidcopy/utag/internal/handler"
	"github.com/solidcopy/utag/internal/tags_file"
)

func ExecuteImport(dir string) {
	fmt.Println("インポート処理を開始します。")

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

	err = tags_file.ReadImageFile(dir, tracks)
	if err != nil {
		fmt.Println(err)
		return
	}

	handler, err := handler.NewHandler(filePaths[0])
	if err != nil {
		fmt.Println(err)
		return
	}

	for i, track := range tracks {
		track.FilePath = filePaths[i]

		err = handler.WriteTrack(track)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	fmt.Println("インポート処理を終了します。")
}
