package main

import (
	. ".."
	"github.com/timob/javabind"
	"fmt"
	"strings"
	"log"
	"time"
)

func main() {
	javabind.SetupJVM("./java_example/out/production/java_example/")
	
	foo, err := NewLocalFoo(false)
	if err != nil {
		log.Fatal(err)
	}
	
	out, err := foo.Method1(false, []string{"alpha"})
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("%v\n", out)
	
	bar := NewLocalBar()
	bar2 := foo.Method2(bar)
	fmt.Printf("%s\n", bar2.Hello())

	fmt.Printf("%s\n", foo.Method3("hello", "world"))
	
	stuff := foo.Method4([]string{"some", "stuff"})
	fmt.Printf("%s\n", strings.Join(stuff, ","))

	input := map[string]int{"a": 1, "b": 2, "c": 3}
	r := foo.Method5(input)
	for k, v := range r {
		fmt.Printf("%s - %s\n", k, v)
	}

	fmt.Printf("%d\n", LocalFooMethod8())
	
	objs := foo.Method11();
	fmt.Println(objs[0].Hello())
	
	fmt.Printf("%d\n", LocalFooAnswer())
	fmt.Printf("%s\n", LocalFooMybar().Hello())
	
	fmt.Printf("%s\n", foo.Method12(time.Now()))

//	x := LocalBar(*foo)
//	fmt.Printf("%v\n", &x)
	bars := foo.Method13()
	fmt.Printf("%s - %s\n", bars[0].Hello(), bars[1].Hello())

    fmt.Printf("%s\n", foo.SaySuper())

    _, err = foo.Method1(true, []string{"alpha"})
	if err != nil {
		log.Fatal(err)
	}	

}
