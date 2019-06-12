package vfs

import "testing"

func TestRouter_Dispatch(t *testing.T) {
	state := ""
	router := &Router{}

	router.Handle("/a/b/c", func(ctx RoutingContext) {
		state = "1"
	})

	router.Handle("/a/{id}/c", func(ctx RoutingContext) {
		state = "2"
	})

	router.Handle("/a/{id}/c/{other}", func(ctx RoutingContext) {
		state = "4"
		if ctx.ValueOf("id") != "x" {
			t.Fatal("got", ctx.ValueOf("id"))
		}

		if ctx.ValueOf("other") != "z" {
			t.Fatal("got", ctx.ValueOf("other"))
		}
	})

	router.Handle("/", func(ctx RoutingContext) {
		state = "3"
	})

	assertState(t, router, "/a/b/z/d", "3", &state)

	assertState(t, router, "/a/b/c", "1", &state)
	assertState(t, router, "/a/b/c/", "1", &state)
	assertState(t, router, "a/b/c/", "1", &state)

	assertState(t, router, "/a/x/c/", "2", &state)

	assertState(t, router, "/a/x/c/z", "4", &state)
}

func assertState(t *testing.T, router *Router, path Path, expect string, dst *string) {
	t.Helper()
	router.Dispatch(path)
	if *dst != expect {
		t.Fatal("expected", expect, "but got", *dst)
	}

}
