package db

import (
	"database/sql"

	"github.com/Rick-Phoenix/querygen/testdata/db/sqlgen"
)

type Store struct {
	db      sqlgen.DBTX
	Queries *sqlgen.Queries
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: sqlgen.New(db),
	}
}

type PostWithUser struct {
	User *sqlgen.User
	Post *sqlgen.Post
}

type UserWithPosts struct {
	*sqlgen.User
	Posts []*sqlgen.Post
}
