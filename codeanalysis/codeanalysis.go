package codeanalysis

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/haibeihabo/auto-uml-for-golang/pkg/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config 配置的结构
type Config struct {
	CodeDir    string
	GopathDir  string
	VendorDir  string
	OutputFile string
	IgnoreDirs []string
}

// AnalysisResult 分析结果接口
type AnalysisResult interface {
	OutputToFile(logfile string)
}

// RunAnalysis runs the analysis
func RunAnalysis() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		config := Config{
			CodeDir:    FormatSlash(viper.GetString("codeargs.codepath")),
			GopathDir:  FormatSlash(viper.GetString("goenv.gopath")),
			VendorDir:  path.Join(viper.GetString("goenv.gopath"), "vendor"),
			OutputFile: FormatSlash(viper.GetString("codeargs.outputpath")),
			IgnoreDirs: viper.GetStringSlice("codeargs.ignoredirs"),
		}

		result := AnalysisCode(config)
		result.OutputToFile(config.OutputFile)
	}
}

// AnalysisCode 分析代码
func AnalysisCode(config Config) AnalysisResult {
	tool := &analysisTool{
		interfaceMetas:              []*interfaceMeta{},
		structMetas:                 []*structMeta{},
		typeAliasMetas:              []*typeAliasMeta{},
		packagePathPackageNameCache: map[string]string{},
		dependencyRelations:         []*DependencyRelation{},
	}
	tool.analysis(config)
	return tool
}

// HasPrefixInSomeElement 元素前缀
func HasPrefixInSomeElement(value string, src []string) bool {
	result := false
	for _, srcValue := range src {
		if strings.HasPrefix(value, srcValue) {
			result = true
			break
		}
	}
	return result
}

func sliceContains(src []string, value string) bool {
	isContain := false
	for _, srcValue := range src {
		if srcValue == value {
			isContain = true
			break
		}
	}
	return isContain
}

func sliceContainsSlice(s []string, s2 []string) bool {
	for _, str := range s2 {
		if !sliceContains(s, str) {
			return false
		}
	}
	return true
}

func mapContains(src map[string]string, key string) bool {
	if _, ok := src[key]; ok {
		return true
	}
	return false
}

func findGoPackageNameInDirPath(dirpath string) string {

	dirList, e := ioutil.ReadDir(dirpath)

	if e != nil {
		logging.Errorf("读取目录%s文件列表失败,%s", dirpath, e)
		return ""
	}

	for _, fileInfo := range dirList {
		if !fileInfo.IsDir() && strings.HasSuffix(fileInfo.Name(), ".go") {
			packageName := ParsePackageNameFromGoFile(path.Join(dirpath, fileInfo.Name()))
			if packageName != "" {
				return packageName
			}
		}
	}

	return ""
}

// ParsePackageNameFromGoFile 解析包
func ParsePackageNameFromGoFile(filepath string) string {

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath, nil, parser.ParseComments)

	if err != nil {
		logging.Errorf("解析文件%s失败, %s", filepath, err)
		return ""
	}

	return file.Name.Name

}

// PathExists 路径是否存在
func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func packagePathToUML(packagePath string) string {
	packagePath = strings.Replace(packagePath, "/", "\\\\", -1)
	packagePath = strings.Replace(packagePath, "-", "_", -1)
	return packagePath
}

type baseInfo struct {
	// go文件路径
	FilePath string
	// 包路径, 例如 github.com/Pingze-github/list-interface
	PackagePath string
}

type interfaceMeta struct {
	baseInfo
	Name string
	// interface的方法签名列表,
	MethodSigns []string
	// UML图节点
	UML string
}

// UniqueNameUML sss
func (i *interfaceMeta) UniqueNameUML() string {
	return packagePathToUML(i.PackagePath) + "." + i.Name
}

type structMeta struct {
	baseInfo
	Name string
	// struct的方法签名列表
	MethodSigns []string
	// UML图节点
	UML string
}

type typeAliasMeta struct {
	baseInfo
	Name           string
	targetTypeName string
}

