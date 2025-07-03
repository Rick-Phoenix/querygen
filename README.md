# What it does

This package automatically generates functions that aggregate results from several sqlc subqueries. This is especially useful when using sqlite since sqlc [does not support json aggregation](https://github.com/sqlc-dev/sqlc/issues/3988#issuecomment-2985800613) with it, unlike postgres. 

So this is meant to generate most (and in some cases, even all) of the boilerplate necessary to define and run those subqueries one by one. 

It receives a schema definition for the subqueries, and then runs them either in a transaction, if specified, or as separate goroutines, and returns the aggregated result.

# Requirements

In order to use this package, you must be using sqlc with the following options:

```yaml
emit_pointers_for_null_types: true
emit_result_struct_pointers: true
```

`goimports` and `gofmt` (or gofumpt) are called on the generated files, so those should also be installed.

# Examples

>[!NOTE]
>The "Store" passed to the querygen.New must be the return value of calling `(your_sqlc_package).New(db_instance)`, or a wrapper struct that holds the sqlc queries under the "Queries" field and the db instance under the "db" field.
>
>The outDir param is where the files will be generated, and the last part of this path will be the package name for the generated files. 
>
>This must be the same package where the store is defined, as this will add the methods to it directly.

From this configuration:

```go
type UserWithPost struct {
	User *sqlgen.User
	Post *sqlgen.Post
}

func TestMain(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	querySchema := QueryGenSchema{
		Name:       "GetUserWithPosts",
		ReturnType: &UserWithPost{},
		Queries: []QueryGroup{
            {
                IsTx: true, 
                Subqueries: []Subquery{
                {Method: "UpdatePost"}, 
                {Method: "UpdateUser", NoReturn: true},
            }},
			{
                Subqueries: []Subquery{
                {Method: "GetUser", SingleParamName: "userId"},
            }},
		},
		OutFile: "testquery",
	}

	gen := New(sqlgen.New(database), "testdata/db")
	gen.makeQuery(querySchema)
}
```

This file will be generated in "testdata/db/testquery.go":

```go
type GetUserWithPostsParams struct {
	UpdatePostParams sqlgen.UpdatePostParams

	UpdateUserParams sqlgen.UpdateUserParams

	userId int64
}

func (s *Store) GetUserWithPosts(ctx context.Context, params GetUserWithPostsParams) (*querygen.UserWithPost, error) {
	tx, err := s.db.(*sql.DB).BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := s.Queries.WithTx(tx)

	post, err := qtx.UpdatePost(ctx, params.UpdatePostParams)
	if err != nil {
        // Uses the variable name in the error message
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	if err := qtx.UpdateUser(ctx, params.UpdateUserParams); err != nil {
        // No variable name available here, so it uses the name of the method
		return nil, fmt.Errorf("error with UpdateUser: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	user, err := s.Queries.GetUser(ctx, params.userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &querygen.UserWithPost{
		User: user,

		Post: post,
	}, nil
}
```

So, a few things to note here:

1. The generator will aggregate all of the parameters required for the various subqueries. If there is only one, it will use that directly. Otherwise, it will create a wrapper struct like it did above. 

2. The name of the parameter to pass to a subquery can be overriden, which can be handy in some situations as a hacky way to avoid having an extra struct param if a query param is used in multiple places. 

>[!NOTE]
> Another example of this is also shown below.

Let's take this example. We need to make two queries: 
- One that gets a user by its id
- One that gets all posts with a certain userId and a certain postId

In this case, the package would generate a wrapper struct that holds the `GetPostsByUserIdParams` and a `userId` field.
But since `userId` is also being used in `GetPostsByUserIdParams`, you can ovverride this by setting `QueryParamName` to `GetPostsByUserIdParams.userId`. 

In this case, the generator will not add this field to the list of params, and you would end up having just a single param for the aggregated query, namely `GetPostsByUserIdParams`, and the userId to pass to the `GetUser` subquery will just get extracted from it, instead of being needlessly repeated. 

3. The variable names for the subqueries are automatically generated from the names of the return types. So for example, 
this: 

```go
post, err := qtx.UpdatePost(ctx, params.UpdatePostParams)
```

Is named "post" because the return type is `sqlgen.Post`. 
This can also be overriden by setting `VarName` to another value.

And in the return type, all fields are automatically assigned to their respective var names:

```go
return &querygen.UserWithPost{
    User: user,

    Post: post,
}, nil
```

This is the result of the generation because `UserWithPost` has these two fields:

```go
type UserWithPost struct {
	User *sqlgen.User
	Post *sqlgen.Post
}
```

This means that you have to be careful with the names of fields and params in order to make it all fit together. 

4. When a subquery has a non-struct parameter (string, int or whatever), you will need to define it in the `SingleParamName` field of the subquery struct (because there is no struct to extract the type name from), so that that can be added to the list of required params. 

If, however, you want to do what i explained in point 2, you can skip defining it like that and you can just use `QueryParamName` instead, so that this specific param will not be added to the list and instead it will just be given to the subquery method with the indicated name.

Now let's try a different kind of query to show an example for point 2 and a few other features. 

```go
type UserWithPosts struct {
	*sqlgen.User
	Posts []*sqlgen.Post
}

querySchema := QueryGenSchema{
    Name:       "GetUserWithPosts",
    ReturnType: &UserWithPosts{},
    Queries: []QueryGroup{
        // Only one subquery here
        {IsTx: true, 
            Subqueries: []Subquery{{Method: "UpdateUser", NoReturn: true}}},
        // Doing what we discussed above, to avoid having userId as a separate param
        {Subqueries: []Subquery{
            {Method: "GetUser", QueryParamName: "GetPostsFromUserIdParams.userId"}, 
            {Method: "GetPostsFromUserId"},
        }},
    },
    OutFile: "testquery",
	}
```

This would generate the following:

```go
type GetUserWithPostsParams struct {
	GetPostsFromUserIdParams sqlgen.GetPostsFromUserIdParams

	UpdateUserParams sqlgen.UpdateUserParams
}

func (s *Store) GetUserWithPosts(ctx context.Context, params GetUserWithPostsParams) (*querygen.UserWithPosts, error) {
    // Only one query here, so no transaction is necessary
	if err := s.Queries.UpdateUser(ctx, params.UpdateUserParams); err != nil {
		return nil, fmt.Errorf("error with UpdateUser: %w", err)
	}

    // Two queries in a non-transaction group, so goroutines are used
	userChan := make(chan *sqlgen.User)

	postsChan := make(chan []*sqlgen.Post)

	errChan := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		user, err := s.Queries.GetUser(ctx, params.GetPostsFromUserIdParams.userId)
		if err != nil {
			errChan <- fmt.Errorf("failed to get user: %w", err)
			return
		}
		userChan <- user
	}()

	go func() {
		defer wg.Done()
		posts, err := s.Queries.GetPostsFromUserId(ctx, params.GetPostsFromUserIdParams)
		if err != nil {
			errChan <- fmt.Errorf("failed to get posts: %w", err)
			return
		}
		postsChan <- posts
	}()

	wg.Wait()

	close(userChan)

	close(postsChan)

	for err := range errChan {
		return nil, err
	}

	user := <-userChan

	posts := <-postsChan

	return &querygen.UserWithPosts{
		User: user,

		Posts: posts,
	}, nil
}
```

Things to note here:

1. As you can see, `userId` is not added as a separate param to the param struct, instead it is "reused" from another param struct.

2. Since there is only one query in the first group, `tx: true` is ignored and the update query is done singularly.

3. Since the second group now has more than one query, they are run as a series of goroutines.
