package service

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/solidcopy/utag/internal/handler"
	"github.com/solidcopy/utag/internal/model"

	"golang.org/x/exp/slices"
)

var AllExtensions []string = []string{
	".flac", ".m4a", ".mp3", ".dsf",
}

func findFiles(dir string) ([]string, error) {

	files := []string{}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && path != dir {
			return filepath.SkipDir
		}

		if slices.Contains(AllExtensions, filepath.Ext(path)) {
			files = append(files, path)
		}

		return nil
	})

	return files, nil
}

func FindAudioFiles(dir string) ([]string, error) {
	filePaths, err := findFiles(dir)
	if err != nil || len(filePaths) == 0 {
		return nil, errors.New("オーディオファイルが見つかりません。")
	}

	// ファイルの種類が混在していたらエラー
	extension := filepath.Ext(filePaths[0])
	for _, file := range filePaths[1:] {
		if filepath.Ext(file) != extension {
			return nil, errors.New("オーディオファイルの種類が混在しています。")
		}
	}

	return filePaths, nil
}

func ReadTracks(dir string) ([]*model.Track, error) {

	filePaths, err := FindAudioFiles(dir)
	if err != nil {
		return nil, err
	}

	handler, err := handler.NewHandler(filePaths[0])
	if err != nil {
		return nil, err
	}

	tracks := make([]*model.Track, 0, len(filePaths))
	for _, filePath := range filePaths {
		track, err := handler.ReadTrack(filePath)
		if err != nil {
			return nil, errors.New("タグ情報の読み込みに失敗しました。")
		}
		tracks = append(tracks, track)
	}

	return tracks, nil
}
