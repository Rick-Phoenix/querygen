package db

import (
	"context"
	"database/sql"
	"fmt"
)

type GetUserWithPostsParams struct {
	UpdatePostParams sqlgen.UpdatePostParams

	UpdateUserParams sqlgen.UpdateUserParams

	userId int64
}

func (s *Store) GetUserWithPosts(ctx context.Context, params GetUserWithPostsParams) (*PostWithUser, error) {
	tx, err := s.db.(*sql.DB).BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := s.Queries.WithTx(tx)

	post, err := qtx.UpdatePost(ctx, params.UpdatePostParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	if err := qtx.UpdateUser(ctx, params.UpdateUserParams); err != nil {
		return nil, fmt.Errorf("error with UpdateUser: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	user, err := s.Queries.GetUser(ctx, params.userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &PostWithUser{
		User: user,

		Post: post,
	}, nil
}
