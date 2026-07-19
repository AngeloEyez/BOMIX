package excel

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"bomix-app/backend/db"
	"bomix-app/backend/types"
)

// BigMatrixReader handles BigMatrix format import
type BigMatrixReader struct {
	db     *gorm.DB
	result *types.ImportResult
}

// Import imports a BigMatrix format file
// See product-spec section 7.2
func (r *BigMatrixReader) Import(f *excelize.File) error {
	sheets := f.GetSheetList()

	// Find BigMatrix sheet
	bigMatrixSheet := ""
	for _, sheet := range sheets {
		if strings.EqualFold(sheet, "BigMatrix") {
			bigMatrixSheet = sheet
			break
		}
	}

	if bigMatrixSheet == "" {
		return errors.New("BigMatrix sheet not found")
	}

	// Parse header
	description, bomCount, date := r.parseHeader(f, bigMatrixSheet)
	_ = description
	_ = bomCount
	_ = date

	// Parse horizontal multi-BOM structure
	// See product-spec section 7.2.2.1
	bomConfigs, err := r.parseBOMConfigs(f, bigMatrixSheet)
	if err != nil {
		return fmt.Errorf("failed to parse BOM configs: %w", err)
	}

	// Parse parts and Matrix selections
	if err := r.parsePartsAndSelections(f, bigMatrixSheet, bomConfigs); err != nil {
		return fmt.Errorf("failed to parse parts and selections: %w", err)
	}

	return nil
}

// parseHeader parses the BigMatrix header
// See product-spec section 7.2.2
func (r *BigMatrixReader) parseHeader(f *excelize.File, sheetName string) (description string, bomCount int, date string) {
	// B3: "BOMs: {number}"
	val, _ := f.GetCellValue(sheetName, "B3")
	if idx := strings.Index(val, "BOMs: "); idx != -1 {
		fmt.Sscanf(strings.TrimSpace(val[idx+len("BOMs: "):]), "%d", &bomCount)
	}

	// B4: "Description: {value}"
	val, _ = f.GetCellValue(sheetName, "B4")
	if idx := strings.Index(val, "Description: "); idx != -1 {
		description = strings.TrimSpace(val[idx+len("Description: "):])
	}

	// E4: "Date: {value}"
	val, _ = f.GetCellValue(sheetName, "E4")
	if idx := strings.Index(val, "Date: "); idx != -1 {
		date = strings.TrimSpace(val[idx+len("Date: "):])
	}

	return description, bomCount, date
}

// BOMConfig represents a single BOM configuration within BigMatrix
type BOMConfig struct {
	ProjectCode string
	RevisionID  int64 // Database ID
	Phase       string
	Version     string
	ModelStart  int // Starting column index (0-based, H=7)
	ModelCount  int // Number of models in this BOM
	Models      []ModelConfig // Model configurations
}

// ModelConfig represents a single model configuration
type ModelConfig struct {
	ModelName string
	Qty       int
	Column    int // Column index within the BOM
}

// parseBOMConfigs parses the horizontal BOM configuration
// See product-spec section 7.2.2.1
func (r *BigMatrixReader) parseBOMConfigs(f *excelize.File, sheetName string) ([]BOMConfig, error) {
	var configs []BOMConfig

	// Start from column H (index 7)
	startCol := 7 // H = 7 (0-indexed)

	for {
		// Get project code from row 2
		projectCode, _ := f.GetCellValue(sheetName, colToCell(startCol+1, 2))
		if projectCode == "" {
			break // No more BOMs
		}

		// Get revision ID from row 3
		revisionStr, _ := f.GetCellValue(sheetName, colToCell(startCol+1, 3))

		// Parse phase and version from revision string (e.g., "PV-0.3")
		var phase, version string
		if idx := strings.Index(revisionStr, "-"); idx != -1 {
			phase = revisionStr[:idx]
			version = revisionStr[idx+1:]
		} else {
			phase = revisionStr
			version = "0.1"
		}

		// Parse models starting from row 4
		var models []ModelConfig
		colIdx := startCol
		for {
			modelName, _ := f.GetCellValue(sheetName, colToCell(colIdx+1, 4))
			if strings.ToUpper(modelName) == "A" && len(models) > 0 {
				// Found start of next BOM
				break
			}

			// Get qty from row 5
			qtyStr, _ := f.GetCellValue(sheetName, colToCell(colIdx+1, 5))
			var qty int
			fmt.Sscanf(qtyStr, "%d", &qty)

			models = append(models, ModelConfig{
				ModelName: strings.TrimSpace(modelName),
				Qty:       qty,
				Column:    colIdx - startCol,
			})

			colIdx++

			// Safety limit
			if colIdx-startCol > 50 {
				break
			}
		}

		if len(models) == 0 {
			break
		}

		config := BOMConfig{
			ProjectCode: projectCode,
			Phase:       phase,
			Version:     version,
			ModelStart:  startCol,
			ModelCount:  len(models),
			Models:      models,
		}

		// Get or create the BOM revision in database
		revisionID, err := r.getOrCreateBOMRevision(config)
		if err != nil {
			return nil, err
		}
		config.RevisionID = revisionID

		configs = append(configs, config)

		// Move to next BOM
		startCol = colIdx
	}

	return configs, nil
}

