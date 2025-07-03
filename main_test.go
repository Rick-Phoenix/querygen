package querygen

import (
	"database/sql"
	"log"
	"testing"

	"github.com/Rick-Phoenix/querygen/db"
	"github.com/Rick-Phoenix/querygen/db/sqlgen"
	_ "modernc.org/sqlite"
)

func TestMain(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	querySchema := QueryGenSchema{
		Name:       "GetUserWithPosts",
		ReturnType: &db.PostWithUser{},
		Queries: []QueryGroup{
			{IsTx: true, Subqueries: []Subquery{{Method: "UpdatePost"}, {Method: "UpdateUser", NoReturn: true}}},
			{Subqueries: []Subquery{{Method: "GetUser", SingleParamName: "userId"}}},
		},
		Store:   sqlgen.New(database),
		OutFile: "tttestquery",
	}

	gen := NewQueryGen("db")
	gen.makeQuery(querySchema)
}
