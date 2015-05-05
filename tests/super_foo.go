package gojvm_gen_package

import "github.com/timob/javabind"

type LocalSuperFoo struct {
	*javabind.Callable
}

// public local.SuperFoo()
func NewLocalSuperFoo() (*LocalSuperFoo) {

	obj, err := javabind.Env.NewInstanceStr("local.SuperFoo")
	if err != nil {
		panic(err)
	}
	return &LocalSuperFoo{&javabind.Callable{obj, javabind.Env}}
}

// public String SaySuper()
func (jbobject *LocalSuperFoo) SaySuper() string {
	jret, err := jbobject.CallObj("SaySuper", "java.lang.String")
	if err != nil {
		panic(err)
	}
	retconv := javabind.NewJavaToGoString()
	dst := new(string)
	retconv.Dest(dst)
	if err := retconv.Convert(jret); err != nil {
		panic(err)
	}
	retconv.CleanUp()
	return *dst
}

