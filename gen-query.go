package querygen

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	u "github.com/Rick-Phoenix/goutils"
)

// A subquery is defined and executed as part of a QueryGroup. It contains all the data that gets used for the file generation.
type Subquery struct {
	// Name of the method that gets called by the subquery (i.e. "GetUser")
	Method string
	// If this query has a single parameter that is not a struct, it will be added to the list of params with this name.
	SingleParamName string
	// An override for the name of the param being passed to the sqlc query that this subquery uses. Can be used, for example, to reuse a param that gets used in another struct param used in another query.
	QueryParamName string
	// Whether this query returns a value or just an error.
	NoReturn bool
	// The name of the variable to assign to this subquery. Defaults to the name of the return type of the query.
	Varname string
	// Whether the first return value should be discarded.
	DiscardReturn bool
}

type subqueryData struct {
	Method        string
	ParamName     string
	VarName       string
	ReturnType    string
	NoReturn      bool
	DiscardReturn bool
	Context       *queryData
}

// The schema for a query aggregator.
type QueryGenSchema struct {
	// The name for the generated query method.
	Name string
	// The list of subqueries to run in this query aggregator.
	Queries []QueryGroup
	// The return type of the aggregator. Must be a pointer to a struct.
	ReturnType any
	// The name of the output file. The ".go" suffix is added automatically. Defaults to the name of the query.
	OutFile string
}

// An aggregator for subqueries that will be run together, either as a transaction or as individual goroutines.
type QueryGroup struct {
	// Whether this query should be part of a transaction. It gets ignored if there is only one subquery.
	IsTx bool
	// The list of subqueries to run as part of this group. If there is only one entry, it will be run as a standalone query that just calls the sqlc method. If there is more than one and IsTx is true, they will be executed in a transaction. Otherwise, they will be executed as separate goroutines.
	Subqueries []Subquery
}

type queryGroupData struct {
	IsTx       bool
	Subqueries []subqueryData
}

type queryData struct {
	Name            string
	FunctionParams  map[string]string
	OutType         string
	OutTypeFields   []string
	Queries         []queryGroupData
	MakeParamStruct bool
	HasTx           bool
	FuncParamName   string
	FuncParamType   string
	Package         string
}

// The struct responsible for generating the queries.
type QueryGen struct {
	tmpl   *template.Template
	outDir string
	pkg    string
	store  reflect.Type
}

//go:embed templates/*
var templateFS embed.FS

// The constructor for the query generator.
// The "store" must be the returned value from running (sqlc_package_name_you_use).New(db_instance), or a wrapper struct that holds the sqlc queries under the "queries" field. The methods defined in the query schemas will be accessed from it.
// outDir is the output directory for the generated files. The last part will be used as the package name for the generated files.
// This must be the same package where the store is defined, as the methods will be assigned to it directly.
func New(store any, outDir string) *QueryGen {
	if outDir == "" {
		log.Fatalf("Missing output dir for generating queries.")
	}

	storeModel := reflect.TypeOf(store)

	if storeModel.Elem().Kind() != reflect.Struct {
		log.Fatalf("Invalid store for query generation. Must be a pointer to a struct (was %q)", storeModel.Name())
	}

	tmpl, err := template.New("protoTemplates").Funcs(funcMap).ParseFS(templateFS, "templates/*")
	if err != nil {
		fmt.Print(fmt.Errorf("Failed to initiate tmpl instance for the generator: %w", err))
		os.Exit(1)
	}

	return &QueryGen{tmpl: tmpl, outDir: outDir, pkg: path.Base(outDir), store: storeModel}
}

func (q *QueryGen) makeQuery(s QueryGenSchema) {
	tmpl := q.tmpl

	queryData := queryData{Name: s.Name, FunctionParams: make(map[string]string), Package: q.pkg}

	if s.ReturnType == nil {
		log.Fatalf("Missing output type for query generation.")
	}

	outModel := reflect.TypeOf(s.ReturnType).Elem()
	store := q.store

	if outModel.Kind() == reflect.Pointer {
		log.Fatalf("Found pointer of pointer for OutType %q when generating query %q", outModel.Name(), s.Name)
	}

	queryData.OutType = getPkgName(outModel, queryData.Package)

	if outModel.Kind() != reflect.Struct {
		log.Fatalf("Output type for query %q is not a struct.", s.Name)
	}

	for i := range outModel.NumField() {
		field := outModel.Field(i)
		queryData.OutTypeFields = append(queryData.OutTypeFields, field.Name)
	}

	for _, queryGroup := range s.Queries {

		queryGroupData := queryGroupData{IsTx: queryGroup.IsTx}

		if queryGroup.IsTx && len(queryGroup.Subqueries) > 1 {
			queryData.HasTx = true
		}

		for _, subQ := range queryGroup.Subqueries {
			subQData := subqueryData{Method: subQ.Method, Context: &queryData, VarName: subQ.Varname, NoReturn: subQ.NoReturn, DiscardReturn: subQ.DiscardReturn}
			method, ok := store.MethodByName(subQ.Method)

			if !ok {
				log.Fatalf("Could not find method %q in %q", subQ.Method, store.String())
			}

			if method.Type.NumIn() >= 3 {
				secondParam := method.Type.In(2)
				if secondParam.Kind() == reflect.Struct {
					subQData.ParamName = secondParam.Name()
					queryData.FunctionParams[secondParam.Name()] = getPkgName(secondParam, queryData.Package)

				} else if subQ.SingleParamName != "" {
					subQData.ParamName = subQ.SingleParamName
					queryData.FunctionParams[subQ.SingleParamName] = secondParam.Name()
				} else if subQ.QueryParamName != "" {
					subQData.ParamName = subQ.QueryParamName
				}
			}

			if len(queryData.FunctionParams) > 1 {
				queryData.MakeParamStruct = true
			}

			if !subQ.NoReturn && method.Type.NumOut() > 0 {
				out := method.Type.Out(0)
				outElem := out.Elem()
				outShortType := outElem.Name()
				outLongType := getPkgName(out, queryData.Package)
				if out.Kind() == reflect.Slice {
					outShortType = outElem.Elem().Name() + "s"
				}
				outShortLower := u.Uncapitalize(outShortType)
				if subQ.NoReturn {
					subQData.VarName = ""
				} else if subQ.DiscardReturn {
					subQData.VarName = "_"
				} else if subQ.Varname == "" {
					subQData.VarName = outShortLower
				}
				subQData.ReturnType = outLongType
			}

			queryGroupData.Subqueries = append(queryGroupData.Subqueries, subQData)
		}

		queryData.Queries = append(queryData.Queries, queryGroupData)
	}

	if len(queryData.FunctionParams) > 1 {
		queryData.FuncParamName = "params"
		queryData.FuncParamType = queryData.Name + "Params"
	} else {
		for name, typ := range queryData.FunctionParams {
			queryData.FuncParamName = name
			queryData.FuncParamType = typ
		}
	}

	outFile := s.OutFile
	if outFile == "" {
		outFile = s.Name
	}
	fullPath := filepath.Join(q.outDir, outFile+".go")
	err := u.ExecTemplateAndFormat(tmpl, "multiQuery", fullPath, queryData)
	if err != nil {
		fmt.Print(err)
	}

	fmt.Printf("✅ Query generated at %q\n", fullPath)
}

func getPkgName(model reflect.Type, pkg string) string {
	if path.Base(model.PkgPath()) == pkg {
		return model.Name()
	}

	return model.String()
}

var funcMap = template.FuncMap{
	"lower": strings.ToLower,
}
