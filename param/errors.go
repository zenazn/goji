package param

import (
	"fmt"
	"reflect"
)

// TypeError is an error type returned when param has difficulty deserializing a
// parameter value.
type TypeError struct {
	// The key that was in error.
	Key string
	// The type that was expected.
	Type reflect.Type
	// The underlying error produced as part of the deserialization process,
	// if one exists.
	Err error
}

func (t TypeError) Error() string {
	return fmt.Sprintf("param: error parsing key %q as %v: %v", t.Key, t.Type,
		t.Err)
}

// SingletonError is an error type returned when a parameter is passed multiple
// times when only a single value is expected. For example, for a struct with
// integer field "foo", "foo=1&foo=2" will return a SingletonError with key
// "foo".
type SingletonError struct {
	// The key that was in error.
	Key string
	// The type that was expected for that key.
	Type reflect.Type
	// The list of values that were provided for that key.
	Values []string
}

func (s SingletonError) Error() string {
	return fmt.Sprintf("param: error parsing key %q: expected single "+
		"value but was given %d: %v", s.Key, len(s.Values), s.Values)
}

// NestingError is an error type returned when a key is nested when the target
// type does not support nesting of the given type. For example, deserializing
// the parameter key "anint[foo]" into a struct that defines an integer param
// "anint" will produce a NestingError with key "anint" and nesting "[foo]".
type NestingError struct {
	// The portion of the key that was correctly parsed into a value.
	Key string
	// The type of the key that was invalidly nested on.
	Type reflect.Type
	// The portion of the key that could not be parsed due to invalid
	// nesting.
	Nesting string
}

func (n NestingError) Error() string {
	return fmt.Sprintf("param: error parsing key %q: invalid nesting "+
		"%q on %s key %q", n.Key+n.Nesting, n.Nesting, n.Type, n.Key)
}

// SyntaxErrorSubtype describes what sort of syntax error was encountered.
type SyntaxErrorSubtype int

const (
	MissingOpeningBracket SyntaxErrorSubtype = iota + 1
	MissingClosingBracket
)

// SyntaxError is an error type returned when a key is incorrectly formatted.
type SyntaxError struct {
	// The key for which there was a syntax error.
	Key string
	// The subtype of the syntax error, which describes what sort of error
	// was encountered.
	Subtype SyntaxErrorSubtype
	// The part of the key (generally the suffix) that was in error.
	ErrorPart string
}

func (s SyntaxError) Error() string {
	prefix := fmt.Sprintf("param: syntax error while parsing key %q: ",
		s.Key)

	switch s.Subtype {
	case MissingOpeningBracket:
		return prefix + fmt.Sprintf("expected opening bracket, got %q",
			s.ErrorPart)
	case MissingClosingBracket:
		return prefix + fmt.Sprintf("expected closing bracket in %q",
			s.ErrorPart)
	default:
		panic("switch is not exhaustive!")
	}
}

// KeyError is an error type returned when an unknown field is set on a struct.
type KeyError struct {
	// The full key that was in error.
	FullKey string
	// The key of the struct that did not have the given field.
	Key string
	// The type of the struct that did not have the given field.
	Type reflect.Type
	// The name of the field which was not present.
	Field string
}

func (k KeyError) Error() string {
	return fmt.Sprintf("param: error parsing key %q: unknown field %q on "+
		"struct %q of type %v", k.FullKey, k.Field, k.Key, k.Type)
}
