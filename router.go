package vfs

import (
	"context"
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

	// Context returns the golang execution context
	Context() context.Context
}

// A Router has a set of patterns which can be registered to be matched in the order of configuration.
type Router struct {
	matchers []matcher
}

// Dispatch tries to find the correct matcher for the given path. The first matching callback is invoked or if
// nothing matches, nothing is called at all (and false is returned). Returns io.EOF if no matcher can be applied.
func (r *Router) Dispatch(ctx context.Context, path Path, args ...interface{}) (interface{}, error) {
	for _, m := range r.matchers {
		matcher, err := m.apply(ctx, path)
		if err != nil {
			continue
		}
		// invoke, if the pattern matches and return
		return matcher.callback(matcher)
	}
	return nil, io.EOF
}

// DispatchResultSet is required to workaround missing generics
func (r *Router) DispatchResultSet(ctx context.Context, path Path) (ResultSet, error) {
	res, err := r.Dispatch(ctx, path)
	if res == nil {
		return nil, err
	}
	if rs, ok := res.(ResultSet); ok {
		return rs, err
	}
	return nil, fmt.Errorf("cannot convert result: %v", err)
}

// DispatchBlob is required to workaround missing generics
func (r *Router) DispatchBlob(ctx context.Context, path Path) (Blob, error) {
	res, err := r.Dispatch(ctx, path)
	if res == nil {
		return nil, err
	}
	if rs, ok := res.(Blob); ok {
		return rs, err
	}
	return nil, fmt.Errorf("cannot convert result: %v", err)
}

// DispatchEntry is required to workaround missing generics
func (r *Router) DispatchEntry(ctx context.Context, path Path, args ...interface{}) (Entry, error) {
	res, err := r.Dispatch(ctx, path, args...)
	if res == nil {
		return nil, err
	}
	if rs, ok := res.(Entry); ok {
		return rs, err
	}
	return nil, fmt.Errorf("cannot convert result: %v", err)
}

// Match registers an arbitrary function with a pattern with injection-like semantics.
//
// Supported patterns are:
//  * * : matches everything
//  * /a/concrete/path : matches the exact path
//  * /{name} : matches anything like /a or /b
//  * /fix/{var}/fix : matches anything like /fix/a/fix or /fix/b/fix
//  * /fix/fix2/* : matches anything like /fix/fix2 or /fix/fix2/a/b/
func (r *Router) Match(pattern string, callback func(ctx RoutingContext) (interface{}, error)) {
	r.matchers = append(r.matchers, matcher{pattern, "", callback, nil, nil})
}

// MatchResultSet is required to workaround missing generics
func (r *Router) MatchResultSet(pattern string, f func(ctx RoutingContext) (ResultSet, error)) {
	r.Match(pattern, func(ctx RoutingContext) (interface{}, error) {
		return f(ctx)
	})
}

// MatchBlob is required to workaround missing generics
func (r *Router) MatchBlob(pattern string, f func(ctx RoutingContext) (Blob, error)) {
	r.Match(pattern, func(ctx RoutingContext) (interface{}, error) {
		return f(ctx)
	})
}

// MatchEntry is required to workaround missing generics
func (r *Router) MatchEntry(pattern string, f func(ctx RoutingContext) (Entry, error)) {
	r.Match(pattern, func(ctx RoutingContext) (i interface{}, e error) {
		return f(ctx)
	})
}

type matcher struct {
	pattern  string
	path     Path
	callback func(ctx RoutingContext) (interface{}, error)
	args     []interface{}
	ctx      context.Context
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
	// there could be out of bounds failure, which we silently ignore
	pathNames := c.path.Names()
	if idxOfName >= len(pathNames) {
		return ""
	}
	return pathNames[idxOfName]
}

func (c matcher) Path() Path {
	return c.path
}

func (c matcher) Args() []interface{} {
	return c.args
}

func (c matcher) apply(ctx context.Context, path Path, args ...interface{}) (matcher, error) {
	if c.pattern == "*" {
		return c.derive(ctx, path), nil
	}

	patternPath := Path(c.pattern)

	if patternPath.Normalize().String() == path.Normalize().String() {
		return c.derive(ctx, path, args...), nil
	}

	patternSegments := patternPath.Names()
	pathSegments := path.Names()

	if len(pathSegments) == len(patternSegments) {
		for i, p := range patternSegments {
			isWildcard := p == "*"
			if isWildcard {
				break
			}
			isNamedVar := strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}")
			if isNamedVar {
				// a named path segment is ignored
				continue
			}
			if pathSegments[i] != patternSegments[i] {
				return c, fmt.Errorf("cannot match path")
			}
		}
		return c.derive(ctx, path, args...), nil
	}

	if len(patternSegments) > 0 && patternSegments[len(patternSegments)-1] == "*" && strings.HasPrefix(path.String(), patternPath.Parent().String()) {
		return c.derive(ctx, path, args...), nil
	}

	return c, fmt.Errorf("cannot match path")

}

func (c matcher) Context() context.Context {
	return c.ctx
}

func (c matcher) derive(ctx context.Context, path Path, args ...interface{}) matcher {
	return matcher{c.pattern, path, c.callback, args, ctx}
}
