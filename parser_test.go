package jag

import (
	"testing"
)

func TestJavaTypeParse(t *testing.T) {
	name := JavaTypeComponents("java.util.Map<java.lang.String, java.util.Map<java.lang.String, java.lang.Integer>>")
	if name[0] != "java.util.Map" || name[1] != "java.lang.String" || name[2] != "java.util.Map<java.lang.String,java.lang.Integer>" {
		t.Fatal()
	}
}
