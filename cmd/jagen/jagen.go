package main

import (
	"jag"
	"flag"
	"os"
	"log"
	"strings"
	"fmt"
)

func main() {
	srcFilename := flag.String("src", "", "set the source file name")
	packageName := flag.String("pkg", "gojvm_gen_package", "set the Go package name")
	flag.Parse()

	var file *os.File
	var err error
	if *srcFilename != "" {
		file, err = os.Open(*srcFilename)
		if err != nil {
			log.Fatal(err)
		}
	}

	handle := &jag.ParserHandle{}
	parser := jag.NewParser(
		handle,
		jag.NewStatements(handle),
		&jag.Tokens{Parser: handle},
		&jag.ClassSig{Parser: handle},
		&jag.JavapParams{Parser: handle},
		jag.NewCommentFilter("Signature:", "\n", `"`, `\`, jag.NewCommentFilter("Compiled from", "\n", `"`, `\`, os.Stdin)),
	)

	parser.Scan()

	if file != nil {
		handle := &jag.ParserHandle{}
		srcParser := jag.NewParser(
			handle,
			jag.NewStatements(handle),
			&jag.Tokens{Parser: handle},
			&jag.ClassSig{Parser: handle},
			&jag.SrcParams{Parser: handle},
			jag.NewCommentFilter("//", "\n", `"`, `\`, jag.NewCommentFilter("/*", "*/", `"`, `\`, file)),
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

	gen := &struct {
		*jag.Translator
		*jag.ClassSig
		*jag.StringGenerator
		} {
		jag.NewTranslator(),
		parser.GetClassSignature(),
		&jag.StringGenerator{Gen: genHandle},
	}
	genHandle.Generator = gen

	gen.Generate()

	if gen.Output() != "" {
		fmt.Println("package " + *packageName + "\n")
		fmt.Print(gen.Output())
	}
}
