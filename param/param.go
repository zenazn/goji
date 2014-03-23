/*
Package param deserializes parameter values into a given struct using magical
reflection ponies. Inspired by gorilla/schema, but uses Rails/jQuery style param
encoding instead of their weird dotted syntax. In particular, this package was
written with the intent of parsing the output of jQuery.param.

This package uses struct tags to guess what names things ought to have. If a
struct value has a "param" tag defined, it will use that. If there is no "param"
tag defined, the name part of the "json" tag will be used. If that is not
defined, the name of the field itself will be used (no case transformation is
performed).

If the name derived in this way is the string "-", param will refuse to set that
value.

The parser is extremely strict, and will return an error if it has any
difficulty whatsoever in parsing any parameter, or if there is any kind of type
mismatch.
*/
package param

import (
	"net/url"
	"reflect"
	"strings"
)

// Parse the given arguments into the the given pointer to a struct object.
func Parse(params url.Values, target interface{}) (err error) {
	v := reflect.ValueOf(target)

	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				panic(err)
			}
		}
	}()

	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		pebkac("Target of param.Parse must be a pointer to a struct. "+
			"We instead were passed a %v", v.Type())
	}

	el := v.Elem()
	t := el.Type()
	cache := cacheStruct(t)

	for key, values := range params {
		sk, keytail := key, ""
		if i := strings.IndexRune(key, '['); i != -1 {
			sk, keytail = sk[:i], sk[i:]
		}
		parseStructField(cache, key, sk, keytail, values, el)
	}

	return nil
}