// colToCell converts column index and row number to Excel cell notation
func colToCell(col int, row int) string {
	// Convert 0-indexed column to Excel column letter
	var colStr string
	col++ // Convert to 1-indexed
	for col > 0 {
		col--
		colStr = string(rune('A'+col%26)) + colStr
		col /= 26
	}
	return fmt.Sprintf("%s%d", colStr, row)
}

// getOrCreateBOMRevision gets or creates a BOM revision in the database
func (r *BigMatrixReader) getOrCreateBOMRevision(config BOMConfig) (int64, error) {
	// Find existing revision
	var revision db.BomRevision
	err := r.db.Where("project_code = ? AND phase = ? AND version = ?",
		config.ProjectCode, config.Phase, config.Version).
		First(&revision).Error

	if err == nil {
		return revision.ID, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	// Create new revision - first get or create project
	var project db.Project
	err = r.db.Where("code = ?", config.ProjectCode).First(&project).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		project = db.Project{
			Code:        config.ProjectCode,
			Description: "Imported from BigMatrix",
		}
		if err := r.db.Create(&project).Error; err != nil {
			return 0, err
		}
	} else if err != nil {
		return 0, err
	}

	revision = db.BomRevision{
		ProjectID: project.ID,
		Phase:     config.Phase,
		Version:   config.Version,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := r.db.Create(&revision).Error; err != nil {
		return 0, err
	}

	return revision.ID, nil
}

// parsePartsAndSelections parses parts and Matrix selections from BigMatrix
// See product-spec section 7.2.3
func (r *BigMatrixReader) parsePartsAndSelections(f *excelize.File, sheetName string, configs []BOMConfig) error {
	// Get all rows from the sheet
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return err
	}

	// First, delete all existing Matrix selections for these revisions
	// See product-spec section 7.0.2
	for _, config := range configs {
		if err := db.DeleteMatrixSelectionsByRevision(r.db, config.RevisionID); err != nil {
			return err
		}

		// Also update Model qty
		if err := r.updateModelQty(config); err != nil {
			return err
		}
	}

	// Process data rows (starting from row 6)
	for i := 6; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}

		// Parse part data
		partData := r.parsePartDataRow(row)
		if partData == nil {
			continue
		}

		// For each BOM config, check Matrix selections
		for _, config := range configs {
			for modelIdx, model := range config.Models {
				// Calculate column for this model
				colIdx := config.ModelStart + modelIdx

				// Check if this cell is checked ("V" or "v")
				cellValue, _ := f.GetCellValue(sheetName, colToCell(colIdx+1, i+1))

				if strings.EqualFold(strings.TrimSpace(cellValue), "V") {
					// This model selected this part
					// Find or create the part in database
					partID, err := r.findOrCreatePart(config.RevisionID, partData)
					if err != nil {
						// Log but continue
						continue
					}

					// Find or create MatrixModel
					matrixModelID, err := r.findOrCreateMatrixModel(config.RevisionID, model.ModelName, model.Qty)
					if err != nil {
						continue
					}

					// Create MatrixSelection
					groupKey := partData.supplier + "|" + partData.supplierPN
					selection := db.MatrixSelection{
						RevisionID:         config.RevisionID,
						ModelID:            matrixModelID,
						PartID:             partID,
						Group:              groupKey,
						Material:           partData.supplier + "|" + partData.supplierPN,
						SelectedSupplier:   partData.supplier,
						SelectedSupplierPn: partData.supplierPN,
						IsAutoSelected:     false,
					}

					if err := r.db.Create(&selection).Error; err != nil {
						// Log but continue
						continue
					}
				}
			}
		}
	}

	return nil
}

