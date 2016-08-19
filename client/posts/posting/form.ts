// Facilities for creating and updating open posts

import PostView, {OPView} from '../view'
import {Post, OP, ThreadData, TextState} from '../models'
import {SpliceResponse} from '../../client'
import {applyMixins, makeFrag, setAttrs} from '../../util'
import {posts, isMobile} from '../../state'
import {parseTerminatedLine} from '../render/body'
import {write} from '../../render'
import {ui} from '../../lang'
import {send, message} from "../../connection"

// Current PostForm and model instances
export let postForm: FormView
export let postModel: FormModel

// Form Model of an OP post
export class OPFormModel extends OP implements FormModel {
	bodyLength: number
	parsedLines: number
	view: FormView
	inputState: TextState

	commitChar: (char: string) => void
	commitBackspace: () => void
	commitSplice: (val: string) => void
	init: () => void
	lastBodyLine: () => string
	parseInput: (val: string) => void

	constructor(id: number) {

		// TODO: Persist id to state.mine

		const oldModel = posts.get(id) as OP,
			oldView = oldModel.view
		oldView.unbind()

		// Copy the parent model's state and data
		super(extractAttrs(oldModel) as ThreadData)

		// Replace old model and view pair with the postForm pair
		posts.addOP(this)
		postForm = new OPFormView(this)
		oldView.el.replaceWith(postForm.el)

		this.init()

		// TODO: Hide [Reply] button

	}
}

// Override mixin for post authoring models
class FormModel {
	bodyLength: number = 0 // Compound length of the input text body
	parsedLines: number = 0 // Number of closed, commited and parsed lines
	body: string
	view: PostView & FormView

	// State of line being edditted. Must be seperated to not affect the
	// asynchronous updates of commited lines
	inputState: TextState

	spliceLine: (line: string, msg: SpliceResponse) => string
	resetState: () => void

	// Initialize state
	init() {
		this.bodyLength = this.parsedLines = 0
		this.inputState = {
			quote: false,
			spoiler: false,
			iDice: 0, // Not used in FormModel. TypeScipt demands it.
			line: "",
		}
		postModel = this
	}

	// Append a character to the model's body and reparse the line, if it's a
	// newline
	append(code: number) {
		const char = String.fromCharCode(code)
		if (char === "\n") {
			this.view.terminateLine(this.parsedLines++)
		}
		this.body += char
	}

	// Remove the last character from the model's body
	backspace() {
		this.body = this.body.slice(0, -1)
	}

	// Splice the last line of the body
	splice(msg: SpliceResponse) {
		this.spliceLine(this.lastBodyLine(), msg)
	}

	// Compare new value to old and generate apropriate commands
	parseInput(val: string): void {
		const old = this.inputState.line,
			lenDiff = val.length - old.length,
			exceeding = this.bodyLength + lenDiff - 2000

		// If exceeding max body lenght, shorten the value, trim $input and try
		// again
		if (exceeding > 0) {
			this.view.trimInput(exceeding)
			return this.parseInput(val.slice(0, -exceeding))
		}

		if (lenDiff === 1 && val.slice(0, -1) === old) {
			return this.commitChar(val.slice(-1))
		}
		if (lenDiff === -1 && old.slice(0, -1) === val) {
			return this.commitBackspace()
		}

		return this.commitSplice(val, lenDiff)
	}

	// Commit a character appendage to the end of the line to the server
	commitChar(char: string) {
		this.bodyLength++
		if (char === "\n") {
			this.resetState()
			this.view.startNewLine()
			this.inputState.line = ""
		} else {
			this.inputState.line += char
		}
		send(message.append, char.charCodeAt(0))
	}

	// Send a message about removing the last character of the line to the
	// server
	commitBackspace() {
		this.inputState.line = this.inputState.line.slice(0, -1)
		this.bodyLength--
		send(message.backspace, null)
	}

	// Commit any other $input change that is not an append or backspace
	commitSplice(val: string, lenDiff: number) {
		const old = this.inputState.line
		let start: number,
			len: number,
			text: string

		// Find first differing character
		for (let i = 0; i < old.length; i++) {
			if (old[i] !== val[i]) {
				start = i
				break
			}
		}

		// Find last common character and the differing part
		const maxLen = Math.max(old.length, val.length),
			vOffset = val.length - maxLen,
			oOffset = old.length - maxLen
		for (let i = maxLen; i >= start; i--) {
			if (old[i + oOffset] !== val[i + vOffset]) {
				len = i + oOffset - start + 1
				text = val.slice(start).slice(0, len - 1)
				break
			}
		}

		send(message.splice, {start, len, text})
		this.bodyLength += lenDiff
		this.inputState.line = val

		// If splice contained newlines, reformat text accordingly
		const lines = val.split("\n")
		if (lines.length > 1) {
			const lastLine = lines[lines.length - 1]
			this.view.injectLines(lines.slice(0, -1), lastLine)
			this.resetState()
			this.inputState.line = lastLine
		}
	}

