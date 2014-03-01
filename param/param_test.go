package param

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

type Everything struct {
	Bool   bool
	Int    int
	Uint   uint
	Float  float64
	Map    map[string]int
	Slice  []int
	String string
	Struct Sub
	Time   time.Time

	PBool   *bool
	PInt    *int
	PUint   *uint
	PFloat  *float64
	PMap    *map[string]int
	PSlice  *[]int
	PString *string
	PStruct *Sub
	PTime   *time.Time

	PPInt **int

	ABool   MyBool
	AInt    MyInt
	AUint   MyUint
	AFloat  MyFloat
	AMap    MyMap
	APtr    MyPtr
	ASlice  MySlice
	AString MyString
}

type Sub struct {
	A int
	B int
}

type MyBool bool
type MyInt int
type MyUint uint
type MyFloat float64
type MyMap map[MyString]MyInt
type MyPtr *MyInt
type MySlice []MyInt
type MyString string

var boolAnswers = map[string]bool{
	"true":  true,
	"false": false,
	"0":     false,
	"1":     true,
	"on":    true,
	"":      false,
}

var testTimeString = "1996-12-19T16:39:57-08:00"
var testTime time.Time

func init() {
	testTime, _ = time.Parse(time.RFC3339, testTimeString)
}

func singletonErrors(t *testing.T, field, valid, invalid string) {
	e := Everything{}

	err := Parse(url.Values{field: {invalid}}, &e)
	if err == nil {
		t.Errorf("Expected error parsing %q as %s", invalid, field)
	}

	err = Parse(url.Values{field + "[]": {valid}}, &e)
	if err == nil {
		t.Errorf("Expected error parsing nested %s", field)
	}

	err = Parse(url.Values{field + "[nested]": {valid}}, &e)
	if err == nil {
		t.Errorf("Expected error parsing nested %s", field)
	}

	err = Parse(url.Values{field: {valid, valid}}, &e)
	if err == nil {
		t.Errorf("Expected error passing %s twice", field)
	}
}

func TestBool(t *testing.T) {
	t.Parallel()

	for val, correct := range boolAnswers {
		e := Everything{}
		e.Bool = !correct
		err := Parse(url.Values{"Bool": {val}}, &e)
		if err != nil {
			t.Error("Parse error on key: ", val)
		}
		assertEqual(t, "e.Bool", correct, e.Bool)
	}
}

func TestBoolTyped(t *testing.T) {
	t.Parallel()

	e := Everything{}
	err := Parse(url.Values{"ABool": {"true"}}, &e)
	if err != nil {
		t.Error("Parse error for typed bool")
	}
	assertEqual(t, "e.ABool", MyBool(true), e.ABool)
}

func TestBoolErrors(t *testing.T) {
	t.Parallel()
	singletonErrors(t, "Bool", "true", "llama")
}

var intAnswers = map[string]int{
	"0":    0,
	"9001": 9001,
	"-42":  -42,
}

func TestInt(t *testing.T) {
	t.Parallel()

	for val, correct := range intAnswers {
		e := Everything{}
		e.Int = 1
		err := Parse(url.Values{"Int": {val}}, &e)
		if err != nil {
			t.Error("Parse error on key: ", val)
		}
		assertEqual(t, "e.Int", correct, e.Int)
	}
}

func TestIntTyped(t *testing.T) {
	t.Parallel()

	e := Everything{}
	err := Parse(url.Values{"AInt": {"1"}}, &e)
	if err != nil {
		t.Error("Parse error for typed int")
	}
	assertEqual(t, "e.AInt", MyInt(1), e.AInt)
}

func TestIntErrors(t *testing.T) {
	t.Parallel()
	singletonErrors(t, "Int", "1", "llama")

	e := Everything{}
	err := Parse(url.Values{"Int": {"4.2"}}, &e)
	if err == nil {
		t.Error("Expected error parsing float as int")
	}
}

var uintAnswers = map[string]uint{
	"0":    0,
	"9001": 9001,
}

func TestUint(t *testing.T) {
	t.Parallel()

	for val, correct := range uintAnswers {
		e := Everything{}
		e.Uint = 1
		err := Parse(url.Values{"Uint": {val}}, &e)
		if err != nil {
			t.Error("Parse error on key: ", val)
		}
		assertEqual(t, "e.Uint", correct, e.Uint)
	}
}

func TestUintTyped(t *testing.T) {
	t.Parallel()

	e := Everything{}
	err := Parse(url.Values{"AUint": {"1"}}, &e)
	if err != nil {
		t.Error("Parse error for typed uint")
	}
	assertEqual(t, "e.AUint", MyUint(1), e.AUint)
}

