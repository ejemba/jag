package jag

import (
	"fmt"
	"strings"
	"log"
	"go/token"
	"io"
	"bufio"
)

type Generator interface {
	GetClassSignature() ClassSigInterface
	Generate()
	TranslatorInterface
	ImportListInterface
	IsAbstractClass(name string) bool
}

type ImportListInterface interface {
	ListImports() (list []string)
}

type TranslatorInterface interface {
	JavaToGoTypeName(string) string
	ConverterForType(prefix, s string) (z string)
	IsGoJVMType(s string) bool
	IsCallableType(s string) bool
	javaNameToGoName(s string) (z string)
}

type GeneratorHandle struct {
	Generator
}


// JNI pimitive arrays, and object arrays (should be done by GoJVM)
// variadic are arrays like Go so just prefix ... to type
var objectConversions = map[string]string {
	"java.lang.Boolean":"bool",
	"java.lang.Long":"int64",
	"java.lang.Integer":"int",
	"java.lang.Float":"float32",
	"java.lang.Double":"float64",
	"java.lang.String":"string",
	"java.net.InetAddress":"string",
	"java.util.Date":"time.Time",
	"...":"...%s",
	"[]":"[]%s",
	"java.util.List":"[]%s",
	"java.util.Collection":"[]%s",
	"java.util.Set":"[]%s",
	"java.util.Iterator":"[]%s",
	"java.util.Map":"map[%s]%s",
	"java.util.Map$Entry":"struct{Key %s; Value %s}",
//	"java.util.Iterator":"struct{func Next() bool, func Value() %s}",
}

// textual map (conversion done by GoJVM)
var typeMap = map[string]string{
	"void":"",
	"int":"int",
	"long":"int64",
	"float":"float32",
	"double":"float64",
	"boolean":"bool",
	"long[]":"[]int64",
	"int[]":"[]int",
}

type Translator struct {
	Gen Generator
	TypeMap map[string]string
	ObjectConversions map[string]string
	trim string
}

func NewTranslator(g Generator, trim string) *Translator {
	return &Translator{g, typeMap, objectConversions, trim}
}

func (t *Translator) JavaToGoTypeName(s string) (z string) {
	if debug {
		log.Printf("translating " + s)
		defer func() {log.Printf("translated to: " + z) }()
	}

	if v, ok := t.TypeMap[s]; ok {
		return v
	}

	jc := JavaTypeComponents(s)
	prefix := jc[0]
	if v, ok := t.ObjectConversions[prefix]; ok {
		gc := make([]interface{}, 0)
		for i := 1; i < len(jc); i++ {
			gc = append(gc, t.Gen.JavaToGoTypeName(jc[i]))
		}
		return fmt.Sprintf(v, gc...)
	}

	return "*" + t.Gen.javaNameToGoName(prefix)
}

func (t *Translator) IsGoJVMType(s string) bool {
	_, ok := t.TypeMap[s]
	return ok
}

func (t *Translator) IsCallableType(s string) bool {
	_, ok := t.ObjectConversions[s]
	return !ok
}

func (t *Translator) javaNameToGoName(s string) (z string) {
	s = strings.TrimPrefix(s, t.trim + ".")
	for _, part := range strings.Split(s, ".") {
		z += capitalize(part)
	}
	return
}

type CallableList struct {
	callables map[string]byte
	TranslatorInterface
}

func NewCallableList(t TranslatorInterface) *CallableList {
	return &CallableList{make(map[string]byte), t}
}

func (c *CallableList) JavaToGoTypeName(s string) (z string) {
	jc := JavaTypeComponents(s)
	if !c.IsGoJVMType(jc[0]) && c.IsCallableType(jc[0]) {
		c.callables[jc[0]] = 1
	}

	return c.TranslatorInterface.JavaToGoTypeName(s)
}

func (c *CallableList) ListCallables() (list []string) {
	for k, _ := range c.callables {
		list = append(list, k)
	}
	return
}

var importMap = map[string]string {
	"time":"time",
}

type ImportList struct {
	importMap map[string]string
	convertedTypes map[string]byte
	TranslatorInterface
}

