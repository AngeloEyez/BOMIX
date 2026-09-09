package db

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"gorm.io/gorm"
)

// MaterialKey 產生標準物料識別鍵 (supplier|supplier_pn)
func MaterialKey(supplier, supplierPN string) string {
	return fmt.Sprintf("%s|%s", strings.TrimSpace(supplier), strings.TrimSpace(supplierPN))
}

// UpsertMaterials 批次 Upsert 全域物料（僅在 EBOM 匯入時呼叫）。
//
// 處理邏輯：
//   1. 依據 (supplier, supplier_pn) 去重與比對現有物料。
//   2. 若物料不存在：執行 INSERT，並記錄 inserted++。
//   3. 若物料已存在：
//      - 空白值防護：HHPN 與 Description 僅在新值非空時才覆寫舊值；
//      - Remark 排除空白防護：若新值為空字串，依然將 DB 既有 Remark 清空為空字串；
//      - 僅在實際欄位有變動時執行 UPDATE，並記錄 updated++。
//   4. 記錄 Log：更新/新增數量以 Info 等級輸出，詳細變更欄位以 Debug 等級輸出。
//
// 參數：
//   - db：GORM 資料庫連線
//   - materials：待寫入/更新的 Material 切片
//   - lg：日誌記錄器（可為 nil）
//
// 回傳：
//   - inserted：實際新增的物料筆數
//   - updated：實際更新的物料筆數
//   - err：若資料庫操作失敗則回傳錯誤
func UpsertMaterials(db *gorm.DB, materials []Material, lg MatrixLogger) (int, int, error) {
	if lg != nil {
		v := reflect.ValueOf(lg)
		if v.Kind() == reflect.Ptr && v.IsNil() {
			lg = nil
		}
	}

	if len(materials) == 0 {
		return 0, 0, nil
	}

	// 1. 記憶體去重（以 supplier|supplier_pn 為鍵，後者覆蓋前者）
	uniqueMaterials := make(map[string]Material, len(materials))
	for _, m := range materials {
		s := strings.TrimSpace(m.Supplier)
		spn := strings.TrimSpace(m.SupplierPN)
		if s == "" && spn == "" {
			continue
		}
		key := fmt.Sprintf("%s|%s", s, spn)
		m.Supplier = s
		m.SupplierPN = spn
		uniqueMaterials[key] = m
	}

	if len(uniqueMaterials) == 0 {
		return 0, 0, nil
	}

	inserted := 0
	updated := 0

	// 2. 於 Transaction 內執行查詢與寫入
	err := db.Transaction(func(tx *gorm.DB) error {
		// 收集所有查詢鍵值
		keys := make([]string, 0, len(uniqueMaterials))
		for k := range uniqueMaterials {
			keys = append(keys, k)
		}

		// 批次查詢既有物料
		existingMap, err := getExistingMaterialsMap(tx, keys)
		if err != nil {
			return fmt.Errorf("查詢既有 Material 失敗: %w", err)
		}

		now := time.Now()
		var toInsert []Material

		for key, newMat := range uniqueMaterials {
			existing, found := existingMap[key]
			if !found {
				// 新增物料
				newMat.CreatedAt = now
				newMat.UpdatedAt = now
				toInsert = append(toInsert, newMat)
			} else {
				// 既有物料，檢查更新
				updates := make(map[string]any)
				var changeLogs []string

				// HHPN 空白值防護：新值非空且與舊值不同時才更新
				trimmedHHPN := strings.TrimSpace(newMat.HHPN)
				if trimmedHHPN != "" && trimmedHHPN != existing.HHPN {
					updates["hhpn"] = trimmedHHPN
					changeLogs = append(changeLogs, fmt.Sprintf("hhpn: '%s' -> '%s'", existing.HHPN, trimmedHHPN))
				}

				// Description 空白值防護：新值非空且與舊值不同時才更新
				trimmedDesc := strings.TrimSpace(newMat.Description)
				if trimmedDesc != "" && trimmedDesc != existing.Description {
					updates["description"] = trimmedDesc
					changeLogs = append(changeLogs, fmt.Sprintf("description: '%s' -> '%s'", existing.Description, trimmedDesc))
				}

				// Remark 排除空白值防護：新值與舊值不同即更新（允許清空為空字串）
				trimmedRemark := strings.TrimSpace(newMat.Remark)
				if trimmedRemark != existing.Remark {
					updates["remark"] = trimmedRemark
					changeLogs = append(changeLogs, fmt.Sprintf("remark: '%s' -> '%s'", existing.Remark, trimmedRemark))
				}

				if len(updates) > 0 {
					updates["updated_at"] = now
					if err := tx.Model(&Material{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
						return fmt.Errorf("更新 Material (ID=%d) 失敗: %w", existing.ID, err)
					}
					updated++
					if lg != nil {
						lg.Debug(fmt.Sprintf("[Material 更新] %s (ID=%d): %s", key, existing.ID, strings.Join(changeLogs, ", ")))
					}
				}
			}
		}

		// 批次插入新物料
		if len(toInsert) > 0 {
			if err := tx.CreateInBatches(&toInsert, 500).Error; err != nil {
				return fmt.Errorf("批次新增 Material 失敗: %w", err)
			}
			inserted = len(toInsert)
			if lg != nil {
				for _, m := range toInsert {
					lg.Debug(fmt.Sprintf("[Material 新增] %s|%s (ID=%d, HHPN=%s)", m.Supplier, m.SupplierPN, m.ID, m.HHPN))
				}
			}
		}

		return nil
	})

	if err != nil {
		return 0, 0, err
	}

	if lg != nil {
		lg.Info(fmt.Sprintf("[Material 批次處理完成] 新增: %d 筆, 更新: %d 筆", inserted, updated))
	}

	return inserted, updated, nil
}

// getExistingMaterialsMap 依據 supplier|supplier_pn 列表查詢已存在的 Material 映射。
func getExistingMaterialsMap(tx *gorm.DB, keys []string) (map[string]Material, error) {
	result := make(map[string]Material, len(keys))
	if len(keys) == 0 {
		return result, nil
	}

	// 為了避免單次 SQL 條件過長，按 500 筆分批查詢
	batchSize := 500
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batchKeys := keys[i:end]

		// 構建 OR 條件或逐批讀取
		var materials []Material
		subQuery := tx.Model(&Material{})
		var conds []string
		var args []any
		for _, k := range batchKeys {
			parts := strings.SplitN(k, "|", 2)
			if len(parts) == 2 {
				conds = append(conds, "(supplier = ? AND supplier_pn = ?)")
				args = append(args, parts[0], parts[1])
			}
		}
		if len(conds) == 0 {
			continue
		}

		if err := subQuery.Where(strings.Join(conds, " OR "), args...).Find(&materials).Error; err != nil {
			return nil, err
		}

		for _, m := range materials {
			k := fmt.Sprintf("%s|%s", m.Supplier, m.SupplierPN)
			result[k] = m
		}
	}

	return result, nil
}

