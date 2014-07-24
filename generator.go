package jag

import (
	"fmt"
	"strings"
)

type Generator interface {
	GetClassSignature() *ClassSig
	JavaToGoTypeName(string) string
	Generate()
}

type GeneratorHandle struct {
	Generator
}

// BT: basic type, GT: go type, T: any non template javatype
/*
var typeMap2 = map[string]string{

	"java.util.Map<BT, T>":"map[GT]GT",
	"java.util.Collection<T>":"[]GT",
	"java.util.Iterator<java.util.Map$Entry<BT, T>>",

	"java.util.Map<java.net.InetAddress, java.lang.Float>":"map[string]float32",
	"java.util.List<java.net.InetAddress>":"[]string",
}
*/

var typeMap = map[string]string{
	"void":"",
	"int":"int",
	"long":"int64",
	"float":"float32",
	"double":"float64",
	"boolean":"bool",
	"java.lang.String":"string",
	"java.lang.String...":"...string",
	"java.lang.String[]":"[]string",
	"java.util.List<java.lang.String>":"[]string",
	"java.util.Map<java.lang.String, java.lang.String>":"map[string]string",
	"java.util.Map<java.lang.String, java.lang.Integer>":"map[string]int",
	"java.util.Collection<java.lang.String>":"[]string",
	"java.util.Map<java.net.InetAddress, java.lang.Float>":"map[string]float32",
	"java.util.List<java.net.InetAddress>":"[]string",
}

type Translator struct {
	TypeMap map[string]string
}

func NewTranslator() *Translator {
	return &Translator{typeMap}
}

func (t *Translator) JavaToGoTypeName(s string) string {
	if v, ok := t.TypeMap[s]; ok {
		return v
	} else {
		return fmt.Sprintf("UNKNOWN %s", s)
	}
}

type StringGenerator struct {
	out string
	Gen Generator
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

	goClassTypeName := ""
	for _, part := range strings.Split(sig.ClassName, ".") {
		goClassTypeName += capitalize(part)
	}
	s.out += fmt.Sprintf("type %s struct {\n\t*gojvmcallable\n}\n\n", goClassTypeName)

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
	obj, err := env.NewInstanceStr("` + sig.ClassName + `", ` + strings.Join(constructor.Params.Names(), ", ") + `)
	if err != nil {
		    return nil, err
	}
	return &` + goClassTypeName + `{&goJVMCallable{obj, env}}, nil
}

`
	}

	for _, method := range sig.Methods {
		s.out += "// " + method.Line + "\n"
		s.out += fmt.Sprintf("func (x *%s) %s", goClassTypeName, capitalize(method.Name))
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
