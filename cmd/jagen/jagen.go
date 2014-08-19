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
	typeFilter := flag.String("filter", "", "filter out functions/methods by parameter/return types")
	trim := flag.String("trim", "", "prefix to trim from generated type names")
	abstractClassesFileName := flag.String("abstract", "", "file with names of abstract/interface classes")
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

	var abstractClassListFile io.Reader
	if *abstractClassesFileName != "" {
		file, err := os.Open(*abstractClassesFileName)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		abstractClassListFile = file
	}

	handle := &jag.ParserHandle{}
	javapSig := &jag.ClassSig{Parser: handle}
	parser := jag.NewParser(
		handle,
		jag.NewStatements(handle),
		&jag.Tokens{Parser: handle},
		javapSig,
		&jag.JavapParams{Parser: handle},
		commentfilter.NewCommentFilter("Signature:", "\n", `"`, `\`, commentfilter.NewCommentFilter("Compiled from", "\n", `"`, `\`, javapReader)),
	)

	parser.Scan()

	if srcReader != nil {
		handle := &jag.ParserHandle{}
		srcSig := &jag.ClassSig{Parser: handle}
		srcParser := jag.NewParser(
			handle,
			jag.NewStatements(handle),
			&jag.Tokens{Parser: handle},
			srcSig,
			&jag.SrcParams{Parser: handle},
			commentfilter.NewCommentFilter("//", "\n", `"`, `\`, commentfilter.NewCommentFilter("/*", "*/", `"`, `\`, srcReader)),
		)
		srcParser.Scan()

		cParamNames := make(map[string]int)
		for i, c := range srcSig.Constructors {
			cParamNames[strings.Join(c.Params.TypeClassNames(), "-")] = i
		}
		mParamNames := make(map[string]int)
		for i, m := range srcSig.Methods {
			mParamNames[m.Name + strings.Join(m.Params.TypeClassNames(), "-")] = i
		}
		for _, c := range javapSig.Constructors {
			if v, ok := cParamNames[strings.Join(c.Params.TypeClassNames(), "-")]; ok {
				for i := range c.Params {
					c.Params[i].Name = srcSig.Constructors[v].Params[i].Name
				}
				c.Line = srcSig.Constructors[v].Line
			}
		}
		for _, m := range javapSig.Methods {
			if v, ok := mParamNames[m.Name + strings.Join(m.Params.TypeClassNames(), "-")]; ok {
				for i := range m.Params {
					m.Params[i].Name = srcSig.Methods[v].Params[i].Name
				}
				m.Line = srcSig.Methods[v].Line
			}
		}
	}

	genHandle := &jag.GeneratorHandle{}

	var t jag.TranslatorInterface
	var list *jag.CallableList

	if *outputTypeDependency {
		list = jag.NewCallableList(jag.NewTranslator(genHandle, *trim))
		t = list
	} else {
		t = jag.NewTranslator(genHandle, *trim)
	}

	importList := jag.NewImportList(t)
	t = importList

	filter := jag.NewClassSigFilter(handle.Parser, *typeFilter)
	handle.Parser = filter

	gen := &struct {
		jag.TranslatorInterface
		jag.ImportListInterface
		*jag.ClassSigFilter
		*jag.StringGenerator
		*jag.AbstractClassList
		} {
		t,
		importList,
		filter,
		&jag.StringGenerator{Gen: genHandle, PkgName: *packageName},
		jag.NewAbstractClassList(abstractClassListFile),
	}
	genHandle.Generator = gen

	gen.Generate()

	if *outputTypeDependency {
		fmt.Println(strings.Join(list.ListCallables(), " "))
	} else {
		fmt.Print(gen.Output())
	}
}
