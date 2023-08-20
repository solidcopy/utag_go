package tags_file

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/solidcopy/utag/internal/model"
)

func ReadTagsFile(dir string) ([]*model.Track, error) {
	tagsFile, err := os.Open(filepath.Join(dir, "tags"))
	if err != nil {
		return nil, errors.New("tagsファイルを読み込めませんでした。")
	}
	defer tagsFile.Close()

	scanner := bufio.NewScanner(tagsFile)
	scanner.Split(bufio.ScanLines)

	allTracks := []*model.Track{}

	if !scanner.Scan() {
		return allTracks, nil
	}
	album := scanner.Text()

	if !scanner.Scan() {
		return allTracks, nil
	}
	albumArtist := scanner.Text()

	if !scanner.Scan() {
		return allTracks, nil
	}
	date := scanner.Text()

	scanner.Scan()
	if scanner.Text() != "" {
		return allTracks, errors.New("tagsファイルの4行目が空白行ではありません。")
	}

	newDisc := true
	tracksByDisc := [][]*model.Track{}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			newDisc = true
			continue
		}

		if newDisc {
			newDisc = false
			tracksByDisc = append(tracksByDisc, []*model.Track{})
		}

		tokens := strings.Split(line, "//")

		track := &model.Track{
			Album:       album,
			AlbumArtist: albumArtist,
			Date:        date,
			Title:       tokens[0],
			Artists:     tokens[1:],
		}

		index := len(tracksByDisc) - 1
		tracksByDisc[index] = append(tracksByDisc[index], track)
	}

	totalDiscs := len(tracksByDisc)
	for i, tracks := range tracksByDisc {
		totalTracks := len(tracks)
		for j, track := range tracks {
			track.DiscNumber = i + 1
			track.TotalDiscs = totalDiscs
			track.TrackNumber = j + 1
			track.TotalTracks = totalTracks
		}
	}

	for _, tracks := range tracksByDisc {
		allTracks = append(allTracks, tracks...)
	}

	return allTracks, nil
}

func ReadImageFile(dir string, tracks []*model.Track) error {
	extsToMime := map[string]string{".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png"}
	for ext, mimeType := range extsToMime {
		imageFilePath := filepath.Join(dir, "Folder"+ext)

		if stat, err := os.Stat(imageFilePath); os.IsNotExist(err) || stat.IsDir() {
			continue
		}

		imageFile, err := os.Open(imageFilePath)
		if err != nil {
			return errors.New("アートワークを読み込めませんでした。")
		}
		defer imageFile.Close()

		imageData, err := io.ReadAll(imageFile)
		if err != nil {
			return errors.New("アートワークを読み込めませんでした。")
		}

		for _, track := range tracks {
			track.Image = &model.Image{MimeType: mimeType, Data: imageData}
		}

		break
	}

	return nil
}
