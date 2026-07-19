package excel

import (
	"bytes"
	"embed"
	"fmt"

	"github.com/xuri/excelize/v2"
	"bomix-app/backend/types"
)

//go:embed template/*.xlsx
var templateFS embed.FS

// TemplateManager manages Excel templates for export
type TemplateManager struct {
	templates map[types.BOMFormat][]byte // Cache template bytes
}

// NewTemplateManager creates a new template manager
func NewTemplateManager() (*TemplateManager, error) {
	tm := &TemplateManager{
		templates: make(map[types.BOMFormat][]byte),
	}

	// Pre-load all templates
	if err := tm.loadAllTemplates(); err != nil {
		return nil, err
	}

	return tm, nil
}

// loadAllTemplates loads all embedded templates
func (tm *TemplateManager) loadAllTemplates() error {
	// Load BigMatrix template
	data, err := templateFS.ReadFile("template/bigmatrix.xlsx")
	if err != nil {
		return fmt.Errorf("failed to load bigmatrix template: %w", err)
	}
	tm.templates[types.FormatBigMatrix] = data

	// Load Matrix template
	data, err = templateFS.ReadFile("template/matrix.xlsx")
	if err != nil {
		return fmt.Errorf("failed to load matrix template: %w", err)
	}
	tm.templates[types.FormatMatrix] = data

	return nil
}

// LoadTemplate loads a template for the specified format
func (tm *TemplateManager) LoadTemplate(format types.BOMFormat) (*excelize.File, error) {
	// Get cached template bytes
	data, ok := tm.templates[format]
	if !ok {
		return nil, ErrInvalidFormat
	}

	// Open from bytes using bytes.NewReader
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to open template: %w", err)
	}

	return f, nil
}

// LoadTemplateFromFile loads a template from a file path (for development/testing)
func LoadTemplateFromFile(path string) (*excelize.File, error) {
	return excelize.OpenFile(path)
}

// Close closes all loaded templates
func (tm *TemplateManager) Close() error {
	// No need to close cached bytes
	return nil
}

// GetTemplateBytes returns the raw bytes of a template
func (tm *TemplateManager) GetTemplateBytes(format types.BOMFormat) ([]byte, error) {
	data, ok := tm.templates[format]
	if !ok {
		return nil, ErrInvalidFormat
	}
	return data, nil
}
