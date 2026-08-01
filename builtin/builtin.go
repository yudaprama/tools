package builtin

import (
	"database/sql"
	"log"

	"github.com/yudaprama/tools"
)

// RegisterAll registers all builtin tools
func RegisterAll(registry *tools.ToolRegistry) error {
	return RegisterAllWithDB(registry, nil)
}

// RegisterAllWithDB registers all builtin tools with optional database connection
// Some tools (like image describe) require database access
func RegisterAllWithDB(registry *tools.ToolRegistry, sqlDB *sql.DB) error {
	log.Println("Registering builtin tools (yzma)...")

	// Register lobe-web-browsing (search, crawlSinglePage, crawlMultiPages)
	if err := RegisterWebBrowsing(registry); err != nil {
		return err
	}
	log.Println("✅ Registered: lobe-web-browsing (search, crawlSinglePage, crawlMultiPages)")

	// Register lobe-local-system (file operations)
	if err := RegisterLocalSystem(registry); err != nil {
		return err
	}
	log.Println("✅ Registered: lobe-local-system (list, read, search, write, rename, move)")

	// Register PDF tool (search/replace)
	if err := RegisterPDF(registry); err != nil {
		return err
	}
	log.Println("✅ Registered: pdf_search_replace, pdf_search_text, pdf_extract_text, pdf_merge, pdf_split, pdf_page_info, pdf_metadata_get, pdf_metadata_set, pdf_extract_images")

	// Register lobe-image-designer (DALL-E compatible)
	if err := RegisterImageDesigner(registry); err != nil {
		return err
	}
	log.Println("✅ Registered: lobe-image-designer (text2image)")

	// Register lobe-code-interpreter (Python execution)
	if err := RegisterCodeInterpreter(registry); err != nil {
		return err
	}
	log.Println("✅ Registered: lobe-code-interpreter (python)")

	// Register calculator
	if err := RegisterCalculator(registry); err != nil {
		return err
	}
	log.Println("✅ Registered: calculator")

	// Register MuninnDB tools (embedded memory operations)
	if err := RegisterMuninnDB(registry); err != nil {
		return err
	}
	log.Println("✅ Registered: muninn_remember, muninn_remember_batch, muninn_recall, muninn_read, muninn_link, muninn_forget, muninn_status")

	// Register office tools (word, excel, powerpoint)
	if err := RegisterOfficeWord(registry); err != nil {
		return err
	}
	if err := RegisterOfficeExcel(registry); err != nil {
		return err
	}
	if err := RegisterOfficePowerPoint(registry); err != nil {
		return err
	}
	log.Println("✅ Registered: office-word, office-excel, office-powerpoint (create, update, read)")

	// Register PostgreSQL tools (postgres_attach, postgres_query, postgres_execute, etc.)
	if err := RegisterPostgres(registry); err != nil {
		log.Printf("⚠️  Failed to register PostgreSQL tools: %v", err)
	} else {
		log.Println("✅ Registered: postgres_attach, postgres_query, postgres_execute, postgres_list_tables, postgres_describe, postgres_detach")
	}

	// Register MySQL tools (mysql_attach, mysql_query, mysql_execute, etc.)
	if err := RegisterMySQL(registry); err != nil {
		log.Printf("⚠️  Failed to register MySQL tools: %v", err)
	} else {
		log.Println("✅ Registered: mysql_attach, mysql_query, mysql_execute, mysql_list_tables, mysql_describe, mysql_detach")
	}

	// Register lobe-image-describe (requires DB for querying VL descriptions)
	if sqlDB != nil {
		if err := RegisterImageDescribe(registry, sqlDB); err != nil {
			return err
		}
		log.Println("✅ Registered: lobe-image-describe (getImageDescription)")

		// Register lobe-video-describe (requires DB for querying Whisper transcriptions)
		if err := RegisterVideoDescribe(registry, sqlDB); err != nil {
			return err
		}
		log.Println("✅ Registered: lobe-video-describe (getVideoTranscription)")
	} else {
		log.Println("⚠️  Skipped: lobe-image-describe (no database connection)")
		log.Println("⚠️  Skipped: lobe-video-describe (no database connection)")
	}

	return nil
}
