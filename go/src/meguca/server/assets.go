package server

import (
	"fmt"
	"io/ioutil"
	"meguca/assets"
	"meguca/auth"
	"meguca/common"
	"meguca/db"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/bakape/thumbnailer"
)

// More performant handler for serving image assets. These are immutable
// (except deletion), so we can also set separate caching policies for them.
func serveImages(w http.ResponseWriter, r *http.Request) {
	path := extractParam(r, "path")
	file, err := os.Open(cleanJoin(common.ImageWebRoot, path))
	if err != nil {
		text404(w)
		return
	}
	defer file.Close()

	head := w.Header()
	for key, val := range imageHeaders {
		head.Set(key, val)
	}

	http.ServeContent(w, r, path, time.Time{}, file)
}

func cleanJoin(a, b string) string {
	return filepath.Clean(filepath.Join(a, b))
}

func serveFile(w http.ResponseWriter, r *http.Request, path string) {
	file, err := os.Open(path)
	if err != nil {
		text404(w)
		return
	}
	defer file.Close()

	stats, err := file.Stat()
	if err != nil {
		text500(w, r, err)
		return
	}
	if stats.IsDir() {
		text404(w)
		return
	}
	modTime := stats.ModTime()
	etag := strconv.FormatInt(modTime.Unix(), 10)

	head := w.Header()
	for key, val := range vanillaHeaders {
		head.Set(key, val)
	}
	head.Set("ETag", etag)
	http.ServeContent(w, r, path, modTime, file)
}

// Server static assets
func serveStatic(w http.ResponseWriter, r *http.Request) {
	serveFile(w, r, cleanJoin(common.WebRoot, extractParam(r, "path")))
}

func serveWorker(w http.ResponseWriter, r *http.Request) {
	serveFile(w, r, filepath.FromSlash(common.WebRoot+"/js/worker.js"))
}

// Set the banners of a board
func setBanners(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		common.MaxNumBanners*common.MaxBannerSize,
	)
	if err := r.ParseMultipartForm(0); err != nil {
		text400(w, err)
		return
	}

	f := r.Form
	board := f.Get("board")
	_, ok := canPerform(w, r, board, auth.BoardOwner, &auth.Captcha{
		CaptchaID: f.Get("captchaID"),
		Solution:  f.Get("captcha"),
	})
	if !ok {
		return
	}

	var (
		opts = thumbnailer.Options{
			JPEGQuality: 0,
			MaxSourceDims: thumbnailer.Dims{
				Width:  300,
				Height: 100,
			},
			ThumbDims: thumbnailer.Dims{
				Width:  300,
				Height: 100,
			},
			AcceptedMimeTypes: map[string]bool{
				"image/jpeg": true,
				"image/png":  true,
				"image/gif":  true,
				"video/webm": true,
			},
		}
		banners = make([]assets.File, 0, common.MaxNumBanners)
		files   = r.MultipartForm.File["banners"]
	)
	for i := 0; i < common.MaxNumBanners && i < len(files); i++ {
		h := files[i]
		file, err := h.Open()
		if err != nil {
			sendFileError(w, h, err.Error())
			return
		}
		defer file.Close()

		buf, err := ioutil.ReadAll(file)
		if err != nil {
			text500(w, r, err)
			return
		}

		if len(buf) > common.MaxBannerSize {
			sendFileError(w, h, "too large")
			return
		}

		src, _, err := thumbnailer.ProcessBuffer(buf, opts)
		switch {
		case err != nil:
			sendFileError(w, h, err.Error())
			return
		case src.HasAudio:
			sendFileError(w, h, "has audio")
			return
		}
		banners = append(banners, assets.File{
			Data: buf,
			Mime: src.Mime,
		})
	}

	if err := db.SetBanners(board, banners); err != nil {
		text500(w, r, err)
	}
}

func sendFileError(w http.ResponseWriter, h *multipart.FileHeader, msg string) {
	http.Error(w, fmt.Sprintf("400 invalid file %s: %s", h.Filename, msg), 400)
}

// Serve board-specific image banner files
func serveBanner(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(extractParam(r, "id"))
	if err != nil {
		text404(w)
		return
	}

	f, ok := assets.Banners.Get(extractParam(r, "board"), id)
	if !ok {
		text404(w)
		return
	}
	h := w.Header()
	h.Set("Content-Type", f.Mime)
	h.Set("Content-Length", strconv.Itoa(len(f.Data)))
	w.Write(f.Data)
}
