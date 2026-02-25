package main

import (
	"fmt"
	"log"

	"github.com/getkawai/tools/htmltomarkdown/converter"
	"github.com/getkawai/tools/htmltomarkdown/plugin/base"
	"github.com/getkawai/tools/htmltomarkdown/plugin/commonmark"
)

func main() {
	input := `<strong>Bold Text</strong>`

	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(
				commonmark.WithStrongDelimiter("__"),
				// ...additional configurations for the plugin
			),
		),
	)

	markdown, err := conv.ConvertString(input)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(markdown)
	// Output: __Bold Text__
}
