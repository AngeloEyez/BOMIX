package excel

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"bomix-app/backend/db"
	"bomix-app/backend/logger"
	"bomix-app/backend/types"
	"bomix-app/backend/view"
)

func TestInvestigateINA234(t *testing.T) {
	ebomPath1 := "testdata/SERENNO-PCM_EZBOM_SI1_0.3_BOM_20260817_1000.WP(compared).xls"
	ebomPath2 := "testdata/TARIS-PCM_EZBOM_SI1_0.3_BOM_20260817_1000.WP(compared).xls"

	if _, err := os.Stat(ebomPath1); os.IsNotExist(err) {
		t.Fatalf("File 1 not found: %s", ebomPath1)
	}
	if _, err := os.Stat(ebomPath2); os.IsNotExist(err) {
		t.Fatalf("File 2 not found: %s", ebomPath2)
	}

	lg := logger.NewLogger(100)

	t.Log("=== 1. Importing both EBOMs into DB ===")
	gdb := setupMatrixTestDB(t)
	reader := NewReader(gdb, lg)

	importResults, err := reader.ImportExcel([]string{ebomPath1, ebomPath2})
	if err != nil {
		t.Fatalf("ImportExcel failed: %v", err)
	}
	t.Logf("Import results: %+v", importResults)

	t.Log("=== 2. Querying DB for INA234 ===")
	var revs []db.BomRevision
	gdb.Order("id asc").Find(&revs)
	revIDs := make([]int64, len(revs))
	for i, r := range revs {
		revIDs[i] = r.ID
	}

	viewSvc := view.NewService(gdb, lg)

	t.Log("=== 3. Querying View Service (ViewCCL) ===")
	cclResult, err := viewSvc.Query(view.ViewQuery{
		RevisionIDs: revIDs,
		ViewType:    view.ViewCCL,
	})
	if err != nil {
		t.Fatalf("ViewCCL query failed: %v", err)
	}

	// 驗證未經過 MergePartGroupsByMaterial 時，包含 Type 維度（分為 SMD 與 BOTTOM 2 筆）
	var rawINAGroups []view.ViewPartGroup
	for _, pg := range cclResult.PartGroups {
		if strings.Contains(pg.MainSupplierPN, "INA234") {
			rawINAGroups = append(rawINAGroups, pg)
		}
	}
	if len(rawINAGroups) != 2 {
		t.Errorf("View 基礎查詢期望 2 筆 INA234 (SMD + BOTTOM)，實際得到 %d 筆", len(rawINAGroups))
	}

	t.Log("=== 4. Testing BigMatrix Export with MergePartGroupsByMaterial ===")
	// 套用二階物料合併
	mergedPartGroups := view.MergePartGroupsByMaterial(cclResult.PartGroups)
	var mergedINAGroups []view.ViewPartGroup
	for _, pg := range mergedPartGroups {
		if strings.Contains(pg.MainSupplierPN, "INA234") {
			mergedINAGroups = append(mergedINAGroups, pg)
		}
	}
	if len(mergedINAGroups) != 1 {
		t.Fatalf("MergePartGroupsByMaterial 後期望 1 筆 INA234，實際得到 %d 筆", len(mergedINAGroups))
	}
	inaMerged := mergedINAGroups[0]
	if inaMerged.Locations != "PSU1,PSU2" {
		t.Errorf("INA234 合併後 Locations 期望 'PSU1,PSU2'，實際得到 %q", inaMerged.Locations)
	}
	if inaMerged.Qty != 2 {
		t.Errorf("INA234 合併後 Qty 期望 2，實際得到 %d", inaMerged.Qty)
	}

	outDir := t.TempDir()
	bmOutPath := outDir + "/Investigate_BigMatrix.xlsx"

	writer, err := NewWriter(lg)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	revDataList := make([]RevisionData, 0, len(cclResult.Revisions))
	for _, vr := range cclResult.Revisions {
		revDataList = append(revDataList, RevisionData{
			ID:               fmt.Sprintf("%d", vr.ID),
			ProjectCode:      vr.ProjectCode,
			Description:      vr.Description,
			SchematicVersion: vr.SchematicVersion,
			PCBVersion:       vr.PCBVersion,
			PCAPN:            vr.PCAPN,
			Phase:            vr.Phase,
			Version:          vr.Version,
			Date:             vr.Date,
			SourceFile:       vr.SourceFile,
			ModelNames:       vr.ModelNames,
			ModelQty:         vr.ModelQty,
			ModelQtyByOrder:  vr.ModelQtyByOrder,
		})
	}

	partDataList := make([]PartData, 0, len(mergedPartGroups))
	for _, pg := range mergedPartGroups {
		ssList := make([]SecondSourceData, 0, len(pg.SecondSources))
		for _, ss := range pg.SecondSources {
			ssList = append(ssList, SecondSourceData{
				HHPN:              ss.HHPN,
				Supplier:          ss.Supplier,
				SupplierPn:        ss.SupplierPN,
				Description:       ss.Description,
				Remark:            ss.Remark,
				SourceRevisionIDs: ss.SourceRevisionIDs,
			})
		}

		partDataList = append(partDataList, PartData{
			Item:              pg.Item,
			HHPN:              pg.HHPN,
			Description:       pg.Description,
			Supplier:          pg.MainSupplier,
			SupplierPn:        pg.MainSupplierPN,
			Qty:               pg.Qty,
			Location:          pg.Locations,
			Type:              pg.Type,
			BOMStatus:         pg.BOMStatus,
			CCL:               pg.CCL,
			Remark:            pg.Remark,
			SecondSources:     ssList,
			SourceRevisionIDs: pg.SourceRevisionIDs,
		})
	}

	exportOpts := ExportOptions{
		Format:      types.FormatBigMatrix,
		RevisionIDs: []string{fmt.Sprintf("%d", revIDs[0]), fmt.Sprintf("%d", revIDs[1])},
		OutputPath:  bmOutPath,
		Revisions:   revDataList,
		PartData:    partDataList,
	}

	outFiles, err := writer.ExportExcel(exportOpts)
	if err != nil {
		t.Fatalf("ExportExcel failed: %v", err)
	}
	t.Logf("BigMatrix Exported files: %v", outFiles)

	wbBM, err := OpenWorkbook(bmOutPath, lg)
	if err != nil {
		t.Fatalf("Failed to open exported BigMatrix: %v", err)
	}
	defer wbBM.Close()

	bmRows, err := wbBM.GetRows("BigMatrix")
	if err != nil {
		t.Fatalf("GetRows BigMatrix failed: %v", err)
	}
	var inaBMRows []string
	for rIdx, row := range bmRows {
		rowStr := strings.Join(row, " | ")
		if strings.Contains(rowStr, "INA234") {
			inaBMRows = append(inaBMRows, rowStr)
			t.Logf("BigMatrix Row %d (1-based: %d): %s", rIdx, rIdx+1, rowStr)
		}
	}
	if len(inaBMRows) != 1 {
		t.Errorf("BigMatrix 匯出中 INA234 期望只有 1 列 Main Source，實際有 %d 列", len(inaBMRows))
	}

	t.Log("=== 5. Testing Matrix Export (maintaining Type split) ===")
	// Matrix 匯出使用原始 cclResult.PartGroups (含 Type)
	matrixPartDataList := make([]PartData, 0, len(cclResult.PartGroups))
	for _, pg := range cclResult.PartGroups {
		ssList := make([]SecondSourceData, 0, len(pg.SecondSources))
		for _, ss := range pg.SecondSources {
			ssList = append(ssList, SecondSourceData{
				HHPN:              ss.HHPN,
				Supplier:          ss.Supplier,
				SupplierPn:        ss.SupplierPN,
				Description:       ss.Description,
				Remark:            ss.Remark,
				SourceRevisionIDs: ss.SourceRevisionIDs,
			})
		}

		matrixPartDataList = append(matrixPartDataList, PartData{
			Item:              pg.Item,
			HHPN:              pg.HHPN,
			Description:       pg.Description,
			Supplier:          pg.MainSupplier,
			SupplierPn:        pg.MainSupplierPN,
			Qty:               pg.Qty,
			Location:          pg.Locations,
			Type:              pg.Type,
			BOMStatus:         pg.BOMStatus,
			CCL:               pg.CCL,
			Remark:            pg.Remark,
			SecondSources:     ssList,
			SourceRevisionIDs: pg.SourceRevisionIDs,
		})
	}

	matrixOutPath := outDir + "/Investigate_Matrix.xlsx"
	matrixExportOpts := ExportOptions{
		Format:      types.FormatMatrix,
		RevisionIDs: []string{fmt.Sprintf("%d", revIDs[0])},
		OutputPath:  matrixOutPath,
		Revisions:   revDataList[:1],
		PartData:    matrixPartDataList,
	}

	outMatrixFiles, err := writer.ExportExcel(matrixExportOpts)
	if err != nil {
		t.Fatalf("Matrix ExportExcel failed: %v", err)
	}
	t.Logf("Matrix Exported files: %v", outMatrixFiles)

	wbMatrix, err := OpenWorkbook(matrixOutPath, lg)
	if err != nil {
		t.Fatalf("Failed to open exported Matrix: %v", err)
	}
	defer wbMatrix.Close()

	smdRows, _ := wbMatrix.GetRows("SMD")
	bottomRows, _ := wbMatrix.GetRows("BOTTOM")

	foundInSMD := false
	for _, r := range smdRows {
		if strings.Contains(strings.Join(r, " | "), "INA234") {
			foundInSMD = true
			t.Logf("Matrix SMD Sheet INA234 Row: %s", strings.Join(r, " | "))
		}
	}
	foundInBottom := false
	for _, r := range bottomRows {
		if strings.Contains(strings.Join(r, " | "), "INA234") {
			foundInBottom = true
			t.Logf("Matrix BOTTOM Sheet INA234 Row: %s", strings.Join(r, " | "))
		}
	}

	if !foundInSMD {
		t.Errorf("Matrix SMD Sheet 應包含 INA234 (PSU1)")
	}
	if !foundInBottom {
		t.Errorf("Matrix BOTTOM Sheet 應包含 INA234 (PSU2)")
	}
}