	// Return the last line of the body
	lastBodyLine(): string {
		const lines = this.body.split("\n")
		return lines[lines.length - 1]
	}
}

applyMixins(OPFormModel, FormModel)

// Post creation and update view
class FormView extends PostView {
	model: Post & FormModel
	inputLock: boolean
	$input: HTMLSpanElement
	$done: HTMLInputElement
	$postControls: Element

	constructor(model: Post) {
		super(model)
		this.renderInputs()
	}

	// Render extra input fields for inputting text and optionally uploading
	// images
	renderInputs() {
		this.$input = document.createElement("span")
		const attrs: {[key: string]: string} = {
			id: "text-input",
			name: "body",
			contenteditable: "",
		}
		if (isMobile) {
			attrs["autocomplete"] = ""
		}
		setAttrs(this.$input, attrs)
		this.$input.textContent = ""
		this.$input.addEventListener("input", (event: Event) =>
			this.onInput((event.target as Element).textContent))
		this.$input.addEventListener("keydown", (event: KeyboardEvent) =>
			this.onKeyDown(event))

		this.$postControls = document.createElement("div")
		this.$postControls.id = "post-controls"

		this.$done = document.createElement("input")
		setAttrs(this.$done, {
			name: "done",
			type: "button",
			value: ui.done,
		})
		this.$postControls.append(this.$done)

		write(() => {
			this.$blockquote.innerHTML = ""
			this.$blockquote.append(this.$input)
			this.el.append(this.$postControls)
			this.$input.focus()
		})

		// TODO: UploadForm integrations

	}

	// Handle input events on $input
	onInput(val: string) {
		if (!this.inputLock) {
			this.model.parseInput(val)
		}
	}

	// Ignore any oninput events on $input during suplied function call
	lockInput(fn: () => void) {
		this.inputLock = true
		fn()
		this.inputLock = false
	}

	// Handle keydown events on $input
	onKeyDown(event: KeyboardEvent) {
		if (event.which === 13) { // Enter
			event.preventDefault()
			return this.onInput(this.model.inputState.line + "\n")
		}
	}

	// Trim $input from the end by the suplied length
	trimInput(length: number) {
		const val = this.$input.textContent.slice(0, -length)
		write(() =>
			this.lockInput(() =>
				this.$input.textContent = val))
	}

	// Start a new line in the input field and close the previous one
	startNewLine() {
		const line = this.model.inputState.line.slice(0, -1),
			frag = makeFrag(parseTerminatedLine(line, this.model))
		write(() => {
			this.$input.before(frag)
			this.lockInput(() =>
				this.$input.textContent = "")
		})
	}

	// Inject lines before $input and set $input contents to lastLine
	injectLines(lines: string[], lastLine: string) {
		const frag = document.createDocumentFragment()
		for (let line of lines) {
			const el = makeFrag(parseTerminatedLine(line, this.model))
			frag.append(el)
		}
		write(() => {
			this.$input.before(frag)
			this.lockInput(() =>
				this.$input.textContent = lastLine)
		})
	}

	// Parse and replace the temporary like closed by $input with a proper
	// parsed line
	terminateLine(num: number) {
		const html = parseTerminatedLine(this.model.lastBodyLine(), this.model),
			frag = makeFrag(html)
		write(() =>
			this.$blockquote.children[num].replaceWith(frag))
	}
}

// FormView of an OP post
class OPFormView extends FormView implements OPView {
	$omit: Element
	model: OP & FormModel
	renderOmit: () => void

	constructor(model: OP) {
		super(model)
	}
}

applyMixins(OPFormView, OPView)

// Extract all non-function attributes from a model
function extractAttrs(src: {[key: string]: any}): {[key: string]: any} {
	const attrs: {[key: string]: any} = {}
	for (let key in src) {
		if (typeof src[key] !== "function") {
			attrs[key] = src[key]
		}
	}
	return attrs
}
