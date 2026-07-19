package db

import (
	"errors"

	"gorm.io/gorm"
)

// GetOrCreateProject gets an existing project or creates a new one
func GetOrCreateProject(db *gorm.DB, seriesID int64, code, description string) (*Project, error) {
	var project Project
	result := db.Where("series_id = ? AND code = ?", seriesID, code).First(&project)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// Create new project
		project = Project{
			SeriesID:    seriesID,
			Code:        code,
			Description: description,
		}
		if err := db.Create(&project).Error; err != nil {
			return nil, err
		}
		return &project, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &project, nil
}

// GetProjects returns all projects for a series
func GetProjects(db *gorm.DB, seriesID int64) ([]Project, error) {
	var projects []Project
	if err := db.Where("series_id = ?", seriesID).Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

// GetProject returns a project by ID
func GetProject(db *gorm.DB, id int64) (*Project, error) {
	var project Project
	if err := db.Preload("Revisions").First(&project, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	return &project, nil
}
