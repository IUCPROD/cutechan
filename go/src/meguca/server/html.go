package server

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"meguca/auth"
	"meguca/cache"
	"meguca/common"
	"meguca/config"
	"meguca/db"
	"meguca/templates"
	"net/http"
	"strconv"
)

var (
	errNoNews = errors.New("can't get news")
)

// Apply headers and write HTML to client
func serveHTML(
	w http.ResponseWriter,
	r *http.Request,
	etag string,
	data []byte,
	err error,
) {
	if err != nil {
		text500(w, r, err)
		return
	}
	head := w.Header()
	for key, val := range vanillaHeaders {
		head.Set(key, val)
	}
	if etag != "" {
		head.Set("ETag", etag)
	}
	head.Set("Content-Type", "text/html")

	writeData(w, r, data)
}

func serveLanding(w http.ResponseWriter, r *http.Request) {
	pos, ok := extractPosition(w, r)
	if !ok {
		return
	}
	html := templates.Landing(pos)
	serveHTML(w, r, "", html, nil)
}

func serve404(w http.ResponseWriter, r *http.Request) {
	html := templates.NotFound()
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(404)
	io.WriteString(w, html)
}

// Serves board HTML to regular or noscript clients
func boardHTML(w http.ResponseWriter, r *http.Request, b string, catalog bool) {
	if !auth.IsBoard(b) {
		serve404(w, r)
		return
	}
	if !assertNotModOnly(w, r, b) {
		return
	}
	if !assertNotBanned(w, r, b) {
		return
	}

	html, data, ctr, err := cache.GetHTML(boardCacheArgs(r, b, catalog))
	switch err {
	case nil:
	case errPageOverflow:
		serve404(w, r)
		return
	default:
		text500(w, r, err)
		return
	}

	pos, ok := extractPosition(w, r)
	if !ok {
		return
	}

	_, hash := config.GetClient()
	etag := formatEtag(ctr, hash, pos)
	if checkClientEtag(w, r, etag) {
		return
	}

	var n, total int
	if !catalog {
		p := data.(pageStore)
		n = p.pageNumber
		total = p.pageTotal
	}
	html = templates.Board(b, n, total, pos, isMinimal(r), catalog, html)
	serveHTML(w, r, etag, html, nil)
}

// Returns, if the minimal query string is not set
func isMinimal(r *http.Request) bool {
	return r.URL.Query().Get("minimal") == "true"
}

// Asserts a thread exists on the specific board and renders the index template
func threadHTML(w http.ResponseWriter, r *http.Request) {
	id, ok := validateThread(w, r)
	if !ok {
		return
	}

	lastN := detectLastN(r)
	k := cache.ThreadKey(id, lastN)
	html, data, ctr, err := cache.GetHTML(k, threadCache)
	if err != nil {
		respondToJSONError(w, r, err)
		return
	}

	pos, ok := extractPosition(w, r)
	if !ok {
		return
	}

	_, hash := config.GetClient()
	etag := formatEtag(ctr, hash, pos)
	if checkClientEtag(w, r, etag) {
		return
	}

	b := extractParam(r, "board")
	title := data.(common.Thread).Subject
	html = templates.Thread(id, b, title, lastN != 0, pos, html)
	serveHTML(w, r, etag, html, nil)
}

// Extract logged in position for HTML request.
// If ok == false, caller should return.
func extractPosition(w http.ResponseWriter, r *http.Request) (
	pos auth.ModerationLevel, ok bool,
) {
	ok = true
	pos = auth.NotLoggedIn
	creds, err := extractLoginCreds(r)

	switch err {
	case nil:
		board := extractParam(r, "board")
		pos, err = db.FindPosition(board, creds.UserID)
		if err != nil {
			text500(w, r, err)
			ok = false
			return
		}
	case common.ErrInvalidCreds:
		return
	default:
		text500(w, r, err)
		ok = false
		return
	}

	return
}

// Render a board selection and navigation panel and write HTML to client
func boardNavigation(w http.ResponseWriter, r *http.Request) {
	staticTemplate(w, r, templates.BoardNavigation)
}

// Execute a simple template, that accepts no arguments
func staticTemplate(
	w http.ResponseWriter,
	r *http.Request,
	fn func() string,
) {
	serveHTML(w, r, "", []byte(fn()), nil)
}

// Serve a form for selecting one of several boards owned by the user
func ownedBoardSelection(w http.ResponseWriter, r *http.Request) {
	creds, ok := isLoggedIn(w, r)
	if !ok {
		return
	}

	owned, err := db.GetOwnedBoards(creds.UserID)
	if err != nil {
		text500(w, r, err)
		return
	}

	ownedTitles := config.GetBoardTitlesByList(owned)
	serveHTML(w, r, "", []byte(templates.OwnedBoard(ownedTitles)), nil)
}

// Renders a form for configuring a board owned by the user
func boardConfigurationForm(w http.ResponseWriter, r *http.Request) {
	conf, isValid := boardConfData(w, r)
	if !isValid {
		return
	}

	serveHTML(w, r, "", []byte(templates.ConfigureBoard(conf)), nil)
}

// Render a form for assigning staff to a board
func staffAssignmentForm(w http.ResponseWriter, r *http.Request) {
	s, err := db.GetStaff(extractParam(r, "board"))
	if err != nil {
		text500(w, r, err)
		return
	}
	html := []byte(templates.StaffAssignment(
		[...][]string{s["owners"], s["moderators"], s["janitors"]},
	))
	serveHTML(w, r, "", html, nil)
}

// Renders a form for creating new boards
func boardCreationForm(w http.ResponseWriter, r *http.Request) {
	staticTemplate(w, r, templates.CreateBoard)
}

// Render the form for configuring the server
func serverConfigurationForm(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(w, r) {
		return
	}

	data := []byte(templates.ConfigureServer((*config.Get())))
	serveHTML(w, r, "", data, nil)
}

// Render a form to change an account password
func changePasswordForm(w http.ResponseWriter, r *http.Request) {
	staticTemplate(w, r, templates.ChangePassword)
}

// Render a form with nothing but captcha and confirmation buttons
func renderCaptcha(w http.ResponseWriter, r *http.Request) {
	staticTemplate(w, r, templates.CaptchaConfirmation)
}

// Render a link to request a new captcha
func noscriptCaptchaLink(w http.ResponseWriter, r *http.Request) {
	staticTemplate(w, r, templates.NoscriptCaptchaLink)
}

func bannerSettingForm(w http.ResponseWriter, r *http.Request) {
	staticTemplate(w, r, templates.BannerForm)
}

// Render the captcha for noscript browsers
func noscriptCaptcha(w http.ResponseWriter, r *http.Request) {
	ip, err := auth.GetIP(r)
	if err != nil {
		text400(w, err)
		return
	}
	serveHTML(w, r, "", []byte(templates.NoscriptCaptcha(ip)), nil)
}

// Redirect the client to the appropriate board through a cross-board redirect
func crossRedirect(w http.ResponseWriter, r *http.Request) {
	idStr := extractParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		serve404(w, r)
		return
	}

	board, op, err := db.GetPostParenthood(id)
	switch err {
	case nil:
		if !assertNotModOnly(w, r, board) {
			return
		}
		url := r.URL
		url.Path = fmt.Sprintf("/%s/%d", board, op)
		http.Redirect(w, r, url.String(), 301)
	case sql.ErrNoRows:
		serve404(w, r)
	default:
		text500(w, r, err)
	}
}