// partData represents parsed part information
type partData struct {
	item         string
	hhpn         string
	description  string
	supplier     string
	supplierPN   string
	qty          int
	location     string
}

// parsePartDataRow parses a part data row from BigMatrix
// See product-spec section 7.2.3.1
func (r *BigMatrixReader) parsePartDataRow(row []string) *partData {
	if len(row) == 0 {
		return nil
	}

	data := &partData{}

	// A: Item
	data.item = strings.TrimSpace(row[0])

	// B: HHPN
	if len(row) > 1 {
		data.hhpn = strings.TrimSpace(row[1])
	}

	// C: Description
	if len(row) > 2 {
		data.description = strings.TrimSpace(row[2])
	}

	// D: Supplier
	if len(row) > 3 {
		data.supplier = strings.TrimSpace(row[3])
	}

	// E: Supplier PN
	if len(row) > 4 {
		data.supplierPN = strings.TrimSpace(row[4])
	}

	// F: Qty
	if len(row) > 5 {
		fmt.Sscanf(strings.TrimSpace(row[5]), "%d", &data.qty)
	}

	// G: Location
	if len(row) > 6 {
		data.location = strings.TrimSpace(row[6])
	}

	// Skip if no supplier info
	if data.supplier == "" && data.supplierPN == "" {
		return nil
	}

	return data
}

// updateModelQty updates the qty for all models in a BOM revision
func (r *BigMatrixReader) updateModelQty(config BOMConfig) error {
	for _, model := range config.Models {
		var matrixModel db.MatrixModel
		err := r.db.Where("revision_id = ? AND model_name = ?", config.RevisionID, model.ModelName).
			First(&matrixModel).Error

		if err == nil {
			// Update existing
			matrixModel.Qty = model.Qty
			if err := r.db.Save(&matrixModel).Error; err != nil {
				return err
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new
			matrixModel = db.MatrixModel{
				RevisionID: config.RevisionID,
				ModelName:  model.ModelName,
				Qty:        model.Qty,
			}
			if err := r.db.Create(&matrixModel).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

// findOrCreatePart finds or creates a part in the database
func (r *BigMatrixReader) findOrCreatePart(revisionID int64, data *partData) (int64, error) {
	// Try to find existing part
	var part db.Part
	err := r.db.Where("revision_id = ? AND supplier = ? AND supplier_pn = ?",
		revisionID, data.supplier, data.supplierPN).
		First(&part).Error

	if err == nil {
		return part.ID, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	// Create new part
	part = db.Part{
		RevisionID:  revisionID,
		Type:        "Main",
		Supplier:    data.supplier,
		SupplierPN:  data.supplierPN,
		Description: data.description,
		Location:    data.location,
		Quantity:    data.qty,
		BOMStatus:   "I",
	}

	if err := r.db.Create(&part).Error; err != nil {
		return 0, err
	}

	return part.ID, nil
}

// findOrCreateMatrixModel finds or creates a MatrixModel in the database
func (r *BigMatrixReader) findOrCreateMatrixModel(revisionID int64, modelName string, qty int) (int64, error) {
	var matrixModel db.MatrixModel
	err := r.db.Where("revision_id = ? AND model_name = ?", revisionID, modelName).
		First(&matrixModel).Error

	if err == nil {
		return matrixModel.ID, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	matrixModel = db.MatrixModel{
		RevisionID: revisionID,
		ModelName:  modelName,
		Qty:        qty,
	}

	if err := r.db.Create(&matrixModel).Error; err != nil {
		return 0, err
	}

	return matrixModel.ID, nil
}
