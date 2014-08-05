package jag

import (
	"gojvm"
	"gojvm/types"
	"reflect"
	"errors"
	"log"
)

//var debug = false
var Env *gojvm.Environment

func setupJVM(classPath ...string) (err error) {
	classPath = append(classPath, gojvm.DefaultJREPath)
	if debug {
		log.Printf("Using classpath %v", classPath)
	}
	jvm, env, err := gojvm.NewJVM(0, gojvm.JvmConfig{classPath})
	if err != nil {
		return err
	}
	if jvm == nil {
		return errors.New("Got a nil context!")
	}
	Env = env

	// expected exceptions are pre-muted/unmuted, but if you're testing something
	// that causes them to throw, and want readable tests, this is the line
	// to uncomment.
	//_Ctx.env.Mute(true)
	return
}

type Callable struct {
	obj	*gojvm.Object
	env *gojvm.Environment
}

func (c *Callable) callObj(method string, retType interface{}, args ...interface{}) *gojvm.Object {
	var t types.Typed
	switch v := retType.(type) {
	case string:
		t = types.Class{types.NewName(v)}
	case types.Typed:
		t = v
	default:
		panic("Callable.callObj unknown retType type")
	}
	obj, err := c.obj.CallObj(c.env, false, method, t, args...)
	if err != nil {
		log.Panic(err)
	}
	return obj
}

func (c *Callable) callCallable(method string, retType interface{}, args ...interface{}) *Callable {
	return &Callable{c.callObj(method, retType, args...), c.env}
}

func (c *Callable) callStr(method string, args ...interface{}) string {
	str, _, err := c.obj.CallString(c.env, false, method, args...)
	if err != nil {
		log.Panic(err)
	}
	return str
}

func (c *Callable) callInt(method string, args ...interface{}) int {
	n, err := c.obj.CallInt(c.env, false, method, args...)
	if err != nil {
		log.Panic(err)
	}
	return n
}

func (c *Callable) callLong(method string, args ...interface{}) int64 {
	n, err := c.obj.CallLong(c.env, false, method, args...)
	if err != nil {
		log.Panic(err)
	}
	return n
}

func (c *Callable) callBool(method string, args ...interface{}) bool {
	b, err := c.obj.CallBool(c.env, false, method, args...)
	if err != nil {
		log.Panic(err)
	}
	return b
}

func (c *Callable) callVoid(method string, args ...interface{}) {
	err := c.obj.CallVoid(c.env, false, method, args...)
	if err != nil {
		log.Panic(err)
	}
}

func (c *Callable) callFloat(method string, args ...interface{}) float32 {
	n, err := c.obj.CallFloat(c.env, false, method, args...)
	if err != nil {
		log.Panic(err)
	}
	return n
}

func (c *Callable) callDouble(method string, args ...interface{}) float64 {
	n, err := c.obj.CallDouble(c.env, false, method, args...)
	if err != nil {
		log.Panic(err)
	}
	return n
}

type callableContainer interface {
	getCallable() *Callable
}

func (c *Callable) getCallable() *Callable {
	return c
}

func size(env *gojvm.Environment, obj *gojvm.Object) (len int, err error) {
	len, err =  obj.CallInt(env, false, "size")
	if err != nil {
		return
	}
	return
}

type ToJavaConverter interface  {
	Convert(value interface{}) error
	Value() *gojvm.Object
	CleanUp() error
}

type FromJavaConverter interface  {
	Dest(ptr interface{})
	Convert(obj *gojvm.Object) error
	CleanUp() error
}

type GoToJavaCallable struct {
	obj	*gojvm.Object
}

func (g *GoToJavaCallable) Convert(value interface{}) (err error) {
	g.obj = value.(callableContainer).getCallable().obj
	return
}

func (g *GoToJavaCallable) Value() *gojvm.Object {
	return g.obj
}

func (g *GoToJavaCallable) CleanUp() error {
	return nil
}

type JavaToGoCallable struct {
	callable *Callable
}

