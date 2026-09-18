package backend

import (
	"fmt"
	"time"

	"bomix-app/backend/db"
)

// ==================== 專案管理 (Project Management) ====================

// GetProjects 取得目前系列中的所有專案清單
//
// 參數：
//   - seriesID: 系列 ID
//
// 回傳：
//   - []*Project: 專案清單 DTO
//   - error: 若未開啟系列資料庫或查詢失敗則回傳錯誤
func (a *App) GetProjects(seriesID int64) ([]*Project, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.db == nil {
		return nil, fmt.Errorf("no series is currently open")
	}

	projects, err := db.GetProjects(a.db, seriesID)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	result := make([]*Project, len(projects))
	for i := range projects {
		result[i] = &Project{
			ID:          projects[i].ID,
			SeriesID:    projects[i].SeriesID,
			Code:        projects[i].Code,
			Description: projects[i].Description,
			CreatedAt:   projects[i].CreatedAt.Format(time.RFC3339),
			UpdatedAt:   projects[i].UpdatedAt.Format(time.RFC3339),
		}
	}

	return result, nil
}

// GetProject 依專案 ID 取得指定專案資訊
//
// 參數：
//   - id: 專案 ID
//
// 回傳：
//   - *Project: 專案資訊 DTO
//   - error: 若未開啟系列資料庫或專案不存在則回傳錯誤
func (a *App) GetProject(id int64) (*Project, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.db == nil {
		return nil, fmt.Errorf("no series is currently open")
	}

	project, err := db.GetProject(a.db, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &Project{
		ID:          project.ID,
		SeriesID:    project.SeriesID,
		Code:        project.Code,
		Description: project.Description,
		CreatedAt:   project.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   project.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// ==================== 版本管理 (Revision Management) ====================

// GetRevisions 取得指定專案的所有 BOM Revision 清單
//
// 參數：
//   - projectID: 專案 ID
//
// 回傳：
//   - []*BomRevision: Revision 清單 DTO（包含 Model 數量統計）
//   - error: 若未開啟系列資料庫或查詢失敗則回傳錯誤
func (a *App) GetRevisions(projectID int64) ([]*BomRevision, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.db == nil {
		return nil, fmt.Errorf("no series is currently open")
	}

	revisions, err := db.GetRevisions(a.db, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get revisions: %w", err)
	}

	result := make([]*BomRevision, len(revisions))
	for i, r := range revisions {
		result[i] = &BomRevision{
			ID:               r.ID,
			ProjectID:        r.ProjectID,
			Phase:            r.Phase,
			Version:          r.Version,
			Description:      r.Description,
			SchematicVersion: r.SchematicVersion,
			PCBVersion:       r.PCBVersion,
			PCAPN:            r.PCAPN,
			Date:             r.Date,
			Mode:             r.Mode,
			SourceFile:       r.SourceFile,
			ModelCount:       len(r.MatrixModels),
			CreatedAt:        r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        r.UpdatedAt.Format(time.RFC3339),
		}
	}

	return result, nil
}

// GetRevision 依 Revision ID 取得指定 BOM Revision 詳細資訊
//
// 參數：
//   - id: Revision ID
//
// 回傳：
//   - *BomRevision: Revision 詳細資訊 DTO
//   - error: 若未開啟系列資料庫或 Revision 不存在則回傳錯誤
func (a *App) GetRevision(id int64) (*BomRevision, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.db == nil {
		return nil, fmt.Errorf("no series is currently open")
	}

	revision, err := db.GetRevision(a.db, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get revision: %w", err)
	}

	return &BomRevision{
		ID:               revision.ID,
		ProjectID:        revision.ProjectID,
		Phase:            revision.Phase,
		Version:          revision.Version,
		Description:      revision.Description,
		SchematicVersion: revision.SchematicVersion,
		PCBVersion:       revision.PCBVersion,
		PCAPN:            revision.PCAPN,
		Date:             revision.Date,
		Mode:             revision.Mode,
		SourceFile:       revision.SourceFile,
		ModelCount:       len(revision.MatrixModels),
		CreatedAt:        revision.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        revision.UpdatedAt.Format(time.RFC3339),
	}, nil
}
