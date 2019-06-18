package vfs

import (
	"context"
	"testing"
)

func TestRouter_Dispatch(t *testing.T) {
	router := &Router{}

	router.Match("/a/b/c", func(ctx RoutingContext) (interface{}, error) {
		return "1", nil
	})

	router.Match("/a/c/*", func(ctx RoutingContext) (i interface{}, e error) {
		return "5", nil
	})

	router.Match("/a/{id}/c", func(ctx RoutingContext) (interface{}, error) {
		return "2", nil
	})

	router.Match("/a/{id}/c/{other}", func(ctx RoutingContext) (interface{}, error) {

		if ctx.ValueOf("id") != "x" {
			t.Fatal("got", ctx.ValueOf("id"))
		}

		if ctx.ValueOf("other") != "z" {
			t.Fatal("got", ctx.ValueOf("other"))
		}
		return "4", nil
	})

	router.Match("*", func(ctx RoutingContext) (interface{}, error) {
		return "3", nil
	})

	assertState(t, router, "/a/b/z/d", "3")

	assertState(t, router, "/a/b/c", "1")
	assertState(t, router, "/a/b/c/", "1")
	assertState(t, router, "a/b/c/", "1")

	assertState(t, router, "/a/x/c/", "2")

	assertState(t, router, "/a/x/c/z", "4")
	assertState(t, router, "/a/c/e", "5")
	assertState(t, router, "/a/c/e/f/g/h", "5")
	assertState(t, router, "/a/c/", "5")
	assertState(t, router, "/a/c", "5")
}

func assertState(t *testing.T, router *Router, path Path, expect string) {
	t.Helper()
	res, err := router.Dispatch(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	str := res.(string)
	if str != expect {
		t.Fatal("expected", expect, "but got", str)
	}

}
