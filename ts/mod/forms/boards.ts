import { View, ViewAttrs } from "../../base"
import { makeFrag, postJSON, uncachedGET } from "../../util"
import { AccountForm } from "./common"
import { loginID } from "../common"

// Render the <select> for picking the owned board you want to manipulate
class OwnedBoardSelection extends View<null> {
  private parent: SelectedBoardForm

  constructor(parent: SelectedBoardForm) {
    super({ tag: "form" })
    this.parent = parent
    this.on("submit", e =>
      this.onSubmit(e))
    this.render()
  }

  private async render() {
    const res = await uncachedGET(`/html/owned-boards/${loginID()}`)
    switch (res.status) {
      case 200:
        this.el.append(makeFrag(await res.text()))
        this.parent.el.append(this.el)
        break
      case 403:
        this.parent.handle403()
        break
      default:
        throw await res.text()
    }
  }

  private onSubmit(e: Event) {
    e.preventDefault()
    e.stopPropagation()
    const board = ((e.target as Element)
      .querySelector("select") as HTMLInputElement)
      .value
    this.parent.renderNext(board)
    this.parent.board = board
    this.remove()
  }
}

// View that performs an action on a selected board
abstract class SelectedBoardForm extends AccountForm {
  public board: string
  protected boardSelector: OwnedBoardSelection

  public abstract renderNext(board: string): void

  constructor(attrs: ViewAttrs) {
    attrs.tag = "form"
    super(attrs)
    this.boardSelector = new OwnedBoardSelection(this)
    this.render()
  }
}

// Board configuration panel
export class BoardConfigForm extends SelectedBoardForm {
  constructor() {
    super({ class: "wide-fields" })
  }

  // Render the configuration input elements
  public async renderNext(board: string) {
    const res = await postJSON(`/html/configure-board/${board}`, null)
    switch (res.status) {
      case 200:
        const frag = makeFrag(await res.text())
        this.el.append(frag)
        this.initCaptcha()
        break
      case 403:
        this.handle403()
        break
      default:
        throw await res.text()
    }
  }

  // Extract form data and send a request to apply the new configs
  protected send() {
    this.postResponse(`/api/configure-board/${this.board}`, req =>
      this.extractForm(req))
  }
}

export class BoardDeletionForm extends SelectedBoardForm {
  constructor() {
    super({})
  }

  public renderNext(board: string) {
    this.renderPublicForm("/html/captcha")
  }

  protected send() {
    this.postResponse("/api/delete-board", req =>
      req["board"] = this.board)
  }
}

export class StaffAssignmentForm extends SelectedBoardForm {
  constructor() {
    super({ class: "divide-rows" })
  }

  public renderNext(board: string) {
    this.renderPublicForm(`/html/assign-staff/${board}`)
  }

  protected send() {
    this.postResponse("/api/assign-staff", req => {
      req["board"] = this.board
      this.extractForm(req)
    })
  }
}

export class BannerForm extends SelectedBoardForm {
  constructor() {
    super({})
  }

  public el: HTMLFormElement

  public renderNext(board: string) {
    this.renderPublicForm("/html/set-banners")
  }

  protected async send() {
    const data = new FormData(this.el)
    data.append("board", this.board)
    if (this.captcha) {
      const c = this.captcha.data()
      for (let k in c) {
        data.append(k, c[k])
      }
    }

    this.handlePostResponse(await fetch("/api/set-banners", {
      method: "POST",
      credentials: "include",
      body: data,
    }))
  }
}

// Panel view for creating boards
export class BoardCreationForm extends AccountForm {
  constructor() {
    super({ tag: "form" })
    this.renderPublicForm("/html/create-board")
  }

  protected send() {
    this.postResponse("/api/create-board", req => {
      req["id"] = this.inputElement('boardName').value
      req["title"] = this.inputElement('boardTitle').value
    })
  }
}
