package db

import (
	"errors"

	"gorm.io/gorm"
)

// CreateMatrixModel creates a new matrix model
func CreateMatrixModel(db *gorm.DB, model *MatrixModel) error {
	return db.Create(model).Error
}

// GetMatrixModels returns all matrix models for a revision
func GetMatrixModels(db *gorm.DB, revisionID int64) ([]MatrixModel, error) {
	var models []MatrixModel
	if err := db.Where("revision_id = ?", revisionID).Preload("Selections").Find(&models).Error; err != nil {
		return nil, err
	}
	return models, nil
}

// GetMatrixModel returns a matrix model by ID
func GetMatrixModel(db *gorm.DB, id int64) (*MatrixModel, error) {
	var model MatrixModel
	if err := db.First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMatrixModelNotFound
		}
		return nil, err
	}
	return &model, nil
}

// UpdateMatrixModel updates a matrix model
func UpdateMatrixModel(db *gorm.DB, model *MatrixModel) error {
	return db.Save(model).Error
}

// CreateMatrixSelections creates matrix selections in batch
func CreateMatrixSelections(db *gorm.DB, selections []MatrixSelection) error {
	return db.Create(&selections).Error
}

// DeleteMatrixSelectionsByRevision deletes all matrix selections for a revision
func DeleteMatrixSelectionsByRevision(db *gorm.DB, revisionID int64) error {
	return db.Where("revision_id = ?", revisionID).Delete(&MatrixSelection{}).Error
}

// DeleteMatrixSelection deletes a matrix selection by ID
func DeleteMatrixSelection(db *gorm.DB, id int64) error {
	return db.Delete(&MatrixSelection{}, id).Error
}

// GetMatrixSelections returns matrix selections for a revision and model
func GetMatrixSelections(db *gorm.DB, revisionID int64, modelID int64) ([]MatrixSelection, error) {
	var selections []MatrixSelection
	query := db.Where("revision_id = ? AND model_id = ?", revisionID, modelID)
	if err := query.Find(&selections).Error; err != nil {
		return nil, err
	}
	return selections, nil
}

// GetMatrixSelectionsByRevision returns all matrix selections for a revision
func GetMatrixSelectionsByRevision(db *gorm.DB, revisionID int64) ([]MatrixSelection, error) {
	var selections []MatrixSelection
	if err := db.Where("revision_id = ?", revisionID).Find(&selections).Error; err != nil {
		return nil, err
	}
	return selections, nil
}

// GetMatrixSelectionsByGroup returns matrix selections by group for a revision
func GetMatrixSelectionsByGroup(db *gorm.DB, revisionID int64, group string) ([]MatrixSelection, error) {
	var selections []MatrixSelection
	if err := db.Where("revision_id = ? AND group = ?", revisionID, group).Find(&selections).Error; err != nil {
		return nil, err
	}
	return selections, nil
}

// GetSecondSourcesByRevision returns all second sources for a revision
func GetSecondSourcesByRevision(db *gorm.DB, revisionID int64) ([]SecondSource, error) {
	var sources []SecondSource
	if err := db.Where("revision_id = ?", revisionID).Find(&sources).Error; err != nil {
		return nil, err
	}
	return sources, nil
}

// CreateSecondSourcesInBatch creates second sources in batch
func CreateSecondSourcesInBatch(db *gorm.DB, sources []SecondSource) error {
	return db.Create(&sources).Error
}

// DeleteSecondSourcesByRevision deletes all second sources for a revision
func DeleteSecondSourcesByRevision(db *gorm.DB, revisionID int64) error {
	return db.Where("revision_id = ?", revisionID).Delete(&SecondSource{}).Error
}

// DeleteSecondSource deletes a second source by ID
func DeleteSecondSource(db *gorm.DB, id int64) error {
	return db.Delete(&SecondSource{}, id).Error
}

// UpdateSecondSource updates a second source
func UpdateSecondSource(db *gorm.DB, source *SecondSource) error {
	return db.Save(source).Error
}

