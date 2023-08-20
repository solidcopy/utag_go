package service

import (
	"fmt"

	"github.com/solidcopy/utag/internal/tags_file"
)

func ExecuteExport(dir string) {
	fmt.Println("エクスポート処理を開始します。")

	tracks, err := ReadTracks(dir)
	if err != nil {
		fmt.Println(err)
		return
	}

	tags_file.WriteTagsFile(tracks)

	tags_file.WriteImageFile(tracks[0])

	fmt.Println("エクスポート処理を完了しました。")
}