func (s *structMeta) UniqueNameUML() string {
	return packagePathToUML(s.PackagePath) + "." + s.Name
}

func (s *structMeta) implInterfaceUML(interfaceMeta1 *interfaceMeta) string {
	return fmt.Sprintf("%s <|- %s\n", interfaceMeta1.UniqueNameUML(), s.UniqueNameUML())
}

type importMeta struct {
	// 例如 main
	Alias string
	// 例如 github.com/Pingze-github/list-interface
	Path string
}

// DependencyRelation 依赖关系
type DependencyRelation struct {
	source *structMeta
	target *structMeta
	uml    string
}

type analysisTool struct {
	config Config

	// 当前解析的go文件, 例如/appdev/go-demo/src/github.com/Pingze-github/list-interface/a.go
	currentFile string
	// 当前解析的go文件,所在包路径, 例如github.com/Pingze-github/list-interface
	currentPackagePath string
	// 当前解析的go文件,引入的其他包
	currentFileImports []*importMeta

	// 所有的interface
	interfaceMetas []*interfaceMeta
	// 所有的struct
	structMetas []*structMeta
	// 所有的别名定义
	typeAliasMetas []*typeAliasMeta
	// package path与package name的映射关系,例如github.com/Pingze-github/list-interface 对应的pakcage name为 main
	packagePathPackageNameCache map[string]string
	// struct之间的依赖关系
	dependencyRelations []*DependencyRelation
}

func (a *analysisTool) analysis(config Config) {

	a.config = config

	if a.config.CodeDir == "" || !PathExists(a.config.CodeDir) {
		logging.Errorf("Cannot find code dir %s\n", a.config.CodeDir)
		return
	}

	if a.config.GopathDir == "" || !PathExists(a.config.GopathDir) {
		logging.Errorf("Cannot find GOAPTH dir %s\n", a.config.GopathDir)
		return
	}

	for _, lib := range stdlibs {
		a.mapPackagePathPackageName(lib, path.Base(lib))
	}

	dirWalkOnce := func(path string, info os.FileInfo, err error) error {
		// 过滤掉测试代码
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "test.go") {
			if config.IgnoreDirs != nil && HasPrefixInSomeElement(path, config.IgnoreDirs) {
				// ignore
			} else {
				logging.Info("Parsing " + path)
				a.visitTypeInFile(path)
			}
		}

		return nil
	}

	filepath.Walk(config.CodeDir, dirWalkOnce)

	dirWalkTwice := func(path string, info os.FileInfo, err error) error {
		// 过滤掉测试代码
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "test.go") {
			if config.IgnoreDirs != nil && HasPrefixInSomeElement(path, config.IgnoreDirs) {
				// ignore
			} else {
				logging.Info("Parsing " + path)
				a.visitFuncInFile(path)
			}
		}

		return nil
	}

	filepath.Walk(config.CodeDir, dirWalkTwice)

}

func (a *analysisTool) initFile(path string) {
	logging.Debug("path=", path)

	a.currentFile = path
	a.currentPackagePath = a.filepathToPackagePath(path)

	if a.currentPackagePath == "" {
		logging.Errorf("packagePath is invalid, currentFile=%s\n", a.currentFile)
	}

}

func (a *analysisTool) mapPackagePathPackageName(packagePath string, packageName string) {
	if packagePath == "" || packageName == "" {
		logging.Errorf("mapPackagePathPackageName, packageName=%s, packagePath=%s\n, current_file=%s",
			packageName, packagePath, a.currentFile)
		return
	}

	if mapContains(a.packagePathPackageNameCache, packagePath) {
		return
	}

	logging.Debugf("mapPackagePathPackageName, packageName=%s, packagePath=%s\n", packageName, packagePath)
	a.packagePathPackageNameCache[packagePath] = packageName

}

func (a *analysisTool) visitTypeInFile(path string) {

	a.initFile(path)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)

	if err != nil {
		logging.Fatal(err)
		return
	}

	a.mapPackagePathPackageName(a.currentPackagePath, file.Name.Name)

	for _, decl := range file.Decls {

		genDecl, ok := decl.(*ast.GenDecl)

		if ok {
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if ok {
					a.visitTypeSpec(typeSpec)
				}
			}
		}
	}
}

