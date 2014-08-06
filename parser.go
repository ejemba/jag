package jag

import (
	"io"	
	"strings"
	"bufio"
	"fmt"
	"bytes"
	"log"
	"sliceutil"
	"regexp"
)

var debug = false

func capitalize(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:]
}

type ParamParser interface {
	GetParams() Params
}

type Parser interface {
	GetStatement() string
	ParseStatement()
	GetToken(int) string
	ScopeDepth() int
	GetCurrentStatement() string
	ParamWords() (count int, start int)
	FindToken(token string) (pos int, found bool)
	Parse()
	Scan()
	io.Reader
	ParamParser
	GetClassSignature() *ClassSig
}

type ParserHandle struct {
		Parser
}


func JavaTypeComponents(j string) (p []string) {
	if strings.HasSuffix(j, "...") {
		return []string{"...", strings.TrimSuffix(string(j), "...")}
	}

	s := bufio.NewScanner(bytes.NewBufferString(strings.Replace(j, " ", "", -1)))
	var scopeDepth int
	s.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
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

var classNameRe = regexp.MustCompile(`[A-z,a-z]+\.`)

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

type Statements struct {
	stmts chan string	
	scopeDepth int
	Parser Parser
}

func NewStatements(g Parser) (a *Statements) {
	a = &Statements{make(chan string, 0), 0, g}
	return
}

func (s *Statements) GetStatement() string {
	return <- s.stmts
}

func (s *Statements) Scan() {
	go s.Parser.Parse()

	scanner := bufio.NewScanner(s.Parser)
	
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if i := bytes.IndexAny(data, ";{}"); i >= 0 {
			if string(data[i]) == "{" {
				s.scopeDepth++
			} else if string(data[i]) == "}" {
				s.scopeDepth--
			}
			return i + 1, data[0:i], nil
		} else if atEOF {
			return len(data), nil, nil
		}
		return 0, nil, nil
	})
	
	for scanner.Scan() {
		stmt := string(bytes.Join(bytes.Fields(scanner.Bytes()), []byte{' '}))
		s.stmts <- stmt
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
		log.Printf("%v", tokens)
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

type ClassSig struct {
	PackageName string
	ClassName string
	Constructors []*struct{
		Params Params
		Throws bool
		Line string
	}
	Methods []*struct{
		Name string
		Params Params
		Return string
		Throws bool
		Line string
	}
	Parser Parser
}

func (c *ClassSig) Parse() {	
	for {		
		c.Parser.ParseStatement()
		
		if c.Parser.GetToken(0) == "package" {
			c.PackageName = c.Parser.GetToken(1)		
		}
		
		if c.Parser.GetToken(0) != "public" ||
		   (c.Parser.GetToken(1) != "class" && c.Parser.GetToken(1) != "interface") {
			continue
		}

		if strings.Contains(c.Parser.GetToken(2), "<") {
			continue
		}

		c.ClassName = c.Parser.GetToken(2)
		for c.Parser.ScopeDepth() > 0 {
			c.Parser.ParseStatement()
			if c.Parser.ScopeDepth() > 2 {
				continue
			}
			if c.Parser.GetToken(0) != "public" {
				continue
			}
			fnk := c.FirstNonKeyWord()
			if c.Parser.GetToken(fnk) == c.ClassName {
				i := sliceutil.Append(&c.Constructors)
				c.Constructors[i].Params = c.Parser.GetParams()
				c.Constructors[i].Throws = c.Throws()
				c.Constructors[i].Line = c.Parser.GetCurrentStatement()

			} else if c.Parser.GetToken(fnk+2) == "(" {
				i := sliceutil.Append(&c.Methods)
				c.Methods[i].Name = c.Parser.GetToken(fnk+1)
				c.Methods[i].Params = c.Parser.GetParams()
				c.Methods[i].Return = c.Parser.GetToken(fnk)
				c.Methods[i].Throws = c.Throws()
				c.Methods[i].Line = c.Parser.GetCurrentStatement()
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
	start, _ = c.Parser.FindToken("(")
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

func (c *ClassSig) GetClassSignature() *ClassSig {
	return c
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
