package jag

import (
	"io"	
	"strings"
	"bufio"
	"fmt"
	"bytes"
	"log"
	"github.com/timob/sliceutil"
	"regexp"
)

var debug = false

func capitalize(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:]
}

type Parser interface {
	GetStatement() string
	ParseStatement()
	GetToken(int) string
	ScopeDepth() int
	GetCurrentStatement() string
	FindToken(token string) (pos int, found bool)
	Scan()
	io.Reader
	ParamParser
	ClassSigInterface
}

type ClassSigInterface interface {
	ParamWords() (count int, start int)
	Parse()
	GetPackageName() string
	GetClassName() string
    GetExtends() string
	GetFields() []*ClassSigField
	GetConstructors() []*ClassSigConstructor
	GetMethods() []*ClassSigMethod
	GetClassSignature() ClassSigInterface
}

type ParamParser interface {
	GetParams() Params
}

type ParserHandle struct {
		Parser
}


func JavaTypeComponents(j string) (p []string) {
	if strings.HasSuffix(j, "...") {
		return []string{"...", strings.TrimSuffix(string(j), "...")}
	} else if strings.HasSuffix(j, "[]") {
		return []string{"[]", strings.TrimSuffix(string(j), "[]")}
	}

	s := bufio.NewScanner(bytes.NewBufferString(strings.Replace(j, " ", "", -1)))
	var scopeDepth int
	s.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
            return 0, nil, nil
        }

        if i := bytes.IndexAny(data, "<>,"); i >= 0 {
			if string(data[i]) == "<" {
				scopeDepth++
			}

			var ret []byte
			if scopeDepth < 2 {
				ret = data[0:i]
			} else {
				ret = data[0:i+1]
			}

			if string(data[i]) == ">" {
				scopeDepth--
			}
			return i + 1, ret, nil
		} else if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})

	p = make([]string, 0)
	var part string
	for s.Scan() {
		part += s.Text()
		if scopeDepth < 2 && part != "" {
			p = append(p, part)
			part = ""
		}
	}
	return
}

type Param struct {
	Name string
	Type string
}

type Params []Param

func (params Params) Names() (names []string) {
	names = make([]string, len(params))
	for i, p := range params {
		names[i] = p.Name
	}
	return
}

func (params Params) Types() (types []string) {
	types = make([]string, len(params))
	for i, p := range params {
		types[i] = p.Type
	}
	return
}

var classNameRe = regexp.MustCompile(`[A-z,a-z,0-9]+\.`)

func className(name string) string {
	return classNameRe.ReplaceAllString(name, "")
}

func (params Params) TypeClassNames() (names []string) {
	names = make([]string, len(params))
	for i, p := range params {
		names[i] = className(p.Type)
	}
	return
}

type stmtMsg struct {
	statement string
	depth int
}

type Statements struct {
	stmts chan *stmtMsg
	scopeDepth int
	Parser Parser
}

func NewStatements(g Parser) (a *Statements) {
	a = &Statements{make(chan *stmtMsg, 0), 0, g}
	return
}

func (s *Statements) GetStatement() string {
	x := <- s.stmts
	s.scopeDepth = x.depth
	return x.statement
}

func (s *Statements) Scan() {
	go s.Parser.Parse()

	scanner := bufio.NewScanner(s.Parser)

	depth := 0
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if i := bytes.IndexAny(data, ";{}"); i >= 0 {
			if string(data[i]) == "{" {
				depth++
			} else if string(data[i]) == "}" {
				depth--
			}
			return i + 1, data[0:i], nil
		} else if atEOF {
			return len(data), nil, nil
		}
		return 0, nil, nil
	})
	
	for scanner.Scan() {
		stmt := string(bytes.Join(bytes.Fields(scanner.Bytes()), []byte{' '}))
		s.stmts <- &stmtMsg{stmt, depth}
	}
}

func (s *Statements) ScopeDepth() int {
	return s.scopeDepth
}

type Tokens struct {
	tokens []string
	Parser Parser
	currentStmt string
}

func (t *Tokens) ParseStatement() {
	tokens := make([]string, 0)

	token := ""
	depth := 0
	t.currentStmt = t.Parser.GetStatement()
	for _, r := range t.currentStmt {
		if r == '<' {
			depth++
		}
		if r == '>' {
			depth--
		}			
		if depth != 0 {
			token += string(r)
			continue
		}			
		
		if r == ' ' {
			if token != "" {
				tokens = append(tokens, token)
			}
			token = ""
		} else if r == ',' {
			if token != "" {
				tokens = append(tokens, token)
			}
			token = ""
		} else if r == '(' {
			if token != "" {
				tokens = append(tokens, token)
			}
			tokens = append(tokens, "(")
			token = ""
		} else if r == ')' {
			if token != "" {
				tokens = append(tokens, token)
			}
			tokens = append(tokens, ")")
			token = ""
		} else {
			token += string(r)
		}
	}

	if token != "" {
		tokens = append(tokens, token)
	}		

	if debug {
		log.Printf("%v depth=%d", tokens, t.Parser.ScopeDepth())
	}

	t.tokens = tokens
}