func (a *analysisTool) visitTypeSpec(typeSpec *ast.TypeSpec) {

	interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
	if ok {
		a.visitInterfaceType(typeSpec.Name.Name, interfaceType)
		return
	}

	structType, ok := typeSpec.Type.(*ast.StructType)
	if ok {
		a.visitStructType(typeSpec.Name.Name, structType)
		return
	}

	// 其他类型别名
	a.typeAliasMetas = append(a.typeAliasMetas, &typeAliasMeta{
		baseInfo: baseInfo{
			FilePath:    a.currentFile,
			PackagePath: a.currentPackagePath,
		},
		Name:           typeSpec.Name.Name,
		targetTypeName: "",
	})

}

func (a *analysisTool) filepathToPackagePath(filepath string) string {
	filepath = FormatSlash(filepath)
	filepath = path.Dir(filepath)

	if a.config.VendorDir != "" {
		if strings.HasPrefix(filepath, a.config.VendorDir) {
			packagePath := strings.TrimPrefix(filepath, a.config.VendorDir)
			packagePath = strings.TrimPrefix(packagePath, "/")
			return packagePath
		}
	}

	if a.config.GopathDir != "" {
		srcdir := path.Join(a.config.GopathDir, "src")
		if strings.HasPrefix(filepath, srcdir) {
			packagePath := strings.TrimPrefix(filepath, srcdir)
			packagePath = strings.TrimPrefix(packagePath, "/")
			return packagePath
		}
	}

	logging.Errorf("Cannot confirm package file path, filepath=%s\n", filepath)

	return ""

}

func (a *analysisTool) visitFuncInFile(path string) {

	a.initFile(path)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)

	if err != nil {
		logging.Fatal(err)
		return
	}

	a.currentFileImports = []*importMeta{}

	if file.Imports != nil {
		for _, import1 := range file.Imports {
			alias := ""
			packagePath := strings.TrimSuffix(strings.TrimPrefix(import1.Path.Value, "\""), "\"")

			if import1.Name != nil {
				alias = import1.Name.Name
			} else {
				aliasCache, ok := a.packagePathPackageNameCache[packagePath]
				logging.Debugf("findAliasInCache,packagePath=%s,alias=%s,ok=%t\n", packagePath, aliasCache, ok)
				if ok {
					alias = aliasCache
				} else {
					alias = a.findAliasByPackagePath(packagePath)
				}
			}

			logging.Debugf("current_file=%s packagePath=%s, alias=%s\n", a.currentFile, packagePath, alias)

			a.currentFileImports = append(
				a.currentFileImports,
				&importMeta{
					Alias: alias,
					Path:  packagePath,
				},
			)
		}
	}

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if ok {
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if ok {
					interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
					if ok {
						a.visitInterfaceFunctions(typeSpec.Name.Name, interfaceType)
					}

					structType, ok := typeSpec.Type.(*ast.StructType)
					if ok {
						a.visitStructFields(typeSpec.Name.Name, structType)
					}
				}
			}
		}
	}

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if ok {
			a.visitFunc(funcDecl)
		}
	}

}

func (a *analysisTool) visitStructType(name string, structType *ast.StructType) {
	strutMeta1 := &structMeta{
		baseInfo: baseInfo{
			FilePath:    a.currentFile,
			PackagePath: a.currentPackagePath,
		},
		Name:        name,
		MethodSigns: []string{},
	}

	a.structMetas = append(a.structMetas, strutMeta1)

}

func (a *analysisTool) visitStructFields(structName string, structType *ast.StructType) {
	sourceStruct1 := a.findStruct(a.currentPackagePath, structName)
	sourceStruct1.UML = a.structToUML(structName, structType)
	for _, field := range structType.Fields.List {
		a.visitStructField(sourceStruct1, field)
	}
}

