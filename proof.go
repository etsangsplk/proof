package proof

import (
	"fmt"
	"reflect"
	"runtime/debug"
	"time"

	"github.com/google/go-cmp/cmp"
)

type T interface {
	Log(args ...interface{})
	Error(args ...interface{})
	FailNow()
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Failed() bool
	Helper()
}

type Prover struct {
	T   T
	lax bool
}

func New(t T) *Prover {
	return &Prover{T: t}
}

func (p *Prover) Equal(x, y interface{}) {
	if !equal(x, y) {
		s := failureStringWithDiff(p.T, "Objects should be equal", x, y)
		p.T.Helper()
		if !p.lax {
			p.T.Fatal(s)
		} else {
			p.T.Error(s)
		}
	}
}

func (p *Prover) EqualsAny(x interface{}, y ...interface{}) {
	var found bool
	for i := range y {
		if equal(x, y[i]) {
			found = true
			break
		}
	}

	if !found {
		s := failureStringWithDiff(p.T, "One of the list of objects should be equal to the first argument", x, y)
		p.T.Helper()
		if !p.lax {
			p.T.Fatal(s)
		} else {
			p.T.Error(s)
		}
	}
}

func (p *Prover) NotEqual(x, y interface{}) {
	if equal(x, y) {
		s := failureStringWithValues(p.T, "Objects should not be equal", x, y)
		p.T.Helper()
		if !p.lax {
			p.T.Fatal(s)
		} else {
			p.T.Error(s)
		}
	}
}

func (p *Prover) Err(err error) {
	if isNil(err) {
		s := failureStringWithValue(p.T, "Error should not be nil", err)
		p.T.Helper()
		if !p.lax {
			p.T.Fatal(s)
		} else {
			p.T.Error(s)
		}
	}
}

func (p *Prover) NotErr(err error) {
	if !isNil(err) {
		s := failureStringWithValue(p.T, "Error should be nil", err)
		p.T.Helper()
		if !p.lax {
			p.T.Fatal(s)
		} else {
			p.T.Error(s)
		}
	}
}

func (p *Prover) Nil(o interface{}) {
	if !isNil(o) {
		s := failureStringWithValue(p.T, "Object should be nil", o)
		p.T.Helper()
		if !p.lax {
			p.T.Fatal(s)
		} else {
			p.T.Error(s)
		}
	}
}

func (p *Prover) NotNil(o interface{}) {
	if isNil(o) {
		s := failureStringWithValue(p.T, "Object should not be nil", o)
		p.T.Helper()
		if !p.lax {
			p.T.Fatal(s)
		} else {
			p.T.Error(s)
		}
	}
}

func (p *Prover) True(b bool) {
	if !b {
		s := failureStringWithValue(p.T, "Bool should be true", b)
		p.T.Helper()
		if !p.lax {
			p.T.Fatal(s)
		} else {
			p.T.Error(s)
		}
	}
}

func (p *Prover) False(b bool) {
	if b {
		s := failureStringWithValue(p.T, "Bool should not be true", b)
		p.T.Helper()
		if !p.lax {
			p.T.Fatal(s)
		} else {
			p.T.Error(s)
		}
	}
}

func (p *Prover) Zero(o interface{}) {
	if !isZero(o) {
		s := failureStringWithValue(p.T, "Object should be zero value", o)
		p.T.Helper()
		if !p.lax {
			p.T.Fatal(s)
		} else {
			p.T.Error(s)
		}
	}
}

func (p *Prover) NotZero(o interface{}) {
	if isZero(o) {
		s := failureStringWithValue(p.T, "Object should not be zero value", o)
		p.T.Helper()
		if !p.lax {
			p.T.Fatal(s)
		} else {
			p.T.Error(s)
		}
	}
}

func (p *Prover) ContainedBySlice(object interface{}, slice interface{}) {
	sv := reflect.ValueOf(slice)
	sk := sv.Kind()
	if sk != reflect.Slice && sk != reflect.Array {
		p.T.Fatalf("ContainedBySlice received non-slice argument")
	}
	for i := 0; i < sv.Len(); i++ {
		if equal(sv.Index(i).Interface(), object) {
			return
		}
	}

	fs := failureStringWithValues(p.T, "Slice does not contain object", slice, object)
	p.T.Helper()
	if !p.lax {
		p.T.Fatal(fs)
	} else {
		p.T.Error(fs)
	}
}

