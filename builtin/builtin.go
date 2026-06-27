package builtin

import (
	"context"
	"log"

	"github.com/cloudwego/eino/components/tool"
)

// All builds every builtin eino tool. A tool group whose backing engine is
// unavailable at runtime is skipped with a warning rather than failing the set.
func All(ctx context.Context) ([]tool.InvokableTool, error) {
	log.Println("Building builtin tools (eino)...")
	var all []tool.InvokableTool

	groups := []string{
		"lobe-web-browsing", "lobe-local-system", "pdf",
		"lobe-image-designer", "lobe-code-interpreter", "calculator",
		"office-word", "office-excel", "office-powerpoint",
	}

	build := func(name string) ([]tool.InvokableTool, error) {
		switch name {
		case "lobe-web-browsing":
			return NewWebBrowsing(ctx)
		case "lobe-local-system":
			return NewLocalSystem(ctx)
		case "pdf":
			return NewPDF(ctx)
		case "lobe-image-designer":
			return NewImageDesigner(ctx)
		case "lobe-code-interpreter":
			return NewCodeInterpreter(ctx)
		case "calculator":
			return NewCalculator(ctx)
		case "office-word":
			return NewOfficeWord(ctx)
		case "office-excel":
			return NewOfficeExcel(ctx)
		case "office-powerpoint":
			return NewOfficePowerPoint(ctx)
		}
		return nil, nil
	}

	for _, name := range groups {
		ts, err := build(name)
		if err != nil {
			return nil, err
		}
		log.Printf("✅ Built: %s", name)
		all = append(all, ts...)
	}

	return all, nil
}
