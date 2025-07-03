package db

import (
	"database/sql"

	"github.com/Rick-Phoenix/protoschema/db/sqlgen"
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

func ToPointer[T any](s []T) []*T {
	out := make([]*T, len(s))
	for i, v := range s {
		out[i] = &v
	}

	return out
}

// func (s *Store) GetUserWithPosts(ctx context.Context, userID int64) (*UserWithPosts, error) {
// 	tx, err := s.db.(*sql.DB).BeginTx(ctx, nil)
// 	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
// 	defer cancel()
//
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to begin transaction: %w", err)
// 	}
// 	defer tx.Rollback()
//
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get user: %w", err)
// 	}
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get user: %w", err)
// 	}
// 	qtx := s.Queries.WithTx(tx)
// 	user, err := qtx.GetUser(ctx, userID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get user: %w", err)
// 	}
//
// 	posts, err := qtx.GetPostsFromUserId(ctx, userID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get post: %w", err)
// 	}
//
// 	if err := tx.Commit(); err != nil {
// 		return nil, fmt.Errorf("failed to commit read transaction: %w", err)
// 	}
//
// 	return &UserWithPosts{
// 		User: user, Posts: ToPointer(posts),
// 	}, nil
// }
