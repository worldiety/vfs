package vfs

import (
	"fmt"
	"strings"
)

type RoutingContext interface {
	// ValueOf returns the string value of a named parameter or the empty string if undefined
	ValueOf(name string) string

	// Path returns the actual path
	Path() Path
}

// A Router has a set of patterns which can be registered to be matched in the order of configuration.
type Router struct {
	matchers []matcher
}

// Dispatch tries to find the correct matcher for the given path. The first matching callback is invoked or if
// nothing matches, nothing is called at all (and false is returned).
func (r *Router) Dispatch(path Path) bool {
	for _, m := range r.matchers {
		matcher, err := m.apply(path)
		if err != nil {
			continue
		}
		// invoke, if the pattern matches and return
		matcher.callback(matcher)
		return true
	}
	return false
}

// Handle registers an arbitrary function with a pattern with injection-like semantics.
//
// Supported patterns are:
//  * / : matches everything
//  * /a/concrete/path : matches the exact path
//  * /{name} : matches anything like /a or /b
//  * /fix/{var}/fix : matches anything like /fix/a/fix or /fix/b/fix
func (r *Router) Handle(pattern string, callback func(ctx RoutingContext)) {
	r.matchers = append(r.matchers, matcher{pattern, "", callback})
}

type matcher struct {
	pattern  string
	path     Path
	callback func(ctx RoutingContext)
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

func (c matcher) apply(path Path) (matcher, error) {
	if c.pattern == "/" {
		return c.derive(path), nil
	}

	patternPath := Path(c.pattern)

	if patternPath.Normalize().String() == path.Normalize().String() {
		return c.derive(path), nil
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
		return c.derive(path), nil
	}

	return c, fmt.Errorf("cannot match path")

}

func (c matcher) derive(path Path) matcher {
	return matcher{c.pattern, path, c.callback}
}
