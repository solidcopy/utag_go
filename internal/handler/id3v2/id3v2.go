package id3v2

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bogem/id3v2/v2"
	"github.com/solidcopy/utag/internal/model"
)

type Id3v2Handler struct {
}

func (h *Id3v2Handler) ReadTrack(filePath string) (*model.Track, error) {

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if filepath.Ext(filePath) == ".dsf" {
		pointer, err := seekToMetadataChunk(file)
		if err != nil {
			return nil, err
		}

		if pointer == 0 {
			return &model.Track{FilePath: filePath}, nil
		}
	}

	tags, err := id3v2.ParseReader(file, id3v2.Options{Parse: true})
	if err != nil {
		return nil, err
	}

	discNumber, totalDiscs := parsePosAndTotal(tags.GetTextFrame("TPOS").Text)
	trackNumber, totalTracks := parsePosAndTotal(tags.GetTextFrame("TRCK").Text)

	var artists []string
	switch tags.Version() {
	case byte(3):
		artists = strings.Split(tags.Artist(), "/")
	case byte(4):
		artists = strings.Split(tags.Artist(), "\x00")
	default:
		artists = []string{tags.Artist()}
	}

	track := &model.Track{
		FilePath:    filePath,
		Album:       tags.Album(),
		AlbumArtist: tags.GetTextFrame("TPE2").Text,
		Date:        tags.GetTextFrame("TDRL").Text,
		Image:       getImage(tags),
		DiscNumber:  discNumber,
		TotalDiscs:  totalDiscs,
		TrackNumber: trackNumber,
		TotalTracks: totalTracks,
		Title:       tags.Title(),
		Artists:     artists,
	}

	return track, nil
}

func getImage(tags *id3v2.Tag) *model.Image {
	frames := tags.GetFrames("APIC")
	if len(frames) == 0 {
		return nil
	}

	var pictureFrame *id3v2.PictureFrame
	for i, frame := range frames {
		p, ok := frame.(id3v2.PictureFrame)
		if !ok {
			continue
		}

		// とりあえず最初の画像を選択し、フロントカバーがあればそちらを優先する
		if i == 0 || p.PictureType == id3v2.PTFrontCover {
			pictureFrame = &p
		}
	}

	if pictureFrame == nil {
		return nil
	}

	mimeType := pictureFrame.MimeType
	if mimeType == "" {
		mimeType = http.DetectContentType(pictureFrame.Picture)
	}

	data := pictureFrame.Picture

	return &model.Image{MimeType: mimeType, Data: data}
}

func (h *Id3v2Handler) WriteTrack(track *model.Track) error {

	if filepath.Ext(track.FilePath) == ".dsf" {
		tags := id3v2.NewEmptyTag()
		SetTags(tags, track)

		var pointer int64
		{
			file, err := os.Open(track.FilePath)
			if err != nil {
				return err
			}
			defer file.Close()

			pointer, err = seekToMetadataChunk(file)
			if err != nil {
				return err
			}
		}

		// 既存のタグを削除する
		if pointer != 0 {
			err := os.Truncate(track.FilePath, pointer)
			if err != nil {
				return err
			}
		}

		file, err := os.OpenFile(track.FilePath, os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		defer file.Close()

		endPointer, err := file.Seek(0, io.SeekEnd)
		if err != nil {
			return err
		}

		writtenSize, err := tags.WriteTo(file)
		if err != nil {
			return err
		}

		buff := make([]byte, 8)

		// ファイル容量を更新する
		file.Seek(12, io.SeekStart)
		binary.LittleEndian.PutUint64(buff, uint64(endPointer)+uint64(writtenSize))
		file.Write(buff)

		// メタデータチャンクの開始位置を更新する
		binary.LittleEndian.PutUint64(buff, uint64(endPointer))
		file.Write(buff)

	} else {
		tags, err := id3v2.Open(track.FilePath, id3v2.Options{Parse: false})
		if err != nil {
			return err
		}

		SetTags(tags, track)

		err = tags.Save()
		if err != nil {
			return err
		}

		file, err := os.Open(track.FilePath)
		if err != nil {
			return err
		}

		stat, err := os.Stat(track.FilePath)
		if err != nil {
			return err
		}

		fileSize := stat.Size()

		if fileSize > 128 {
			pointer := fileSize - 128
			_, err = file.Seek(pointer, io.SeekStart)
			if err != nil {
				return err
			}

			var buff [3]byte
			_, err = file.Read(buff[:])
			if err != nil {
				return err
			}

			err = file.Close()
			if err != nil {
				return err
			}

			if string(buff[:]) == "TAG" {
				err = os.Truncate(track.FilePath, pointer)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func SetTags(tags *id3v2.Tag, track *model.Track) {
	tags.SetVersion(4)
	tags.SetAlbum(track.Album)
	tags.AddTextFrame("TPE2", id3v2.EncodingUTF8, track.AlbumArtist)
	tags.AddTextFrame("TDRL", id3v2.EncodingUTF8, track.Date)

	if track.Image != nil {
		tags.AddAttachedPicture(id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    track.Image.MimeType,
			PictureType: id3v2.PTFrontCover,
			Description: "Cover",
			Picture:     track.Image.Data,
		})
	}

	tags.AddTextFrame("TPOS", id3v2.EncodingUTF8, formatPosAndTotal(track.DiscNumber, track.TotalDiscs))
	tags.AddTextFrame("TRCK", id3v2.EncodingUTF8, formatPosAndTotal(track.TrackNumber, track.TotalTracks))
	tags.SetTitle(track.Title)

	artists := []string{}
	if track.AlbumArtist != "" {
		artists = append(artists, track.AlbumArtist)
	}
	artists = append(artists, track.Artists...)
	// v2.4で保存するので、\x00区切りにする
	tags.SetArtist(strings.Join(artists, "\x00"))
}

// DSFはID3v2がファイルの先頭ではなく末尾にある。
// メタデータチャンクの開始位置を取得して、
// その位置までファイルの読み込み位置を進める。
func seekToMetadataChunk(file *os.File) (int64, error) {

	// "DSD "の4バイト、チャンクサイズの8バイト、合計ファイルサイズの8バイトの
	// 計20バイトをスキップする
	ret, err := file.Seek(20, 0)
	if err != nil {
		return 0, err
	}
	if ret < 20 {
		return 0, errors.New("DSFファイルの形式が不正です。")
	}

	// メタデータチャンクの先頭ポインタを取得する
	buff := make([]byte, 4)
	_, err = file.Read(buff)
	if err != nil {
		return 0, errors.New("DSFファイルの読み込みに失敗しました。")
	}
	pointer := int64(binary.LittleEndian.Uint32(buff))

	if pointer != 0 {
		file.Seek(pointer, 0)
	}

	return pointer, nil
}

func formatPosAndTotal(pos, total int) string {
	if total == 0 {
		return strconv.Itoa(pos)
	}

	return fmt.Sprintf("%d/%d", pos, total)
}

func parsePosAndTotal(s string) (int, int) {
	if s == "" {
		return 0, 0
	}

	split := strings.Split(s, "/")

	pos, err := strconv.Atoi(split[0])
	if err != nil {
		return 0, 0
	}

	total := 0
	if len(split) > 1 {
		total, err = strconv.Atoi(split[1])
		if err != nil {
			return 0, 0
		}
	}

	return pos, total
}
