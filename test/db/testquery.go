package db

import (
	"context"
	"fmt"
	"sync"
)

type GetUserWithPostsParams struct {
	GetPostsFromUserIdParams GetPostsFromUserIdParams

	GetUserParams GetUserParams
}

func (q *Queries) GetUserWithPosts(ctx context.Context, params GetUserWithPostsParams) (*UserWithPosts, error) {
	postsChan := make(chan []*Post)

	userChan := make(chan *User)
	errChan := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		posts, err := q.GetPostsFromUserId(ctx, params.GetPostsFromUserIdParams)
		if err != nil {
			errChan <- fmt.Errorf("failed to get posts: %w", err)
			return
		}
		postsChan <- posts
	}()

	go func() {
		defer wg.Done()
		user, err := q.GetUser(ctx, params.GetUserParams)
		if err != nil {
			errChan <- fmt.Errorf("failed to get user: %w", err)
			return
		}
		userChan <- user
	}()

	wg.Wait()

	close(postsChan)

	close(userChan)

	for err := range errChan {
		return nil, err
	}

	posts := <-postsChan

	user := <-userChan

	return &UserWithPosts{
		User: user,

		Posts: posts,
	}, nil
}