func (a *analysisTool) visitStructField(sourceStruct1 *structMeta, field *ast.Field) {

	fieldNames := a.IdentsToString(field.Names)

	targetStruct1, isarray := a.analysisTypeForDependencyRelation(field.Type)

	if targetStruct1 != nil {

		if fieldNames == "" {

			d := DependencyRelation{
				source: sourceStruct1,
				target: targetStruct1,
				uml:    sourceStruct1.UniqueNameUML() + " -|> " + targetStruct1.UniqueNameUML(),
			}

			a.dependencyRelations = append(a.dependencyRelations, &d)

		} else {

			if isarray {

				d := DependencyRelation{
					source: sourceStruct1,
					target: targetStruct1,
					uml:    sourceStruct1.UniqueNameUML() + " ---> \"*\" " + targetStruct1.UniqueNameUML() + " : " + fieldNames,
				}

				a.dependencyRelations = append(a.dependencyRelations, &d)

			} else {
				d := DependencyRelation{
					source: sourceStruct1,
					target: targetStruct1,
					uml:    sourceStruct1.UniqueNameUML() + " ---> " + targetStruct1.UniqueNameUML() + " : " + fieldNames,
				}

				a.dependencyRelations = append(a.dependencyRelations, &d)

			}

		}

	}

}

func (a *analysisTool) isGoBaseType(type1 string) bool {

	baseTypes := []string{"bool", "byte", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "complex64", "complex128", "string", "uintptr", "rune", "error"}

	if sliceContains(baseTypes, type1) {
		return true
	}

	return false
}

func (a *analysisTool) findStructByAliasAndStructName(alias string, structName string) *structMeta {

	if alias == "" && a.isGoBaseType(structName) {
		return nil
	}

	packagepath := a.findPackagePathByAlias(alias, structName)

	if packagepath != "" {
		return a.findStruct(packagepath, structName)
	}

	return nil
}

func (a *analysisTool) analysisTypeForDependencyRelation(t ast.Expr) (structMeta1 *structMeta, isArray bool) {

	structMeta1 = nil
	isArray = false

	ident, ok := t.(*ast.Ident)
	if ok {
		structMeta1 = a.findStructByAliasAndStructName("", ident.Name)
		isArray = false
		return
	}

	starExpr, ok := t.(*ast.StarExpr)
	if ok {
		structMeta1, isArray = a.analysisTypeForDependencyRelation(starExpr.X)
		return
	}

	arrayType, ok := t.(*ast.ArrayType)
	if ok {
		eleStructName, _ := a.analysisTypeForDependencyRelation(arrayType.Elt)
		structMeta1 = eleStructName
		isArray = true
		return
	}

	mapType, ok := t.(*ast.MapType)
	if ok {
		valueStructMeta1, _ := a.analysisTypeForDependencyRelation(mapType.Value)
		structMeta1 = valueStructMeta1
		isArray = true
		return
	}

	selectorExpr, ok := t.(*ast.SelectorExpr)
	if ok {
		alias := a.typeToString(selectorExpr.X, false)
		structMeta1 = a.findStructByAliasAndStructName(alias, a.typeToString(selectorExpr.Sel, false))
		isArray = false
		return
	}

	return
}

func (a *analysisTool) structToUML(name string, structType *ast.StructType) string {
	classUML := "class " + name + " " + a.structBodyToString(structType)
	return fmt.Sprintf("namespace %s {\n %s \n}", a.packagePathToUML(a.currentPackagePath), classUML)
}

func (a *analysisTool) packagePathToUML(packagePath string) string {
	return packagePathToUML(packagePath)
}

func (a *analysisTool) structBodyToString(structType *ast.StructType) string {
	result := "{\n"

	for _, field := range structType.Fields.List {
		result += "  " + a.fieldToString(field) + "\n"
	}
	result += "}"

	return result
}

func (a *analysisTool) visitInterfaceType(name string, interfaceType *ast.InterfaceType) {
	interfaceInfo1 := &interfaceMeta{
		baseInfo: baseInfo{
			FilePath:    a.currentFile,
			PackagePath: a.currentPackagePath,
		},
		Name: name,
	}

	a.interfaceMetas = append(a.interfaceMetas, interfaceInfo1)
}

