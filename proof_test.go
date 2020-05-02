package proof

import (
	"errors"
	"testing"
	"time"
)

type nopT struct {
	failed bool
}

func (t *nopT) Log(args ...interface{})                   {}
func (t *nopT) Error(args ...interface{})                 { t.failed = true }
func (t *nopT) FailNow()                                  { t.failed = true }
func (t *nopT) Fatal(args ...interface{})                 { t.failed = true }
func (t *nopT) Fatalf(format string, args ...interface{}) { t.failed = true }
func (t *nopT) Failed() bool                              { return t.failed }
func (t *nopT) Helper()                                   {}

func TestEqual(t *testing.T) {
	t.Parallel()

	type testStruct struct {
		Val string
	}

	testEqual := []struct {
		name string
		x    interface{}
		y    interface{}
	}{
		{name: "int", x: 1, y: 1},
		{name: "float", x: 1.0, y: 1.0},
		{name: "bool", x: true, y: true},
		{name: "int32/int64", x: int64(1), y: int32(1)},
		{name: "string", x: "fizzbuzz", y: "fizzbuzz"},
		{name: "slice", x: []string{"fizzbuzz"}, y: []string{"fizzbuzz"}},
		{name: "map", x: map[string]string{"fizz": "buzz"}, y: map[string]string{"fizz": "buzz"}},
		{name: "struct", x: testStruct{Val: "struct"}, y: testStruct{Val: "struct"}},
	}

	for _, test := range testEqual {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			prove := New(&nopT{})
			prove.lax = true

			prove.Equal(test.x, test.y)
			if prove.T.Failed() {
				t.Fatalf("'%s' arguments should have been considered equal", test.name)
			}

			prove.NotEqual(test.x, test.y)
			if !prove.T.Failed() {
				t.Fatalf("'%s' arguments should not have been considered equal", test.name)
			}
		})
	}

}

func TestNotEqual(t *testing.T) {
	t.Parallel()

	type testStruct struct {
		Val string
	}

	testEqual := []struct {
		name string
		x    interface{}
		y    interface{}
	}{
		{name: "int", x: 1, y: 2},
		{name: "float", x: 1.0, y: 2.0},
		{name: "bool", x: true, y: false},
		{name: "int32/int64", x: int64(1), y: int32(2)},
		{name: "string", x: "fizzbuzz", y: "buzzfizz"},
		{name: "slice", x: []string{"fizzbuzz"}, y: []string{"buzzfizz"}},
		{name: "map", x: map[string]string{"fizz": "buzz"}, y: map[string]string{"buzz": "fizz"}},
		{name: "struct", x: testStruct{Val: "struct"}, y: testStruct{Val: "val"}},
	}

	for _, test := range testEqual {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			prove := New(&nopT{})
			prove.lax = true

			prove.NotEqual(test.x, test.y)
			if prove.T.Failed() {
				t.Fatalf("'%s' arguments should have been considered not equal", test.name)
			}

			prove.Equal(test.x, test.y)
			if !prove.T.Failed() {
				t.Fatalf("'%s' arguments should not have been considered equal", test.name)
			}
		})
	}

}

func TestVarious(t *testing.T) {
	t.Parallel()

	prove := New(t)

	val := 10
	pVal := &val

	prove.EqualsAny(val, 10, 20)

	prove.Err(errors.New("error"))
	prove.NotErr(nil)
	prove.Nil(nil)
	prove.NotNil(pVal)
	prove.True(true)
	prove.False(false)
	prove.Zero("")
	prove.NotZero("not zero")

	slice := []string{"one", "two"}
	object := "two"
	prove.ContainedBySlice(object, slice)

	prove.Len([]string{"len"}, 1)
	prove.Panic(func() { panic("now is the perfect time to panic!") })

	start := time.Now()
	prove.Retry(200*time.Millisecond, func() bool {
		if time.Since(start) < 50*time.Millisecond {
			return false
		}
		return true
	})

	prove = New(&nopT{})
	prove.Lax(func(lax *Prover) {
		lax.Equal(1, 1)
		lax.Equal(1, 2)
	})
	if !prove.T.Failed() {
		t.Fatalf("lax call should triggered failed state")
	}
}

func TestRecover(t *testing.T) {
	t.Parallel()

	nt := &nopT{}

	defer func() {
		if !nt.Failed() {
			t.Fatal("Panic should have been recovered and set test state to failed.")
		}
	}()

	defer Recover(nt)

	panic("recover me!")
}
