/**
 * @vitest-environment jsdom
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import useSettingsStore from '../../../src/renderer/stores/useSettingsStore.js';

// Mock window.api
global.window = Object.assign(global.window || {}, {
    api: {
        settings: {
            get: vi.fn().mockResolvedValue({ success: true, data: {} }),
            save: vi.fn().mockResolvedValue({ success: true }),
        },
        theme: {
            getAttributes: vi.fn().mockResolvedValue({ success: true, data: { colors: {} } }),
            getList: vi.fn().mockResolvedValue({ success: true, data: [] })
        }
    }
});

describe('Settings Store', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        // Reset store
        useSettingsStore.setState({
            bomSidebarWidth: 250,
            isBomSidebarCollapsed: false,
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
        expect(state.bomSidebarWidth).toBe(300);
        expect(state.theme).toBe('dark');
        expect(state.theme).toBe('dark'); // Synced top-level
    });

    it('should toggle theme and persist', async () => {
        // Initial state is light
        expect(useSettingsStore.getState().theme).toBe('light');

        await useSettingsStore.getState().toggleTheme();

        const state = useSettingsStore.getState();
        expect(state.theme).toBe('dark');
        expect(state.theme).toBe('dark');

        expect(window.api.settings.save).toHaveBeenCalledWith(expect.objectContaining({
            theme: 'dark'
        }));
    });

    it('should set theme ID and persist', async () => {
        await useSettingsStore.getState().setThemeId('emerald');

        const state = useSettingsStore.getState();
        expect(state.activeThemeId).toBe('emerald');
        expect(state.activeThemeId).toBe('emerald');

        expect(window.api.settings.save).toHaveBeenCalledWith(expect.objectContaining({ themeId: 'emerald' }));
    });
});
