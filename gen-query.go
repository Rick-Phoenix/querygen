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
	"github.com/Rick-Phoenix/querygen/_test/db"
)

// A subquery is defined and executed as part of a QueryGroup. It contains all the data that gets used for the file generation.
type Subquery struct {
	// Name of the method that gets called by the subquery (i.e. "GetUser")
	Method string
	// An override for the name of the param being passed to the sqlc query that this subquery uses. Can be used, for example, to reuse a param that gets used in another struct param used in another query.
	QueryParamName string
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
	IsErr         bool
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
	// Whether this query should be part of a transaction. If false, goroutines will be used instead. It gets ignored if there is only one subquery.
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
	tmpl      *template.Template
	outDir    string
	pkg       string
	queryData map[string]*db.QueryData
}

type QueryStruct interface {
	GetPkg() string
	ExtractMethods() map[string]*db.QueryData
}

//go:embed templates/*
var templateFS embed.FS

// The constructor for the query generator.
// The "store" must be the *Queries sqlc-generated instance, augmented with the GetPkg and ExtractMethods methods.
// outDir is the output directory for the generated files, which must be the same as the sqlc package.
func New(queryStruct QueryStruct, outDir string) *QueryGen {
	if outDir == "" {
		log.Fatalf("Missing output dir for generating queries.")
	}

	tmpl, err := template.New("protoTemplates").Funcs(funcMap).ParseFS(templateFS, "templates/*")
	if err != nil {
		fmt.Print(fmt.Errorf("Failed to initiate tmpl instance for the generator: %w", err))
		os.Exit(1)
	}

	return &QueryGen{tmpl: tmpl, outDir: outDir, pkg: filepath.Base(outDir), queryData: queryStruct.ExtractMethods()}
}

func (q *QueryGen) makeQuery(s QueryGenSchema) {
	tmpl := q.tmpl

	queryData := queryData{Name: s.Name, FunctionParams: make(map[string]string), Package: q.pkg}

	if s.ReturnType == nil {
		log.Fatalf("Missing output type for query generation.")
	}

	returnType := reflect.TypeOf(s.ReturnType)

	if returnType.Kind() != reflect.Pointer || returnType.Elem().Kind() != reflect.Struct {
		log.Fatalf("The returnType must be a pointer to a struct.")
	} else {
		returnType = returnType.Elem()
	}

	queryData.OutType = returnType.Name()

	for i := range returnType.NumField() {
		field := returnType.Field(i)
		queryData.OutTypeFields = append(queryData.OutTypeFields, field.Name)
	}

	for _, queryGroup := range s.Queries {

		queryGroupData := queryGroupData{IsTx: queryGroup.IsTx}

		if queryGroup.IsTx && len(queryGroup.Subqueries) > 1 {
			queryData.HasTx = true
		}

		for _, subQ := range queryGroup.Subqueries {
			method, ok := q.queryData[subQ.Method]
			fmt.Printf("DEBUG: %+v\n", method)
			if !ok {
				log.Fatalf("Could not find method %q in the queryData map.", subQ.Method)
			}

			subQData := subqueryData{Method: subQ.Method, Context: &queryData, VarName: subQ.Varname, IsErr: method.IsErr, DiscardReturn: subQ.DiscardReturn, ReturnType: method.ReturnTypes[0]}

			if len(queryData.FunctionParams) > 1 {
				queryData.MakeParamStruct = true
			}

			if !subQData.IsErr {
				if subQ.DiscardReturn {
					subQData.VarName = "_"
				} else if subQ.Varname == "" {
					subQData.VarName = method.ReturnTypes[0]
				}
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

	err := u.ExecTemplate(tmpl, "multiQuery", fullPath, queryData)
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
