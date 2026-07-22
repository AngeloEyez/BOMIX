package excel

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"bomix-app/backend/db"
	"bomix-app/backend/logger"
	"bomix-app/backend/types"
)

// EBOMReader handles EBOM format import
type EBOMReader struct {
	db        *gorm.DB
	result    *types.ImportResult
	revisionID int64 // Will be set after creating/updating revision
	logger    *logger.Logger
}

// Import imports an EBOM format file
func (r *EBOMReader) Import(f Workbook) error {
	sheets := f.GetSheetList()

	// Phase 1: Read all parts from SMD, PTH, BOTTOM sheets
	// Phase 2: Read status from NI, PROTO, MP sheets
	// See product-spec section 7.1.5

	// First pass: collect all parts from SMD, PTH, BOTTOM
	var allParts []db.Part
	var allSecondSources []parsedSecondSource

	smdSheet := r.findSheetCaseInsensitive(sheets, "SMD")
	pthSheet := r.findSheetCaseInsensitive(sheets, "PTH")
	bottomSheet := r.findSheetCaseInsensitive(sheets, "BOTTOM")

	// Parse header from SMD sheet
	var phase, version, description, schematicVersion, pcbVersion, pcaPn, date, projectCode string
	var err error
	if smdSheet != "" {
		phase, version, description, schematicVersion, pcbVersion, pcaPn, date, projectCode, err = r.parseHeader(f, smdSheet)
		if err != nil {
			return fmt.Errorf("failed to parse header: %w", err)
		}
	}

	// Determine NPI/MP mode
	mode := r.determineMode(f, sheets)

	// Create or update the BOM revision
	revisionID, err := r.createOrUpdateRevision(projectCode, phase, version, description, schematicVersion, pcbVersion, pcaPn, date, mode)
	if err != nil {
		return fmt.Errorf("failed to create/update revision: %w", err)
	}
	r.revisionID = revisionID

	// Process SMD sheet
	if smdSheet != "" {
		parts, secondSrcs := r.parseSheet(f, smdSheet, "SMD", len(allParts))
		allParts = append(allParts, parts...)
		allSecondSources = append(allSecondSources, secondSrcs...)
		r.result.PartsCount += len(parts)
		r.result.SecondSources += len(secondSrcs)
	}

	// Process PTH sheet
	if pthSheet != "" {
		parts, secondSrcs := r.parseSheet(f, pthSheet, "PTH", len(allParts))
		allParts = append(allParts, parts...)
		allSecondSources = append(allSecondSources, secondSrcs...)
		r.result.PartsCount += len(parts)
		r.result.SecondSources += len(secondSrcs)
	}

	// Process BOTTOM sheet
	if bottomSheet != "" {
		parts, secondSrcs := r.parseSheet(f, bottomSheet, "BOTTOM", len(allParts))
		allParts = append(allParts, parts...)
		allSecondSources = append(allSecondSources, secondSrcs...)
		r.result.PartsCount += len(parts)
		r.result.SecondSources += len(secondSrcs)
	}

	// Process NI sheet (bom_status = X)
	niSheet := r.findSheetCaseInsensitive(sheets, "NI")
	if niSheet != "" {
		niParts := r.parseStatusSheet(f, niSheet, "X", "")
		allParts = append(allParts, niParts...)
		r.result.PartsCount += len(niParts)
	}

	// Process PROTO sheet (bom_status = P)
	protoSheet := r.findSheetCaseInsensitive(sheets, "PROTO")
	if protoSheet != "" {
		protoParts := r.parseStatusSheet(f, protoSheet, "P", mode)
		allParts = append(allParts, protoParts...)
		r.result.PartsCount += len(protoParts)
	}

	// Process MP sheet (bom_status = M)
	mpSheet := r.findSheetCaseInsensitive(sheets, "MP")
	if mpSheet != "" {
		mpParts := r.parseStatusSheet(f, mpSheet, "M", mode)
		allParts = append(allParts, mpParts...)
		r.result.PartsCount += len(mpParts)
	}

	// Save all parts and second sources to database
	// See product-spec section 7.0.1: Delete old parts and recreate
	secondSources, err := r.saveParts(allParts, allSecondSources)
	if err != nil {
		return fmt.Errorf("failed to save parts: %w", err)
	}

	// Apply merge algorithm if this was an update
	// See product-spec section 7.1.7
	if err := r.applyMergeAlgorithm(r.revisionID, secondSources); err != nil {
		return fmt.Errorf("failed to apply merge algorithm: %w", err)
	}

	// Auto-import previous Matrix selections if configured
	// See product-spec section 7.1.8
	// We need to get the full revision for this
	var revision db.BomRevision
	if err := r.db.Where("id = ?", r.revisionID).First(&revision).Error; err == nil {
		if err := r.autoImportPreviousMatrix(revision); err != nil {
			// Log warning but don't fail the import
			fmt.Printf("Warning: failed to auto-import previous Matrix: %v\n", err)
		}
	}

	return nil
}

