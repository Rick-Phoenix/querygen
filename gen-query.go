package querygen

import (
	"fmt"
	"path"
	"reflect"

	u "github.com/Rick-Phoenix/goutils"
	"github.com/labstack/gommon/log"
	_ "modernc.org/sqlite"
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
	OutType    any
	Store      any
	OutputPath string
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

func (p *ProtoPackage) makeQuery(s QueryGenSchema) {
	tmpl := p.tmpl

	if s.OutputPath == "" {
		log.Fatalf("Missing output path for query generation.")
	}

	if s.OutType == nil {
		log.Fatalf("Missing output type for query generation.")
	}

	store := reflect.TypeOf(s.Store)

	if store.Elem().Kind() != reflect.Struct {
		log.Fatalf("Invalid store for query generation. Must be a pointer to a struct (was %q)", store.Name())
	}

	queryData := queryData{Name: s.Name, FunctionParams: make(map[string]string), Package: path.Dir(s.OutputPath)}

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

	outModel := reflect.TypeOf(s.OutType).Elem()

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

	if len(queryData.FunctionParams) > 1 {
		queryData.FuncParamName = "params"
		queryData.FuncParamType = queryData.Name + "Params"
	} else {
		for name, typ := range queryData.FunctionParams {
			queryData.FuncParamName = name
			queryData.FuncParamType = typ
		}
	}

	err := u.ExecTemplateAndFormat(tmpl, "multiQuery", s.OutputPath, queryData)
	if err != nil {
		fmt.Print(err)
	}

	fmt.Printf("✅ Query generated at %q\n", s.OutputPath)
}

func getPkgName(model reflect.Type, pkg string) string {
	if path.Base(model.PkgPath()) == pkg {
		return model.Name()
	}

	return model.String()
}