func NewImportList(t TranslatorInterface) *ImportList {
	return &ImportList{importMap, make(map[string]byte), t}
}

func (c *ImportList) JavaToGoTypeName(s string) (z string) {
	name :=  c.TranslatorInterface.JavaToGoTypeName(s)
	jc := JavaTypeComponents(s)
	if !c.IsGoJVMType(jc[0]) && !c.IsCallableType(jc[0]) {
		c.convertedTypes[name] = 1
	}

	return name
}

func (c *ImportList) ListImports() (list []string) {
	for k, _ := range c.convertedTypes {
		for name, importedName := range c.importMap {
			if strings.HasPrefix(k, name + ".") {
				list = append(list, importedName)
			}
		}
	}
	return
}

// NewGoToJavaList(NewGoToJavaString())
// NewGoToJavaList(NewGoToJavaList(NewGoToJavaString())
func (t *Translator) ConverterForType(prefix, s string) (z string) {
	jc := JavaTypeComponents(s)

	var name string
	if jc[0] == "..." || jc[0] == "[]" {
		name = "ObjectArray"
	} else if t.IsCallableType(jc[0]) {
		return prefix + "Callable()"
	} else {
		name = strings.Replace(className(jc[0]), "$", "_", -1)
	}
	z = prefix + name + "("

	for i := 1; i < len(jc); i++ {
		if i != 1 {
			z += ", "
		}
		z += t.ConverterForType(prefix, jc[i])
	}
	z += ")"
	return
}

type AbstractClassList struct {
	list map[string]byte
}

func NewAbstractClassList(reader io.Reader) (a *AbstractClassList) {
	a = new(AbstractClassList)
	a.list = make(map[string]byte)
	if reader == nil {
		return
	}
	lineScanner := bufio.NewScanner(reader)
	for lineScanner.Scan() {
		a.list[lineScanner.Text()] = 1
	}
	return
}

func (a *AbstractClassList) IsAbstractClass(name string) bool {
	_, ok := a.list[name]
	return ok
}

func javaToGoIdentifier(s string) (z string) {
	if token.Lookup(s).IsKeyword() {
		return s + "_gen"
	}
	return s
}

type StringGenerator struct {
	out string
	Gen Generator
	PkgName string
}

func (s *StringGenerator) printParams(params Params) {
	for i, p := range params {
		if i != 0 {
			s.out += ", "
		}
		var typeName string
		firstComponent := JavaTypeComponents(p.Type)[0]
		if s.Gen.IsAbstractClass(firstComponent) {
			typeName = "interface{}"
		} else {
			typeName = s.Gen.JavaToGoTypeName(p.Type)
		}
		s.out += javaToGoIdentifier(p.Name) + " " + typeName
	}
}

func (s *StringGenerator) GenerateParamConversion(p Params) {
	conversions := make([]string, 0)
	for _, param := range p {
		if s.Gen.IsGoJVMType(param.Type) {
			continue
		}
		s.out += "\tconv_" + param.Name + " := " + s.Gen.ConverterForType("javabind.NewGoToJava", param.Type) + "\n"
		conversions = append(conversions, param.Name)
	}

	for _, param := range conversions {
		s.out += "\tif err := conv_" + param + ".Convert(" + javaToGoIdentifier(param) + "); err != nil {\n\t\tpanic(err)\n\t}\n"
	}
	return
}

func (s *StringGenerator) GenerateParamConversionCleanup(p Params) {
	for _, param := range p {
		if s.Gen.IsGoJVMType(param.Type) {
			continue
		}
		s.out += "\tconv_" + param.Name + ".CleanUp()\n"
	}
}

