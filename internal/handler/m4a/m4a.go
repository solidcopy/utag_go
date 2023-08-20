package m4a

import (
	"bytes"
	"encoding/binary"
	"net/http"
	"os"

	"github.com/abema/go-mp4"
	"github.com/solidcopy/utag/internal/model"
	"golang.org/x/exp/slices"
)

type M4aHandler struct{}

func (h *M4aHandler) ReadTrack(filePath string) (*model.Track, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	track := &model.Track{FilePath: filePath}

	parents := []string{"moov", "udta", "meta", "ilst"}
	target := []string{"(c)nam", "(c)ART", "(c)alb", "(c)day", "aART", "trkn", "disk", "covr"}

	var itemName string

	_, err = mp4.ReadBoxStructure(file, func(h *mp4.ReadHandle) (interface{}, error) {

		if h.BoxInfo.IsSupportedType() {

			typeName := h.BoxInfo.Type.String()

			if slices.Contains(parents, typeName) || slices.Contains(target, typeName) {
				itemName = typeName
				return h.Expand()
			}

			if typeName == "data" {

				buff := new(bytes.Buffer)
				h.ReadData(buff)

				// 最初の8バイトはデータ本体ではなさそうなので削除
				data := buff.Bytes()[8:]

				switch itemName {
				case "(c)nam":
					track.Title = string(data)
				case "(c)ART":
					track.Artists = append(track.Artists, string(data))
				case "(c)alb":
					track.Album = string(data)
				case "(c)day":
					track.Date = string(data)
				case "aART":
					track.AlbumArtist = string(data)
				case "trkn":
					track.TrackNumber = int(binary.BigEndian.Uint16(data[2:4]))
					track.TotalTracks = int(binary.BigEndian.Uint16(data[4:6]))
				case "disk":
					track.DiscNumber = int(binary.BigEndian.Uint16(data[2:4]))
					track.TotalDiscs = int(binary.BigEndian.Uint16(data[4:6]))
				case "covr":
					mimeType := http.DetectContentType(data)
					track.Image = &model.Image{MimeType: mimeType, Data: data}
				}
			}
		}
		return nil, nil
	})

	if err != nil {
		return nil, err
	}

	return track, nil
}

func (h *M4aHandler) WriteTrack(track *model.Track) error {

	file, err := os.Open(track.FilePath)
	if err != nil {
		return err
	}

	newFilePath := track.FilePath + ".utag_temp"
	newFile, err := os.OpenFile(newFilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}

	w := mp4.NewWriter(newFile)

	metaBoxes, err := mp4.ExtractBox(file, nil, mp4.BoxPath{mp4.BoxTypeMoov(), mp4.BoxTypeUdta(), mp4.BoxTypeMeta()})
	if err != nil {
		return err
	}
	noMetaBox := len(metaBoxes) == 0

	_, err = mp4.ReadBoxStructure(file, func(h *mp4.ReadHandle) (interface{}, error) {
		switch h.BoxInfo.Type {
		case mp4.BoxTypeMoov(), mp4.BoxTypeUdta(), mp4.BoxTypeMeta():
			_, err := w.StartBox(&h.BoxInfo)
			if err != nil {
				return nil, err
			}

			box, _, err := h.ReadPayload()
			if err != nil {
				return nil, err
			}

			_, err = mp4.Marshal(w, box, h.BoxInfo.Context)
			if err != nil {
				return nil, err
			}

			createMetaBox := noMetaBox && h.BoxInfo.Type == mp4.BoxTypeUdta()
			if createMetaBox {
				_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeMeta()})
				if err != nil {
					return nil, err
				}

				meta := mp4.Meta{}
				_, err = mp4.Marshal(w, &meta, mp4.Context{UnderUdta: true})
				if err != nil {
					return nil, err
				}
			}

			if createMetaBox || h.BoxInfo.Type == mp4.BoxTypeMeta() {
				_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeIlst()})
				if err != nil {
					return nil, err
				}

				addStringTag(w, "\251alb", track.Album)
				addStringTag(w, "aART", track.AlbumArtist)
				addStringTag(w, "\251day", track.Date)
				addNumberAndTotalTag(w, "trkn", track.TrackNumber, track.TotalTracks)
				addNumberAndTotalTag(w, "disk", track.DiscNumber, track.TotalDiscs)
				addStringTag(w, "\251nam", track.Title)
				for _, artist := range track.Artists {
					addStringTag(w, "\251ART", artist)
				}
				addBytesTag(w, "covr", track.Image.Data)

				_, err = w.EndBox()
				if err != nil {
					return nil, err
				}
			}

			if createMetaBox {
				_, err = w.EndBox()
				if err != nil {
					return nil, err
				}
			}

			_, err = h.Expand()
			if err != nil {
				return nil, err
			}

			_, err = w.EndBox()
			return nil, err

		case mp4.BoxTypeIlst():
			return nil, nil
		default:
			return nil, w.CopyBox(file, &h.BoxInfo)
		}
	})

	if err != nil {
		return err
	}

	file.Close()
	newFile.Close()

	os.Remove(track.FilePath)
	os.Rename(newFilePath, track.FilePath)

	return nil
}

func addStringTag(w *mp4.Writer, name string, value string) error {

	err := startTagBox(w, name)
	if err != nil {
		return err
	}

	boxData := mp4.Data{DataType: mp4.DataTypeStringUTF8, Data: []byte(value)}

	_, err = mp4.Marshal(w, &boxData, mp4.Context{UnderIlstMeta: true})
	if err != nil {
		return err
	}

	err = endTagBox(w)
	if err != nil {
		return err
	}

	return nil
}

func addBytesTag(w *mp4.Writer, name string, value []byte) error {

	err := startTagBox(w, name)
	if err != nil {
		return err
	}

	boxData := mp4.Data{DataType: mp4.DataTypeBinary, Data: value}

	_, err = mp4.Marshal(w, &boxData, mp4.Context{UnderIlstMeta: true})
	if err != nil {
		return err
	}

	err = endTagBox(w)
	if err != nil {
		return err
	}

	return nil
}

func addNumberAndTotalTag(w *mp4.Writer, name string, number int, total int) error {

	err := startTagBox(w, name)
	if err != nil {
		return err
	}

	data := make([]byte, 8)
	binary.BigEndian.PutUint16(data[2:4], uint16(number))
	binary.BigEndian.PutUint16(data[4:6], uint16(total))

	boxData := mp4.Data{DataType: mp4.DataTypeBinary, Data: data}

	_, err = mp4.Marshal(w, &boxData, mp4.Context{UnderIlstMeta: true})
	if err != nil {
		return err
	}

	err = endTagBox(w)
	if err != nil {
		return err
	}

	return nil
}

func startTagBox(w *mp4.Writer, name string) error {

	_, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxType([]byte(name))})
	if err != nil {
		return err
	}

	_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeData()})
	if err != nil {
		return err
	}

	return nil
}

func endTagBox(w *mp4.Writer) error {

	_, err := w.EndBox()
	if err != nil {
		return err
	}

	_, err = w.EndBox()
	if err != nil {
		return err
	}

	return nil
}
