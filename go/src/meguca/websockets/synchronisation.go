// Synchronization management message handlers

package websockets

import (
	"errors"

	"meguca/common"
	"meguca/config"
	"meguca/db"
	"meguca/feeds"
)

var (
	errInvalidBoard  = errors.New("invalid board")
	errInvalidThread = errors.New("invalid thread")
)

type syncRequest struct {
	Thread uint64
	Board  string
}

type reclaimRequest struct {
	ID       uint64
	Password string
}

// Synchronise the client to a certain thread, assign it's ID and prepare to
// receive update messages.
func (c *Client) synchronise(data []byte) error {
	var msg syncRequest
	err := decodeMessage(data, &msg)
	switch {
	case err != nil:
		return err
	case !config.IsServeBoard(msg.Board):
		return errInvalidBoard
	case msg.Thread != 0:
		valid, err := db.ValidateOP(msg.Thread, msg.Board)
		switch {
		case err != nil:
			return err
		case !valid:
			return errInvalidThread
		}
	}

	return c.registerSync(msg.Thread, msg.Board)
}

// Register fresh client sync or change from previous sync
func (c *Client) registerSync(id uint64, board string) (err error) {
	c.feed, err = feeds.SyncClient(c, id, board)
	if err != nil {
		return
	}

	// Still sending something for consistency, but there is no actual syncing
	// to board pages
	if id == 0 {
		return c.sendMessage(common.MessageSynchronise, nil)
	}
	return
}
