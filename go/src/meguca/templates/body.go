package templates

import (
	"html"
	"meguca/common"
	"meguca/util"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/valyala/quicktemplate"
)

// Embeddable URL types
const (
	youTube = iota
	soundCloud
	vimeo
)

var (
	linkRegexp      = regexp.MustCompile(`^>>(>*)(\d+)$`)

	providers = map[int]string{
		youTube:    "Youtube",
		soundCloud: "SoundCloud",
		vimeo:      "Vimeo",
	}
	embedPatterns = [...]struct {
		typ  int
		patt *regexp.Regexp
	}{
		{
			youTube,
			regexp.MustCompile(`https?:\/\/(?:[^\.]+\.)?youtube\.com\/watch\/?\?(?:.+&)?v=([^&]+)`),
		},
		{
			youTube,
			regexp.MustCompile(`https?:\/\/(?:[^\.]+\.)?(?:youtu\.be|youtube\.com\/embed)\/([a-zA-Z0-9_-]+)`),
		},
		{
			soundCloud,
			regexp.MustCompile(`https?:\/\/soundcloud.com\/.*`),
		},
		{
			vimeo,
			regexp.MustCompile(`https?:\/\/(?:www\.)?vimeo\.com\/.+`),
		},
	}

	// URLs supported for linkification
	urlPrefixes = map[byte]string{
		'h': "http",
		'm': "magnet:?",
		'i': "irc",
		'f': "ftp",
		'b': "bitcoin",
	}
)

type bodyContext struct {
	index bool     // Rendered for an index page
	state struct { // Body parser state
		spoiler, quote, lastLineEmpty, code bool
	}
	common.Post
	OP uint64
	quicktemplate.Writer
}

// Render the text body of a post
func streambody(w *quicktemplate.Writer, p common.Post, op uint64, index bool) {
	c := bodyContext{
		index:  index,
		Post:   p,
		OP:     op,
		Writer: *w,
	}

	var fn func(string)
	if c.Editing {
		fn = c.parseOpenLine
	} else {
		fn = c.parseTerminatedLine
	}

	lines := strings.Split(c.Body, "\n")
	last := len(lines) - 1
	for i, l := range lines {
		// Prevent successive empty lines
		if len(l) == 0 {
			// Don't break, if body ends with newline
			if !c.state.lastLineEmpty && i != last {
				c.string("<br>")
			}
			c.state.lastLineEmpty = true
			c.state.quote = false
			continue
		}
		c.state.lastLineEmpty = false

		c.initLine(l[0])
		fn(l)
		c.terminateTags(i != last)
	}
}

// Write string without escaping
func (c *bodyContext) string(s string) {
	c.N().S(s)
}

// Escape and write string
func (c *bodyContext) escape(s string) {
	c.E().S(s)
}

// Write a byte without heap allocations or escaping
func (c *bodyContext) byte(b byte) {
	buf := [1]byte{b}
	c.N().SZ(buf[:])
}

// Parse a line that is no longer being edited
func (c *bodyContext) parseTerminatedLine(line string) {
	c.parseCode(line, (*c).parseFragment)
}

// Open a new line container and check for quotes
func (c *bodyContext) initLine(first byte) {
	c.state.quote = false
	c.state.lastLineEmpty = false
	if first == '>' {
		c.string("<em>")
		c.state.quote = true
	}
	if c.state.spoiler {
		c.string("<del>")
	}
}

// Detect code tags
func (c *bodyContext) parseCode(frag string, fn func(string)) {
	for {
		i := strings.Index(frag, "``")
		if i != -1 {
			c.formatCode(frag[:i], fn)
			frag = frag[i+2:]
			c.state.code = !c.state.code
		} else {
			c.formatCode(frag, fn)
			break
		}
	}
}

func (c *bodyContext) formatCode(frag string, fn func(string)) {
	if c.state.code {
		// Strip quotes
		for len(frag) != 0 && frag[0] == '>' {
			c.string(`&gt;`)
			frag = frag[1:]
		}
		c.N().Z(highlightSyntax(frag))
	} else {
		c.parseSpoilers(frag, fn)
	}
}

