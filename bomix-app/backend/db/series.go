package db

import (
	"errors"

	"gorm.io/gorm"
)

// CreateSeries creates a new series
func CreateSeries(db *gorm.DB, name, description string) (*Series, error) {
	series := &Series{
		Name:        name,
		Description: description,
	}
	if err := db.Create(series).Error; err != nil {
		return nil, err
	}
	return series, nil
}

// GetSeriesInfo returns the series information
func GetSeriesInfo(db *gorm.DB) (*Series, error) {
	var series Series
	if err := db.First(&series).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSeriesNotFound
		}
		return nil, err
	}
	return &series, nil
}

// GetSeries returns a series by ID
func GetSeries(db *gorm.DB, id int64) (*Series, error) {
	var series Series
	if err := db.Preload("Projects").First(&series, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSeriesNotFound
		}
		return nil, err
	}
	return &series, nil
}
