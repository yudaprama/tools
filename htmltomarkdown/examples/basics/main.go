package main

import (
	"fmt"
	"log"

	htmltomarkdown "github.com/yudaprama/tools/htmltomarkdown"
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