func (a *analysisTool) interfaceToUML(name string, interfaceType *ast.InterfaceType) string {
	interfaceUML := "interface " + name + " " + a.interfaceBodyToString(interfaceType)
	return fmt.Sprintf("namespace %s {\n %s \n}", a.packagePathToUML(a.currentPackagePath), interfaceUML)
}

func (a *analysisTool) funcParamsResultsToString(funcType *ast.FuncType) string {

	funcString := "("

	if funcType.Params != nil {
		for index, field := range funcType.Params.List {
			if index != 0 {
				funcString += ","
			}

			funcString += a.fieldToString(field)
		}
	}

	funcString += ")"

	if funcType.Results != nil {

		if len(funcType.Results.List) >= 2 {
			funcString += "("
		}

		for index, field := range funcType.Results.List {
			if index != 0 {
				funcString += ","
			}

			funcString += a.fieldToString(field)
		}

		if len(funcType.Results.List) >= 2 {
			funcString += ")"
		}
	}

	return funcString

}

func (a *analysisTool) findStruct(packagePath string, structName string) *structMeta {

	for _, structMeta1 := range a.structMetas {
		if structMeta1.Name == structName && structMeta1.PackagePath == packagePath {
			return structMeta1
		}
	}

	return nil
}

func (a *analysisTool) findTypeAlias(packagePath string, structName string) *typeAliasMeta {

	for _, typeAliasMeta1 := range a.typeAliasMetas {
		if typeAliasMeta1.Name == structName && typeAliasMeta1.PackagePath == packagePath {
			return typeAliasMeta1
		}
	}

	return nil
}

func (a *analysisTool) findInterfaceMeta(packagePath string, interfaceName string) *interfaceMeta {

	for _, interfaceMeta := range a.interfaceMetas {
		if interfaceMeta.Name == interfaceName && interfaceMeta.PackagePath == packagePath {
			return interfaceMeta
		}
	}

	return nil
}

func (a *analysisTool) visitFunc(funcDecl *ast.FuncDecl) {

	a.debugFunc(funcDecl)

	packageAlias, structName := a.findStructTypeOfFunc(funcDecl)

	if structName != "" {

		packagePath := ""
		if packageAlias == "" {
			packagePath = a.currentPackagePath
		}

		structMeta := a.findStruct(packagePath, structName)
		if structMeta != nil {
			methodSign := a.createMethodSign(funcDecl.Name.Name, funcDecl.Type)
			structMeta.MethodSigns = append(structMeta.MethodSigns, methodSign)
		}
	}

}

func (a *analysisTool) visitInterfaceFunctions(name string, interfaceType *ast.InterfaceType) {
	methods := []string{}

	for _, field := range interfaceType.Methods.List {
		funcType, ok := field.Type.(*ast.FuncType)
		if ok {
			methods = append(methods, a.createMethodSign(field.Names[0].Name, funcType))
		}
	}

	interfaceMeta := a.findInterfaceMeta(a.currentPackagePath, name)
	interfaceMeta.MethodSigns = methods
	interfaceMeta.UML = a.interfaceToUML(name, interfaceType)
}

func (a *analysisTool) findStructTypeOfFunc(funcDecl *ast.FuncDecl) (packageAlias string, structName string) {
	if funcDecl.Recv != nil {
		for _, field := range funcDecl.Recv.List {
			t := field.Type
			ident, ok := t.(*ast.Ident)
			if ok {
				packageAlias = ""
				structName = ident.Name
			}

			starExpr, ok := t.(*ast.StarExpr)
			if ok {
				ident, ok := starExpr.X.(*ast.Ident)
				if ok {
					packageAlias = ""
					structName = ident.Name
				}
			}
		}
	}

	return
}

