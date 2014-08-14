package jag

import (
	"fmt"
	"strings"
	"log"
	"go/token"
)

type Generator interface {
	GetClassSignature() ClassSigInterface
	Generate()
	TranslatorInterface
}

type TranslatorInterface interface {
	JavaToGoTypeName(string) string
	ConverterForType(prefix, s string) (z string)
	IsGoJVMType(s string) bool
	IsCallableType(s string) bool
}

type GeneratorHandle struct {
	Generator
}


// JNI pimitive arrays, and object arrays (should be done by GoJVM)
// variadic are arrays like Go so just prefix ... to type
var objectConversions = map[string]string {
	"java.lang.Long":"int64",
	"java.lang.Integer":"int",
	"java.lang.Float":"float32",
	"java.lang.String":"string",
	"java.net.InetAddress":"string",
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
}

func NewTranslator(g Generator) *Translator {
	return &Translator{g, typeMap, objectConversions}
}

func (t *Translator) JavaToGoTypeName(s string) (z string) {
	if debug {
		log.Printf("translating " + s)
		defer func() {log.Printf("translated to: " + z) }()
	}

	jc := JavaTypeComponents(s)
	gc := make([]interface{}, 0)
	for i := 1; i < len(jc); i++ {
		gc = append(gc, t.Gen.JavaToGoTypeName(jc[i]))
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

type CallableList struct {
	callables []string
	*Translator
}

func (c *CallableList) JavaToGoTypeName(s string) (z string) {
	jc := JavaTypeComponents(s)
	if !c.IsGoJVMType(jc[0]) && c.IsCallableType(jc[0]) {
		c.callables = append(c.callables, jc[0])
	}

	return c.Translator.JavaToGoTypeName(s)
}

func (c *CallableList) ListCallables() (list []string) {
	set := make(map[string]byte, len(c.callables))
	list = make([]string, 0, len(set))
	for _, name := range c.callables {
		if _, ok := set[name]; !ok {
			list = append(list, name)
		}
		set[name] = 0
	}
	return
}

// NewGoToJavaList(NewGoToJavaString())
// NewGoToJavaList(NewGoToJavaList(NewGoToJavaString())
func (t *Translator) ConverterForType(prefix, s string) (z string) {
	jc := JavaTypeComponents(s)
	if jc[0] == "..." || jc[0] == "[]"{
		z += "javabind.NewGoToGoObjectArray("
	} else {
		var name string
		if t.IsCallableType(jc[0]) {
			name = "Callable"
		} else {
			name = strings.Replace(className(jc[0]), "$", "_", -1)
		}
		z += prefix + name + "("
	}

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
		s.out += javaToGoIdentifier(p.Name) + " " + s.Gen.JavaToGoTypeName(p.Type)
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

func (s *StringGenerator) Generate() {
	sig := s.Gen.GetClassSignature()
	if sig.GetClassName() == "" {
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

	goClassTypeName := javaNameToGoName(sig.GetClassName())
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
		if method.Static {
			s.out += "javabind.CallStatic"
		} else {
			s.out += "jbobject.Call"
		}
		if s.Gen.IsGoJVMType(method.Return) {
			s.out += capitalize(strings.Replace(method.Return, "[]", "Array", -1))
		} else {
			s.out += "Obj"
		}
		callArgs := make([]string, 0)
		if method.Static {
			callArgs = append(callArgs, `"` + sig.GetClassName() + `"`)
		}
		callArgs = append(callArgs, `"` + method.Name + `"`)
		if !s.Gen.IsGoJVMType(method.Return) {
			callArgs = append(callArgs, `"` + JavaTypeComponents(method.Return)[0]  + `"`)
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
			var extra string
			if method.Throws {
				extra = ", nil"
			}
			if s.Gen.IsGoJVMType(method.Return) {
				s.out += "\treturn jret" +extra+ "\n"
			} else {
				s.out += "\tretconv := " + s.Gen.ConverterForType("javabind.NewJavaToGo", method.Return) + "\n"
				firstRetComponent := JavaTypeComponents(method.Return)[0]
				if s.Gen.IsCallableType(firstRetComponent) {
					s.out += "\tdst := &javabind.Callable{}\n"
				} else {
					s.out += "\tdst := new("+ret+")\n"
				}
				s.out += "\tretconv.Dest(dst)\n\tif err := retconv.Convert(jret); err != nil {\n\t\tpanic(err)\n\t}\n"
				s.out += "\tretconv.CleanUp()\n"
				if s.Gen.IsCallableType(firstRetComponent) {
					s.out += "\treturn &" + javaNameToGoName(method.Return) + "{dst}"+extra+"\n"
				} else {
					s.out += "\treturn *dst"+extra+"\n"
				}
			}
		} else if method.Throws {
			s.out += "\treturn nil\n"
		}
		s.out += "}\n\n"
	}
}

func (s *StringGenerator) Output() string {
	return s.out
}