func (s *StringGenerator) GenerateReturnConversion(jtype string) {
	if s.Gen.IsGoJVMType(jtype) {
		s.out += "\treturn jret"
	} else {
		s.out += "\tretconv := " + s.Gen.ConverterForType("javabind.NewJavaToGo", jtype) + "\n"
		jretcomp := JavaTypeComponents(jtype)
		firstRetComponent := jretcomp[0]
		if s.Gen.IsCallableType(firstRetComponent) {
			s.out += "\tdst := &javabind.Callable{}\n"
		} else {
			s.out += "\tdst := new("+s.Gen.JavaToGoTypeName(jtype)+")\n"
		}
		s.out += "\tretconv.Dest(dst)\n\tif err := retconv.Convert(jret); err != nil {\n\t\tpanic(err)\n\t}\n"
		s.out += "\tretconv.CleanUp()\n"
		if s.Gen.IsCallableType(firstRetComponent) {
			s.out += "\treturn &" + s.Gen.javaNameToGoName(firstRetComponent) + "{dst}"
		} else {
			s.out += "\treturn *dst"
		}
	}
}

func (s *StringGenerator) GenerateCallArgs(p Params) (args []string) {
	args = make([]string, len(p))
	for i, param := range p {
 		if s.Gen.IsGoJVMType(param.Type) {
			args[i] = param.Name
		} else if strings.HasSuffix(param.Type, "...") {
			name := strings.TrimSuffix(param.Type, "...")
			args[i] = "javabind.ObjectArray(conv_"+param.Name+".Value(), \""+JavaTypeComponents(name)[0]+"\")"
		} else if strings.HasSuffix(param.Type, "[]") {
			name := strings.TrimSuffix(param.Type, "[]")
			args[i] = "javabind.ObjectArray(conv_" + param.Name + ".Value(), \"" + JavaTypeComponents(name)[0] + "\")"
		} else {
			args[i] = "javabind.CastObject(conv_" + param.Name + ".Value(), \"" + JavaTypeComponents(param.Type)[0] + "\")"
		}
	}
	return
}

func (s *StringGenerator) GenerateFuncName(t interface{}) {
	var static bool
	var prefix string
	var Type string
	switch v := t.(type) {
	case *ClassSigMethod:
		static = v.Static
		prefix = "Call"
		Type = v.Return
	case *ClassSigField:
		static = v.Static
		prefix = "GetField"
		Type = v.Type
	}
	if static {
		s.out += "javabind"
	} else {
		s.out += "jbobject"
	}
	s.out += "." + prefix
	if static {
		s.out += "Static"
	}

	if s.Gen.IsGoJVMType(Type) {
		s.out += capitalize(strings.Replace(Type, "[]", "Array", -1))
	} else {
		s.out += "Obj"
		if JavaTypeComponents(Type)[0] == "[]" {
			s.out += "Array"
		}
	}
}