// Injects spoiler tags and calls fn on the remaining parts
func (c *bodyContext) parseSpoilers(frag string, fn func(string)) {
	for {
		i := strings.Index(frag, "**")
		if i != -1 {
			fn(frag[:i])
			if c.state.quote {
				c.string("</em>")
			}
			if c.state.spoiler {
				c.string("</del>")
			} else {
				c.string("<del>")
			}
			if c.state.quote {
				c.string("<em>")
			}

			c.state.spoiler = !c.state.spoiler
			frag = frag[i+2:]
		} else {
			fn(frag)
			break
		}
	}
}

// Parse a line fragment
func (c *bodyContext) parseFragment(frag string) {
	// Leading and trailing punctuation, if any
	var leadPunct, trailPunct byte

	for i, word := range strings.Split(frag, " ") {
		if i != 0 {
			c.byte(' ')
		}

		// Strip leading and trailing punctuation and commit separately
		leadPunct, word, trailPunct = util.SplitPunctuationString(word)
		if leadPunct != 0 {
			c.byte(leadPunct)
		}

		if len(word) == 0 {
			goto end
		}
		switch word[0] {
		case '>': // Links
			if m := linkRegexp.FindStringSubmatch(word); m != nil {
				// Post links
				c.parsePostLink(m)
				goto end
			}
		default: // Generic HTTP(S) URLs and magnet links
			// Checking the first byte is much cheaper than a function call. Do
			// that first, as most cases won't match.
			pre, ok := urlPrefixes[word[0]]
			if ok && strings.HasPrefix(word, pre) {
				c.parseURL(word)
				goto end
			}
		}
		c.escape(word)

	end:
		// Write trailing punctuation, if any
		if trailPunct != 0 {
			c.byte(trailPunct)
		}
	}
}

// Parse a potential link to a post
func (c *bodyContext) parsePostLink(m []string) {
	if c.Links == nil {
		c.escape(m[0])
		return
	}

	id, _ := strconv.ParseUint(string(m[2]), 10, 64)
	var op uint64
	for _, l := range c.Links {
		if l[0] == id {
			op = l[1]
			break
		}
	}
	if op == 0 {
		c.escape(m[0])
		return
	}

	if len(m[1]) != 0 { // Write extra quotes
		c.escape(m[1])
	}
	c.string(postLink(id, op != c.OP, c.index))
}

// Format and anchor link that opens in a new tab
func (c *bodyContext) newTabLink(href, text string) {
	c.string(`<a rel="noreferrer" href="`)
	c.escape(href)
	c.string(`" target="_blank">`)
	c.escape(text)
	c.string(`</a>`)
}

// Parse generic URLs and magnet links
func (c *bodyContext) parseURL(bit string) {
	s := string(bit)
	u, err := url.Parse(s)
	switch {
	case err != nil || u.Path == s: // Invalid or empty path
		c.escape(bit)
	case c.parseEmbeds(bit):
	case bit[0] == 'm': // Don't open a new tab for magnet links
		s = html.EscapeString(s)
		c.string(`<a rel="noreferrer" href="`)
		c.string(s)
		c.string(`">`)
		c.string(s)
		c.string(`</a>`)
	default:
		c.newTabLink(s, s)
	}
}

// Parse select embeddable URLs. Returns, if any found.
func (c *bodyContext) parseEmbeds(s string) bool {
	for _, t := range embedPatterns {
		if !t.patt.MatchString(s) {
			continue
		}

		c.string(`<em><a rel="noreferrer" class="embed" target="_blank" data-type="`)
		c.N().D(t.typ)
		c.string(`" href="`)
		c.escape(s)
		c.string(`">[`)
		c.string(providers[t.typ])
		c.string(`] ???</a></em>`)

		return true
	}
	return false
}

func (c *bodyContext) uint64(i uint64) {
	c.string(strconv.FormatUint(i, 10))
}

// Close any open HTML tags
func (c *bodyContext) terminateTags(newLine bool) {
	if c.state.spoiler {
		c.string("</del>")
	}
	if c.state.quote {
		c.string("</em>")
	}
	if newLine {
		c.string("<br>")
	}
}

// Parse a line that is still being edited
func (c *bodyContext) parseOpenLine(line string) {
	c.parseCode(line, func(s string) {
		c.escape(s)
	})
}
