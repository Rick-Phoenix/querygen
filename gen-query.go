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

type Subquery struct {
	Method          string
	SingleParamName string
	QueryParamName  string
	NoReturn        bool
	Varname         string
	DiscardReturn   bool
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

type QueryGenSchema struct {
	Name       string
	Queries    []QueryGroup
	ReturnType any
	Store      any
	OutFile    string
}

type QueryGroup struct {
	IsTx       bool
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

type QueryGen struct {
	tmpl   *template.Template
	outDir string
	pkg    string
}

//go:embed templates/*
var templateFS embed.FS

func NewQueryGen(outDir string) *QueryGen {
	if outDir == "" {
		log.Fatalf("Missing output dir for generating queries.")
	}
	tmpl, err := template.New("protoTemplates").Funcs(funcMap).ParseFS(templateFS, "templates/*")
	if err != nil {
		fmt.Print(fmt.Errorf("Failed to initiate tmpl instance for the generator: %w", err))
		os.Exit(1)
	}

	return &QueryGen{tmpl: tmpl, outDir: outDir, pkg: path.Base(outDir)}
}

func (q *QueryGen) makeQuery(s QueryGenSchema) {
	tmpl := q.tmpl

	queryData := queryData{Name: s.Name, FunctionParams: make(map[string]string), Package: q.pkg}

	if s.ReturnType == nil {
		log.Fatalf("Missing output type for query generation.")
	}

	store := reflect.TypeOf(s.Store)

	if store.Elem().Kind() != reflect.Struct {
		log.Fatalf("Invalid store for query generation. Must be a pointer to a struct (was %q)", store.Name())
	}

	outModel := reflect.TypeOf(s.ReturnType).Elem()

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
