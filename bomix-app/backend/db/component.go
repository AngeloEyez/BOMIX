package db

import (
	"errors"

	"gorm.io/gorm"
)

// CreateComponentsInBatch 批次建立 RevisionComponent 記錄。
// GORM 將自動回填每筆記錄產生的 ID。
//
// 參數：
//   - db：GORM 資料庫連線
//   - comps：待建立的 RevisionComponent 切片
//
// 回傳：建立失敗則回傳錯誤
func CreateComponentsInBatch(db *gorm.DB, comps []RevisionComponent) error {
	if len(comps) == 0 {
		return nil
	}
	return db.CreateInBatches(&comps, 500).Error
}

// DeleteComponentsByRevision 刪除指定 Revision 的所有 RevisionComponent 記錄。
// 同時會連帶清理關聯的 PartLocation 記錄。
//
// 參數：
//   - db：GORM 資料庫連線
//   - revisionID：要刪除的 BOM Revision ID
//
// 回傳：刪除失敗則回傳錯誤
func DeleteComponentsByRevision(db *gorm.DB, revisionID int64) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// 先刪除關聯的 PartLocation
		if err := DeletePartLocationsByRevision(tx, revisionID); err != nil {
			return err
		}
		// 再刪除 RevisionComponent
		return tx.Where("revision_id = ?", revisionID).Delete(&RevisionComponent{}).Error
	})
}

// GetComponentsByRevision 查詢指定 Revision 的所有 RevisionComponent 記錄。
//
// 參數：
//   - db：GORM 資料庫連線
//   - revisionID：要查詢的 BOM Revision ID
//
// 回傳：RevisionComponent 切片或錯誤
func GetComponentsByRevision(db *gorm.DB, revisionID int64) ([]RevisionComponent, error) {
	var comps []RevisionComponent
	if err := db.Where("revision_id = ?", revisionID).Find(&comps).Error; err != nil {
		return nil, err
	}
	return comps, nil
}

// GetComponentsByRevisionWithLocations 查詢指定 Revision 的所有 RevisionComponent 記錄，並 Preload 其 PartLocations。
//
// 參數：
//   - db：GORM 資料庫連線
//   - revisionID：要查詢的 BOM Revision ID
//
// 回傳：含 Locations 的 RevisionComponent 切片或錯誤
func GetComponentsByRevisionWithLocations(db *gorm.DB, revisionID int64) ([]RevisionComponent, error) {
	var comps []RevisionComponent
	if err := db.Preload("Locations").Where("revision_id = ?", revisionID).Find(&comps).Error; err != nil {
		return nil, err
	}
	return comps, nil
}

// GetComponent 依 ID 查詢單一 RevisionComponent。
//
// 參數：
//   - db：GORM 資料庫連線
//   - id：Component 主鍵 ID
//
// 回傳：RevisionComponent 指標或 ErrComponentNotFound
func GetComponent(db *gorm.DB, id int64) (*RevisionComponent, error) {
	var comp RevisionComponent
	if err := db.First(&comp, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrComponentNotFound
		}
		return nil, err
	}
	return &comp, nil
}

// ─────────────────────────────────────────────
// PartLocation CRUD
// ─────────────────────────────────────────────

// CreatePartLocationsInBatch 批次建立 PartLocation 記錄。
// 呼叫前須確保每筆記錄的 ComponentID 已正確指定對應主料的 RevisionComponent.ID。
//
// 參數：
//   - db：GORM 資料庫連線
//   - locations：要建立的 PartLocation 切片
//
// 回傳：建立失敗則回傳錯誤
func CreatePartLocationsInBatch(db *gorm.DB, locations []PartLocation) error {
	if len(locations) == 0 {
		return nil
	}
	return db.CreateInBatches(&locations, 500).Error
}

// GetPartLocationsByComponentIDs 批量查詢指定 Component IDs 的所有 PartLocation 記錄。
//
// 參數：
//   - db：GORM 資料庫連線
//   - componentIDs：要查詢的 Component ID 列表
//
// 回傳：PartLocation 切片或錯誤
func GetPartLocationsByComponentIDs(db *gorm.DB, componentIDs []int64) ([]PartLocation, error) {
	if len(componentIDs) == 0 {
		return nil, nil
	}
	var locations []PartLocation
	if err := db.Where("component_id IN ?", componentIDs).Find(&locations).Error; err != nil {
		return nil, err
	}
	return locations, nil
}

// DeletePartLocationsByRevision 刪除指定 Revision 下所有 Component 關聯的 PartLocation 記錄。
//
// 參數：
//   - db：GORM 資料庫連線
//   - revisionID：要刪除的 BOM Revision ID
//
// 回傳：刪除失敗則回傳錯誤
func DeletePartLocationsByRevision(db *gorm.DB, revisionID int64) error {
	return db.Where(
		"component_id IN (?)",
		db.Model(&RevisionComponent{}).Select("id").Where("revision_id = ?", revisionID),
	).Delete(&PartLocation{}).Error
}
