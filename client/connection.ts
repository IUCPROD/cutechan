// Handles Websocket connectivity and messaging

import FSM from './fsm'
import { debug, page } from './state'
import { sync as lang } from './lang'
import { write } from './render'
import { authenticate } from './mod/login'
import { PostData } from "./posts/models"
import { insertPost } from "./client"
import { fetchTime, refetch } from "./page/thread"

// A reqeust message to synchronise or resynchronise (after a connection loss)
// to the server
type SyncRequest = {
	board: string
	thread: number
	id?: string
}

// Message types of the WebSocket communication protocol
export const enum message {
	invalid,

	// 1 - 29 modify post model state
	insertThread,
	insertPost,
	append,
	backspace,
	splice,
	closePost,
	link,
	backlink,
	command,
	insertImage,
	spoiler,

	// >= 30 are miscelenious and do not write to post models
	synchronise = 30,
	resynchronise,
	switchSync,

	// Account management
	register,
	login,
	authenticate,
	logout,
	logoutAll,
	changePassword,

	// Board and server administration
	configServer,
	createBoard,

	// Send new post ID to client
	postID,

	// Concatenation of multiple websocket messages to reduce transport overhead
	concat,
}

export type MessageHandler = (msg: {}) => void

// Websocket message handlers. Each handler responds to its distinct message
// type.
export const handlers: { [type: number]: MessageHandler } = {}

// Websocket connection and syncronisation with server states
const enum syncStatus { disconnected, connecting, syncing, synced, desynced }

// States of the connection finite state machine
export const enum connState {
	loading, connecting, syncing, synced, reconnecting, dropped, desynced
}

// Events passable to the connection FSM
export const enum connEvent {
	start, open, close, retry, error, sync,
}

// Finite state machine for managing websocket connectivity
export const connSM = new FSM<connState, connEvent>(connState.loading)

let socket: WebSocket,
	attempts: number,
	attemptTimer: number

const syncEl = document.getElementById('sync')
const path =
	(location.protocol === 'https:' ? 'wss' : 'ws')
	+ `://${location.host}/socket`

function connect() {
	nullSocket()
	if (window.location.protocol == 'file:') {
		console.error("Page downloaded locally. Refusing to sync.")
		return
	}
	socket = new WebSocket(path)
	socket.onopen = connSM.feeder(connEvent.open)
	socket.onclose = connSM.feeder(connEvent.close)
	socket.onerror = connSM.feeder(connEvent.close)
	socket.onmessage = ({data}) =>
		onMessage(data, false)
	if (debug) {
		(window as any).socket = socket
	}
}

// Strip all handlers and remove references from Websocket instance
function nullSocket() {
	if (socket) {
		socket.onclose
			= socket.onmessage
			= socket.onopen
			= socket.onclose
			= socket.onerror
			= socket
			= null
	}
}

// Render connction status indicator
function renderStatus(status: syncStatus) {
	write(() =>
		syncEl.textContent = lang[status])
}

// Send a message to the server. If msg is null, it is omitted from sent
// websocket message.
export function send(type: message, msg: any) {
	if (socket.readyState !== 1) {
		console.warn("Attempting to send while socket closed")
		return
	}

	let str = leftPad(type)
	if (msg !== null) {
		str += JSON.stringify(msg)
	}

	if (debug) {
		console.log('<', str)
	}
	socket.send(str)
}

// Ensure message type is always a 2 characters long string
function leftPad(type: message): string {
	let str = type.toString()
	if (str.length === 1) {
		str = '0' + str
	}
	return str
}

// Routes messages from the server to the respective handler
function onMessage(data: string, extracted: boolean) {
	// First two charecters of a message define its type
	const type = parseInt(data.slice(0, 2))

	if (debug) {
		console.log(extracted ? ">>" : ">", data)
	}

	// Split several concatenated messages
	if (type === message.concat) {
		for (let msg of data.slice(2).split('\u0000')) {
			onMessage(msg, true)
		}
		return
	}

	const handler = handlers[type]
	if (handler) {
		handler(JSON.parse(data.slice(2)))
	}
}

function prepareToSync() {
	renderStatus(syncStatus.connecting)
	synchronise()
	authenticate()
	attemptTimer = setTimeout(() => resetAttempts(), 10000)
	return connState.syncing
}

// Send a requests to the server to syschronise to the current page and
// subscribe to the apropriate event feeds.
export async function synchronise() {
	const msg: SyncRequest = {
		board: page.board,
		thread: page.thread,
	}
	let type = message.synchronise

	// If thread data is too old because of disconnect, computer suspention or
	// resuming old tabs, refetch and rerender the thread. The actual deadline
	// is 30 seconds, but a ten second buffer is probably sound.
	if (page.thread && Date.now() - fetchTime > 20000) {
		await refetch()
	}

	// TODO: Resynchronisation logic, with open post right retrieval
	// // If clientID is set, then this attempt to synchronise comes after a
	// // connection loss. Attempt to recover lost server-side state.
	// if (clientID) {
	// 	msg.id = clientID
	// 	type = message.resynchronise
	// }

	send(type, msg)
}

// Reset the reconnection attempt counter and timers
function resetAttempts() {
	if (attemptTimer) {
		clearTimeout(attemptTimer)
		attemptTimer = 0
	}
	attempts = 0
}

function clearModuleState() {
	nullSocket()
	if (attemptTimer) {
		clearTimeout(attemptTimer)
		attemptTimer = 0
	}
}

export function start() {
	connSM.feed(connEvent.start)
}

// Work arround browser slowing down/suspending tabs and keep the FSM up to date
// with the actual status.
function onWindowFocus() {
	if (connSM.state !== connState.desynced && navigator.onLine) {
		connSM.feed(connEvent.retry)
	}
}

connSM.act(connState.loading, connEvent.start, () => {
	renderStatus(syncStatus.connecting)
	attempts = 0
	connect()
	return connState.connecting
})

for (let state of [connState.connecting, connState.reconnecting]) {
	connSM.act(state, connEvent.open, prepareToSync)
}

// Syncronise to the server and start receiving updates on the apropriate
// channel. If there are any missed meessages, fetch them.
handlers[message.synchronise] = (backlog: { [id: number]: PostData }) => {
	if (page.thread) {
		for (let id in backlog) {
			insertPost(backlog[id])
		}
	}
	connSM.feed(connEvent.sync)
}

connSM.act(connState.syncing, connEvent.sync, () => {
	renderStatus(syncStatus.synced)
	return connState.synced
})

connSM.wildAct(connEvent.close, event => {
	clearModuleState()
	if (debug) {
		console.error(event)
	}
	renderStatus(syncStatus.disconnected)

	// Wait maxes out at ~1min
	const wait = 500 * Math.pow(1.5, Math.min(Math.floor(++attempts / 2), 12))
	setTimeout(connSM.feeder(connEvent.retry), wait)

	return connState.dropped
})

connSM.act(connState.dropped, connEvent.retry, () => {
	connect()

	// Don't show this immediately so we don't thrash on network loss
	setTimeout(() =>
		connSM.state === connState.reconnecting
		&& renderStatus(syncStatus.connecting)
		, 100)
	return connState.reconnecting
})

// Invalid message or some other critical error
connSM.wildAct(connEvent.error, () => {
	renderStatus(syncStatus.desynced)
	clearModuleState()
	return connState.desynced
})

document.addEventListener('visibilitychange', event => {
	if (!(event.target as Document).hidden) {
		onWindowFocus()
	}
})

window.addEventListener('online', () => {
	resetAttempts()
	connSM.feed(connEvent.retry)
})
window.addEventListener('offline', connSM.feeder(connEvent.close))
