import { describe, it, expect, vi, beforeEach } from 'vitest';
import useBomStore from '../../../src/renderer/stores/useBomStore.js';
import useSettingsStore from '../../../src/renderer/stores/useSettingsStore.js';

// Mock window.api
global.window = {
    document: {
        documentElement: {
            classList: {
                add: vi.fn(),
                remove: vi.fn()
            }
        },
        head: {
            appendChild: vi.fn()
        },
        getElementById: vi.fn().mockReturnValue(null),
        createElement: vi.fn().mockReturnValue({ id: '', textContent: '' })
    },
    api: {
        settings: {
            get: vi.fn().mockResolvedValue({ success: true, data: {} }),
            save: vi.fn().mockResolvedValue({ success: true }),
        },
        bom: {
            getView: vi.fn().mockResolvedValue({ success: true, data: [] }),
            getRevisions: vi.fn().mockResolvedValue({ success: true, data: [] }),
        },
        theme: {
            getList: vi.fn().mockResolvedValue({ success: true, data: [] }),
            getAttributes: vi.fn().mockResolvedValue({ success: true, data: {} }),
        }
    },
};

describe('Integration: Stores & UI Logic', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        // Reset stores
        useBomStore.setState({
            selectedRevisionId: null,
            selectedRevisionIds: new Set(),
            bomMode: 'BOM',
            viewCache: {}
        });
        useSettingsStore.setState({
            settings: { bomSidebarWidth: 250, isBomSidebarCollapsed: false, theme: 'light', activeThemeId: 'default' }
        });
    });

    describe('Dashboard -> BomPage Navigation', () => {
        it('should select single BOM and switch to BOM mode on toggleRevisionSelection(id, false)', async () => {
            const bomId = 101;
            const mockRevision = { id: bomId, phase_name: 'PV', version: '1.0' };

            // Setup mock state
            useBomStore.setState({ revisions: [mockRevision] });

            // Simulate Dashboard click (single select)
            await useBomStore.getState().toggleRevisionSelection(bomId, false);

            const state = useBomStore.getState();
            expect(state.selectedRevisionId).toBe(bomId);
            expect(state.selectedRevisionIds.has(bomId)).toBe(true);
            expect(state.selectedRevisionIds.size).toBe(1);
            expect(state.bomMode).toBe('BOM'); // Default single selection mode

            // Should trigger fetch
            expect(window.api.bom.getView).toHaveBeenCalled();
        });

        it('should handle multi-selection and switch to BIGBOM mode if not Matrix', async () => {
            const bomId1 = 101;
            const bomId2 = 102;
            const mockRevisions = [
                { id: bomId1, phase_name: 'PV', version: '1.0' },
                { id: bomId2, phase_name: 'PV', version: '1.1' }
            ];

            useBomStore.setState({ revisions: mockRevisions });

            // Select first
            await useBomStore.getState().toggleRevisionSelection(bomId1, false);
            // Select second (multi)
            await useBomStore.getState().toggleRevisionSelection(bomId2, true);

            const state = useBomStore.getState();
            expect(state.selectedRevisionIds.size).toBe(2);
            expect(state.selectedRevisionId).toBeNull(); // Single ID is null for multi
            expect(state.bomMode).toBe('BIGBOM'); // Auto-switch for multi
        });
    });

    describe('Sidebar Settings', () => {
        it('should load sidebar settings correctly', async () => {
            const mockSettings = { bomSidebarWidth: 400 };
            window.api.settings.get.mockResolvedValue({ success: true, data: mockSettings });

            await useSettingsStore.getState().loadSettings();

            const state = useSettingsStore.getState();
            // Store structure changed: properties are top-level now, not inside `settings` object
            expect(state.bomSidebarWidth).toBe(400);
        });

        it('should safely default if settings are empty', async () => {
            window.api.settings.get.mockResolvedValue({ success: true, data: {} });

            await useSettingsStore.getState().loadSettings();

            const state = useSettingsStore.getState();
            expect(state.bomSidebarWidth).toBe(250);
        });
    });
});