func TestUintErrors(t *testing.T) {
	t.Parallel()
	singletonErrors(t, "Uint", "1", "llama")

	e := Everything{}
	err := Parse(url.Values{"Uint": {"4.2"}}, &e)
	if err == nil {
		t.Error("Expected error parsing float as uint")
	}

	err = Parse(url.Values{"Uint": {"-42"}}, &e)
	if err == nil {
		t.Error("Expected error parsing negative number as uint")
	}
}

var floatAnswers = map[string]float64{
	"0":         0,
	"9001":      9001,
	"-42":       -42,
	"9001.0":    9001.0,
	"4.2":       4.2,
	"-9.000001": -9.000001,
}

func TestFloat(t *testing.T) {
	t.Parallel()

	for val, correct := range floatAnswers {
		e := Everything{}
		e.Float = 1
		err := Parse(url.Values{"Float": {val}}, &e)
		if err != nil {
			t.Error("Parse error on key: ", val)
		}
		assertEqual(t, "e.Float", correct, e.Float)
	}
}

func TestFloatTyped(t *testing.T) {
	t.Parallel()

	e := Everything{}
	err := Parse(url.Values{"AFloat": {"1.0"}}, &e)
	if err != nil {
		t.Error("Parse error for typed float")
	}
	assertEqual(t, "e.AFloat", MyFloat(1.0), e.AFloat)
}

func TestFloatErrors(t *testing.T) {
	t.Parallel()
	singletonErrors(t, "Float", "1.0", "llama")
}

func TestMap(t *testing.T) {
	t.Parallel()
	e := Everything{}

	err := Parse(url.Values{
		"Map[one]":   {"1"},
		"Map[two]":   {"2"},
		"Map[three]": {"3"},
	}, &e)
	if err != nil {
		t.Error("Parse error in map: ", err)
	}

	for k, v := range map[string]int{"one": 1, "two": 2, "three": 3} {
		if mv, ok := e.Map[k]; !ok {
			t.Errorf("Key %q not in map", k)
		} else {
			assertEqual(t, "Map["+k+"]", v, mv)
		}
	}
}

func TestMapTyped(t *testing.T) {
	t.Parallel()

	e := Everything{}
	err := Parse(url.Values{"AMap[one]": {"1"}}, &e)
	if err != nil {
		t.Error("Parse error for typed map")
	}
	assertEqual(t, "e.AMap[one]", MyInt(1), e.AMap[MyString("one")])
}

func TestMapErrors(t *testing.T) {
	t.Parallel()
	e := Everything{}

	err := Parse(url.Values{"Map[]": {"llama"}}, &e)
	if err == nil {
		t.Error("expected error parsing empty map key")
	}

	err = Parse(url.Values{"Map": {"llama"}}, &e)
	if err == nil {
		t.Error("expected error parsing map without key")
	}

	err = Parse(url.Values{"Map[": {"llama"}}, &e)
	if err == nil {
		t.Error("expected error parsing map with malformed key")
	}
}

func testPtr(t *testing.T, key, in string, out interface{}) {
	e := Everything{}

	err := Parse(url.Values{key: {in}}, &e)
	if err != nil {
		t.Errorf("Parse error while parsing pointer e.%s: %v", key, err)
	}
	fieldKey := key
	if i := strings.IndexRune(fieldKey, '['); i >= 0 {
		fieldKey = fieldKey[:i]
	}
	v := reflect.ValueOf(e).FieldByName(fieldKey)
	if v.IsNil() {
		t.Errorf("Expected param to allocate pointer for e.%s", key)
	} else {
		assertEqual(t, "*e."+key, out, v.Elem().Interface())
	}
}

func TestPtr(t *testing.T) {
	t.Parallel()
	testPtr(t, "PBool", "true", true)
	testPtr(t, "PInt", "2", 2)
	testPtr(t, "PUint", "2", uint(2))
	testPtr(t, "PFloat", "2.0", 2.0)
	testPtr(t, "PMap[llama]", "4", map[string]int{"llama": 4})
	testPtr(t, "PSlice[]", "4", []int{4})
	testPtr(t, "PString", "llama", "llama")
	testPtr(t, "PStruct[B]", "2", Sub{0, 2})
	testPtr(t, "PTime", testTimeString, testTime)

	foo := 2
	testPtr(t, "PPInt", "2", &foo)
}

func TestPtrTyped(t *testing.T) {
	t.Parallel()

	e := Everything{}
	err := Parse(url.Values{"APtr": {"1"}}, &e)
	if err != nil {
		t.Error("Parse error for typed pointer")
	}
	assertEqual(t, "e.APtr", MyInt(1), *e.APtr)
}

