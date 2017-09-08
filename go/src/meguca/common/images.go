//go:generate easyjson --all --no_std_marshalers $GOFILE

package common

// Supported file formats
const (
	JPEG uint8 = iota
	PNG
	GIF
	WEBM
	PDF
	SVG
	MP4
	MP3
	OGG
	ZIP
	SevenZip
	TGZ
	TXZ
)

// Extensions maps internal file types to their canonical file extensions
var Extensions = map[uint8]string{
	JPEG:     "jpg",
	PNG:      "png",
	GIF:      "gif",
	MP3:      "mp3",
	MP4:      "mp4",
	WEBM:     "webm",
	OGG:      "ogg",
	PDF:      "pdf",
	ZIP:      "zip",
	SevenZip: "7z",
	TGZ:      "tar.gz",
	TXZ:      "tar.xz",
}

type Images []Image

// Image contains a post's image and thumbnail data
type Image struct {
	ImageCommon
}

// ImageCommon contains the common data shared between multiple post referencing
// the same image
type ImageCommon struct {
	APNG  bool `json:"apng,omitempty"`
	Audio bool `json:"audio,omitempty"`
	// Only used for file formats like OGG and MP4 that may or may not contain
	// video
	Video     bool      `json:"video,omitempty"`
	FileType  uint8     `json:"fileType"`
	ThumbType uint8     `json:"thumbType"`
	Length    uint32    `json:"length,omitempty"`
	Dims      [4]uint16 `json:"dims"`
	Size      int       `json:"size"`
	Artist    string    `json:"artist,omitempty"`
	Title     string    `json:"title,omitempty"`
	MD5       string
	SHA1      string
}
