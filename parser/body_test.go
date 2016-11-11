package parser

import (
	"testing"

	"github.com/bakape/meguca/config"
	. "github.com/bakape/meguca/test"
	"github.com/bakape/meguca/types"
)

func TestParseLine(t *testing.T) {
	config.SetBoardConfigs(config.BoardConfigs{
		ID: "a",
	})

	t.Run("commands disabled", func(t *testing.T) {
		links, com, err := ParseLine([]byte("#flip"), "a")
		if err != nil {
			t.Fatal(err)
		}
		if links != nil {
			t.Fatalf("unexpected links: %#v", links)
		}
		AssertDeepEquals(t, com, types.Command{})
	})

	t.Run("commands enabled", func(t *testing.T) {
		config.SetBoardConfigs(config.BoardConfigs{
			ID: "a",
			BoardPublic: config.BoardPublic{
				PostParseConfigs: config.PostParseConfigs{
					HashCommands: true,
				},
			},
		})

		links, com, err := ParseLine([]byte("#flip"), "a")
		if err != nil {
			t.Fatal(err)
		}
		if links != nil {
			t.Fatalf("unexpected links: %#v", links)
		}
		if com.Type != types.Flip {
			t.Fatalf("unexpected command type: %d", com.Type)
		}
	})
}

func TestParseBody(t *testing.T) {
	assertTableClear(t, "posts")
	assertInsert(t, "posts", []types.DatabasePost{
		{
			StandalonePost: types.StandalonePost{
				Post: types.Post{
					ID: 8,
				},
				OP:    2,
				Board: "a",
			},
		},
		{
			StandalonePost: types.StandalonePost{
				Post: types.Post{
					ID: 6,
				},
				OP:    2,
				Board: "a",
			},
		},
	})
	config.SetBoardConfigs(config.BoardConfigs{
		ID: "a",
		BoardPublic: config.BoardPublic{
			PostParseConfigs: config.PostParseConfigs{
				HashCommands: true,
			},
		},
	})

	links, com, err := ParseBody([]byte("#flip\n>>8\n>>>6 #flip\n#flip"), "a")
	if err != nil {
		t.Fatal(err)
	}
	if l := len(com); l != 2 {
		t.Errorf("unexpected command count: %d", l)
	}
	AssertDeepEquals(t, links, types.LinkMap{
		8: types.Link{
			OP:    2,
			Board: "a",
		},
		6: types.Link{
			OP:    2,
			Board: "a",
		},
	})
}