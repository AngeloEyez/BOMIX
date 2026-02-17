import { describe, it, expect, vi, beforeEach } from 'vitest';
import matrixService from '../../../src/main/services/matrix.service.js';
import matrixModelRepo from '../../../src/main/database/repositories/matrix-model.repo.js';
import matrixSelectionRepo from '../../../src/main/database/repositories/matrix-selection.repo.js';
import bomService from '../../../src/main/services/bom.service.js';

vi.mock('../../../src/main/database/repositories/matrix-model.repo.js');
vi.mock('../../../src/main/database/repositories/matrix-selection.repo.js');
vi.mock('../../../src/main/services/bom.service.js');

describe('Matrix Service', () => {
    const bomRevisionId = 1;

    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe('createModels', () => {
        it('should create default models if none provided', () => {
            matrixModelRepo.create.mockImplementation((data) => ({ id: Math.random(), ...data }));

            const result = matrixService.createModels(bomRevisionId);

            expect(result).toHaveLength(3);
            expect(result[0].name).toBe('Model A');
            expect(result[1].name).toBe('Model B');
            expect(result[2].name).toBe('Model C');
            expect(matrixModelRepo.create).toHaveBeenCalledTimes(3);
        });

        it('should create provided models', () => {
            const models = [{ name: 'Test A' }, { name: 'Test B' }];
            matrixModelRepo.create.mockImplementation((data) => ({ id: Math.random(), ...data }));

            const result = matrixService.createModels(bomRevisionId, models);

            expect(result).toHaveLength(2);
            expect(result[0].name).toBe('Test A');
            expect(matrixModelRepo.create).toHaveBeenCalledTimes(2);
        });
    });

    describe('deleteModel', () => {
        it('should delete model if no selections exist', () => {
            matrixSelectionRepo.countByModelId.mockReturnValue(0);
            matrixModelRepo.delete.mockReturnValue(true);

            const result = matrixService.deleteModel(1);
            expect(result.success).toBe(true);
        });

        it('should throw error if selections exist', () => {
            matrixSelectionRepo.countByModelId.mockReturnValue(5);
            expect(() => matrixService.deleteModel(1)).toThrow('該 Model 已有選擇紀錄');
        });

        it('should throw error if delete fails', () => {
            matrixSelectionRepo.countByModelId.mockReturnValue(0);
            matrixModelRepo.delete.mockReturnValue(false);
            expect(() => matrixService.deleteModel(1)).toThrow('刪除失敗');
        });
    });

    describe('getMatrixData', () => {
        it('should return implicit selections for single source groups', () => {
            // Mock Models
            const models = [{ id: 10, name: 'Model A' }];
            matrixModelRepo.findByBomRevisionId.mockReturnValue(models);

            // Mock Explicit Selections (Empty)
            matrixSelectionRepo.findByBomRevisionId.mockReturnValue([]);

            // Mock BOM Items (from executeView)
            // One item with only Main Source (Implicit candidate)
            // One item with Second Source (No implicit)
            const bomItems = [
                {
                    id: 999, // Representative ID
                    supplier: 'S1',
                    supplier_pn: 'P1',
                    second_sources: []
                },
                {
                    id: 888,
                    supplier: 'S2',
                    supplier_pn: 'P2',
                    second_sources: [{ supplier: 'SS1' }]
                }
            ];
            bomService.executeView.mockReturnValue(bomItems);

            const result = matrixService.getMatrixData(bomRevisionId);

            expect(result.models).toEqual(models);
            expect(result.selections).toHaveLength(1); // Only one implicit selection

            const implicit = result.selections[0];
            expect(implicit.is_implicit).toBe(true);
            expect(implicit.group_key).toBe('S1|P1');
            expect(implicit.matrix_model_id).toBe(10);
            expect(implicit.selected_id).toBe(999);
        });

        it('should calculate summary status correctly', () => {
             const models = [{ id: 10, name: 'Model A' }];
             matrixModelRepo.findByBomRevisionId.mockReturnValue(models);

             // One explicit selection for the item with second source
             const explicit = [{
                 matrix_model_id: 10,
                 group_key: 'S2|P2',
                 selected_type: 'second_source',
                 selected_id: 200
             }];
             matrixSelectionRepo.findByBomRevisionId.mockReturnValue(explicit);

             const bomItems = [
                { id: 999, supplier: 'S1', supplier_pn: 'P1', second_sources: [] }, // Implicit
                { id: 888, supplier: 'S2', supplier_pn: 'P2', second_sources: [{}] } // Explicitly selected
            ];
            bomService.executeView.mockReturnValue(bomItems);

            const result = matrixService.getMatrixData(bomRevisionId);

            // Both groups are covered (one implicit, one explicit)
            expect(result.summary.modelStatus[10].isComplete).toBe(true);
            expect(result.summary.isSafe).toBe(true);
            expect(result.selections).toHaveLength(2); // 1 explicit + 1 implicit
        });

        it('should indicate incomplete status', () => {
             const models = [{ id: 10, name: 'Model A' }];
             matrixModelRepo.findByBomRevisionId.mockReturnValue(models);
             matrixSelectionRepo.findByBomRevisionId.mockReturnValue([]); // No explicit

             const bomItems = [
                { id: 888, supplier: 'S2', supplier_pn: 'P2', second_sources: [{}] } // Needs selection, none provided
            ];
            bomService.executeView.mockReturnValue(bomItems);

            const result = matrixService.getMatrixData(bomRevisionId);

            expect(result.summary.modelStatus[10].isComplete).toBe(false);
            expect(result.summary.isSafe).toBe(false);
        });
    });
});
