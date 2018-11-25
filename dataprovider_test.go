package vfs

import "testing"

func TestEmptyPath(t *testing.T) {
	cases := []string{"", "/"}
	for _, str := range cases {
		p := Path(str)
		if p.NameCount() != 0 {
			t.Fatal("expected 0 but got", p.NameCount())
		}

		if len(p.Names()) != 0 {
			t.Fatal("expected 0 but got", len(p.Names()))
		}

		if p.Parent().NameCount() != 0 {
			t.Fatal("expected 0 but got", p.Parent().NameCount())
		}

		if p.String() != "/" {
			t.Fatal("expected / but got", p.String())
		}
	}
}

func Test1Path(t *testing.T) {
	cases := []string{"a", "/a", "a/", "/a/"}
	for _, str := range cases {
		p := Path(str)
		if p.NameCount() != 1 {
			t.Fatal("expected 1 but got", p.NameCount(), " => "+str)
		}

		if len(p.Names()) != 1 {
			t.Fatal("expected 1 but got", len(p.Names()), " => "+str)
		}

		if p.NameAt(0) != "a" {
			t.Fatal("expected a but got", p.NameAt(0))
		}

		if p.Parent().NameCount() != 0 {
			t.Fatal("expected 0 but got", p.Parent().NameCount())
		}

		if p.String() != "/a" {
			t.Fatal("expected /a but got", p.String())
		}
	}
}

func Test2Path(t *testing.T) {
	cases := []string{"a/b", "/a/b", "a/b", "/a/b/"}
	for _, str := range cases {
		p := Path(str)
		if p.NameCount() != 2 {
			t.Fatal("expected 1 but got", p.NameCount(), " => "+str)
		}

		if len(p.Names()) != 2 {
			t.Fatal("expected 1 but got", len(p.Names()), " => "+str)
		}

		if p.NameAt(0) != "a" {
			t.Fatal("expected a but got", p.NameAt(0))
		}

		if p.NameAt(1) != "b" {
			t.Fatal("expected b but got", p.NameAt(1))
		}

		if p.Parent().NameCount() != 1 {
			t.Fatal("expected 1 but got", p.Parent().NameCount())
		}

		if p.String() != "/a/b" {
			t.Fatal("expected /a/b but got", p.String())
		}
	}
}

func TestModPath(t *testing.T) {
	p := ConcatPaths("a/b/", "/c")
	if p.String() != "/a/b/c" {
		t.Fatal("expected /a/b/c but got", p)
	}

	p = p.Child("d")
	if p.String() != "/a/b/c/d" {
		t.Fatal("expected /a/b/c/d but got", p)
	}

	p = p.TrimPrefix(Path("a/b/c"))
	if p.String() != "/d" {
		t.Fatal("expected /d but got", p)
	}
}
