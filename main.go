package querygen

import (
	"database/sql"
	"testing"

	"github.com/Rick-Phoenix/querygen/db"
	"github.com/Rick-Phoenix/querygen/db/sqlgen"
	"github.com/labstack/gommon/log"
	_ "modernc.org/sqlite"
)

func TestMain(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	querySchema := QueryGenSchema{
		Name:    "GetUserWithPosts",
		OutType: &db.PostWithUser{},
		Queries: []QueryGroup{
			{IsTx: true, Subqueries: []Subquery{{Method: "UpdatePost"}, {Method: "UpdateUser", NoReturn: true}}},
			{Subqueries: []Subquery{{Method: "GetUser", SingleParamName: "userId"}}},
		},
		Store:      sqlgen.New(database),
		OutputPath: "db/tttestquery.go",
	}
}