// findSheetCaseInsensitive finds a sheet by name (case-insensitive & trimmed)
func (r *EBOMReader) findSheetCaseInsensitive(sheets []string, name string) string {
	for _, sheet := range sheets {
		if strings.EqualFold(strings.TrimSpace(sheet), name) {
			return sheet
		}
	}
	return ""
}

// parseHeader parses the header row from SMD sheet
// See product-spec section 7.1.1
// Returns revision data and project code separately
func (r *EBOMReader) parseHeader(f Workbook, sheetName string) (phase, version, description, schematicVersion, pcbVersion, pcaPn, date, projectCode string, err error) {
	// B3: Project Code - "Product Code: {value}"
	val, _ := f.GetCellValue(sheetName, "B3")
	if idx := strings.Index(val, "Product Code: "); idx != -1 {
		projectCode = strings.TrimSpace(val[idx+len("Product Code: "):])
	}

	// B4: Description - "Description: {value}"
	val, _ = f.GetCellValue(sheetName, "B4")
	if idx := strings.Index(val, "Description: "); idx != -1 {
		description = strings.TrimSpace(val[idx+len("Description: "):])
	}

	// D3: Schematic Version - "Schematic Version: {value}"
	val, _ = f.GetCellValue(sheetName, "D3")
	if idx := strings.Index(val, "Schematic Version: "); idx != -1 {
		schematicVersion = strings.TrimSpace(val[idx+len("Schematic Version: "):])
	}

	// D4: Phase - "Phase: {value}"
	val, _ = f.GetCellValue(sheetName, "D4")
	if idx := strings.Index(val, "Phase: "); idx != -1 {
		phase = strings.TrimSpace(val[idx+len("Phase: "):])
	}

	// F3: PCB Version - "PCB Version: {value}"
	val, _ = f.GetCellValue(sheetName, "F3")
	if idx := strings.Index(val, "PCB Version: "); idx != -1 {
		pcbVersion = strings.TrimSpace(val[idx+len("PCB Version: "):])
	}

	// F4: PCA PN - "PCA PN: {value}"
	val, _ = f.GetCellValue(sheetName, "F4")
	if idx := strings.Index(val, "PCA PN: "); idx != -1 {
		pcaPn = strings.TrimSpace(val[idx+len("PCA PN: "):])
	}

	// H3: Version - "BOM Version: {value}"
	val, _ = f.GetCellValue(sheetName, "H3")
	if idx := strings.Index(val, "BOM Version: "); idx != -1 {
		version = strings.TrimSpace(val[idx+len("BOM Version: "):])
	}

	// H4: Date - "Date: {value}"
	val, _ = f.GetCellValue(sheetName, "H4")
	if idx := strings.Index(val, "Date: "); idx != -1 {
		date = strings.TrimSpace(val[idx+len("Date: "):])
	}

	return phase, version, description, schematicVersion, pcbVersion, pcaPn, date, projectCode, nil
}

type parsedSecondSource struct {
	partIndex    int
	secondSource db.SecondSource
}

