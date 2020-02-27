package main

import (
	"bufio"
	"log"
	"io"
)

func removeBOMCharacter(reader io.Reader) io.Reader {
	bufferReader := bufio.NewReader(reader)
	r, _, err := bufferReader.ReadRune()
	if err != nil {
		log.Fatal(err)
	}

	if r != '\uFEFF' {
		bufferReader.UnreadRune() // Not a BOM -- put the rune back
	}
	return bufferReader
}
