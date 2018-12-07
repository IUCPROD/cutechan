// File package provides file backend abstraction.
package file

import (
	"net/http"
	"strings"

	"meguca/common"
	"meguca/config"
)

// Current file backend.
var Backend FileBackend

type Config struct {
	Backend  string
	Dir      string
	Username string
	Password string
	AuthURL  string
}

type FileBackend interface {
	IsServable() bool
	Serve(w http.ResponseWriter, r *http.Request)
	Write(sha1 string, fileType, thumbType uint8, src, thumb []byte) error
	Delete(sha1 string, fileType, thumbType uint8) error
}

const (
	DefaultUploadsPath = "/uploads"
	srcDir             = "src"
	thumbDir           = "thumb"
)

func StartBackend(conf Config) (err error) {
	if conf.Backend == "fs" {
		Backend, err = makeFSBackend(conf)
	} else if conf.Backend == "swift" {
		Backend, err = makeSwiftBackend(conf)
	} else {
		panic("unknown backend")
	}
	return
}

func imageRoot() string {
	r := config.Get().ImageRootOverride
	if r != "" {
		return r
	}
	return DefaultUploadsPath
}

func imagePath(root string, dir string, typ uint8, SHA1 string) string {
	return strings.Join([]string{
		root,
		dir,
		SHA1[:2],
		SHA1[2:] + "." + common.Extensions[typ],
	}, "/")
}

func SourcePath(fileType uint8, SHA1 string) string {
	return imagePath(imageRoot(), srcDir, fileType, SHA1)
}

func ThumbPath(thumbType uint8, SHA1 string) string {
	return imagePath(imageRoot(), thumbDir, thumbType, SHA1)
}