// parseSheet parses data rows from a sheet
// See product-spec sections 7.1.2, 7.1.3, 7.1.4
func (r *EBOMReader) parseSheet(f Workbook, sheetName, sheetType string, basePartIndices ...int) ([]db.Part, []parsedSecondSource) {
	basePartIndex := 0
	if len(basePartIndices) > 0 {
		basePartIndex = basePartIndices[0]
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, nil
	}

	var parts []db.Part
	var secondSources []parsedSecondSource

	currentMainPartIndex := -1

	// Start from row 6 (index 5)
	for i := 5; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}

		// Get item number (column A)
		item := strings.TrimSpace(row[0])

		// Determine if this is a Main Source or 2nd Source
		// See product-spec section 7.1.3
		if item != "" {
			// This is a Main Source
			part := r.parsePartRow(row, sheetType)
			parts = append(parts, part)
			currentMainPartIndex = basePartIndex + len(parts) - 1
		} else if currentMainPartIndex >= 0 {
			// This is a 2nd Source (empty item, follows a Main Source)
			secondSource := r.parseSecondSourceRow(row)
			secondSources = append(secondSources, parsedSecondSource{
				partIndex:    currentMainPartIndex,
				secondSource: secondSource,
			})
		}
	}

	return parts, secondSources
}

// parsePartRow parses a single part data row
// See product-spec section 7.1.2
func (r *EBOMReader) parsePartRow(row []string, sheetType string) db.Part {
	var part db.Part

	// A: Item (not stored in database, just for reference)
	_ = strings.TrimSpace(row[0])

	// B: HHPN (not stored in Part model, skip for now)
	_ = strings.TrimSpace(row[1])

	// E: Description
	if len(row) > 4 {
		part.Description = strings.TrimSpace(row[4])
	}

	// F: Supplier
	if len(row) > 5 {
		part.Supplier = strings.TrimSpace(row[5])
	}

	// G: Supplier PN
	if len(row) > 6 {
		part.SupplierPN = strings.TrimSpace(row[6])
	}

	// I: Location (comma-separated)
	if len(row) > 8 {
		locationStr := strings.TrimSpace(row[8])
		// Atomize location: split by comma and create individual parts
		locations := r.atomizeLocation(locationStr)
		part.Location = locations
	}

	// J: CCL
	if len(row) > 9 {
		part.CCL = strings.TrimSpace(row[9])
	}

	// L: Remark
	if len(row) > 11 {
		part.Remark = strings.TrimSpace(row[11])
	}

	// Set type based on sheet
	part.Type = sheetType

	// Set default BOM status
	part.BOMStatus = "I" // Default to "Install"

	return part
}

// atomizeLocation splits comma-separated locations into individual entries
// See product-spec section 7.1.4
func (r *EBOMReader) atomizeLocation(locationStr string) string {
	// Split by comma and trim spaces
	parts := strings.Split(locationStr, ",")
	var atomized []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			atomized = append(atomized, trimmed)
		}
	}
	return strings.Join(atomized, ",")
}

// parseSecondSourceRow parses a second source row
func (r *EBOMReader) parseSecondSourceRow(row []string) db.SecondSource {
	var source db.SecondSource

	// B: HHPN (not in SecondSource model, skip)
	_ = strings.TrimSpace(row[1])

	// E: Description
	if len(row) > 4 {
		source.Description = strings.TrimSpace(row[4])
	}

	// F: Supplier
	if len(row) > 5 {
		source.Supplier = strings.TrimSpace(row[5])
	}

	// G: Supplier PN
	if len(row) > 6 {
		source.SupplierPN = strings.TrimSpace(row[6])
	}

	return source
}

