package db

import (
	"errors"

	"gorm.io/gorm"
)

// CreateRevision creates a new BOM revision
func CreateRevision(db *gorm.DB, projectID int64, phase, version, description string) (*BomRevision, error) {
	revision := &BomRevision{
		ProjectID:   projectID,
		Phase:       phase,
		Version:     version,
		Description: description,
	}
	if err := db.Create(revision).Error; err != nil {
		return nil, err
	}
	return revision, nil
}

// GetRevision returns a revision by ID
func GetRevision(db *gorm.DB, id int64) (*BomRevision, error) {
	var revision BomRevision
	if err := db.First(&revision, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRevisionNotFound
		}
		return nil, err
	}
	return &revision, nil
}

// GetRevisions returns all revisions for a project
func GetRevisions(db *gorm.DB, projectID int64) ([]BomRevision, error) {
	var revisions []BomRevision
	if err := db.Where("project_id = ?", projectID).Order("created_at DESC").Find(&revisions).Error; err != nil {
		return nil, err
	}
	return revisions, nil
}

// FindRevision finds a revision by project ID, phase, and version
func FindRevision(db *gorm.DB, projectID int64, phase, version string) (*BomRevision, error) {
	var revision BomRevision
	if err := db.Where("project_id = ? AND phase = ? AND version = ?", projectID, phase, version).First(&revision).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRevisionNotFound
		}
		return nil, err
	}
	return &revision, nil
}

// UpdateRevision updates a revision
func UpdateRevision(db *gorm.DB, revision *BomRevision) error {
	return db.Save(revision).Error
}

// FindPreviousRevision finds the previous revision in the same phase with version less than current
func FindPreviousRevision(db *gorm.DB, projectID int64, phase, currentVersion string) (*BomRevision, error) {
	var revision BomRevision
	// Find the revision with the highest version number that is less than currentVersion
	if err := db.Where("project_id = ? AND phase = ? AND version < ?", projectID, phase, currentVersion).
		Order("version DESC").
		First(&revision).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No previous revision found
		}
		return nil, err
	}
	return &revision, nil
}
