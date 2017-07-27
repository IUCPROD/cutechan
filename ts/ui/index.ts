export { default as FormView } from "./forms"
export { postAdded } from "./tab"
export { default as notifyAboutReply, OverlayNotification } from "./notification"
export { default as CaptchaView } from "./captcha"
import { HeaderModal } from "../base"

import { init as initKeyboard } from "./keyboard"
import { init as initTab } from "./tab"
import OptionPanel from "./options"

export function init() {
  initKeyboard()
  initTab()
  new OptionPanel()
  new HeaderModal(document.getElementById("FAQ"))
}