func (t *Tokens) GetToken(i int) string {
	if i >= len(t.tokens) {
		return ""
	} else {
		return t.tokens[i]
	}
}

func (t *Tokens) FindToken(token string) (pos int, found bool) {
	for i := 0; t.GetToken(i) != ""; i++ {
		if t.GetToken(i) == token {
			return i, true
		}
	}
	return 0, false
}

func (t *Tokens) GetCurrentStatement() string {
	return t.currentStmt
}

type ClassSigConstructor struct {
	Params Params
	Throws bool
	Line string
}

type ClassSigMethod struct {
	Name string
    Params Params
	Return string
	Throws bool
	Line string
	Static bool
}

type ClassSigField struct {
	Name string
	Type string
	Static bool
}

type ClassSig struct {
	PackageName string
	ClassName string
    Extends string
	Constructors []*ClassSigConstructor
	Methods []*ClassSigMethod
	Fields []*ClassSigField
	Parser Parser
}

func (c *ClassSig) GetClassSignature() ClassSigInterface {
	return c.Parser
}

func (c *ClassSig) GetPackageName() string {
	return c.PackageName
}

func (c *ClassSig) GetClassName() string {
	return c.ClassName
}

func (c *ClassSig) GetExtends() string {
    return c.Extends
}

func (c *ClassSig) Parse() {
	for {
		c.Parser.ParseStatement()
		
		if c.Parser.GetToken(0) == "package" {
			c.PackageName = c.Parser.GetToken(1)
		}

		_, found := c.Parser.FindToken("public")
		if !found {
			continue
		}

		declarePos, found := c.Parser.FindToken("class")
		if !found {
			declarePos, found = c.Parser.FindToken("interface")
		}
		if !found {
			declarePos, found = c.Parser.FindToken("enum")
		}
		if !found {
			continue
		}

		c.ClassName = c.Parser.GetToken(declarePos + 1)

		if strings.Contains(c.ClassName, "<") {
			continue
		}

        if pos, found := c.Parser.FindToken("extends"); found  {
            c.Extends = c.Parser.GetToken(pos +1 )
        }

		for c.Parser.ScopeDepth() > 0 {
			c.Parser.ParseStatement()
			if c.Parser.ScopeDepth() > 2 {
				continue
			}
			if c.Parser.GetToken(0) != "public" {
				continue
			}

			_, static := c.Parser.FindToken("static")
			_, fun := c.Parser.FindToken("(");
			typePos := c.FirstNonKeyWord()
			t := c.Parser.GetToken(typePos)
			if t[0] == '<' && t[len(t) - 1] == '>' {
				continue
//				typePos++
			}

			if fun {
				if c.Parser.GetToken(typePos) == c.ClassName && !static  && c.Parser.GetToken(typePos+1) == "(" {
					i := sliceutil.Append(&c.Constructors)
					c.Constructors[i].Params = c.Parser.GetParams()
					c.Constructors[i].Throws = c.Throws()
					c.Constructors[i].Line = c.Parser.GetCurrentStatement()
				} else {
					i := sliceutil.Append(&c.Methods)
					c.Methods[i].Name = c.Parser.GetToken(typePos+1)
					c.Methods[i].Params = c.Parser.GetParams()
					c.Methods[i].Return = c.Parser.GetToken(typePos)
					c.Methods[i].Throws = c.Throws()
					c.Methods[i].Line = c.Parser.GetCurrentStatement()
					c.Methods[i].Static = static
				}
			} else if static {
				i := sliceutil.Append(&c.Fields)
				c.Fields[i].Name = c.Parser.GetToken(typePos+1)
				c.Fields[i].Type = c.Parser.GetToken(typePos)
				c.Fields[i].Static = static
			}

		}
	}
}

func (c *ClassSig) Throws() bool {
	var foundClose bool
	for i := 0; c.Parser.GetToken(i) != ""; i++ {
		if c.Parser.GetToken(i) == ")" {
			foundClose = true
		}
		if c.Parser.GetToken(i) == "throws" && foundClose {
			return true
		}
	}
	return false
}

