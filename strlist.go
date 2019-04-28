package vfs

// A List covers an interface slice
type List struct {
	entries []interface{}
}

//
func (l *List) Size() int {
	return len(l.entries)
}

func (l *List) Add(v interface{}) {
	l.entries = append(l.entries, v)
}

// A StrList is an ArrayList of strings
type StrList struct {
	List
}

func (l *StrList) Add(v string) {
	l.List.Add(v)
}


type AttrList struct{
	List
}