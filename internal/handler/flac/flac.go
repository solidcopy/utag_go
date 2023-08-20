package flac

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	"github.com/go-flac/go-flac"
	utag "github.com/solidcopy/utag/internal"
	"github.com/solidcopy/utag/internal/model"
)

type FlacHandler struct {
}

type Blocks = []*flac.MetaDataBlock

func (h *FlacHandler) ReadTrack(filePath string) (*model.Track, error) {
	flacFile, err := flac.ParseFile(filePath)
	if err != nil {
		return nil, err
	}

	blocks := flacFile.Meta
	comments := getVorbisComments(blocks)

	track := &model.Track{
		FilePath:    filePath,
		Album:       getString(comments, "ALBUM"),
		AlbumArtist: getString(comments, "ALBUMARTIST"),
		Date:        getString(comments, "DATE"),
		Image:       getImage(blocks),
		DiscNumber:  getInt(comments, "DISCNUMBER"),
		TotalDiscs:  getInt(comments, "DISCTOTAL"),
		TrackNumber: getInt(comments, "TRACKNUMBER"),
		TotalTracks: getInt(comments, "TRACKTOTAL"),
		Title:       getString(comments, "TITLE"),
		Artists:     getValues(comments, "ARTIST"),
	}

	return track, nil
}

func (h *FlacHandler) WriteTrack(track *model.Track) error {
	flacFile, err := flac.ParseFile(track.FilePath)
	if err != nil {
		return err
	}

	blocks := removeVorbisCommentsAndPicture(flacFile.Meta)

	flacFile.Meta, err = addVorbisCommentsAndPicture(blocks, track)
	if err != nil {
		return err
	}

	err = flacFile.Save(track.FilePath)
	if err != nil {
		return err
	}

	return nil
}

func getVorbisComments(blocks Blocks) map[string][]string {
	for _, block := range blocks {
		if block.Type != flac.VorbisComment {
			continue
		}

		comment, err := flacvorbis.ParseFromMetaDataBlock(*block)
		if err != nil {
			continue
		}

		vorbisComments := make(map[string][]string, len(comment.Comments))

		for _, comment := range comment.Comments {
			split := strings.SplitN(comment, "=", 2)
			if len(split) == 2 {
				name := strings.ToUpper(split[0])
				values, ok := vorbisComments[name]
				if !ok {
					values = []string{}
				}
				vorbisComments[name] = append(values, split[1])
			}
		}

		return vorbisComments
	}

	return map[string][]string{}
}

func getValues(comments map[string][]string, commentName string) []string {
	values, ok := comments[commentName]
	if !ok {
		return []string{}
	}
	return values
}

func getString(comments map[string][]string, commentName string) string {
	values := getValues(comments, commentName)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

func getInt(comments map[string][]string, commentName string) int {
	value, err := strconv.Atoi(getString(comments, commentName))
	if err != nil {
		value = 0
	}
	return value
}

func getImage(blocks Blocks) *model.Image {
	var picture *flacpicture.MetadataBlockPicture
	for _, block := range blocks {
		if block.Type == flac.Picture {
			parsedPicture, err := flacpicture.ParseFromMetaDataBlock(*block)
			if err != nil {
				continue
			}
			if picture == nil || picture.PictureType == flacpicture.PictureTypeFrontCover {
				picture = parsedPicture
				if picture.PictureType == flacpicture.PictureTypeFrontCover {
					break
				}
			}
		}
	}

	if picture == nil {
		return nil
	}

	mimeType := picture.MIME
	if mimeType == "" {
		mimeType = http.DetectContentType(picture.ImageData)
	}

	return &model.Image{MimeType: mimeType, Data: picture.ImageData}
}

func removeVorbisCommentsAndPicture(blocks Blocks) Blocks {
	newBlocks := Blocks{}
	for _, block := range blocks {
		if block.Type != flac.VorbisComment && block.Type != flac.Picture && block.Type != flac.Padding {
			newBlocks = append(newBlocks, block)
		}
	}
	return newBlocks
}

func addVorbisCommentsAndPicture(blocks Blocks, track *model.Track) (Blocks, error) {

	vorbisComment := flacvorbis.New()

	vorbisComment.Vendor = "utag " + utag.Version

	vorbisComment.Add("ALBUM", track.Album)
	vorbisComment.Add("ALBUMARTIST", track.AlbumArtist)
	vorbisComment.Add("DATE", track.Date)
	setInt(vorbisComment, "DISCNUMBER", track.DiscNumber)
	setInt(vorbisComment, "DISCTOTAL", track.TotalDiscs)
	setInt(vorbisComment, "TRACKNUMBER", track.TrackNumber)
	setInt(vorbisComment, "TRACKTOTAL", track.TotalTracks)
	vorbisComment.Add("TITLE", track.Title)
	vorbisComment.Add("ARTIST", track.AlbumArtist)
	for _, artist := range track.Artists {
		vorbisComment.Add("ARTIST", artist)
	}

	vorbisCommentBlock := vorbisComment.Marshal()
	blocks = append(blocks, &vorbisCommentBlock)

	if track.Image != nil {
		picture, err := flacpicture.NewFromImageData(flacpicture.PictureTypeFrontCover, "", track.Image.Data, track.Image.MimeType)
		if err == nil {
			pictureBlock := picture.Marshal()
			blocks = append(blocks, &pictureBlock)
		}
	}

	padding := flac.MetaDataBlock{Type: flac.Padding, Data: make([]byte, 64)}
	blocks = append(blocks, &padding)

	return blocks, nil
}

func setInt(vorbisComment *flacvorbis.MetaDataBlockVorbisComment, name string, value int) {
	if value != 0 {
		vorbisComment.Add(name, strconv.Itoa(value))
	}
}
