// FrontEnds for using the inbuilt post cache

package server

import (
	"net/http"
	"strconv"

	"meguca/cache"
	"meguca/common"
	"meguca/db"
	"meguca/templates"
)

// Contains data of a board page
type pageStore struct {
	pageNumber, pageTotal int
	json                  []byte
	data                  common.Board
}

// var newsCache = cache.FrontEnd{
// 	GetCounter: func(k cache.Key) (uint64, error) {
// 		// Update once per 5 minutes.
// 		ctr := time.Now().Unix() / 60 / 5
// 		return uint64(ctr), nil
// 	},

// 	GetFresh: func(k cache.Key) (interface{}, error) {
// 		return db.GetNews()
// 	},

// 	EncodeJSON: func(data interface{}) ([]byte, error) {
// 		// Not needed.
// 		return nil, nil
// 	},

// 	RenderHTML: func(data interface{}, json []byte, k cache.Key) []byte {
// 		return []byte(templates.News(data.([]common.NewsEntry)))
// 	},
// }

var threadCache = cache.FrontEnd{
	GetCounter: func(k cache.Key) (uint64, error) {
		return db.ThreadCounter(k.ID)
	},

	GetFresh: func(k cache.Key) (interface{}, error) {
		return db.GetThread(k.ID, int(k.LastN))
	},

	RenderHTML: func(data interface{}, json []byte, k cache.Key) []byte {
		last100 := k.LastN == numPostsOnRequest
		return []byte(templates.ThreadPosts(k.Lang, data.(common.Thread), json, last100))
	},
}

var catalogCache = cache.FrontEnd{
	GetCounter: func(k cache.Key) (uint64, error) {
		if k.Board == "all" {
			return db.AllBoardCounter()
		}
		return db.BoardCounter(k.Board)
	},

	GetFresh: func(k cache.Key) (interface{}, error) {
		if k.Board == "all" {
			return db.GetAllBoardCatalog()
		}
		return db.GetBoardCatalog(k.Board)
	},

	RenderHTML: func(data interface{}, json []byte, k cache.Key) []byte {
		all := k.Board == "all"
		return []byte(templates.CatalogThreads(data.(common.Board), json, all))
	},
}

var boardCache = cache.FrontEnd{
	GetCounter: func(k cache.Key) (uint64, error) {
		if k.Board == "all" {
			return db.AllBoardCounter()
		}
		return db.BoardCounter(k.Board)
	},

	// Board pages are built as a list of individually fetched and cached
	// threads with up to 3 replies each.
	GetFresh: func(k cache.Key) (interface{}, error) {
		// Get thread IDs in the right order
		var (
			ids []uint64
			err error
		)
		if k.Board == "all" {
			ids, err = db.GetAllThreadsIDs()
		} else {
			ids, err = db.GetThreadIDs(k.Board)
		}
		if err != nil {
			return nil, err
		}

		// Get data and JSON for these views and paginate
		var (
			pages = make([]pageStore, 0, (len(ids)-1)/20+1)
			page  pageStore
		)
		closePage := func() {
			if page.data != nil {
				page.json = append(page.json, ']')
				pages = append(pages, page)
			}
		}
		for i, id := range ids {
			// Start a new page
			if i%20 == 0 {
				closePage()
				page = pageStore{
					pageNumber: len(pages),
					json:       append(make([]byte, 0, 1<<10), '['),
					data:       make(common.Board, 0, 20),
				}
			}

			k := cache.ThreadKey(k.Lang, id, numPostsAtIndex)
			json, data, _, err := cache.GetJSONAndData(k, threadCache)
			if err != nil {
				return nil, err
			}
			if len(page.json) != 1 {
				page.json = append(page.json, ',')
			}
			page.json = append(page.json, json...)
			page.data = append(page.data, data.(common.Thread))
		}
		closePage()

		// Record total page count in all stores
		l := len(pages)
		if l == 0 { // Empty board
			l = 1
			pages = []pageStore{
				{
					json: []byte("[]"),
				},
			}
		}
		for i := range pages {
			pages[i].pageTotal = l
		}

		return pages, nil
	},

	Size: func(data interface{}, _, _ []byte) (s int) {
		for _, p := range data.([]pageStore) {
			s += len(p.json) * 2
		}
		return
	},
}

// For individual pages of a board index
var boardPageCache = cache.FrontEnd{
	GetCounter: func(k cache.Key) (uint64, error) {
		// Get the counter of the parent board
		k.Page = -1
		_, _, ctr, err := cache.GetJSONAndData(k, boardCache)
		return ctr, err
	},

	GetFresh: func(k cache.Key) (interface{}, error) {
		i := int(k.Page)
		k.Page = -1
		_, data, _, err := cache.GetJSONAndData(k, boardCache)
		if err != nil {
			return nil, err
		}

		pages := data.([]pageStore)
		if i > len(pages)-1 {
			return nil, errPageOverflow
		}
		return pages[i], nil
	},

	EncodeJSON: func(data interface{}) ([]byte, error) {
		return data.(pageStore).json, nil
	},

	RenderHTML: func(data interface{}, json []byte, k cache.Key) []byte {
		all := k.Board == "all"
		return []byte(templates.IndexThreads(k.Lang, data.(pageStore).data, json, all))
	},

	Size: func(_ interface{}, _, html []byte) int {
		// Only the HTML is owned by this store. All other data is just
		// borrowed from boardCache.
		return len(html)
	},
}

// Returns arguments for accessing the board page JSON/HTML cache
func boardCacheArgs(r *http.Request, board string, catalog bool) (
	k cache.Key, f cache.FrontEnd,
) {
	var page int64
	if !catalog {
		p, err := strconv.ParseUint(r.URL.Query().Get("page"), 10, 64)
		if err == nil {
			page = int64(p)
		}
	}

	k = cache.BoardKey(getReqLang(r), board, page, !catalog)
	if catalog {
		f = catalogCache
	} else {
		f = boardPageCache
	}
	return
}