// GetMaterial 依主鍵 ID 查詢單一 Material。
//
// 參數：
//   - db：GORM 資料庫連線
//   - id：物料主鍵 ID
//
// 回傳：Material 指標或 ErrMaterialNotFound
func GetMaterial(db *gorm.DB, id int64) (*Material, error) {
	var m Material
	if err := db.First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMaterialNotFound
		}
		return nil, err
	}
	return &m, nil
}

// GetMaterialBySupplierPN 依據 (supplier, supplier_pn) 查詢單一 Material。
//
// 參數：
//   - db：GORM 資料庫連線
//   - supplier：廠牌
//   - supplierPN：廠牌料號
//
// 回傳：Material 指標或 ErrMaterialNotFound
func GetMaterialBySupplierPN(db *gorm.DB, supplier, supplierPN string) (*Material, error) {
	var m Material
	err := db.Where("supplier = ? AND supplier_pn = ?", strings.TrimSpace(supplier), strings.TrimSpace(supplierPN)).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMaterialNotFound
		}
		return nil, err
	}
	return &m, nil
}

// GetMaterialsByIDs 批量查詢指定 IDs 的 Material 清單（供 View 系統 Late-Binding 使用）。
//
// 參數：
//   - db：GORM 資料庫連線
//   - ids：物料 ID 切片
//
// 回傳：Material 切片或錯誤
func GetMaterialsByIDs(db *gorm.DB, ids []int64) ([]Material, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var materials []Material
	if err := db.Where("id IN ?", ids).Find(&materials).Error; err != nil {
		return nil, err
	}
	return materials, nil
}

