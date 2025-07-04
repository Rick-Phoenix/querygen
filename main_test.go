package querygen

import (
	"database/sql"
	"log"
	"testing"

	"github.com/Rick-Phoenix/querygen/testdata/db/sqlgen"
	_ "modernc.org/sqlite"
)

type UserWithPost struct {
	User *sqlgen.User
	Post *sqlgen.Post
}

type UserWithPosts struct {
	*sqlgen.User
	Posts []*sqlgen.Post
}

func TestMain(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	querySchema := QueryGenSchema{
		Name:       "GetUserWithPosts",
		ReturnType: &UserWithPosts{},
		Queries: []QueryGroup{
			{IsTx: true, Subqueries: []Subquery{
				{Method: "UpdatePost"},
				{Method: "UpdateUser", NoReturn: true},
			}},
			{Subqueries: []Subquery{
				{Method: "GetUser", QueryParamName: "GetPostsFromUserIdParams.userId"},
				{Method: "GetPostsFromUserId"},
			}},
		},
		OutFile: "testquery",
	}

	gen := New(sqlgen.New(database), "_test/db")
	gen.makeQuery(querySchema)
}
