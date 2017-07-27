interface ChildNode {
	after(...nodes: (Node | string)[]): void
	before(...nodes: (Node | string)[]): void
	replaceWith(...nodes: (Node | string)[]): void
}

interface ParentNode {
	append(...nodes: (Node | string)[]): void
	prepend(...nodes: (Node | string)[]): void
}

interface NodeSelector {
	querySelector(sel: string): HTMLElement

	// Hack. Modern browsers have Symbol.iterator on NodeList
	querySelectorAll(sel: string): HTMLElement[]
}

interface EventTarget {
	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean | EventListenerOptions
	): void
}

interface EventListenerOptions {
	capture?: boolean
	once?: boolean
	passive?: boolean
}

interface Element extends ChildNode, ParentNode {
	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean | EventListenerOptions
	): void
}

interface HTMLElement {
	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean | EventListenerOptions
	): void
}

interface HTMLInputElement {
	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean | EventListenerOptions
	): void
}

interface HTMLAnchorElement {
	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean | EventListenerOptions
	): void
}

interface Window {
	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean | EventListenerOptions
	): void
}

interface Node extends ChildNode, ParentNode {
	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean | EventListenerOptions
	): void
}

interface Document extends ChildNode, ParentNode {
	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean | EventListenerOptions
	): void
}

interface History {
	scrollRestoration: "auto" | "manual"
}

interface ArrayBufferTarget extends EventTarget {
	result: ArrayBuffer
}

interface ArrayBufferLoadEvent extends Event {
	target: ArrayBufferTarget
}

interface NotificationOptions {
	vibrate?: boolean
}

interface Array<T> {
	includes(item: T): boolean
}
