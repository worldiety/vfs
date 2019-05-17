package vfs

import (
	"io"
	"log"
	"unsafe"
)

// tries to close and prints silently the closer in case of an error
func silentClose(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Printf("failed to close: %v\n", err)
	}
}

// EqualsByReference compares the references of the given interfaces, just as a == in Java would do. Note that
// in contrast to Java, there is no unboxing of referenced primitives. If both interfaces are nil, true is returned.
// The most important question is, if two nil Data pointers with a different type are equal or not:
//   * the first answer is "yes, they are equal because they are both pointing to the same (nil) and nil is
//     applicable to both". However with the rule that Go has different semantics for the same nil value depending
//     on the type, this cannot be true (e.g. different listener implementations on distinct structs).
//   * the seconds answer is "no, they are not equal because they have different types". However that is also
//     not true, because due to polymorphism Go can point to a nil value with distinct interface types but
//     the same callable contract, so actually they are identical (e.g. due to coercing the same nil type through
//     different interface types).
//
// At the end, this implementation asserts that the type pointers are also identical, which is probably the most
// correct decision but will falsely report non-equal for some nil cases.
func EqualsByReference(a interface{}, b interface{}) bool {
	type iface struct {
		Type, Data unsafe.Pointer
	}

	if a == nil && b == nil {
		return true
	}
	if (a == nil && b != nil) || (a != nil && b == nil) {
		return false
	}

	iFaceA := *(*iface)(unsafe.Pointer(&a))
	iFaceB := *(*iface)(unsafe.Pointer(&b))
	return iFaceA.Data == iFaceB.Data && iFaceA.Type == iFaceB.Type
}
