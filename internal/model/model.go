package model

type Track struct {
	FilePath string
	// アルバム情報
	Album       string
	AlbumArtist string
	Date        string
	Image       *Image
	// ディスク情報
	DiscNumber int
	TotalDiscs int
	// トラック情報
	TrackNumber int
	TotalTracks int
	Title       string
	Artists     []string
}

type Image struct {
	MimeType string
	Data     []byte
}
