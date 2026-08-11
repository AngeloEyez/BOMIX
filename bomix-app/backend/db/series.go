package db

import (
	"errors"
	"strings"

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

// UpdateProjectExportOrder 更新 Series 的 Project 匯出排序紀錄
//
// 參數：
//   - db：GORM 資料庫連線
//   - projectOrder：JSON 序列化後的 Project Code 順序字串
//
// 回傳：
//   - error：若更新失敗則回傳錯誤
func UpdateProjectExportOrder(db *gorm.DB, projectOrder string) error {
	err := db.Model(&Series{}).Where("id = ?", 1).Update("project_export_order", projectOrder).Error
	if err != nil && strings.Contains(err.Error(), "no such column") {
		// 若遇到舊版 DB 缺少欄位，自動補建 schema 並重試一次
		if migErr := AutoMigrate(db); migErr == nil {
			return db.Model(&Series{}).Where("id = ?", 1).Update("project_export_order", projectOrder).Error
		}
	}
	return err
}

