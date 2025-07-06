# What it does

This package automatically generates functions that aggregate results from several sqlc subqueries. This is especially useful when using sqlite since sqlc [does not support json aggregation](https://github.com/sqlc-dev/sqlc/issues/3988#issuecomment-2985800613) with it, unlike postgres. 

So this is meant to generate most (and in some cases, even all) of the boilerplate necessary to define and run those subqueries one by one. 

It receives a schema definition for the subqueries, and then runs them either in a transaction, if specified, or as separate goroutines, and returns the aggregated result.

# Requirements

In order to use this package, you must be using sqlc with the following options:

```yaml
emit_pointers_for_null_types: true
emit_result_struct_pointers: true
query_parameter_limit: 0
```

In order to use this package you must extend the sqlc-generated package's "Queries" struct with the following methods:

```go
type QueryData struct {
	Name         string
	ParamName    string
	Params       map[string]string
	ReturnTypes  []string
	ReturnFields map[string]string
	IsResult     bool
	IsErr        bool
	SliceReturn  bool
}

func (q *Queries) GetPkg() string {
	if q == nil {
		return ""
	}

	return reflect.TypeOf(q).Elem().PkgPath()
}

func (q *Queries) ExtractMethods() map[string]*QueryData {
	output := make(map[string]*QueryData)
	model := reflect.TypeOf(q)
	ignoredMethods := []string{"WithTx", "ExtractMethods", "GetPkg"}
	for i := range model.NumMethod() {
		method := model.Method(i)
		data := &QueryData{
			Params:       make(map[string]string),
			ReturnFields: make(map[string]string),
		}
		if slices.Contains(ignoredMethods, method.Name) {
			continue
		}
		data.Name = method.Name
		if method.Type.NumOut() == 1 {
			data.IsErr = true
			data.ReturnTypes = append(data.ReturnTypes, "error")
		} else {
			firstReturn := method.Type.Out(0)

			if firstReturn == reflect.TypeOf((*sql.Result)(nil)).Elem() {
				data.IsResult = true
				data.ReturnTypes = append(data.ReturnTypes, "sql.Result")
			} else {
				var target reflect.Type
				if firstReturn.Kind() == reflect.Slice {
					data.SliceReturn = true
					target = firstReturn.Elem().Elem()
				} else if firstReturn.Kind() == reflect.Pointer {
					target = firstReturn.Elem()
				}

				if target != nil && target.Kind() == reflect.Struct {
					for i := range target.NumField() {
						field := target.Field(i)
						data.ReturnFields[field.Name] = field.Type.Name()
					}
				}

				data.ReturnTypes = append(data.ReturnTypes, target.Name())
				data.ReturnTypes = append(data.ReturnTypes, "error")
			}
		}

		if method.Type.NumIn() > 2 {
			queryParam := method.Type.In(2)
			data.ParamName = queryParam.Name()
			for i := range queryParam.NumField() {
				field := queryParam.Field(i)
				data.Params[field.Name] = field.Type.Name()
			}
		}
		output[data.Name] = data
	}

	return output
}

```


# Examples

>[!NOTE]
>The outDir param is where the files will be generated, and the last part of this path will be the package name for the generated files. 
>
>This must be the same package where the store is defined, as this will add the methods to it directly.

From this configuration:

```go
import (
	"database/sql"
	"log"
	"testing"

	"github.com/Rick-Phoenix/querygen/test/db"
	_ "modernc.org/sqlite"
)

func TestMain(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	querySchema := QueryGenSchema{
		Name:       "GetUserWithPosts",
		ReturnType: &db.UserWithPosts{},
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
	gen := New(store, "test/db")
	gen.makeQuery(querySchema)
}
```

This file will be generated in "testdata/db/testquery.go":

```go
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
```

So, a few things to note here:

1. The generator will aggregate all of the parameters required for the various subqueries. If there is only one, it will use that directly. Otherwise, it will create a wrapper struct like it did above. 

2. In the return type, each field is automatically assigned to the variable of its name. The variable names for the subqueries are extracted automatically from the names of their returned structs, but they can also overridden.

3. If a subquery returns an array of slices, it will automatically receive the "s" suffix.
