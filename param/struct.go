package param

import (
	"reflect"
	"strings"
	"sync"
)

// We decode a lot of structs (since it's the top-level thing this library
// decodes) and it takes a fair bit of work to reflect upon the struct to figure
// out what we want to do. Instead of doing this on every invocation, we cache
// metadata about each struct the first time we see it. The upshot is that we
// save some work every time. The downside is we are forced to briefly acquire
// a lock to access the cache in a thread-safe way. If this ever becomes a
// bottleneck, both the lock and the cache can be sharded or something.
type structCache map[string]cacheLine
type cacheLine struct {
	offset int
	parse  func(string, string, []string, reflect.Value)
}

var cacheLock sync.RWMutex
var cache = make(map[reflect.Type]structCache)

func cacheStruct(t reflect.Type) structCache {
	cacheLock.RLock()
	sc, ok := cache[t]
	cacheLock.RUnlock()

	if ok {
		return sc
	}

	// It's okay if two people build struct caches simultaneously
	sc = make(structCache)
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		// Only unexported fields have a PkgPath; we want to only cache
		// exported fields.
		if sf.PkgPath != "" {
			continue
		}
		name := extractName(sf)
		if name != "-" {
			sc[name] = cacheLine{i, extractHandler(t, sf)}
		}
	}

	cacheLock.Lock()
	cache[t] = sc
	cacheLock.Unlock()

	return sc
}

// Extract the name of the given struct field, looking at struct tags as
// appropriate.
func extractName(sf reflect.StructField) string {
	name := sf.Tag.Get("param")
	if name == "" {
		name = sf.Tag.Get("json")
		idx := strings.IndexRune(name, ',')
		if idx >= 0 {
			name = name[:idx]
		}
	}
	if name == "" {
		name = sf.Name
	}

	return name
}

func extractHandler(s reflect.Type, sf reflect.StructField) func(string, string, []string, reflect.Value) {
	if reflect.PtrTo(sf.Type).Implements(textUnmarshalerType) {
		return parseTextUnmarshaler
	}

	switch sf.Type.Kind() {
	case reflect.Bool:
		return parseBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return parseInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return parseUint
	case reflect.Float32, reflect.Float64:
		return parseFloat
	case reflect.Map:
		return parseMap
	case reflect.Ptr:
		return parsePtr
	case reflect.Slice:
		return parseSlice
	case reflect.String:
		return parseString
	case reflect.Struct:
		return parseStruct

	default:
		pebkac("struct %v has illegal field %q (type %v, kind %v).",
			s, sf.Name, sf.Type, sf.Type.Kind())
		return nil
	}
}

// We have to parse two types of structs: ones at the top level, whose keys
// don't have square brackets around them, and nested structs, which do.
func parseStructField(cache structCache, key, sk, keytail string, values []string, target reflect.Value) {
	l, ok := cache[sk]
	if !ok {
		panic(KeyError{
			FullKey: key,
			Key:     kpath(key, keytail),
			Type:    target.Type(),
			Field:   sk,
		})
	}
	f := target.Field(l.offset)

	l.parse(key, keytail, values, f)
}
