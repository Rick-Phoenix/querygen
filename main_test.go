package querygen

import (
	"database/sql"
	"log"
	"testing"

	"github.com/Rick-Phoenix/querygen/_test/db"
	_ "modernc.org/sqlite"
)

type UserWithPost struct {
	User *db.User
	Post *db.Post
}

type UserWithPosts struct {
	*db.User
	Posts *db.Post
}

type PostsSlice struct {
	Posts *db.Post
}

func TestMain(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	querySchema := QueryGenSchema{
		Name:       "GetUserWithPosts",
		ReturnType: &PostsSlice{},
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
	gen := New(store, "_test/db")
	gen.makeQuery(querySchema)
}