func (p *Prover) Len(o interface{}, length int) {
	oLen, hasLen := getLen(o)
	if hasLen && oLen == length {
		return
	}

	var s string
	if !hasLen {
		s = failureStringWithValue(p.T, "Object was not of type array, slice, map or chan", o)
	} else {
		s = fmt.Sprintf("Expected object of length %d to be length %d", oLen, length)
		s = failureStringWithValue(p.T, s, o)
	}

	p.T.Helper()
	if !p.lax {
		p.T.Fatal(s)
	} else {
		p.T.Error(s)
	}
}

func (p *Prover) Panic(f func()) {
	defer func() {
		r := recover()
		if r == nil {
			p.T.Helper()
			if !p.lax {
				p.T.Fatal("Expected function to panic")
			} else {
				p.T.Error("Expected function to panic")
			}
		}
	}()
	f()
}

func (p *Prover) Retry(duration time.Duration, f func() bool) {
	after := time.After(duration)
	for {
		select {
		case <-after:
			p.T.Helper()
			if !p.lax {
				p.T.Fatalf("Expected function to return true within duration %v", duration)
			} else {
				p.T.Error("Expected function to panic")
			}
			return
		default:
			if f() {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (p *Prover) Lax(fn func(lax *Prover)) {
	lax := &Prover{
		T:   p.T,
		lax: true,
	}

	fn(lax)

	if lax.T.Failed() {
		p.T.FailNow()
	}
}

func Recover(t T) {
	if r := recover(); r != nil {

		stack := debug.Stack()

		if !t.Failed() {
			t.Helper()
			fmt.Printf(fmt.Sprintf("panic: %v [recovered]\n\n", r))
			fmt.Println(string(stack))
			t.FailNow()
		}
	}
}

func equal(x, y interface{}) bool {
	allowAllUnexported := cmp.Exporter(func(reflect.Type) bool { return true })
	return cmp.Equal(x, y, equateConvertibleTypes(), allowAllUnexported)
}

func getLen(o interface{}) (int, bool) {
	v := reflect.ValueOf(o)
	k := v.Type().Kind()

	if o == nil ||
		(k != reflect.Array &&
			k != reflect.Slice &&
			k != reflect.Map &&
			k != reflect.Chan) {
		return 0, false
	}

	return v.Len(), true
}

func isNil(o interface{}) bool {
	if o == nil {
		return true
	}
	return isNilValue(reflect.ValueOf(o))
}

func isNilValue(v reflect.Value) bool {
	kind := v.Kind()
	if kind >= reflect.Chan &&
		kind <= reflect.Slice &&
		v.IsNil() {
		return true
	}
	return false
}

func isZero(o interface{}) bool {
	if o == nil {
		return true
	}

	v := reflect.ValueOf(o)

	if isNilValue(v) {
		return true
	}

	switch v.Kind() {
	case reflect.Ptr:
		return equal(o, zeroPtr(v))
	case reflect.Slice, reflect.Array, reflect.Map:
		if v.Len() == 0 {
			return true
		}
		return false
	default:
		return equal(o, zeroVal(v))
	}
}

func zeroPtr(v reflect.Value) interface{} {
	return reflect.New(v.Type().Elem()).Interface()
}

func zeroVal(v reflect.Value) interface{} {
	return reflect.Zero(v.Type()).Interface()
}

func failureStringWithValue(t T, msg string, o interface{}) string {
	s := fmt.Sprintf("(%T) %s (%v)", o, msg, o)
	return s
}

func failureStringWithValues(t T, msg string, x, y interface{}) string {
	s := fmt.Sprintf("(%T, %T) %s (%v, %v)", x, y, msg, x, y)
	return s
}

func failureStringWithDiff(t T, msg string, x, y interface{}) string {
	ds := diff(x, y)
	s := fmt.Sprintf("(%T, %T) %s", x, y, msg)
	if ds != "" {
		s += ":\n" + ds
	}
	return s
}

func diff(x, y interface{}) string {
	return cmp.Diff(x, y)
}

func equateConvertibleTypes() cmp.Option {
	return cmp.FilterValues(shouldTreatAsConvertibleTypes, cmp.Comparer(equateAfterConverting))
}

func shouldTreatAsConvertibleTypes(x, y interface{}) bool {
	if x == nil || y == nil {
		return false
	}

	xt := reflect.TypeOf(x)
	yt := reflect.TypeOf(y)

	if xt.Name() == yt.Name() {
		return false
	}

	if !xt.ConvertibleTo(yt) {
		return false
	}

	return true
}

func equateAfterConverting(x, y interface{}) bool {
	xv := reflect.ValueOf(x)
	yv := reflect.ValueOf(y)
	return cmp.Equal(xv.Convert(yv.Type()).Interface(), y)
}
