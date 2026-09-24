// @vitest-environment jsdom
import {beforeEach, describe, expect, it, vi} from 'vitest';

let dark = false;
let reduced = false;

beforeEach(() => {
    localStorage.clear();
    dark = false;
    reduced = false;
    vi.stubGlobal('matchMedia', vi.fn(query => ({
        matches: query.includes('color-scheme') ? dark : reduced,
        addEventListener: vi.fn(), removeEventListener: vi.fn(),
    })));
    vi.resetModules();
});

describe('central preferences adapter', () => {
    it('applies system theme, language, timezone, density, and device motion accessibility', async () => {
        dark = true;
        reduced = true;
        const preferences = await import('./preferences');
        preferences.applyPreferences({theme: 'system', language: 'es-ES', timezone: 'Europe/Madrid', date_format: 'YYYY-MM-DD', compact_mode: true, reduce_motion: false, accent_color: '#E39827'});
        expect(document.documentElement.dataset.theme).toBe('dark');
        expect(document.documentElement.lang).toBe('es-ES');
        expect(document.documentElement.dataset.reduceMotion).toBe('true');
        expect(document.documentElement.dataset.compact).toBe('true');
        expect(preferences.formatDateTime('2026-01-02T12:30:00Z')).toMatch(/^2026-01-02 /);
    });

    it('rejects inaccessible colors and preserves unknown fields in the offline cache', async () => {
        const preferences = await import('./preferences');
        preferences.applyPreferences({theme: 'light', accent_color: '#000000', future_option: {enabled: true}});
        expect(document.documentElement.style.getPropertyValue('--accent-color')).toBe('#E39827');
        expect(JSON.parse(localStorage.getItem('fsm_preferences')).future_option).toEqual({enabled: true});
    });
});

