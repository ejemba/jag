package jag

import (
	"bufio"
	"bytes"
	"io"
)

const (
	dataState = iota
	commentState = iota
	literalState = iota
)

type commentFilter struct {
	start, end, strLit string
	scanner *bufio.Scanner
	buf *bytes.Buffer
	state int
	tokenType string
	err error
}

func NewCommentFilter(start, end, strLit, quote string, r io.Reader) *commentFilter {
	scanner := bufio.NewScanner(r)
	filter := &commentFilter{start, end, strLit, scanner, new(bytes.Buffer), dataState, "", nil}
	nextTokenType := ""

	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		offset := len(nextTokenType)
		search := data[offset:]
		filter.tokenType = nextTokenType
		min := len(search)
		for _, str := range []string{start, end, strLit} {
			n := bytes.Index(search, []byte(str))
			for str == strLit && n >= len(quote) && string(search[n-len(quote):n+len(strLit)]) == quote + strLit {
				n = bytes.Index(search[n+len(strLit):], []byte(strLit))
			}

			if n != -1 && n < min {
				nextTokenType = str
				min = n
			}
		}

		if min != len(search) {
			return offset+min, data[0:offset+min], nil
		} else if atEOF {
			return len(data), data, nil
		} else {
			return 0, nil, nil
		}
	})

	return filter
}

func (c *commentFilter) Read(p []byte) (n int, err error) {
	toRead := len(p) - c.buf.Len()

	for toRead > 0 {
		if c.scanner.Scan() == false {
			break
		}
		bytes := c.scanner.Bytes()

		if c.tokenType == c.start {
			if c.state != literalState {
				c.state = commentState
			}
		} else if c.tokenType == c.end {
			if c.state == commentState {
				c.state = dataState
				bytes = bytes[len(c.end):]
			}
		} else if c.tokenType == c.strLit {
			if c.state == dataState {
				c.state = literalState
			} else if c.state == literalState {
				c.state = dataState
			}
		}

		if c.state != commentState {
			toRead -= len(bytes)
			_, err = c.buf.Write(bytes)
			if err != nil {
				return
			}
		}
	}

	if c.buf.Len() == 0 {
		return 0, io.EOF
	}

	//	log.Printf("BUFFER: %s\n", c.buf.Bytes())

	writeP := p[0:0]
	n64, err := io.CopyN(bytes.NewBuffer(writeP), c.buf, int64(len(p)))
	return int(n64), err
}

