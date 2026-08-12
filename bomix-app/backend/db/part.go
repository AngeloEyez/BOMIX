package db

import (
	"errors"

	"gorm.io/gorm"
)

// CreatePartsInBatch 批次建立 Part 記錄（純化物料，不含 Location）
//
// 參數：
//   - db：GORM 資料庫連線
//   - parts：要建立的 Part 切片（GORM 會回填 ID）
//
// 回傳：建立失敗則回傳錯誤
func CreatePartsInBatch(db *gorm.DB, parts []Part) error {
	return db.Create(&parts).Error
}

// DeletePartsByRevision 刪除指定 Revision 的所有 Part 記錄。
// 由於 Part.Locations 設定了 OnDelete:CASCADE，關聯的 PartLocation 將自動被刪除。
//
// 參數：
//   - db：GORM 資料庫連線
//   - revisionID：要刪除的 BOM Revision ID
func DeletePartsByRevision(db *gorm.DB, revisionID int64) error {
	return db.Where("revision_id = ?", revisionID).Delete(&Part{}).Error
}

// GetPartsByRevision 查詢指定 Revision 的所有 Part 記錄（不含 PartLocation）。
//
// 參數：
//   - db：GORM 資料庫連線
//   - revisionID：要查詢的 BOM Revision ID
//
// 回傳：Part 切片或錯誤
func GetPartsByRevision(db *gorm.DB, revisionID int64) ([]Part, error) {
	var parts []Part
	if err := db.Where("revision_id = ?", revisionID).Find(&parts).Error; err != nil {
		return nil, err
	}
	return parts, nil
}

// GetPartsByRevisionWithLocations 查詢指定 Revision 的所有 Part 記錄，並 Preload 關聯的 PartLocations。
//
// 參數：
//   - db：GORM 資料庫連線
//   - revisionID：要查詢的 BOM Revision ID
//
// 回傳：含 Locations 的 Part 切片或錯誤
func GetPartsByRevisionWithLocations(db *gorm.DB, revisionID int64) ([]Part, error) {
	var parts []Part
	if err := db.Preload("Locations").Where("revision_id = ?", revisionID).Find(&parts).Error; err != nil {
		return nil, err
	}
	return parts, nil
}

// GetPartsByRevisionAndType 查詢指定 Revision 且含有符合製程類型 Location 的 Part 記錄。
//
// 參數：
//   - db：GORM 資料庫連線
//   - revisionID：要查詢的 BOM Revision ID
//   - partType：製程類型（SMD / PTH / BOTTOM）
//
// 回傳：Part 切片或錯誤
func GetPartsByRevisionAndType(db *gorm.DB, revisionID int64, partType string) ([]Part, error) {
	var parts []Part
	if err := db.Joins("JOIN part_locations ON part_locations.part_id = parts.id").
		Where("parts.revision_id = ? AND part_locations.type = ?", revisionID, partType).
		Group("parts.id").Find(&parts).Error; err != nil {
		return nil, err
	}
	return parts, nil
}

// GetPart 依 ID 查詢單一 Part 記錄。
//
// 參數：
//   - db：GORM 資料庫連線
//   - id：Part 主鍵 ID
//
// 回傳：Part 指標或錯誤（找不到時回傳 ErrPartNotFound）
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

// ─────────────────────────────────────────────
// PartLocation CRUD
// ─────────────────────────────────────────────

// CreatePartLocationsInBatch 批次建立 PartLocation 記錄。
// 呼叫前須確保每筆記錄的 PartID 已正確設定。
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

// GetPartLocationsForParts 批量查詢指定 Part IDs 的所有 PartLocation 記錄。
// 採用 IN 子句，避免 N+1 查詢問題。
//
// 參數：
//   - db：GORM 資料庫連線
//   - partIDs：要查詢的 Part ID 列表
//
// 回傳：PartLocation 切片或錯誤
func GetPartLocationsForParts(db *gorm.DB, partIDs []int64) ([]PartLocation, error) {
	if len(partIDs) == 0 {
		return nil, nil
	}
	var locations []PartLocation
	if err := db.Where("part_id IN ?", partIDs).Find(&locations).Error; err != nil {
		return nil, err
	}
	return locations, nil
}

// DeletePartLocationsByRevision 刪除指定 Revision 下所有 Part 關聯的 PartLocation 記錄。
// 使用子查詢定位屬於此 Revision 的 Part IDs。
//
// 參數：
//   - db：GORM 資料庫連線
//   - revisionID：要刪除的 BOM Revision ID
//
// 回傳：刪除失敗則回傳錯誤
func DeletePartLocationsByRevision(db *gorm.DB, revisionID int64) error {
	return db.Where(
		"part_id IN (?)",
		db.Model(&Part{}).Select("id").Where("revision_id = ?", revisionID),
	).Delete(&PartLocation{}).Error
}
