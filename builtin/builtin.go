package builtin

import (
	"context"
	"database/sql"
	"log"

	"github.com/cloudwego/eino/components/tool"
)

// All builds every builtin eino tool. Tools that need a database connection
// (image/video describe) are only included when sqlDB is non-nil. Tools whose
// backing engine is unavailable (e.g. the DuckDB postgres/mysql extension) are
// skipped with a warning rather than failing the whole set.
func All(ctx context.Context, sqlDB *sql.DB) ([]tool.InvokableTool, error) {
	log.Println("Building builtin tools (eino)...")
	var all []tool.InvokableTool

	// Each entry: build the group, append on success, skip on failure.
	groups := []struct {
		name string
	}{
		{"lobe-web-browsing"}, {"lobe-local-system"}, {"pdf"},
		{"lobe-image-designer"}, {"lobe-code-interpreter"}, {"calculator"},
		{"muninndb"}, {"office-word"}, {"office-excel"}, {"office-powerpoint"},
		{"postgres"}, {"mysql"},
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
		case "muninndb":
			return NewMuninnDB(ctx)
		case "office-word":
			return NewOfficeWord(ctx)
		case "office-excel":
			return NewOfficeExcel(ctx)
		case "office-powerpoint":
			return NewOfficePowerPoint(ctx)
		case "postgres":
			return NewPostgres(ctx)
		case "mysql":
			return NewMySQL(ctx)
		}
		return nil, nil
	}

	optionalEngines := map[string]bool{"postgres": true, "mysql": true}

	for _, g := range groups {
		ts, err := build(g.name)
		if err != nil {
			if optionalEngines[g.name] {
				log.Printf("⚠️  Skipped: %s (%v)", g.name, err)
				continue
			}
			return nil, err
		}
		log.Printf("✅ Built: %s", g.name)
		all = append(all, ts...)
	}

	if sqlDB != nil {
		imgTools, err := NewImageDescribe(ctx, sqlDB)
		if err != nil {
			return nil, err
		}
		all = append(all, imgTools...)
		log.Println("✅ Built: lobe-image-describe")

		vidTools, err := NewVideoDescribe(ctx, sqlDB)
		if err != nil {
			return nil, err
		}
		all = append(all, vidTools...)
		log.Println("✅ Built: lobe-video-describe")
	} else {
		log.Println("⚠️  Skipped: lobe-image-describe (no database connection)")
		log.Println("⚠️  Skipped: lobe-video-describe (no database connection)")
	}

	return all, nil
}