// parseStatusSheet parses NI/PROTO/MP sheets
// See product-spec section 7.1.5
func (r *EBOMReader) parseStatusSheet(f Workbook, sheetName, bomStatus, mode string) []db.Part {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil
	}

	var parts []db.Part

	// Start from row 6 (index 5)
	for i := 5; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}

		// Create part with status
		part := db.Part{
			Type:      "", // Status sheets don't have type
			BOMStatus: bomStatus,
		}

		// B: HHPN (not in Part model, skip)
		_ = strings.TrimSpace(row[1])

		// E: Description
		if len(row) > 4 {
			part.Description = strings.TrimSpace(row[4])
		}

		// F: Supplier
		if len(row) > 5 {
			part.Supplier = strings.TrimSpace(row[5])
		}

		// G: Supplier PN
		if len(row) > 6 {
			part.SupplierPN = strings.TrimSpace(row[6])
		}

		// I: Location
		if len(row) > 8 {
			locationStr := strings.TrimSpace(row[8])
			part.Location = r.atomizeLocation(locationStr)
		}

		// J: CCL
		if len(row) > 9 {
			part.CCL = strings.TrimSpace(row[9])
		}

		parts = append(parts, part)
	}

	return parts
}

// determineMode determines NPI or MP mode based on PROTO overlap
// See product-spec section 7.1.6
func (r *EBOMReader) determineMode(f Workbook, sheets []string) string {
	protoSheet := r.findSheetCaseInsensitive(sheets, "PROTO")
	if protoSheet == "" {
		return "MP" // No PROTO sheet means MP mode
	}

	// Get all locations from SMD, PTH, BOTTOM sheets
	mainLocations := make(map[string]bool)

	smdSheet := r.findSheetCaseInsensitive(sheets, "SMD")
	pthSheet := r.findSheetCaseInsensitive(sheets, "PTH")
	bottomSheet := r.findSheetCaseInsensitive(sheets, "BOTTOM")

	for _, sheet := range []string{smdSheet, pthSheet, bottomSheet} {
		if sheet == "" {
			continue
		}
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}

		for i := 5; i < len(rows); i++ {
			row := rows[i]
			if len(row) > 8 {
				locationStr := strings.TrimSpace(row[8])
				locations := strings.Split(locationStr, ",")
				for _, loc := range locations {
					trimmed := strings.TrimSpace(loc)
					if trimmed != "" {
						mainLocations[trimmed] = true
					}
				}
			}
		}
	}

	// Check if any PROTO location overlaps with main locations
	protoRows, err := f.GetRows(protoSheet)
	if err != nil {
		return "MP"
	}

	for i := 5; i < len(protoRows); i++ {
		row := protoRows[i]
		if len(row) > 8 {
			locationStr := strings.TrimSpace(row[8])
			locations := strings.Split(locationStr, ",")
			for _, loc := range locations {
				trimmed := strings.TrimSpace(loc)
				if trimmed != "" && mainLocations[trimmed] {
					return "NPI" // Found overlap
				}
			}
		}
	}

	return "MP" // No overlap found
}

