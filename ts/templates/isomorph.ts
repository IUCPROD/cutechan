// MUST BE KEPT IN SYNC WITH go/src/meguca/templates/isomorph.go!
// Template structs, helper routines and context providers.

import * as Mustache from "mustache"
import templates from "templates"
import { ln, lang } from "../lang"
import { config } from "../state"
import { Thread, Post, thumbPath, sourcePath, parseBody } from "../posts"
import { Dict, makeEl, pad } from "../util"

export class TemplateContext {
	private template: string
	private ctx: Dict

	constructor(name: string, ctx: Dict) {
		this.template = templates[name]
		this.ctx = ctx
	}

	render(): string {
		return Mustache.render(this.template, this.ctx)
	}

	renderNode(): HTMLElement {
		return makeEl(this.render())
	}
}

// TODO(Kagami): Capitalize model keys for better compatibility?
export function makePostContext(t: Thread, p: Post): TemplateContext {
	const ctx: Dict = {
		ID: p.id,
		TID: t.id,
		Index: false,  // Currently we render only in-thread posts on the client
		OP: t.id == p.id,
		Badge: false,
		Board: p.board,
		Subject: p.subject,
		Staff: !!p.auth,
		Auth: ln.Common.Posts[p.auth],
		backlinks: null,
		post: p,
	}

	ctx.PostClass = () => {
		const classes = ["post"]
		if (ctx.OP) {
			classes.push("post_op")
		}
		if (ctx.post.image) {
			classes.push("post_file")
		}
		return classes.join(" ")
	}

	ctx.URL = () => {
		let url = ""
		if (!ctx.OP) {
			url = `#${ctx.ID}`
		}
		if (ctx.Index) {
			url = `/${ctx.Board}/${ctx.TID}${url}`
		}
		return url
	}

	// NOOP because we need to re-render based on relativeTime setting.
	ctx.Time = ""

	ctx.File = () => {
		const img = p.image
		if (!img) return ""
		return new TemplateContext("post-file", {
			HasArtist: !!img.artist,
			Artist: img.artist,
			HasTitle: !!img.title,
			LCopy: ln.Common.Posts["clickToCopy"],
			Title: img.title,
			HasVideo: img.video,
			HasAudio: img.audio,
			Length: duration(img.length),
			Size: fileSize(img.size),
			Width: img.dims[0],
			Height: img.dims[1],
			TWidth: img.dims[2],
			THeight: img.dims[3],
			SourcePath: sourcePath(img.fileType, img.SHA1),
			ThumbPath: thumbPath(img.thumbType, img.SHA1),
		}).render()
	}

	ctx.Body = parseBody(p)

	// NOOP because we will need to update already rendered posts so avoid
	// code duplication.
	ctx.Backlinks = ""

	return new TemplateContext("post", ctx)
}

const PLURAL_FORMS: { [key: string]: (n: number) => number } = {
	default: n => n == 1 ? 0 : 1,
	ru: n => n%10==1 && n%100!=11 ? 0 : n%10>=2 && n%10<=4 && (n%100<10 || n%100>=20) ? 1 : 2,
}

// Return pluralize form for various languages.
export function pluralize(num: number, plurals: [string]): string {
	const getForm = PLURAL_FORMS[config.defaultLang] || PLURAL_FORMS.default
	return plurals[getForm(num)]
}

// Renders classic absolute timestamp.
export function readableTime(time: number): string {
	const d = new Date(time * 1000)
	return `${pad(d.getDate())} ${lang.time.calendar[d.getMonth()]} `
		+ `${d.getFullYear()} (${lang.time.week[d.getDay()]}) `
		+ `${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function duration(l: number): string {
  return pad(Math.floor(l / 60)) + ":" + pad(Math.floor(l % 60))
}

// Formats a human-readable representation of file size.
export function fileSize(size: number): string {
	const sizes = ln.Common.Sizes
	if (size < 1024) {
		return size + sizes["b"]
	} else if (size < 1024 * 1024) {
		return (size / 1024).toFixed(2) + sizes["kb"]
	} else {
		return (size / 1024 / 1024).toFixed(2) + sizes["mb"]
	}
}
