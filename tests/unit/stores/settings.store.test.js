import { describe, it, expect, vi, beforeEach } from 'vitest';
import useSettingsStore from '../../../src/renderer/stores/useSettingsStore.js';

// Mock window.api
global.window = {
    api: {
        settings: {
            get: vi.fn(),
            save: vi.fn().mockResolvedValue({ success: true }),
        },
    },
};

describe('Settings Store', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        // Reset store
        useSettingsStore.setState({
            settings: { bomSidebarWidth: 250, isBomSidebarCollapsed: false, theme: 'light', activeThemeId: 'default' },
            theme: 'light',
            activeThemeId: 'default',
            isLoading: false,
            error: null
        });
    });

    it('should load settings and update state', async () => {
        const mockSettings = {
            bomSidebarWidth: 300,
            theme: 'dark'
        };
        window.api.settings.get.mockResolvedValue({ success: true, data: mockSettings });

        await useSettingsStore.getState().loadSettings();

        const state = useSettingsStore.getState();
        expect(state.settings.bomSidebarWidth).toBe(300);
        expect(state.settings.theme).toBe('dark');
        expect(state.theme).toBe('dark'); // Synced top-level
    });

    it('should toggle theme and persist', async () => {
        // Initial state is light
        expect(useSettingsStore.getState().theme).toBe('light');

        useSettingsStore.getState().toggleTheme();

        const state = useSettingsStore.getState();
        expect(state.theme).toBe('dark');
        expect(state.settings.theme).toBe('dark');

        expect(window.api.settings.save).toHaveBeenCalledWith(expect.objectContaining({
            theme: 'dark'
        }));
    });

    it('should set theme ID and persist', async () => {
        useSettingsStore.getState().setThemeId('emerald');

        const state = useSettingsStore.getState();
        expect(state.activeThemeId).toBe('emerald');
        expect(state.settings.activeThemeId).toBe('emerald');

        expect(window.api.settings.save).toHaveBeenCalledWith(expect.objectContaining({
            activeThemeId: 'emerald'
        }));
    });
});
