import PostView from "../view"
import FormModel from "./model"
import { Post } from "../model"
import { setAttrs, importTemplate, firstChild } from "../../util"
import { postSM, postEvent } from "."
import UploadForm from "./upload"
import { CaptchaView } from "../../ui"
import { message, send } from "../../connection"
import { isStaff, getMyAuth } from "../../mod";

// Element at the bottom of the thread to keep the fixed reply form from
// overlapping any other posts, when scrolled till bottom
let bottomSpacer: HTMLElement

// Post creation and update view
export default class FormView extends PostView {
  public model: FormModel
  private input: HTMLTextAreaElement
  private observer: MutationObserver
  private previousHeight: number
  public upload: UploadForm
  public captcha: CaptchaView

  constructor(model: Post) {
    super(model, null)
    this.renderInputs()
    this.initDraft()
  }

  // Render extra input fields for inputting text and optionally
  // uploading images.
  private renderInputs() {
    this.input = document.createElement("textarea")
    setAttrs(this.input, {
      id: "text-input",
      name: "body",
      rows: "1",
      maxlength: "2000",
    })
    this.el.append(importTemplate("post-controls"))
    this.resizeInput()

    this.input.addEventListener("input", e => {
      e.stopImmediatePropagation()
      this.onInput()
    })
    this.onClick({
      "input[name=\"done\"]": postSM.feeder(postEvent.done),
      "input[name=\"cancel\"]": postSM.feeder(postEvent.done),
    })

    this.upload = new UploadForm(this.model, this.el)
    this.upload.input.addEventListener("change", () =>
      this.model.uploadFile())
    this.inputElement("done").hidden = !this.model.nonLive

    const bq = this.el.querySelector("blockquote")
    bq.innerHTML = ""
    bq.append(this.input)

    const captcha = this.el.querySelector(".antispam-captcha")
    if (this.model.needCaptcha) {
      if (captcha) {
        this.renderCaptcha(captcha)
      } else {
        // Page's captcha setting has desynced from the server
        location.reload(true)
      }
    } else {
      if (captcha) {
        captcha.style.display = "none"
      }
      requestAnimationFrame(() =>
        this.input.focus())
    }

    if (isStaff()) {
      this.renderStaffControl()
    }
  }

  private renderStaffControl() {
    const staffEl = this.el.querySelector("[name=staffTitle]") as HTMLInputElement
    const onStaffChange = () => {
      localStorage.setItem("staffTitle", staffEl.checked.toString())
      this.model.auth = staffEl.checked ? getMyAuth() : ""
      // this.renderName()
    }
    staffEl.addEventListener("change", onStaffChange)
    staffEl.checked = localStorage.getItem("staffTitle") === "true"
    if (staffEl.checked) {
      onStaffChange()
    }
  }

  // Request a captcha to be filled out, before the post is submitted
  private renderCaptcha(el: HTMLElement) {
    const cont = el.querySelector(".captcha-container")
    this.captcha = new CaptchaView(cont)

    // Hide all other post controls till the captcha is submitted
    const controls = [
      this.el.querySelector(".post-container"),
      this.el.querySelector(".post-controls"),
    ]
    for (const c of controls) {
      c.style.display = "none"
    }

    el.addEventListener("submit", e => {
      e.preventDefault()
      e.stopImmediatePropagation()

      send(message.captcha, this.captcha.data())

      el.remove()
      for (const c of controls) {
        c.style.display = ""
      }
      this.input.focus()
      postSM.feed(postEvent.captchaSolved)
    })

    requestAnimationFrame(() =>
      (cont
        .querySelector("input[type=number]") as HTMLElement)
        .focus())
  }

  // Show button for closing allocated posts
  private showDone() {
    const c = firstChild(this.el.querySelector(".post-controls"), ch =>
      ch.getAttribute("name") === "cancel")
    if (c) {
      c.remove()
    }
    const d = this.inputElement("done")
    if (d) {
      d.hidden = false
    }
  }

  // Initialize extra elements for a draft unallocated post
  private initDraft() {
    bottomSpacer = document.getElementById("bottom-spacer")
    this.el.classList.add("reply-form")
    this.el.querySelector("header").classList.add("temporary")

    // Keep this post and bottomSpacer the same height
    this.observer = new MutationObserver(() =>
      this.resizeSpacer())
    this.observer.observe(this.el, {
      childList: true,
      attributes: true,
      characterData: true,
      subtree: true,
    })

    document.getElementById("thread-container").append(this.el)
    this.resizeSpacer()
    // this.setEditing(false)
  }

  // Resize bottomSpacer to the same top position as this post
  private resizeSpacer() {
    // Not a reply
    if (!bottomSpacer) {
      return
    }

    const { height } = this.el.getBoundingClientRect()
    // Avoid needless writes
    if (this.previousHeight === height) {
      return
    }
    this.previousHeight = height
    bottomSpacer.style.height = `calc(${height}px - 2.1em)`
  }

  private removeUploadForm() {
    this.upload.input.remove()
    this.upload.status.remove()
  }

  // Handle input events on this.input
  public onInput() {
    if (!this.input) {
      return
    }
    this.resizeInput()
    this.model.parseInput(this.input.value)
  }

  // Resize textarea to content width and adjust height
  private resizeInput() {
    const el = this.input,
      s = el.style
    s.width = "0px"
    s.height = "0px"
    el.wrap = "off"
    // Make the line slightly larger, so there is enough space for the next
    // character. This prevents wrapping on type.
    s.width = Math.max(260, el.scrollWidth + 5) + "px"
    el.wrap = "soft"
    s.height = Math.max(16, el.scrollHeight) + "px"
  }

  // Trim input from the end by the supplied length
  public trimInput(length: number) {
    this.input.value = this.input.value.slice(0, -length)
  }

  // Replace the current body and set the cursor to the input's end.
  // commit sets, if the onInput method should be run.
  public replaceText(body: string, commit: boolean) {
    const el = this.input
    el.value = body
    if (commit) {
      this.onInput()
    } else {
      this.resizeInput()
    }
    requestAnimationFrame(() => {
      el.focus()
      el.setSelectionRange(body.length, body.length)

      // Because Firefox refocuses the clicked <a>
      requestAnimationFrame(() =>
        el.focus())
    })
  }

  // Transform form into a generic post. Removes any dangling form controls
  // and frees up references.
  public cleanUp() {
    if (this.upload && this.upload.isUploading) {
      this.upload.cancel()
    }
    this.el.classList.remove("reply-form")
    const pc = this.el.querySelector(".post-controls")
    if (pc) {
      pc.remove()
    }
    if (bottomSpacer) {
      bottomSpacer.style.height = ""
      // if (atBottom) {
      //   scrollToBottom()
      // }
    }
    if (this.observer) {
      this.observer.disconnect()
    }
    bottomSpacer
      = this.observer
      = this.upload
      = null
  }

  // Clean up on form removal
  public remove() {
    super.remove()
    this.cleanUp()
  }

  // Lock the post form after a critical error occurs
  public renderError() {
    this.el.classList.add("errored")
    this.input.setAttribute("contenteditable", "false")
  }

  // Transition into allocated post
  public renderAlloc() {
    this.id = this.el.id = `post${this.model.id}`
    this.el.querySelector("header").classList.remove("temporary")
    // this.renderHeader()
    this.showDone()
  }

  // Insert image into an open post
  public insertImage() {
    // this.renderImage(false)
    this.resizeInput()
    this.removeUploadForm()
  }
}
