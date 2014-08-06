package jag

import (
	"fmt"
	"strings"
	"log"
)

type Generator interface {
	GetClassSignature() *ClassSig
	JavaToGoTypeName(string) string
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
//	"java.lang.String...":"...string",
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
		z = "*"
		for _, part := range strings.Split(s, ".") {
			z += capitalize(part)
		}
		return
	}
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

func (s *StringGenerator) Generate() {
	sig := s.Gen.GetClassSignature()
	if sig.ClassName == "" {
		return
	}

	s.out += "package " + s.PkgName + "\n\n"
	s.out += "import \"jag\"\n\n"

	/*
	goClassTypeName := ""
	for _, part := range strings.Split(sig.ClassName, ".") {
		goClassTypeName += capitalize(part)
	}
	*/

	goClassTypeName := s.Gen.JavaToGoTypeName(sig.ClassName)
	s.out += fmt.Sprintf("type %s struct {\n\t*jag.Callable\n}\n\n", goClassTypeName)

	for i, constructor := range sig.Constructors {
		s.out += "// " + constructor.Line + "\n"
		s.out += "func New" + goClassTypeName
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
		s.out += ")"
		s.out += ` {
	obj, err := jag.Env.NewInstanceStr("` + sig.ClassName + `", ` + strings.Join(constructor.Params.Names(), ", ") + `)
	if err != nil {
		return nil, err
	}
	return &` + goClassTypeName + `{&goJVMCallable{obj, jag.Env}}, nil
}

`
	}

	for _, method := range sig.Methods {
		s.out += "// " + method.Line + "\n"
		s.out += fmt.Sprintf("func (x %s) %s", goClassTypeName, capitalize(method.Name))
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
		s.out += " {\n}\n\n"
	}
}

func (s *StringGenerator) Output() string {
	return s.out
}
