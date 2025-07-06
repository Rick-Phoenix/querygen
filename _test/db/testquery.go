package db

import (
	"context"
	"fmt"
	"sync"
)

func (q *Queries) GetUserWithPosts(ctx context.Context) (*PostsSlice, error) {
	PostChan := make(chan Post)

	UserChan := make(chan User)

	errChan := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		Post, err := q.GetPostsFromUserId(ctx)
		if err != nil {
			errChan <- fmt.Errorf("failed to get Post: %w", err)
			return
		}
		PostChan <- Post
	}()

	go func() {
		defer wg.Done()
		User, err := q.GetUser(ctx)
		if err != nil {
			errChan <- fmt.Errorf("failed to get User: %w", err)
			return
		}
		UserChan <- User
	}()

	wg.Wait()

	close(PostChan)

	close(UserChan)

	for err := range errChan {
		return nil, err
	}

	Post := <-PostChan

	User := <-UserChan

	return &PostsSlice{
		Posts: posts,
	}, nil
}
