package tags_file

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/solidcopy/utag/internal/model"
	"golang.org/x/exp/slices"
)

func WriteTagsFile(tracks []*model.Track) error {

	track := tracks[0]

	dir := filepath.Dir(track.FilePath)

	tagsFilePath := filepath.Join(dir, "tags")

	tagsFile, err := os.Create(tagsFilePath)
	if err != nil {
		return err
	}
	defer tagsFile.Close()

	tagsFile.WriteString(track.Album)
	tagsFile.WriteString("\n")
	tagsFile.WriteString(track.AlbumArtist)
	tagsFile.WriteString("\n")
	tagsFile.WriteString(track.Date)
	tagsFile.WriteString("\n")

	tagsFile.WriteString("\n")

	checkDiscNumber := isDiscNumberConsistent(tracks)
	curDiscNumber := 1

	for _, track := range tracks {
		if checkDiscNumber && track.DiscNumber != curDiscNumber {
			tagsFile.WriteString("\n")
			curDiscNumber = track.DiscNumber
		}

		tagsFile.WriteString(track.Title)

		artists := slices.DeleteFunc(track.Artists, func(a string) bool {
			return a == "" || a == track.AlbumArtist
		})
		if len(artists) > 0 {
			tagsFile.WriteString("//")
			tagsFile.WriteString(strings.Join(artists, "//"))
		}

		tagsFile.WriteString("\n")
	}

	return nil
}

func isDiscNumberConsistent(tracks []*model.Track) bool {
	currentDiscNumber := 1
	for _, track := range tracks {
		discNumber := track.DiscNumber
		if discNumber != currentDiscNumber && discNumber != currentDiscNumber+1 {
			return false
		}
		currentDiscNumber = discNumber
	}

	return true
}

func WriteImageFile(track *model.Track) error {

	if track.Image == nil {
		return nil
	}

	dir := filepath.Dir(track.FilePath)

	mimeType := track.Image.MimeType

	var imageFileName string
	switch mimeType {
	case "image/jpeg", "image/jpg":
		imageFileName = "Folder.jpg"
	case "image/png":
		imageFileName = "Folder.png"
	case "image/gif":
		imageFileName = "Folder.gif"
	default:
		return nil
	}
	imageFilePath := filepath.Join(dir, imageFileName)

	imageFile, err := os.Create(imageFilePath)
	if err != nil {
		return err
	}
	defer imageFile.Close()

	imageFile.Write(track.Image.Data)

	return nil
}