// GetMaterialsMapByIDs 批量查詢指定 IDs 的 Material，並轉為 map[int64]Material 方便檢索。
//
// 參數：
//   - db：GORM 資料庫連線
//   - ids：物料 ID 切片
//
// 回傳：以 Material.ID 為鍵的 Material 字典
func GetMaterialsMapByIDs(db *gorm.DB, ids []int64) (map[int64]Material, error) {
	materials, err := GetMaterialsByIDs(db, ids)
	if err != nil {
		return nil, err
	}
	res := make(map[int64]Material, len(materials))
	for _, m := range materials {
		res[m.ID] = m
	}
	return res, nil
}

// GetMaterialMapBySupplierPNs 依據 (supplier|supplier_pn) 鍵值列表批量查詢已存在的 Material。
// 供 Matrix / BigMatrix 匯入比對使用。
//
// 參數：
//   - db：GORM 資料庫連線
//   - keys：格式為 "supplier|supplier_pn" 的字串切片
//
// 回傳：以 "supplier|supplier_pn" 為鍵的 Material 字典
func GetMaterialMapBySupplierPNs(db *gorm.DB, keys []string) (map[string]Material, error) {
	return getExistingMaterialsMap(db, keys)
}

// UpdateMaterialNotes 依據 (supplier|supplier_pn) 鍵值映射批次更新 Material 的 notes 欄位。
// 僅在 BigMatrix 匯入時呼叫。
//
// 處理原則：
//   1. 僅更新資料庫中已存在的 Material，若物料不存在則忽略，不新增資料。
//   2. 僅更新 notes 與 updated_at，不變動 supplier, supplier_pn, hhpn, description, remark 等其餘屬性。
//   3. 當 notes 內容與既有值不同時（包含清空為空字串）才執行 UPDATE。
//
// 參數：
//   - db：GORM 資料庫連線
//   - notesMap：以 "supplier|supplier_pn" 為鍵，notes 內容為值的字典
//   - lg：日誌記錄器（可為 nil）
//
// 回傳：
//   - updated：實際更新的 Material 筆數
//   - err：資料庫操作錯誤
func UpdateMaterialNotes(db *gorm.DB, notesMap map[string]string, lg MatrixLogger) (int, error) {
	if lg != nil {
		v := reflect.ValueOf(lg)
		if v.Kind() == reflect.Ptr && v.IsNil() {
			lg = nil
		}
	}

	if len(notesMap) == 0 {
		return 0, nil
	}

	keys := make([]string, 0, len(notesMap))
	for k := range notesMap {
		keys = append(keys, k)
	}

	updated := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		existingMap, err := getExistingMaterialsMap(tx, keys)
		if err != nil {
			return fmt.Errorf("查詢既有 Material 失敗: %w", err)
		}

		now := time.Now()
		for key, newNotes := range notesMap {
			existing, found := existingMap[key]
			if !found {
				// 規格要求：僅更新既有物料，不新增，亦不變動其他屬性
				continue
			}

			// 若 notes 內容有變更，才執行更新（包含清空為空字串）
			if strings.TrimSpace(existing.Notes) != newNotes {
				updates := map[string]any{
					"notes":      newNotes,
					"updated_at": now,
				}
				if err := tx.Model(&Material{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
					return fmt.Errorf("更新 Material (ID=%d) notes 失敗: %w", existing.ID, err)
				}
				updated++
				if lg != nil {
					lg.Debug(fmt.Sprintf("[Material Notes 更新] %s (ID=%d): '%s' -> '%s'",
						key, existing.ID, existing.Notes, newNotes))
				}
			}
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	if lg != nil && updated > 0 {
		lg.Info(fmt.Sprintf("[Material Notes 更新完成] 成功更新: %d 筆", updated))
	}

	return updated, nil
}