func (a *analysisTool) debugFunc(funcDecl *ast.FuncDecl) {
	logging.Debug("func name=", funcDecl.Name)

	if funcDecl.Recv != nil {
		for _, field := range funcDecl.Recv.List {
			logging.Debug("func recv, name=", field.Names, " type=", field.Type)
		}
	}

	if funcDecl.Type.Params != nil {
		for _, field := range funcDecl.Type.Params.List {
			logging.Debug("func param, name=", field.Names, " type=", field.Type)
		}
	}

	if funcDecl.Type.Results != nil {
		for _, field := range funcDecl.Type.Results.List {
			logging.Debug("func result, type=", field.Type)
		}
	}
}

func (a *analysisTool) IdentsToString(names []*ast.Ident) string {
	r := ""
	for index, name := range names {
		if index != 0 {
			r += ","
		}
		r += name.Name
	}

	return r
}

// 创建方法签名
func (a *analysisTool) createMethodSign(methodName string, funcType *ast.FuncType) string {
	methodSign := methodName + "("

	if funcType.Params != nil {
		for index, field := range funcType.Params.List {
			if index != 0 {
				methodSign += ","
			}
			methodSign += a.fieldToStringInMethodSign(field)
		}
	}

	methodSign += ")"

	if funcType.Results != nil {

		if len(funcType.Results.List) >= 2 {
			methodSign += "("
		}

		for index, field := range funcType.Results.List {
			if index != 0 {
				methodSign += ","
			}
			methodSign += a.fieldToStringInMethodSign(field)
		}

		if len(funcType.Results.List) >= 2 {
			methodSign += ")"
		}
	}

	return methodSign
}

func (a *analysisTool) fieldToStringInMethodSign(f *ast.Field) string {
	argCount := len(f.Names)

	if argCount == 0 {
		argCount = 1
	}

	sign := ""

	for i := 0; i < argCount; i++ {
		if i != 0 {
			sign += ","
		}
		sign += a.typeToString(f.Type, true)
	}

	return sign
}

func (a *analysisTool) fieldToString(f *ast.Field) string {
	r := ""

	if len(f.Names) > 0 {

		for index, name := range f.Names {
			if index != 0 {
				r += ","
			}

			r += name.Name
		}

		r += " "

	}

	r += a.typeToString(f.Type, false)

	return r
}

func (a *analysisTool) typeToString(t ast.Expr, convertTypeToUnqiueType bool) string {

	ident, ok := t.(*ast.Ident)
	if ok {
		if convertTypeToUnqiueType {
			return a.addPackagePathWhenStruct(ident.Name)
		}

		return ident.Name
	}

	starExpr, ok := t.(*ast.StarExpr)
	if ok {
		return "*" + a.typeToString(starExpr.X, convertTypeToUnqiueType)
	}

	arrayType, ok := t.(*ast.ArrayType)
	if ok {
		return "[]" + a.typeToString(arrayType.Elt, convertTypeToUnqiueType)
	}

	mapType, ok := t.(*ast.MapType)
	if ok {
		return "map[" + a.typeToString(mapType.Key, convertTypeToUnqiueType) + "]" + a.typeToString(mapType.Value, convertTypeToUnqiueType)
	}

	chanType, ok := t.(*ast.ChanType)
	if ok {
		return "chan " + a.typeToString(chanType.Value, convertTypeToUnqiueType)
	}

	funcType, ok := t.(*ast.FuncType)
	if ok {
		return "func" + a.funcParamsResultsToString(funcType)
	}

	interfaceType, ok := t.(*ast.InterfaceType)
	if ok {
		return "interface " + strings.Replace(a.interfaceBodyToString(interfaceType), "\n", " ", -1)
	}

	selectorExpr, ok := t.(*ast.SelectorExpr)
	if ok {
		if convertTypeToUnqiueType {
			return a.findPackagePathByAlias(a.selectorExprToString(selectorExpr.X), selectorExpr.Sel.Name) + "." + selectorExpr.Sel.Name
		}
		return a.typeToString(selectorExpr.X, true) + "." + selectorExpr.Sel.Name
	}

	structType, ok := t.(*ast.StructType)
	if ok {
		return "struct " + strings.Replace(a.structBodyToString(structType), "\n", " ", -1)
	}

	ellipsis, ok := t.(*ast.Ellipsis)
	if ok {
		return "... " + a.typeToString(ellipsis.Elt, convertTypeToUnqiueType)
	}

	parenExpr, ok := t.(*ast.ParenExpr)
	if ok {
		return " (" + a.typeToString(parenExpr.X, convertTypeToUnqiueType) + ")"
	}

	logging.Error("typeToString ", reflect.TypeOf(t), " file=", a.currentFile, " expr=", a.content(t))

	return ""
}