// createOrUpdateRevision creates or updates a BOM revision
// Returns the revision ID
func (r *EBOMReader) createOrUpdateRevision(projectCode, phase, version, description, schematicVersion, pcbVersion, pcaPn, date, mode string) (int64, error) {
	if r.db == nil {
		return 0, errors.New("db is nil")
	}

	// Find active series
	series, err := db.GetSeriesInfo(r.db)
	if err != nil {
		return 0, fmt.Errorf("failed to get active series: %w", err)
	}

	// Find or create project with valid SeriesID
	projectPtr, err := db.GetOrCreateProject(r.db, series.ID, projectCode, description)
	if err != nil {
		return 0, fmt.Errorf("failed to get or create project: %w", err)
	}
	project := *projectPtr

	// Find existing revision
	var existing db.BomRevision
	err = r.db.Where("project_id = ? AND phase = ? AND version = ?",
		project.ID, phase, version).
		First(&existing).Error

	if err == nil {
		// Update existing revision
		if r.logger != nil {
			r.logger.Info("BOM 覆蓋: 更新現有版本", "project", projectCode, "phase", phase, "version", version)
		}
		existing.Description = description
		existing.SchematicVersion = schematicVersion
		existing.PCBVersion = pcbVersion
		existing.PCAPN = pcaPn
		existing.Date = date
		existing.Mode = mode
		existing.UpdatedAt = time.Now()
		if err := r.db.Save(&existing).Error; err != nil {
			return 0, err
		}
		return existing.ID, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	// Create new revision
	revision := db.BomRevision{
		ProjectID:        project.ID,
		Phase:            phase,
		Version:          version,
		Description:      description,
		SchematicVersion: schematicVersion,
		PCBVersion:       pcbVersion,
		PCAPN:            pcaPn,
		Date:             date,
		Mode:             mode,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := r.db.Create(&revision).Error; err != nil {
		return 0, err
	}

	if r.logger != nil {
		r.logger.Info("BOM 新增: 建立全新版本", "project", projectCode, "phase", phase, "version", version)
	}

	return revision.ID, nil
}

// saveParts saves all parts and second sources to the database
func (r *EBOMReader) saveParts(parts []db.Part, parsedSecondSources []parsedSecondSource) ([]db.SecondSource, error) {
	if r.revisionID == 0 {
		return nil, errors.New("revision ID not set")
	}

	// Set RevisionID for all parts
	for i := range parts {
		parts[i].RevisionID = r.revisionID
	}

	// Delete old parts for this revision
	// See product-spec section 7.0.1
	if err := r.db.Where("revision_id = ?", r.revisionID).Delete(&db.Part{}).Error; err != nil {
		return nil, err
	}

	// Batch insert new parts - GORM assigns parts[i].ID
	if len(parts) > 0 {
		if err := r.db.CreateInBatches(&parts, 500).Error; err != nil {
			return nil, err
		}
	}

	// Now map second sources to parent parts' newly generated DB IDs
	secondSources := make([]db.SecondSource, 0, len(parsedSecondSources))
	for _, pss := range parsedSecondSources {
		ss := pss.secondSource
		ss.RevisionID = r.revisionID
		if pss.partIndex >= 0 && pss.partIndex < len(parts) {
			ss.PartID = parts[pss.partIndex].ID
		}
		secondSources = append(secondSources, ss)
	}

	// Batch insert second sources
	if len(secondSources) > 0 {
		if err := r.db.CreateInBatches(&secondSources, 500).Error; err != nil {
			return nil, err
		}
	}

	return secondSources, nil
}

// applyMergeAlgorithm applies the merge algorithm for EBOM re-import
// See product-spec section 7.1.7
func (r *EBOMReader) applyMergeAlgorithm(revisionID int64, newSecondSources []db.SecondSource) error {
	// Step 1: Update BomRevision metadata (already done in createOrUpdateRevision)

	// Step 2: Parts are already deleted and recreated (in saveParts)

	// Step 3: Diff SecondSources
	// Load old second sources
	var oldSecondSources []db.SecondSource
	if err := r.db.Where("revision_id = ?", revisionID).Find(&oldSecondSources).Error; err != nil {
		return err
	}
	// Group by (main_supplier, main_supplier_pn) - using the Part's supplier info
	oldGroups := make(map[string][]db.SecondSource)
	for _, ss := range oldSecondSources {
		// Get the main part's supplier info for the group key
		var mainPart db.Part
		if err := r.db.Where("id = ?", ss.PartID).First(&mainPart).Error; err != nil {
			if r.logger != nil {
				r.logger.Warn("Diff 二源料件時未找到對應主料件", "partID", ss.PartID, "error", err)
			}
			continue
		}
		key := fmt.Sprintf("%s|%s", mainPart.Supplier, mainPart.SupplierPN)
		oldGroups[key] = append(oldGroups[key], ss)
	}

	// Group new second sources by main part
	newGroups := make(map[string][]db.SecondSource)
	for _, ss := range newSecondSources {
		// Get the main part's supplier info for the group key
		var mainPart db.Part
		if err := r.db.Where("id = ?", ss.PartID).First(&mainPart).Error; err != nil {
			if r.logger != nil {
				r.logger.Warn("Diff 新二源料件時未找到對應主料件", "partID", ss.PartID, "error", err)
			}
			continue
		}
		key := fmt.Sprintf("%s|%s", mainPart.Supplier, mainPart.SupplierPN)
		newGroups[key] = append(newGroups[key], ss)
	}

	// Find groups to delete and update
	var groupsToDelete []string
	var secondSourcesToDelete []db.SecondSource
	var secondSourcesToUpdate []db.SecondSource
	var secondSourcesToCreate []db.SecondSource

	// Check old groups
	for key, oldSrcs := range oldGroups {
		if _, exists := newGroups[key]; !exists {
			// Group no longer exists - delete all
			groupsToDelete = append(groupsToDelete, key)
			secondSourcesToDelete = append(secondSourcesToDelete, oldSrcs...)
		} else {
			// Group exists - diff individual sources
			newSrcs := newGroups[key]
			oldMap := make(map[string]db.SecondSource)
			for _, ss := range oldSrcs {
				srcKey := fmt.Sprintf("%s|%s", ss.Supplier, ss.SupplierPN)
				oldMap[srcKey] = ss
			}
			newMap := make(map[string]db.SecondSource)
			for _, ss := range newSrcs {
				srcKey := fmt.Sprintf("%s|%s", ss.Supplier, ss.SupplierPN)
				newMap[srcKey] = ss
			}

			// Find deleted sources
			for srcKey, oldSS := range oldMap {
				if _, exists := newMap[srcKey]; !exists {
					secondSourcesToDelete = append(secondSourcesToDelete, oldSS)
				}
			}

			// Find new sources and updates
			for srcKey, newSS := range newMap {
				if _, exists := oldMap[srcKey]; !exists {
					secondSourcesToCreate = append(secondSourcesToCreate, newSS)
				} else {
					// Update existing - use the oldSS ID to update
					oldSS := oldMap[srcKey]
					// Update the fields that might have changed
					oldSS.Description = newSS.Description
					oldSS.Supplier = newSS.Supplier
					oldSS.SupplierPN = newSS.SupplierPN
					secondSourcesToUpdate = append(secondSourcesToUpdate, oldSS)
				}
			}
		}
	}

	// Execute deletions
	if len(secondSourcesToDelete) > 0 {
		var ids []int64
		for _, ss := range secondSourcesToDelete {
			ids = append(ids, ss.ID)
		}
		if err := r.db.Where("id IN ?", ids).Delete(&db.SecondSource{}).Error; err != nil {
			return err
		}
	}

	// Execute updates
	for _, ss := range secondSourcesToUpdate {
		if err := r.db.Save(&ss).Error; err != nil {
			return err
		}
	}

	// Execute creates
	if len(secondSourcesToCreate) > 0 {
		if err := r.db.CreateInBatches(&secondSourcesToCreate, 500).Error; err != nil {
			return err
		}
	}

	// Step 4: Clean invalid MatrixSelections
	// See product-spec section 7.1.7 step 4
	removedGroups := groupsToDelete
	var removedMaterials []string
	for _, ss := range secondSourcesToDelete {
		removedMaterials = append(removedMaterials, ss.Supplier+"|"+ss.SupplierPN)
	}

	return db.DeleteInvalidSelections(r.db, revisionID, removedGroups, removedMaterials)
}

// autoImportPreviousMatrix automatically imports Matrix selections from previous revision
// See product-spec section 7.1.8
func (r *EBOMReader) autoImportPreviousMatrix(revision db.BomRevision) error {
	// Find previous revision in same project and phase
	var previous db.BomRevision
	err := r.db.Where("project_id = ? AND phase = ? AND version < ?",
		revision.ProjectID, revision.Phase, revision.Version).
		Order("version DESC").
		First(&previous).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// No previous revision found
		return nil
	}
	if err != nil {
		return err
	}

	// Import Matrix selections from previous to current
	return db.ImportMatrixSelections(r.db, previous.ID, revision.ID)
}
