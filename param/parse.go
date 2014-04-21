package param

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

var textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

// Generic parse dispatcher. This function's signature is the interface of all
// parse functions. `key` is the entire key that is currently being parsed, such
// as "foo[bar][]". `keytail` is the portion of the string that the current
// parser is responsible for, for instance "[bar][]". `values` is the list of
// values assigned to this key, and `target` is where the resulting typed value
// should be Set() to.
func parse(key, keytail string, values []string, target reflect.Value) {
	t := target.Type()
	if reflect.PtrTo(t).Implements(textUnmarshalerType) {
		parseTextUnmarshaler(key, keytail, values, target)
		return
	}

	switch k := target.Kind(); k {
	case reflect.Bool:
		parseBool(key, keytail, values, target)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parseInt(key, keytail, values, target)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parseUint(key, keytail, values, target)
	case reflect.Float32, reflect.Float64:
		parseFloat(key, keytail, values, target)
	case reflect.Map:
		parseMap(key, keytail, values, target)
	case reflect.Ptr:
		parsePtr(key, keytail, values, target)
	case reflect.Slice:
		parseSlice(key, keytail, values, target)
	case reflect.String:
		parseString(key, keytail, values, target)
	case reflect.Struct:
		parseStruct(key, keytail, values, target)

	default:
		pebkac("unsupported object of type %v and kind %v.",
			target.Type(), k)
	}
}

// We pass down both the full key ("foo[bar][]") and the part the current layer
// is responsible for making sense of ("[bar][]"). This computes the other thing
// you probably want to know, which is the path you took to get here ("foo").
func kpath(key, keytail string) string {
	l, t := len(key), len(keytail)
	return key[:l-t]
}

// Helper for validating that a value has been passed exactly once, and that the
// user is not attempting to nest on the key.
func primitive(tipe, key, keytail string, values []string) {
	if keytail != "" {
		perr("expected %s for key %q, got nested value", tipe,
			kpath(key, keytail))
	}
	if len(values) != 1 {
		perr("expected %s for key %q, but key passed %v times", tipe,
			kpath(key, keytail), len(values))
	}
}

func keyed(tipe reflect.Type, key, keytail string) (string, string) {
	idx := strings.IndexRune(keytail, ']')
	if keytail[0] != '[' || idx == -1 {
		perr("expected a square bracket delimited index for %q "+
			"(of type %v)", kpath(key, keytail), tipe)
	}
	return keytail[1:idx], keytail[idx+1:]
}

func parseTextUnmarshaler(key, keytail string, values []string, target reflect.Value) {
	primitive("encoding.TextUnmarshaler", key, keytail, values)

	tu := target.Addr().Interface().(encoding.TextUnmarshaler)
	err := tu.UnmarshalText([]byte(values[0]))
	if err != nil {
		perr("error while calling UnmarshalText on %v for key %q: %v",
			target.Type(), kpath(key, keytail), err)
	}
}

func parseBool(key, keytail string, values []string, target reflect.Value) {
	primitive("bool", key, keytail, values)

	switch values[0] {
	case "true", "1", "on":
		target.SetBool(true)
	case "false", "0", "":
		target.SetBool(false)
	default:
		perr("could not parse key %q as bool", kpath(key, keytail))
	}
}

func parseInt(key, keytail string, values []string, target reflect.Value) {
	primitive("int", key, keytail, values)

	t := target.Type()
	i, err := strconv.ParseInt(values[0], 10, t.Bits())
	if err != nil {
		perr("error parsing key %q as int: %v", kpath(key, keytail),
			err)
	}
	target.SetInt(i)
}

func parseUint(key, keytail string, values []string, target reflect.Value) {
	primitive("uint", key, keytail, values)

	t := target.Type()
	i, err := strconv.ParseUint(values[0], 10, t.Bits())
	if err != nil {
		perr("error parsing key %q as uint: %v", kpath(key, keytail),
			err)
	}
	target.SetUint(i)
}

func parseFloat(key, keytail string, values []string, target reflect.Value) {
	primitive("float", key, keytail, values)

	t := target.Type()
	f, err := strconv.ParseFloat(values[0], t.Bits())
	if err != nil {
		perr("error parsing key %q as float: %v", kpath(key, keytail),
			err)
	}

	target.SetFloat(f)
}

func parseString(key, keytail string, values []string, target reflect.Value) {
	primitive("string", key, keytail, values)

	target.SetString(values[0])
}

func parseSlice(key, keytail string, values []string, target reflect.Value) {
	// BUG(carl): We currently do not handle slices of nested types. If
	// support is needed, the implementation probably could be fleshed out.
	if keytail != "[]" {
		perr("unexpected array nesting for key %q: %q",
			kpath(key, keytail), keytail)
	}
	t := target.Type()

	slice := reflect.MakeSlice(t, len(values), len(values))
	kp := kpath(key, keytail)
	for i := range values {
		// We actually cheat a little bit and modify the key so we can
		// generate better debugging messages later
		key := fmt.Sprintf("%s[%d]", kp, i)
		parse(key, "", values[i:i+1], slice.Index(i))
	}
	target.Set(slice)
}

func parseMap(key, keytail string, values []string, target reflect.Value) {
	t := target.Type()
	mapkey, maptail := keyed(t, key, keytail)

	// BUG(carl): We don't support any map keys except strings, although
	// there's no reason we shouldn't be able to throw the value through our
	// unparsing stack.
	var mk reflect.Value
	if t.Key().Kind() == reflect.String {
		mk = reflect.ValueOf(mapkey).Convert(t.Key())
	} else {
		pebkac("key for map %v isn't a string (it's a %v).", t, t.Key())
	}

	if target.IsNil() {
		target.Set(reflect.MakeMap(t))
	}

	val := target.MapIndex(mk)
	if !val.IsValid() || !val.CanSet() {
		// It's a teensy bit annoying that the value returned by
		// MapIndex isn't Set()table if the key exists.
		val = reflect.New(t.Elem()).Elem()
	}
	parse(key, maptail, values, val)
	target.SetMapIndex(mk, val)
}

func parseStruct(key, keytail string, values []string, target reflect.Value) {
	t := target.Type()
	sk, skt := keyed(t, key, keytail)
	cache := cacheStruct(t)

	parseStructField(cache, key, sk, skt, values, target)
}

func parsePtr(key, keytail string, values []string, target reflect.Value) {
	t := target.Type()

	if target.IsNil() {
		target.Set(reflect.New(t.Elem()))
	}
	parse(key, keytail, values, target.Elem())
}