func TestSlice(t *testing.T) {
	t.Parallel()

	e := Everything{}
	err := Parse(url.Values{"Slice[]": {"3", "1", "4"}}, &e)
	if err != nil {
		t.Error("Parse error for slice")
	}
	if e.Slice == nil {
		t.Fatal("Expected param to allocate a slice")
	}
	if len(e.Slice) != 3 {
		t.Fatal("Expected a slice of length 3")
	}

	assertEqual(t, "e.Slice[0]", 3, e.Slice[0])
	assertEqual(t, "e.Slice[1]", 1, e.Slice[1])
	assertEqual(t, "e.Slice[2]", 4, e.Slice[2])
}

func TestSliceTyped(t *testing.T) {
	t.Parallel()
	e := Everything{}
	err := Parse(url.Values{"ASlice[]": {"3", "1", "4"}}, &e)
	if err != nil {
		t.Error("Parse error for typed slice")
	}
	if e.ASlice == nil {
		t.Fatal("Expected param to allocate a slice")
	}
	if len(e.ASlice) != 3 {
		t.Fatal("Expected a slice of length 3")
	}

	assertEqual(t, "e.ASlice[0]", MyInt(3), e.ASlice[0])
	assertEqual(t, "e.ASlice[1]", MyInt(1), e.ASlice[1])
	assertEqual(t, "e.ASlice[2]", MyInt(4), e.ASlice[2])
}

func TestSliceErrors(t *testing.T) {
	t.Parallel()
	e := Everything{}
	err := Parse(url.Values{"Slice": {"1"}}, &e)
	if err == nil {
		t.Error("expected error parsing slice without key")
	}

	err = Parse(url.Values{"Slice[llama]": {"1"}}, &e)
	if err == nil {
		t.Error("expected error parsing slice with string key")
	}

	err = Parse(url.Values{"Slice[": {"1"}}, &e)
	if err == nil {
		t.Error("expected error parsing malformed slice key")
	}
}

var stringAnswer = "This is the world's best string"

func TestString(t *testing.T) {
	t.Parallel()
	e := Everything{}

	err := Parse(url.Values{"String": {stringAnswer}}, &e)
	if err != nil {
		t.Error("Parse error in string: ", err)
	}

	assertEqual(t, "e.String", stringAnswer, e.String)
}

func TestStringTyped(t *testing.T) {
	t.Parallel()

	e := Everything{}
	err := Parse(url.Values{"AString": {"llama"}}, &e)
	if err != nil {
		t.Error("Parse error for typed string")
	}
	assertEqual(t, "e.AString", MyString("llama"), e.AString)
}

func TestStruct(t *testing.T) {
	t.Parallel()
	e := Everything{}

	err := Parse(url.Values{
		"Struct[A]": {"1"},
	}, &e)
	if err != nil {
		t.Error("Parse error in struct: ", err)
	}
	assertEqual(t, "e.Struct.A", 1, e.Struct.A)
	assertEqual(t, "e.Struct.B", 0, e.Struct.B)

	err = Parse(url.Values{
		"Struct[A]": {"4"},
		"Struct[B]": {"2"},
	}, &e)
	if err != nil {
		t.Error("Parse error in struct: ", err)
	}
	assertEqual(t, "e.Struct.A", 4, e.Struct.A)
	assertEqual(t, "e.Struct.B", 2, e.Struct.B)
}

func TestStructErrors(t *testing.T) {
	t.Parallel()
	e := Everything{}

	err := Parse(url.Values{"Struct[]": {"llama"}}, &e)
	if err == nil {
		t.Error("expected error parsing empty struct key")
	}

	err = Parse(url.Values{"Struct": {"llama"}}, &e)
	if err == nil {
		t.Error("expected error parsing struct without key")
	}

	err = Parse(url.Values{"Struct[": {"llama"}}, &e)
	if err == nil {
		t.Error("expected error parsing malformed struct key")
	}

	err = Parse(url.Values{"Struct[C]": {"llama"}}, &e)
	if err == nil {
		t.Error("expected error parsing unknown")
	}
}

func TestTextUnmarshaler(t *testing.T) {
	t.Parallel()
	e := Everything{}

	err := Parse(url.Values{"Time": {testTimeString}}, &e)
	if err != nil {
		t.Error("parse error for TextUnmarshaler (Time): ", err)
	}
	assertEqual(t, "e.Time", testTime, e.Time)
}

func TestTextUnmarshalerError(t *testing.T) {
	t.Parallel()
	e := Everything{}

	err := Parse(url.Values{"Time": {"llama"}}, &e)
	if err == nil {
		t.Error("expected error parsing llama as time")
	}
}
