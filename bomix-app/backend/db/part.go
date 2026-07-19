package db

import (
	"errors"

	"gorm.io/gorm"
)

// CreatePartsInBatch creates parts in batch
func CreatePartsInBatch(db *gorm.DB, parts []Part) error {
	return db.Create(&parts).Error
}

// DeletePartsByRevision deletes all parts for a revision
func DeletePartsByRevision(db *gorm.DB, revisionID int64) error {
	return db.Where("revision_id = ?", revisionID).Delete(&Part{}).Error
}

// GetPartsByRevision returns all parts for a revision
func GetPartsByRevision(db *gorm.DB, revisionID int64) ([]Part, error) {
	var parts []Part
	if err := db.Where("revision_id = ?", revisionID).Find(&parts).Error; err != nil {
		return nil, err
	}
	return parts, nil
}

// GetPartsByRevisionAndType returns parts for a revision filtered by type
func GetPartsByRevisionAndType(db *gorm.DB, revisionID int64, partType string) ([]Part, error) {
	var parts []Part
	if err := db.Where("revision_id = ? AND type = ?", revisionID, partType).Find(&parts).Error; err != nil {
		return nil, err
	}
	return parts, nil
}

// GetPart returns a part by ID
func GetPart(db *gorm.DB, id int64) (*Part, error) {
	var part Part
	if err := db.First(&part, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPartNotFound
		}
		return nil, err
	}
	return &part, nil
}
