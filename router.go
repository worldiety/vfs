package vfs

import (
	"fmt"
	"io"
	"strings"
)

// The RoutingContext is used to represent a dispatching context or state.
type RoutingContext interface {
	// ValueOf returns the string value of a named parameter or the empty string if undefined
	ValueOf(name string) string

	// Path returns the actual path
	Path() Path

	// Args may contains additional arguments passed by invocation/dispatching
	Args() []interface{}
}

// A Router has a set of patterns which can be registered to be matched in the order of configuration.
type Router struct {
	matchers []matcher
}

// Dispatch tries to find the correct matcher for the given path. The first matching callback is invoked or if
// nothing matches, nothing is called at all (and false is returned). Returns io.EOF if no matcher can be applied.
func (r *Router) Dispatch(path Path, args ...interface{}) (interface{}, error) {
	for _, m := range r.matchers {
		matcher, err := m.apply(path)
		if err != nil {
			continue
		}
		// invoke, if the pattern matches and return
		return matcher.callback(matcher)
	}
	return nil, io.EOF
}

// DispatchReadBucket is required to workaround missing generics
func (r *Router) DispatchReadBucket(path Path, f func(ctx RoutingContext) (ResultSet, error)) (ResultSet, error) {
	res, err := r.Dispatch(path)
	if res == nil {
		return nil, err
	}
	if rs, ok := res.(ResultSet); ok {
		return rs, err
	}
	return nil, fmt.Errorf("cannot convert result: %v", err)
}

// DispatchOpen is required to workaround missing generics
func (r *Router) DispatchOpen(path Path, f func(ctx RoutingContext) (ResultSet, error)) (Blob, error) {
	res, err := r.Dispatch(path)
	if res == nil {
		return nil, err
	}
	if rs, ok := res.(Blob); ok {
		return rs, err
	}
	return nil, fmt.Errorf("cannot convert result: %v", err)
}

// Handle registers an arbitrary function with a pattern with injection-like semantics.
//
// Supported patterns are:
//  * * : matches everything
//  * /a/concrete/path : matches the exact path
//  * /{name} : matches anything like /a or /b
//  * /fix/{var}/fix : matches anything like /fix/a/fix or /fix/b/fix
func (r *Router) Handle(pattern string, callback func(ctx RoutingContext) (interface{}, error)) {
	r.matchers = append(r.matchers, matcher{pattern, "", callback, nil})
}

type matcher struct {
	pattern  string
	path     Path
	callback func(ctx RoutingContext) (interface{}, error)
	args     []interface{}
}

func (c matcher) ValueOf(name string) string {
	varName := "{" + name + "}"
	idxOfName := -1
	for i, elem := range Path(c.pattern).Names() {
		if elem == varName {
			idxOfName = i
			break
		}
	}
	if idxOfName < 0 {
		return ""
	}
	// this will cause out of bounds panic, if our matching fails
	return c.path.NameAt(idxOfName)
}

func (c matcher) Path() Path {
	return c.path
}

func (c matcher) Args() []interface{} {
	return c.args
}

func (c matcher) apply(path Path, args ...interface{}) (matcher, error) {
	if c.pattern == "*" {
		return c.derive(path), nil
	}

	patternPath := Path(c.pattern)

	if patternPath.Normalize().String() == path.Normalize().String() {
		return c.derive(path, args...), nil
	}

	patternSegments := patternPath.Names()
	pathSegments := path.Names()

	if len(pathSegments) == len(patternSegments) {
		for i, p := range patternSegments {
			isNamedVar := strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}")
			if isNamedVar {
				// a named path segment is ignored
				continue
			}
			if pathSegments[i] != patternSegments[i] {
				return c, fmt.Errorf("cannot match path")
			}
		}
		return c.derive(path, args...), nil
	}

	return c, fmt.Errorf("cannot match path")

}

func (c matcher) derive(path Path, args ...interface{}) matcher {
	return matcher{c.pattern, path, c.callback, args}
}