func (j *JavaToGoCallable) Dest(ptr interface{}) {
	j.callable = ptr.(callableContainer).getCallable()
}

func (j *JavaToGoCallable) Convert(obj *gojvm.Object) (err error) {
	j.callable.obj = obj
	return
}

func (j *JavaToGoCallable) CleanUp() error {
	return nil
}

type GoToJavaString struct {
	obj	*gojvm.Object
	env *gojvm.Environment
}

func NewGoToJavaString() *GoToJavaString {
	return &GoToJavaString{env: Env}
}

func (g *GoToJavaString) Convert(value interface{}) (err error) {
	g.obj, err = g.env.NewStringObject(value.(string))
	return
}

func (g *GoToJavaString) Value() *gojvm.Object {
	return g.obj
}

func (g *GoToJavaString) CleanUp() error {
	return nil
}

type JavaToGoString struct {
	str *string
	env *gojvm.Environment
}

func NewJavaToGoString() *JavaToGoString {
	return &JavaToGoString{env: Env}
}

func (j *JavaToGoString) Dest(ptr interface{}) {
	j.str = ptr.(*string)
}

func (j *JavaToGoString) Convert(obj *gojvm.Object) (err error) {
	x, _, err := j.env.ToString(obj)
	*j.str = x
	return
}

func (j *JavaToGoString) CleanUp() (err error) {
	return
}

type GoToJavaList struct {
	obj	*gojvm.Object
	env *gojvm.Environment
	item ToJavaConverter
}

func NewGoToJavaList(item ToJavaConverter) *GoToJavaList {
	return &GoToJavaList{env: Env, item: item}
}

func (g *GoToJavaList) Convert(value interface{}) (err error) {
	listObj, err := g.env.NewInstanceStr("java/util/ArrayList")
	if err != nil {
		return
	}

	r_value := reflect.ValueOf(value)
	if r_value.Type().Kind() != reflect.Slice {
		return errors.New("GoToJavaList.Convert: value not slice")
	}
	n := r_value.Len()
	for i := 0; i < n; i++ {
		if err = g.item.Convert(r_value.Index(i).Interface()); err != nil {
			return
		}
		listObj.CallBool(g.env, false, "add", &gojvm.CastObject{g.item.Value(), types.JavaLangObject})
		if err = g.item.CleanUp(); err != nil {
			return
		}
	}

	return
}

func (g *GoToJavaList) Value() *gojvm.Object {
	return g.obj
}

func (g *GoToJavaList) CleanUp() error {
	return nil
}

type JavaToGoList struct {
	list interface{}
	env *gojvm.Environment
	item FromJavaConverter
}

func NewJavaToGoList(item FromJavaConverter) *JavaToGoList {
	return &JavaToGoList{env: Env, item: item}
}

func (j *JavaToGoList) Dest(ptr interface{}) {
	j.list = ptr
}

func (j *JavaToGoList) Convert(obj *gojvm.Object) (err error) {
	r_value := reflect.ValueOf(j.list)

	if r_value.Type().Kind() != reflect.Ptr {
		return errors.New("JavaToGoList.Convert: dest not ptr")
	}

	r_slice := reflect.Indirect(r_value)
	if r_slice.Type().Kind() != reflect.Slice {
		return errors.New("JavaToGoList.Convert: dest ptr , does not point to slice")
	}

	len, err := size(j.env, obj)
	if err != nil {
		return
	}
	for i := 0; i < len; i++ {
		itemObj, err := obj.CallObj(j.env, false, "get", types.Class{types.JavaLangObject}, i)
		if err != nil {
			return err
		}

		r_newElem := reflect.Indirect(reflect.New(r_slice.Type().Elem()))
		j.item.Dest(r_newElem.Addr().Interface())
		if err = j.item.Convert(itemObj); err != nil {
			return err
		}
		if err = j.item.CleanUp(); err != nil {
			return err
		}

		r_newSlice := reflect.Append(r_slice, r_newElem)
		r_slice.Set(r_newSlice)
	}

	return
}

func (j *JavaToGoList) CleanUp() (err error) {
	return
}
