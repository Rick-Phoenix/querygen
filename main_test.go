package querygen

import (
	"database/sql"
	"log"
	"testing"

	"github.com/Rick-Phoenix/querygen/test/db"
	_ "modernc.org/sqlite"
)

func TestMain(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	querySchema := QueryGenSchema{
		Name:       "GetUserWithPosts",
		ReturnType: &db.UserWithPosts{},
		Queries: []QueryGroup{
			{Subqueries: []Subquery{
				{Method: "GetPostsFromUserId"}, {
					Method: "GetUser",
				},
			}},
		},
		OutFile: "testquery",
	}

	store := db.New(database)
	gen := New(store, "test/db")
	gen.makeQuery(querySchema)
}