// DeleteInvalidSelections deletes invalid matrix selections based on removed groups or materials
// This is used during EBOM merge to clean up selections for parts that no longer exist
func DeleteInvalidSelections(db *gorm.DB, revisionID int64, removedGroups []string, removedMaterials []string) error {
	if len(removedGroups) == 0 && len(removedMaterials) == 0 {
		return nil
	}

	query := db.Where("revision_id = ?", revisionID)

	// Delete selections where the group is in the removed groups list
	// Note: "group" is a reserved keyword in SQL, so we use "group" in quotes
	if len(removedGroups) > 0 {
		query = query.Where("`group` IN ?", removedGroups)
	}

	// Delete selections where the material is in the removed materials list
	// Note: This is an OR condition - if either group or material is removed, delete the selection
	if len(removedMaterials) > 0 {
		if len(removedGroups) > 0 {
			query = query.Where("`group` IN ? OR material IN ?", removedGroups, removedMaterials)
		} else {
			query = query.Where("material IN ?", removedMaterials)
		}
	}

	return query.Delete(&MatrixSelection{}).Error
}

// ImportMatrixSelections imports Matrix selections from a source revision to a target revision
// See product-spec section 7.4
// This function copies Matrix selection states from sourceRevisionID to targetRevisionID
// based on matching group keys (main_supplier + main_supplier_pn) and model names
func ImportMatrixSelections(db *gorm.DB, sourceRevisionID, targetRevisionID int64) error {
	// Get all MatrixModels from target revision
	var targetModels []MatrixModel
	if err := db.Where("revision_id = ?", targetRevisionID).Find(&targetModels).Error; err != nil {
		return err
	}

	// Create a map of model name to model ID for quick lookup
	targetModelMap := make(map[string]int64)
	for _, model := range targetModels {
		targetModelMap[model.ModelName] = model.ID
	}

	// Get all MatrixSelections from source revision
	var sourceSelections []MatrixSelection
	if err := db.Where("revision_id = ?", sourceRevisionID).Find(&sourceSelections).Error; err != nil {
		return err
	}

	// Create a map of model ID to model name
	sourceModelMap := make(map[int64]string)

	// Process each source selection
	var selectionsToCreate []MatrixSelection
	for _, sourceSel := range sourceSelections {
		// Get the model name from sourceModelMap or fetch it
		modelName, exists := sourceModelMap[sourceSel.ModelID]
		if !exists {
			var model MatrixModel
			if err := db.Where("id = ?", sourceSel.ModelID).First(&model).Error; err != nil {
				continue
			}
			modelName = model.ModelName
			sourceModelMap[sourceSel.ModelID] = modelName
		}

		// Check if target has a model with the same name
		targetModelID, exists := targetModelMap[modelName]
		if !exists {
			// Target doesn't have this model, skip this selection
			continue
		}

		// Get the part associated with this selection to get supplier info
		var sourcePart Part
		if err := db.Where("id = ?", sourceSel.PartID).First(&sourcePart).Error; err != nil {
			continue
		}

		// Check if target revision has the same group (main supplier + supplier PN)
		var targetParts []Part
		if err := db.Where("revision_id = ? AND supplier = ? AND supplier_pn = ?",
			targetRevisionID, sourcePart.Supplier, sourcePart.SupplierPN).Find(&targetParts).Error; err != nil {
			continue
		}

		if len(targetParts) == 0 {
			// Target doesn't have this part group, skip
			continue
		}

		// Find the target part that matches the selected material
		// The selected material could be the main source or a second source
		var selectedPart *Part
		for _, tp := range targetParts {
			if tp.Supplier == sourceSel.SelectedSupplier && tp.SupplierPN == sourceSel.SelectedSupplierPn {
				selectedPart = &tp
				break
			}
		}

		// If the selected part doesn't exist in target, check if it exists as a second source
		if selectedPart == nil {
			var secondSource SecondSource
			err := db.Where("revision_id = ? AND supplier = ? AND supplier_pn = ?",
				targetRevisionID, sourceSel.SelectedSupplier, sourceSel.SelectedSupplierPn).
				First(&secondSource).Error
			if err == nil {
				// Found as second source, use the first matching part
				selectedPart = &targetParts[0]
			}
		}

		if selectedPart == nil {
			// Selected material doesn't exist in target, skip this selection
			continue
		}

		// Create the selection in target revision
		selectionsToCreate = append(selectionsToCreate, MatrixSelection{
			RevisionID:          targetRevisionID,
			ModelID:             targetModelID,
			PartID:              selectedPart.ID,
			Group:               sourceSel.Group,
			Material:            sourceSel.Material,
			SelectedSupplier:    sourceSel.SelectedSupplier,
			SelectedSupplierPn:  sourceSel.SelectedSupplierPn,
			IsAutoSelected:      true,
		})
	}

	// Batch insert new selections
	if len(selectionsToCreate) > 0 {
		if err := db.CreateInBatches(&selectionsToCreate, 100).Error; err != nil {
			return err
		}
	}

	return nil
}
