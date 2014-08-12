package main

import (
	"jag"
	"flag"
	"os"
	"log"
	"strings"
	"fmt"
	"io"
	"github.com/timob/commentfilter"
)

func main() {
	inputFilename := flag.String("in", "", "javap output file")
	srcFilename := flag.String("src", "", "set the source file name")
	packageName := flag.String("pkg", "gojvm_gen_package", "set the Go package name")
	outputTypeDependency := flag.Bool("d", false, "display type dependency")
	flag.Parse()

	var javapReader io.Reader
	if *inputFilename != "" {
		file, err := os.Open(*inputFilename)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		javapReader = file
	} else {
		javapReader = os.Stdin
	}

	var srcReader io.Reader
	if *srcFilename != "" {
		file, err := os.Open(*srcFilename)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		srcReader = file
	}

	handle := &jag.ParserHandle{}
	parser := jag.NewParser(
		handle,
		jag.NewStatements(handle),
		&jag.Tokens{Parser: handle},
		&jag.ClassSig{Parser: handle},
		&jag.JavapParams{Parser: handle},
		commentfilter.NewCommentFilter("Signature:", "\n", `"`, `\`, commentfilter.NewCommentFilter("Compiled from", "\n", `"`, `\`, javapReader)),
	)

	parser.Scan()

	if srcReader != nil {
		handle := &jag.ParserHandle{}
		srcParser := jag.NewParser(
			handle,
			jag.NewStatements(handle),
			&jag.Tokens{Parser: handle},
			&jag.ClassSig{Parser: handle},
			&jag.SrcParams{Parser: handle},
			commentfilter.NewCommentFilter("//", "\n", `"`, `\`, commentfilter.NewCommentFilter("/*", "*/", `"`, `\`, srcReader)),
		)
		srcParser.Scan()

		sig := srcParser.GetClassSignature()
		cParamNames := make(map[string]int)
		for i, c := range sig.Constructors {
			cParamNames[strings.Join(c.Params.TypeClassNames(), "-")] = i
		}
		mParamNames := make(map[string]int)
		for i, m := range sig.Methods {
			mParamNames[m.Name + strings.Join(m.Params.TypeClassNames(), "-")] = i
		}
		sig2 := parser.GetClassSignature()
		for _, c := range sig2.Constructors {
			if v, ok := cParamNames[strings.Join(c.Params.TypeClassNames(), "-")]; ok {
				for i := range c.Params {
					c.Params[i].Name = sig.Constructors[v].Params[i].Name
				}
				c.Line = sig.Constructors[v].Line
			}
		}
		for _, m := range sig2.Methods {
			if v, ok := mParamNames[m.Name + strings.Join(m.Params.TypeClassNames(), "-")]; ok {
				for i := range m.Params {
					m.Params[i].Name = sig.Methods[v].Params[i].Name
				}
				m.Line = sig.Methods[v].Line
			}
		}
	}

	genHandle := &jag.GeneratorHandle{}

	var t jag.TranslatorInterface
	var list *jag.CallableList

	if *outputTypeDependency {
		list = &jag.CallableList{Translator: jag.NewTranslator(genHandle)}
		t = list
	} else {
		t = jag.NewTranslator(genHandle)
	}

	gen := &struct {
		jag.TranslatorInterface
		*jag.ClassSig
		*jag.StringGenerator
		} {
		t,
		parser.GetClassSignature(),
		&jag.StringGenerator{Gen: genHandle, PkgName: *packageName},
	}
	genHandle.Generator = gen

	gen.Generate()

	if *outputTypeDependency {
		fmt.Println(strings.Join(list.ListCallables(), " "))
	} else {
		fmt.Print(gen.Output())
	}
}
