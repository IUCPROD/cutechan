// Template structs, helper routines and context providers.
// MUST BE KEPT IN SYNC WITH ts/templates/isomorph.ts!
package templates

import (
	"fmt"
	"time"
	"sort"
	"strings"
	"strconv"
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"meguca/imager/assets"
	"meguca/lang"
)

type PostContext struct {
	ID uint64
	TID uint64
	Index bool
	OP bool
	Badge bool
	Board string
	Subject string
	Staff bool
	Auth string
	Time string
	post common.Post
	backlinks common.Backlinks
}

type FileContext struct {
	HasArtist bool
	Artist string
	HasTitle bool
	LCopy string
	Title string
	HasVideo bool
	HasAudio bool
	Length string
	Size string
	TWidth uint16
	THeight uint16
	Width uint16
	Height uint16
	SourcePath string
	ThumbPath string
}

type PostLinkContext struct {
	ID string
	URL string
	Cross bool
}

type BacklinksContext struct {
	LReplies string
	Backlinks []string
}

func MakePostContext(t common.Thread, p common.Post, bls common.Backlinks, index bool, all bool) PostContext {
	ln := lang.Get()
	postTime := time.Unix(p.Time, 0)
	return PostContext{
		ID: p.ID,
		TID: t.ID,
		Index: index,
		OP: t.ID == p.ID,
		Badge: t.ID == p.ID && index && all,
		Board: t.Board,
		Subject: t.Subject,
		Staff: p.Auth != "",
		Auth: ln.Common.Posts[p.Auth],
		Time: readableTime(postTime),
		post: p,
		backlinks: bls,
	}
}

func (ctx PostContext) Render() string {
	return renderMustache("post", &ctx)
}

func (ctx *PostContext) PostClass() string {
	classes := []string{"post"}
	if ctx.OP {
		classes = append(classes, "post_op")
	}
	if ctx.post.Image != nil {
		classes = append(classes, "post_file")
	}
	return strings.Join(classes, " ")
}

func (ctx *PostContext) URL() (url string) {
	if !ctx.OP {
		url = fmt.Sprintf("#%d", ctx.ID)
	}
	if ctx.Index {
		url = fmt.Sprintf("/%s/%d%s", ctx.Board, ctx.TID, url)
	}
	return
}

// Stringify an int and left-pad to at least double digits.
func pad(buf []byte, i int) []byte {
	if i < 10 {
		buf = append(buf, '0')
	}
	return append(buf, strconv.Itoa(i)...)
}

func readableTime(t time.Time) string {
	ln := lang.Get().Common.Time
	year, m, day := t.Date()
	weekday := ln["week"][int(t.Weekday())]
	// Months are 1-indexed for some fucking reason.
	month := ln["calendar"][int(m)-1]

	// Premature optimization.
	buf := make([]byte, 0, 17+len(weekday)+len(month))
	buf = pad(buf, day)
	buf = append(buf, ' ')
	buf = append(buf, month...)
	buf = append(buf, ' ')
	buf = append(buf, strconv.Itoa(year)...)
	buf = append(buf, " ("...)
	buf = append(buf, weekday...)
	buf = append(buf, ") "...)
	buf = pad(buf, t.Hour())
	buf = append(buf, ':')
	buf = pad(buf, t.Minute())

	return string(buf)
}

func duration(l uint32) string {
	return fmt.Sprintf("%02d:%02d", l / 60, l % 60)
}

// Formats a human-readable representation of file size.
func fileSize(size int) string {
	sizes := lang.Get().Common.Sizes
	switch {
	case size < 1024:
		return fmt.Sprintf("%d%s", size, sizes["b"])
	case size < 1024 * 1024:
		return fmt.Sprintf("%.2f%s", float32(size) / 1024, sizes["kb"])
	default:
		return fmt.Sprintf("%.2f%s", float32(size) / 1024 / 1024, sizes["mb"])
	}
}

func (ctx *PostContext) File() string {
	if ctx.post.Image == nil {
		return ""
	}
	ln := lang.Get()
	img := ctx.post.Image
	fileCtx := FileContext{
		HasArtist: img.Artist != "",
		Artist: img.Artist,
		HasTitle: img.Title != "",
		LCopy: ln.Common.Posts["clickToCopy"],
		Title: img.Title,
		HasVideo: img.Video,
		HasAudio: img.Audio,
		Length: duration(img.Length),
		Size: fileSize(img.Size),
		Width: img.Dims[0],
		Height: img.Dims[1],
		TWidth: img.Dims[2],
		THeight: img.Dims[3],
		SourcePath: assets.SourcePath(img.FileType, img.SHA1),
		ThumbPath: assets.ThumbPath(img.ThumbType, img.SHA1),
	}
	return renderMustache("post-file", &fileCtx)
}

func (ctx *PostContext) Body() string {
	return renderBody(ctx.post, ctx.TID, ctx.Index)
}

// Render a link to another post. Can optionally be cross-thread.
func renderPostLink(id uint64, cross, index bool) string {
	idStr := strconv.FormatUint(id, 10)
	url := ""
	if cross || index {
		url += "/all/" + idStr
	}
	url += "#" + idStr
	linkCtx := PostLinkContext{
		ID: idStr,
		URL: url,
		Cross: cross,
	}
	return renderMustache("post-link", &linkCtx)
}

func (ctx *PostContext) Backlinks() string {
	links := ctx.backlinks[ctx.ID]
	if links == nil {
		return ""
	}

	// Backlink ids always grow.
	ids := make(sortableUInt64, len(links))
	i := 0
	for id := range links {
		ids[i] = id
		i++
	}
	sort.Sort(ids)

	rendered := make([]string, len(links))
	for i, id := range ids {
		op := links[id]
		rendered[i] = renderPostLink(id, op != ctx.TID, ctx.Index)
	}

	ln := lang.Get()
	linkCtx := BacklinksContext{
		LReplies: ln.Common.UI["replies"],
		Backlinks: rendered,
	}
	return renderMustache("post-backlinks", &linkCtx)
}

func getPluralFormIndex(langCode string, n int) int {
	switch langCode {
	case "ru":
		if n%10==1 && n%100!=11 {
			return 0
		} else if n%10>=2 && n%10<=4 && (n%100<10 || n%100>=20) {
			return 1
		} else {
			return 2
		}
	default:
		if n == 1 {
			return 0
		} else {
			return 1
		}
	}
}

// Return either the singular or plural form of a translation, depending on
// number
func pluralize(num int, plurals []string) string {
	langCode := config.Get().DefaultLang
	return plurals[getPluralFormIndex(langCode, num)]
}

// Return pluar form for two numbers.
func pluralize2(n1, n2 int, plurals []string) string {
	if n1 + n2 == 1 {
		return plurals[0]
	} else {
		return plurals[1]
	}
}

func posClass(pos auth.ModerationLevel) string {
	posStr := pos.String()
	if posStr == "" {
		posStr = "notstaff"
	}
	return "pos-" + posStr
}