func (a *analysisTool) selectorExprToString(t ast.Expr) string {

	ident, ok := t.(*ast.Ident)
	if ok {
		return ident.Name
	}

	logging.Error("selectorExprToString ", reflect.TypeOf(t), " file=", a.currentFile, " expr=", a.content(t))

	return ""
}

func (a *analysisTool) addPackagePathWhenStruct(fieldType string) string {

	searchPackages := []string{a.currentPackagePath}

	for _, import1 := range a.currentFileImports {
		if import1.Alias == "." {
			searchPackages = append(searchPackages, import1.Path)
		}
	}

	for _, meta := range a.structMetas {
		if sliceContains(searchPackages, meta.PackagePath) && meta.Name == fieldType {
			return meta.PackagePath + "." + fieldType
		}
	}

	for _, meta := range a.interfaceMetas {
		if sliceContains(searchPackages, meta.PackagePath) && meta.Name == fieldType {
			return meta.PackagePath + "." + fieldType
		}
	}

	return fieldType
}

func (a *analysisTool) findAliasByPackagePath(packagePath string) string {
	result := ""

	if a.config.VendorDir != "" {
		absPath := path.Join(a.config.VendorDir, packagePath)
		if PathExists(absPath) {
			result = findGoPackageNameInDirPath(absPath)
		}
	}

	if a.config.GopathDir != "" {
		absPath := path.Join(a.config.GopathDir, "src", packagePath)
		if PathExists(absPath) {
			result = findGoPackageNameInDirPath(absPath)
		}
	}

	logging.Debugf("packagepath=%s, alias=%s\n", packagePath, result)

	return result
}

func (a *analysisTool) existStructOrInterfaceInPackage(typeName string, packageName string) bool {
	structMeta1 := a.findStruct(a.currentPackagePath, typeName)
	if structMeta1 != nil {
		return true
	}

	interfaceMeta1 := a.findInterfaceMeta(a.currentPackagePath, typeName)
	if interfaceMeta1 != nil {
		return true
	}

	return false
}

func (a *analysisTool) existTypeAliasInPackage(typeName string, packageName string) bool {
	meta1 := a.findTypeAlias(a.currentPackagePath, typeName)
	if meta1 != nil {
		return true
	}

	return false
}

