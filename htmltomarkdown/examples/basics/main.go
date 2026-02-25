package main

import (
	"fmt"
	"log"

	htmltomarkdown "github.com/getkawai/tools/htmltomarkdown"
)

func main() {
	input := `<strong>Bold Text</strong>`

	markdown, err := htmltomarkdown.ConvertString(input)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(markdown)
	// Output: **Bold Text**
}
