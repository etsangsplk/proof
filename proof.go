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
	t   T
	lax bool
}

func New(t T) *Prover {
	return &Prover{t: t}
}

func (p *Prover) Equal(x, y interface{}) {
	if !equal(x, y) {
		s := failureStringWithDiff(p.t, "Objects should be equal", x, y)
		p.t.Helper()
		if !p.lax {
			p.t.Fatal(s)
		} else {
			p.t.Error(s)
		}
	}
}

func (p *Prover) NotEqual(x, y interface{}) {
	if equal(x, y) {
		s := failureStringWithValues(p.t, "Objects should not be equal", x, y)
		p.t.Helper()
		if !p.lax {
			p.t.Fatal(s)
		} else {
			p.t.Error(s)
		}
	}
}

func (p *Prover) Err(err error) {
	if isNil(err) {
		s := failureStringWithValue(p.t, "Error should not be nil", err)
		p.t.Helper()
		if !p.lax {
			p.t.Fatal(s)
		} else {
			p.t.Error(s)
		}
	}
}

func (p *Prover) NotErr(err error) {
	if !isNil(err) {
		s := failureStringWithValue(p.t, "Error should be nil", err)
		p.t.Helper()
		if !p.lax {
			p.t.Fatal(s)
		} else {
			p.t.Error(s)
		}
	}
}

func (p *Prover) Nil(o interface{}) {
	if !isNil(o) {
		s := failureStringWithValue(p.t, "Object should be nil", o)
		p.t.Helper()
		if !p.lax {
			p.t.Fatal(s)
		} else {
			p.t.Error(s)
		}
	}
}

func (p *Prover) NotNil(o interface{}) {
	if isNil(o) {
		s := failureStringWithValue(p.t, "Object should not be nil", o)
		p.t.Helper()
		if !p.lax {
			p.t.Fatal(s)
		} else {
			p.t.Error(s)
		}
	}
}

func (p *Prover) True(b bool) {
	if !b {
		s := failureStringWithValue(p.t, "Bool should be true", b)
		p.t.Helper()
		if !p.lax {
			p.t.Fatal(s)
		} else {
			p.t.Error(s)
		}
	}
}

func (p *Prover) False(b bool) {
	if b {
		s := failureStringWithValue(p.t, "Bool should not be true", b)
		p.t.Helper()
		if !p.lax {
			p.t.Fatal(s)
		} else {
			p.t.Error(s)
		}
	}
}

func (p *Prover) Zero(o interface{}) {
	if !isZero(o) {
		s := failureStringWithValue(p.t, "Object should be zero value", o)
		p.t.Helper()
		if !p.lax {
			p.t.Fatal(s)
		} else {
			p.t.Error(s)
		}
	}
}

func (p *Prover) NotZero(o interface{}) {
	if isZero(o) {
		s := failureStringWithValue(p.t, "Object should not be zero value", o)
		p.t.Helper()
		if !p.lax {
			p.t.Fatal(s)
		} else {
			p.t.Error(s)
		}
	}
}

func (p *Prover) ContainedBySlice(object interface{}, slice interface{}) {
	sv := reflect.ValueOf(slice)
	sk := sv.Kind()
	if sk != reflect.Slice && sk != reflect.Array {
		p.t.Fatalf("ContainedBySlice received non-slice argument")
	}
	for i := 0; i < sv.Len(); i++ {
		if equal(sv.Index(i).Interface(), object) {
			return
		}
	}

	fs := failureStringWithValues(p.t, "Slice does not contain object", slice, object)
	p.t.Helper()
	if !p.lax {
		p.t.Fatal(fs)
	} else {
		p.t.Error(fs)
	}
}

func (p *Prover) Len(o interface{}, length int) {
	oLen, hasLen := getLen(o)
	if hasLen && oLen == length {
		return
	}

	var s string
	if !hasLen {
		s = failureStringWithValue(p.t, "Object was not of type array, slice, map or chan", o)
	} else {
		s = fmt.Sprintf("Expected object of length %d to be length %d", oLen, length)
		s = failureStringWithValue(p.t, s, o)
	}

	p.t.Helper()
	if !p.lax {
		p.t.Fatal(s)
	} else {
		p.t.Error(s)
	}
}

func (p *Prover) Panic(f func()) {
	defer func() {
		r := recover()
		if r == nil {
			p.t.Helper()
			if !p.lax {
				p.t.Fatal("Expected function to panic")
			} else {
				p.t.Error("Expected function to panic")
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
			p.t.Helper()
			if !p.lax {
				p.t.Fatalf("Expected function to return true within duration %v", duration)
			} else {
				p.t.Error("Expected function to panic")
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
		t:   p.t,
		lax: true,
	}

	fn(lax)

	if lax.t.Failed() {
		p.t.FailNow()
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
	return cmp.Equal(x, y, equateConvertibleNumbers())
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

func equateConvertibleNumbers() cmp.Option {
	return cmp.FilterValues(shouldTreatAsConvertibleNumbers, cmp.Comparer(equateAfterConverting))
}

func shouldTreatAsConvertibleNumbers(x, y interface{}) bool {
	xv := reflect.ValueOf(x)
	yv := reflect.ValueOf(y)

	if xv.Kind() == yv.Kind() {
		return false
	}

	if !isConvertibleNumber(xv) || !isConvertibleNumber(yv) {
		return false
	}
	if !xv.Type().ConvertibleTo(yv.Type()) {
		return false
	}

	return true
}

var convertibleNumberKinds = map[reflect.Kind]struct{}{
	reflect.Int:     {},
	reflect.Int8:    {},
	reflect.Int16:   {},
	reflect.Int32:   {},
	reflect.Int64:   {},
	reflect.Uint:    {},
	reflect.Uint8:   {},
	reflect.Uint16:  {},
	reflect.Uint32:  {},
	reflect.Uint64:  {},
	reflect.Uintptr: {},
	reflect.Float32: {},
	reflect.Float64: {},
}

func isConvertibleNumber(v reflect.Value) bool {
	_, exists := convertibleNumberKinds[v.Kind()]
	return exists
}

func equateAfterConverting(x, y interface{}) bool {
	xv := reflect.ValueOf(x)
	yv := reflect.ValueOf(y)
	return cmp.Equal(xv.Convert(yv.Type()).Interface(), y)
}
