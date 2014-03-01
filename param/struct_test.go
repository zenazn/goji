package param

import (
	"reflect"
	"testing"
)

type Fruity struct {
	A bool
	B int            `json:"banana"`
	C uint           `param:"cherry"`
	D float64        `json:"durian" param:"dragonfruit"`
	E int            `json:"elderberry" param:"-"`
	F map[string]int `json:"-" param:"fig"`
	G *int           `json:"grape,omitempty"`
	H []int          `param:"honeydew" json:"huckleberry"`
	I string         `foobar:"iyokan"`
	J Cheesy         `param:"jackfruit" cheese:"jarlsberg"`
}

type Cheesy struct {
	A int `param:"affinois"`
	B int `param:"brie"`
	C int `param:"camembert"`
	D int `param:"delice d'argental"`
}

type Private struct {
	Public, private int
}

var fruityType = reflect.TypeOf(Fruity{})
var cheesyType = reflect.TypeOf(Cheesy{})
var privateType = reflect.TypeOf(Private{})

var fruityNames = []string{
	"A", "banana", "cherry", "dragonfruit", "-", "fig", "grape", "honeydew",
	"I", "jackfruit",
}

var fruityCache = map[string]cacheLine{
	"A":           {0, parseBool},
	"banana":      {1, parseInt},
	"cherry":      {2, parseUint},
	"dragonfruit": {3, parseFloat},
	"fig":         {5, parseMap},
	"grape":       {6, parsePtr},
	"honeydew":    {7, parseSlice},
	"I":           {8, parseString},
	"jackfruit":   {9, parseStruct},
}

func assertEqual(t *testing.T, what string, e, a interface{}) {
	if !reflect.DeepEqual(e, a) {
		t.Errorf("Expected %s to be %v, was actually %v", what, e, a)
	}
}

func TestNames(t *testing.T) {
	t.Parallel()

	for i, val := range fruityNames {
		name := extractName(fruityType.Field(i))
		assertEqual(t, "tag", val, name)
	}
}

func TestCacheStruct(t *testing.T) {
	t.Parallel()

	sc := cacheStruct(fruityType)

	if len(sc) != len(fruityCache) {
		t.Errorf("Cache has %d keys, but expected %d", len(sc),
			len(fruityCache))
	}

	for k, v := range fruityCache {
		sck, ok := sc[k]
		if !ok {
			t.Errorf("Could not find key %q in cache", k)
			continue
		}
		if sck.offset != v.offset {
			t.Errorf("Cache for %q: expected offset %d but got %d",
				k, sck.offset, v.offset)
		}
		// We want to compare function pointer equality, and this
		// appears to be the only way
		a := reflect.ValueOf(sck.parse)
		b := reflect.ValueOf(v.parse)
		if a.Pointer() != b.Pointer() {
			t.Errorf("Parse mismatch for %q: %v, expected %v", k, a,
				b)
		}
	}
}

func TestPrivate(t *testing.T) {
	t.Parallel()

	sc := cacheStruct(privateType)
	if len(sc) != 1 {
		t.Error("Expected Private{} to have one cachable field")
	}
}