func (s *StringGenerator) Generate() {
	sig := s.Gen.GetClassSignature()
	if sig.GetClassName() == "" {
		return
	}

	/*
	goClassTypeName := ""
	for _, part := range strings.Split(sig.ClassName, ".") {
		goClassTypeName += capitalize(part)
	}
	*/

	goClassTypeName := s.Gen.javaNameToGoName(JavaTypeComponents(sig.GetClassName())[0])
	s.out += fmt.Sprintf("type %s struct {\n\t*javabind.Callable\n}\n\n", goClassTypeName)

	for i, constructor := range sig.GetConstructors() {
		s.out += "// "+constructor.Line+"\n"
		s.out += "func New"+goClassTypeName
		if i > 0 {
			s.out += fmt.Sprintf("%d", i+1)
		}
		s.out += "("
		s.printParams(constructor.Params)
		s.out += ")"
		s.out += fmt.Sprintf(" (*%s", goClassTypeName)
		if constructor.Throws {
			s.out += ", error"
		}
		s.out += ") {\n"
		s.GenerateParamConversion(constructor.Params)
		newInstanceArgs := make([]string, 0)
		newInstanceArgs = append(newInstanceArgs, `"`+sig.GetClassName()+`"`)
		newInstanceArgs = append(newInstanceArgs, s.GenerateCallArgs(constructor.Params)...)
		var onError string
		if constructor.Throws {
			onError = "return nil, err"
		} else {
			onError = "panic(err)"
		}
		s.out += `
	obj, err := javabind.Env.NewInstanceStr(`+strings.Join(newInstanceArgs, ", ")+`)
	if err != nil {
		`+onError+`
	}` + "\n"
		s.GenerateParamConversionCleanup(constructor.Params)
		s.out += "\treturn &"+goClassTypeName+"{&javabind.Callable{obj, javabind.Env}}"

		if constructor.Throws {
			s.out += ", nil"
		}
		s.out += "\n}\n\n"
	}

	methodCount := make(map[string]int)
	for _, method := range sig.GetMethods() {
		s.out += "// " + method.Line + "\n"
		if method.Static {
			s.out += fmt.Sprintf("func %s", goClassTypeName + capitalize(method.Name))
		} else {
			s.out += fmt.Sprintf("func (jbobject *%s) %s", goClassTypeName, capitalize(method.Name))
		}
		if v , ok := methodCount[method.Name]; ok {
			v++
			s.out += fmt.Sprintf("%d", v)
			methodCount[method.Name] = v
		} else {
			methodCount[method.Name] = 1
		}
		s.out += "("
		s.printParams(method.Params)
		s.out += ") "
		ret := s.Gen.JavaToGoTypeName(method.Return)
		if ret != "" {
			if method.Throws {
				s.out += "(" + ret  + ", error)"
			} else {
				s.out += ret
			}
		} else {
			if method.Throws {
				s.out += "error"
			}
		}
		s.out += " {\n"
		s.GenerateParamConversion(method.Params)
		s.out += "\t"
		if ret != "" {
			s.out += "jret, "
		}
		s.out += "err := "
		s.GenerateFuncName(method)
		callArgs := make([]string, 0)
		if method.Static {
			callArgs = append(callArgs, `"` + sig.GetClassName() + `"`)
		}
		callArgs = append(callArgs, `"` + method.Name + `"`)
		jretcomp := JavaTypeComponents(method.Return)
		if !s.Gen.IsGoJVMType(method.Return) {
			comp := jretcomp[0]
			if comp == "[]" {
				comp = jretcomp[1]
			}
			callArgs = append(callArgs, `"` + comp  + `"`)
		}
		callArgs = append(callArgs, s.GenerateCallArgs(method.Params)...)
		s.out += "(" + strings.Join(callArgs, ", ") + ")\n"
		s.out += "\tif err != nil {\n\t\t"
		if method.Throws {
			if ret != "" {
				s.out += "var zero "+ret+"\n"
				s.out += "\t\treturn zero, "
			} else {
				s.out += "return "
			}
			s.out += "err\n"
		} else {
			s.out += "panic(err)\n"
		}
		s.out += "\t}\n"
		s.GenerateParamConversionCleanup(method.Params)
		if ret != "" {
			s.GenerateReturnConversion(method.Return)
			if method.Throws {
				s.out += ", nil"
			}
		} else if method.Throws {
			s.out += "\treturn nil"
		}
		s.out += "\n}\n\n"
	}

	for _ , field := range sig.GetFields() {
		if field.Static == false {
			continue
		}
		ret := s.Gen.JavaToGoTypeName(field.Type)
		s.out += "func " + goClassTypeName + capitalize(field.Name) + "() " +ret+ " {\n"
		s.out += "\tjret, err := "
		s.GenerateFuncName(field)
		s.out += "(\"" + sig.GetClassName() + "\", \"" + field.Name + "\""
		if !s.Gen.IsGoJVMType(field.Type) {
			jretcomp := JavaTypeComponents(field.Type)
			comp := jretcomp[0]
			if comp == "[]" {
				comp = jretcomp[1]
			}
			s.out += ", \"" + comp + "\""
		}
		s.out += ")\n"
		s.out += "\tif err != nil {\n\t\tpanic(err)\n\t}\n"
		s.GenerateReturnConversion(field.Type)
		s.out += "\n}\n\n"
	}

	prefix := "package " + s.PkgName + "\n\n"
	prefix += "import \"github.com/timob/javabind\"\n"
	for _, importName := range s.Gen.ListImports() {
		prefix += "import \"" + importName + "\"\n"
	}
	s.out = prefix + "\n" + s.out
}

func (s *StringGenerator) Output() string {
	return s.out
}
