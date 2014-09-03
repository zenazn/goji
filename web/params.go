package web

import (
	"sort"

	"code.google.com/p/go.net/context"
)

type key int

const (
	paramKey key = iota
	// The key used to communicate to the NotFound handler what methods would have
	// been allowed if they'd been provided.
	validMethodsKey
)

func URLParams(ctx context.Context) map[string]string {
	if u, ok := ctx.Value(paramKey).(map[string]string); ok {
		return u
	}
	return nil
}

func withURLParams(ctx context.Context, v map[string]string) context.Context {
	return context.WithValue(ctx, paramKey, v)
}

// ValidMethods can be used in a NotFound handler to get the list of valid methods
func ValidMethods(ctx context.Context) []string {
	if ms, ok := ctx.Value(validMethodsKey).(methodSet); ok {
		var methodsList []string
		for mname, meth := range validMethodsMap {
			if ms&methodSet(meth) != 0 {
				methodsList = append(methodsList, mname)
			}
		}
		sort.Strings(methodsList)

		return methodsList
	}
	return nil
}
