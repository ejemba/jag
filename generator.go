package jag

import (
	"fmt"
	"strings"
	"log"
)

type Generator interface {
	GetClassSignature() *ClassSig
	JavaToGoTypeName(string) string
	ConverterForType(prefix, s string) (z string)
	IsGoJVMType(s string) bool
	IsCallableType(s string) bool
	Generate()
}

type GeneratorHandle struct {
	Generator
}


// JNI pimitive arrays, and object arrays (should be done by GoJVM)

var objectConversions = map[string]string {
	"java.lang.String":"string",
	"...":"...%s",
	"java.util.List":"[]%s",
	"java.util.Collection":"[]%s",
	"java.util.Map":"map[%s]%s",
	"java.util.Map$Entry":"struct{key %s; value %s}",
	"java.util.Iterator":"struct{func Next() bool, func Value() %s}",
}

// textual map (conversion done by GoJVM)
// variadic are arrays like Go so just prefix ... to type
var typeMap = map[string]string{
	"void":"",
	"int":"int",
	"long":"int64",
	"float":"float32",
	"double":"float64",
	"boolean":"bool",
}

type Translator struct {
	TypeMap map[string]string
	ObjectConversions map[string]string
}

func NewTranslator() *Translator {
	return &Translator{typeMap, objectConversions}
}

func (t *Translator) JavaToGoTypeName(s string) (z string) {
	if debug {
		log.Printf("translating " + s)
		defer func() {log.Printf("translated to: " + z) }()
	}

	jc := JavaTypeComponents(s)
	gc := make([]interface{}, 0)
	for i := 1; i < len(jc); i++ {
		gc = append(gc, t.JavaToGoTypeName(jc[i]))
	}

	prefix := jc[0]
	if v, ok := t.TypeMap[prefix]; ok {
		return v
	} else if v, ok := t.ObjectConversions[prefix]; ok {
		return fmt.Sprintf(v, gc...)
	} else {
		z = "*" + javaNameToGoName(s)
		return
	}
}

func (t *Translator) IsGoJVMType(s string) bool {
	_, ok := t.TypeMap[s]
	return ok
}

func (t *Translator) IsCallableType(s string) bool {
	_, ok := t.ObjectConversions[s]
	return !ok
}

// NewGoToJavaList(NewGoToJavaString())
// NewGoToJavaList(NewGoToJavaList(NewGoToJavaString())
func (t *Translator) ConverterForType(prefix, s string) (z string) {
	jc := JavaTypeComponents(s)
	var name string
	if t.IsCallableType(jc[0]) {
		name = "Callable"
	} else {
		name = className(jc[0])
	}
	z += prefix + name + "("

	for i := 1; i < len(jc); i++ {
		if i != 1 {
			z += ", "
		}
		z += t.ConverterForType(prefix, jc[i])
	}
	z += ")"
	return
}

func javaNameToGoName(s string) (z string) {
	for _, part := range strings.Split(s, ".") {
		z += capitalize(part)
	}
	return
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
		s.out += p.Name + " " + s.Gen.JavaToGoTypeName(p.Type)
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
		s.out += "\tif err := conv_" + param + ".Convert(" + param + "); err != nil {\n\t\tpanic(err)\n\t}\n"
	}
	return
}

func (s *StringGenerator) GenerateCallArgs(p Params) (args []string) {
	args = make([]string, len(p))
	for i, param := range p {
		if s.Gen.IsGoJVMType(param.Type) {
			args[i] = param.Name
		} else {
			args[i] = "javabind.CastObject(conv_" + param.Name + ".Value(), \"" + JavaTypeComponents(param.Type)[0] + "\")"
		}
	}
	return
}

func (s *StringGenerator) Generate() {
	sig := s.Gen.GetClassSignature()
	if sig.ClassName == "" {
		return
	}

	s.out += "package " + s.PkgName + "\n\n"
	s.out += "import \"javabind\"\n\n"

	/*
	goClassTypeName := ""
	for _, part := range strings.Split(sig.ClassName, ".") {
		goClassTypeName += capitalize(part)
	}
	*/

	goClassTypeName := javaNameToGoName(sig.ClassName)
	s.out += fmt.Sprintf("type %s struct {\n\t*javabind.Callable\n}\n\n", goClassTypeName)

	for i, constructor := range sig.Constructors {
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
		newInstanceArgs = append(newInstanceArgs, `"`+sig.ClassName+`"`)
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
	}
	return &`+goClassTypeName+`{&javabind.Callable{obj, javabind.Env}}`

		if constructor.Throws {
			s.out += ", nil"
		}
		s.out += "\n}\n\n"
	}

	for _, method := range sig.Methods {
		s.out += "// " + method.Line + "\n"
		s.out += fmt.Sprintf("func (jbobject *%s) %s", goClassTypeName, capitalize(method.Name))
		s.out += "("
		s.printParams(method.Params)
		s.out += ") "
		ret := s.Gen.JavaToGoTypeName(method.Return)
		if ret != "" {
			if method.Throws {
				s.out += "(ret " + ret  + ", err error)"
			} else {
				s.out += "(ret " + ret + ")"
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
		s.out += "jbobject.Call"
		if s.Gen.IsGoJVMType(method.Return) {
			s.out += method.Return
		} else {
			s.out += "Obj"
		}
		callArgs := make([]string, 0)
		callArgs = append(callArgs, `"` + method.Name + `"`)
		if !s.Gen.IsGoJVMType(method.Return) {
			callArgs = append(callArgs, `"` + JavaTypeComponents(method.Return)[0]  + `"`)
		}
		callArgs = append(callArgs, s.GenerateCallArgs(method.Params)...)
		s.out += "(" + strings.Join(callArgs, ", ") + ")\n"
		s.out += "\tif err != nil {\n\t\t"
		if method.Throws {
			s.out += "return\n"
		} else {
			s.out += "panic(err)\n"
		}
		s.out += "\t}\n"
		if ret != "" {
			if s.Gen.IsGoJVMType(method.Return) {
				s.out += "\tret = jret\n"
			} else {
				s.out += "\tretconv := " + s.Gen.ConverterForType("javabind.NewJavaToGo", method.Return) + "\n"
				firstRetComponent := JavaTypeComponents(method.Return)[0]
				if s.Gen.IsCallableType(firstRetComponent) {
					s.out += "\tdst := &javabind.Callable{}\n"
				} else {
					s.out += "\tdst := &ret\n"
				}
				s.out += "\tretconv.Dest(dst)\n\tif err := retconv.Convert(jret); err != nil {\n\t\tpanic(err)\n\t}\n"
				if s.Gen.IsCallableType(firstRetComponent) {
					s.out += "\tret = &" + javaNameToGoName(method.Return) + "{dst}\n"
				}
			}
		}
		s.out += "\treturn\n}\n\n"
	}
}

func (s *StringGenerator) Output() string {
	return s.out
}