func (a *analysisTool) findPackagePathByAlias(alias string, structName string) string {
	if alias == "" {

		if a.existStructOrInterfaceInPackage(structName, a.currentPackagePath) {
			return a.currentPackagePath
		}

		if a.existTypeAliasInPackage(structName, a.currentPackagePath) {
			// 忽略别名类型
			return ""
		}

		matchedImportMetas := []*importMeta{}

		for _, importMeta := range a.currentFileImports {
			if importMeta.Alias == "." {
				matchedImportMetas = append(matchedImportMetas, importMeta)
			}
		}

		if len(matchedImportMetas) > 1 {
			for _, matchedImportMeta := range matchedImportMetas {

				if a.existStructOrInterfaceInPackage(structName, matchedImportMeta.Path) {
					logging.Debugf("findPackagePathByAlias, alias=%s, packagePath=%s\n", alias, matchedImportMeta.Path)
					return matchedImportMeta.Path
				}

				if a.existTypeAliasInPackage(structName, matchedImportMeta.Path) {
					// 忽略别名类型
					return ""
				}
			}
		}

		currentFileImportsjson, _ := json.Marshal(a.currentFileImports)
		logging.Warnf("Cannot find package full path. Package=%s，type name=%s, file=%s, matchedImportMetas=%d, currentFileImports=%s", alias, structName, a.currentFile, len(matchedImportMetas), currentFileImportsjson)

		return alias

	}

	for _, importMeta := range a.currentFileImports {
		if importMeta.Path == alias {
			logging.Debugf("findPackagePathByAlias, alias=%s, packagePath=%s\n", alias, alias)
			return alias
		}
	}

	matchedImportMetas := []*importMeta{}

	for _, importMeta := range a.currentFileImports {
		if importMeta.Alias == alias {
			matchedImportMetas = append(matchedImportMetas, importMeta)
		}
	}

	if len(matchedImportMetas) == 1 {
		logging.Debugf("findPackagePathByAlias, alias=%s, packagePath=%s\n", alias, matchedImportMetas[0].Path)
		return matchedImportMetas[0].Path
	}

	if len(matchedImportMetas) > 1 {

		for _, matchedImportMeta := range matchedImportMetas {

			if a.existStructOrInterfaceInPackage(structName, matchedImportMeta.Path) {
				logging.Debugf("findPackagePathByAlias, alias=%s, packagePath=%s\n", alias, matchedImportMeta.Path)
				return matchedImportMeta.Path
			}

			if a.existTypeAliasInPackage(structName, matchedImportMeta.Path) {
				// 忽略别名类型
				return ""
			}

		}

	}

	currentFileImportsjson, _ := json.Marshal(a.currentFileImports)
	logging.Warnf("Cannot find package full path. Package=%s，type name=%s, file=%s, matchedImportMetas=%d, currentFileImports=%s", alias, structName, a.currentFile, len(matchedImportMetas), currentFileImportsjson)

	return alias
}

func (a *analysisTool) interfaceBodyToString(interfaceType *ast.InterfaceType) string {
	result := " {\n"
	for _, field := range interfaceType.Methods.List {

		funcType, ok := field.Type.(*ast.FuncType)

		if ok {
			result += "  " + a.IdentsToString(field.Names) + a.funcParamsResultsToString(funcType) + "\n"
		}

	}

	result += "}"
	return result
}

func (a *analysisTool) content(t ast.Expr) string {
	bytes, err := ioutil.ReadFile(a.currentFile)
	if err != nil {
		logging.Error("Read file", a.currentFile, "fail", err)
		return ""
	}

	return string(bytes[t.Pos()-1 : t.End()-1])
}

//查找哪些Struct实现了interface
func (a *analysisTool) findInterfaceImpls(interfaceMeta1 *interfaceMeta) []*structMeta {
	metas := []*structMeta{}

	for _, structMeta1 := range a.structMetas {
		if sliceContainsSlice(structMeta1.MethodSigns, interfaceMeta1.MethodSigns) {
			metas = append(metas, structMeta1)
		}
	}

	return metas
}

func (a *analysisTool) UML() string {
	uml := ""

	for _, structMeta1 := range a.structMetas {
		uml += structMeta1.UML
		uml += "\n"
	}

	for _, interfaceMeta1 := range a.interfaceMetas {
		uml += interfaceMeta1.UML
		uml += "\n"
	}

	for _, d := range a.dependencyRelations {
		uml += d.uml
		uml += "\n"
	}

	for _, interfaceMeta1 := range a.interfaceMetas {
		structMetas := a.findInterfaceImpls(interfaceMeta1)
		for _, structMeta := range structMetas {
			uml += structMeta.implInterfaceUML(interfaceMeta1)
		}
	}

	return "@startuml\n" + uml + "@enduml"
}

func (a *analysisTool) OutputToFile(logfile string) {
	uml := a.UML()
	ioutil.WriteFile(logfile, []byte(uml), 0666)
	logging.Infof("The UML file saved to %s\n", logfile)

}

// FormatSlash fromaslash
func FormatSlash(pathstr string) string {
	reg, _ := regexp.Compile("\\\\")
	return reg.ReplaceAllString(pathstr, "/")
}
