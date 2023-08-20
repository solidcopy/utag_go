package handler

import (
	"path/filepath"

	"github.com/solidcopy/utag/internal/handler/flac"
	"github.com/solidcopy/utag/internal/handler/id3v2"
	"github.com/solidcopy/utag/internal/handler/m4a"
	"github.com/solidcopy/utag/internal/model"
)

type FileHandler interface {
	ReadTrack(filePath string) (*model.Track, error)
	WriteTrack(track *model.Track) error
}

func NewHandler(filePath string) (FileHandler, error) {
	extension := filepath.Ext(filePath)
	switch extension {
	case ".mp3", ".dsf":
		return &id3v2.Id3v2Handler{}, nil
	case ".flac":
		return &flac.FlacHandler{}, nil
	case ".m4a":
		return &m4a.M4aHandler{}, nil
	default:
		return nil, nil
	}
}