func (c *ClassSig) ParamWords() (count int, start int) {
	start, found := c.Parser.FindToken("(")
	if !found {
		panic("ParamWords called while invalid statement")
	}
	start++
	i := start
	for {
		token := c.Parser.GetToken(i)
		if token == ")" {
			break
		}
		if !javaKeyWord(token) {
			count++
		}
		i++
	}
	return
}

func (c *ClassSig) FirstNonKeyWord() int {
	for i := 0; c.Parser.GetToken(i) != ""; i++ {
		if !javaKeyWord(c.Parser.GetToken(i)) {
			return i
		}
	}
	return -1
}

func (c *ClassSig) GetConstructors() []*ClassSigConstructor {
	return c.Constructors
}

func (c *ClassSig) GetMethods() []*ClassSigMethod {
	return c.Methods
}

func (c *ClassSig) GetFields() []*ClassSigField {
	return c.Fields
}

type ClassSigFilter struct {
	Parser
	filter map[string]byte
}

func NewClassSigFilter(p Parser, filterStr string) *ClassSigFilter {
	filter := make(map[string]byte)
	for _, name := range strings.Split(filterStr, " ") {
		filter[name] = 0
	}
	return &ClassSigFilter{p, filter}
}

func (c *ClassSigFilter) GetConstructors() []*ClassSigConstructor {
	ret := make([]*ClassSigConstructor, 0, len(c.Parser.GetConstructors()))
A:
	for _, v := range c.Parser.GetConstructors() {
		for _, v2 := range v.Params {
			for _, v3 := range JavaTypeComponents(v2.Type) {
				if _, ok := c.filter[v3]; ok {
					continue A
				}
			}
		}
		ret = append(ret, v)
	}
	return ret
}

func (c *ClassSigFilter) GetMethods() []*ClassSigMethod {
	ret := make([]*ClassSigMethod, 0, len(c.Parser.GetMethods()))
A:
	for _, v := range c.Parser.GetMethods() {
		for _, v2 := range v.Params {
			for _, v3 := range JavaTypeComponents(v2.Type) {
				if _, ok := c.filter[v3]; ok {
					continue A
				}
			}
		}
		for _, v2 := range JavaTypeComponents(v.Return) {
			if _, ok := c.filter[v2]; ok {
				continue A
			}
		}
		ret = append(ret, v)
	}
	return ret
}

func (c *ClassSigFilter) GetFields() []*ClassSigField {
	ret := make([]*ClassSigField, 0, len(c.Parser.GetFields()))
	for _, v := range c.Parser.GetFields() {
		if _, ok := c.filter[v.Type]; ok {
			continue
		}
		ret = append(ret, v)
	}
	return ret
}

func (c *ClassSigFilter) GetExtends() string {
    for _, v := range JavaTypeComponents(c.Parser.GetExtends()) {
        if _, ok := c.filter[v]; ok {
            return ""
        }
    }

    return c.Parser.GetExtends()
}

type SrcParams struct {
	Parser Parser
}

func (c *SrcParams) GetParams() Params {
	paramLen, startToken := c.Parser.ParamWords()
	paramLen = paramLen / 2

	params := make(Params, paramLen)
	for i := range params {
		if javaKeyWord(c.Parser.GetToken(startToken + i*2)) {
			startToken++
		}
		params[i].Name =  c.Parser.GetToken(startToken+1 + i*2)
		params[i].Type =  c.Parser.GetToken(startToken + i*2)
		if params[i].Name == "..." {
			startToken++
			params[i].Name =  c.Parser.GetToken(startToken+1 + i*2)
			params[i].Type = "..." + params[i].Type
		}
	}
	return params
}

type JavapParams struct {
	Parser Parser
}

func (c *JavapParams) GetParams() Params {
	paramLen, startToken := c.Parser.ParamWords()
	params := make(Params, paramLen)
	r := 'a'
	for i := range params {	
		params[i].Name =  fmt.Sprintf("%c", r)
		params[i].Type =  c.Parser.GetToken(startToken + i)
		r++
	}
	return params
}

var javaKeyWords = map[string]bool {
	"final":true,
	"static":true,
	"abstract":true,
	"public":true,
}

func javaKeyWord(w string) bool {
	if _, ok := javaKeyWords[w]; ok {
		return true
	}
	return false
}

func NewParser(h *ParserHandle, s *Statements, t *Tokens, c *ClassSig, p ParamParser, r io.Reader) Parser {
	o := &struct {
		*Statements
		*Tokens
		*ClassSig
		ParamParser
		io.Reader
	} {s, t, c, p, r}
	h.Parser = o
	return o
}
