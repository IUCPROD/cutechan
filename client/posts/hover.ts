/**
 * Post and image hover previews.
 */

import API from "../api"
import options from "../options"
import { posts } from "../state"
import { View } from "../base"
import { Post } from "./model"
import PostView from "./view"
import * as popup from "./popup"
import {
	POST_LINK_SEL, POST_FILE_LINK_SEL, POST_FILE_THUMB_SEL,
	POST_HOVER_TIMEOUT_SECS,
} from "../vars"
import {
	getID, getClosestID,
	emitChanges, ChangeEmitter, HOOKS, hook,
} from "../util"

interface MouseMove extends ChangeEmitter {
	event: MouseEvent
}

const overlay = document.getElementById("hover-overlay")

// Currently displayed previews, if any.
const postPreviews = [] as [PostPreview]
let imagePreview = null as HTMLElement

let clearPostTID = 0

// Centralized mousemove target tracking.
const mouseMove = emitChanges<MouseMove>({
	event: {
		target: null,
	},
} as MouseMove)

// Clone a post element as a preview.
// TODO(Kagami): Render mustache template instead?
function clonePost(el: HTMLElement): HTMLElement {
	const preview = el.cloneNode(true) as HTMLElement
	preview.removeAttribute("id")
	preview.classList.add("post_hover")
	return preview
}

// Post hover preview view.
class PostPreview extends View<Post> {
	public el: HTMLElement
	public parent: HTMLElement

	constructor(model: Post, parent: HTMLElement) {
		const { el } = model.view
		super({el: clonePost(el)})
		this.parent = parent
		this.model = Object.assign({}, model)
		this.render()
		parent.addEventListener("click", clearPostPreviews)
	}

	private render() {
		// Underline reverse post links in preview.
		const re = new RegExp("[>\/]" + getClosestID(this.parent))
		for (const el of this.el.querySelectorAll(POST_LINK_SEL)) {
			if (re.test(el.textContent)) {
				el.classList.add("post-link_ref")
			}
		}
		overlay.append(this.el)
		this.position()
	}

	// Position the preview element relative to it's parent link.
	private position() {
		const rect = this.parent.getBoundingClientRect()

		// The preview will never take up more than 100% screen width, so no
		// need for checking horizontal overflow. Must be applied before
		// reading the height, so it takes into account post resizing to
		// viewport edge.
		this.el.style.left = rect.left + "px"

		const height = this.el.offsetHeight
		let top = rect.top - height - 5

		// If post gets cut off at the top, put it bellow the link.
		if (top < 0) {
			top += height + 23
		}
		this.el.style.top = top + "px"
	}

	// Remove this view.
	public remove() {
		this.parent.removeEventListener("click", clearPostPreviews)
		super.remove()
	}
}

async function renderPostPreview(event: MouseEvent) {
	const target = event.target as HTMLElement
	if (!target.matches || !target.matches(POST_LINK_SEL)) {
		if (postPreviews.length && !clearPostTID) {
			clearPostTID = window.setTimeout(
				clearInactivePostPreviews,
				POST_HOVER_TIMEOUT_SECS * 1000
			)
		}
		return
	}

	const id = getID(target)
	if (!id) return

	let post = posts.get(id)
	if (!post) {
		// Fetch from server, if this post is not currently displayed due to
		// lastN or in a different thread.
		const data = await API.post.get(id)
		post = new Post(data)
		new PostView(post, null)
	}

	const preview = new PostPreview(post, target)
	postPreviews.push(preview)
}

function renderImagePreview(event: MouseEvent) {
	if (!options.imageHover) return
	if (popup.isOpen()) return

	const target = event.target as HTMLElement
	if (!target.matches || !target.matches(POST_FILE_THUMB_SEL)) return

	const link = target.closest(POST_FILE_LINK_SEL)
	if (!link) return
	const src = link.getAttribute("href")
	const ext = src.slice(src.lastIndexOf(".") + 1)

	if (ext === "jpg" || ext === "png" || ext === "gif") {
		const el = document.createElement("img")
		el.src = src
		imagePreview = el
		overlay.append(el)
	}
}

function clearInactivePostPreviews() {
	clearPostTID = 0
	const target = mouseMove.event.target as HTMLElement
	for (let i = postPreviews.length - 1; i >= 0; i--) {
		const preview = postPreviews[i]
		if (target === preview.parent || preview.el.contains(target)) return
		postPreviews.pop().remove()
	}
}

function clearPostPreviews() {
	let preview
	while (preview = postPreviews.pop()) {
		preview.remove()
	}
}

function clearImagePreview() {
	if (imagePreview) {
		imagePreview.remove()
		imagePreview = null
	}
}

function clearPreviews() {
	clearPostPreviews()
	clearImagePreview()
}

function onMouseMove(event: MouseEvent) {
	if (event.target !== mouseMove.event.target) {
		clearImagePreview()
		mouseMove.event = event
	}
}

export function init() {
	document.addEventListener("mousemove", onMouseMove, {passive: true})
	mouseMove.onChange("event", renderPostPreview)
	mouseMove.onChange("event", renderImagePreview)
	hook(HOOKS.openPostPopup, clearPreviews)
}
