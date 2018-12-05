package vfs

import (
	"io"
	"log"
)

// tries to close and prints silently the closer in case of an error
func silentClose(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Printf("failed to close: %v\n", err)
	}
}
