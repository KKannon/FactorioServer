// @vitest-environment jsdom
import {beforeEach, describe, expect, it, vi} from 'vitest';

beforeEach(() => {
    localStorage.clear();
    vi.stubGlobal('matchMedia', vi.fn(() => ({
        matches: false,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
    })));
    vi.resetModules();
});

const loadFor = async language => {
    const preferences = await import('./preferences');
    preferences.applyPreferences({language});
    return import('./serverSettings');
};

describe('localized server settings', () => {
    it('uses Portuguese labels, descriptions, and enum values', async () => {
        const settings = await loadFor('pt-BR');
        expect(settings.localizedServerSetting('game_password')).toEqual({
            label: 'Senha do jogo',
            description: 'Senha opcional que os jogadores devem informar para entrar.',
        });
        expect(settings.localizedVisibility('public')).toBe('Lista pública de servidores');
        expect(settings.commandOptions()).toContainEqual(['admins-only', 'Somente administradores']);
    });

    it('supports English and Spanish and keeps unknown backend fields readable', async () => {
        let settings = await loadFor('en-US');
        expect(settings.localizedServerSetting('auto_pause').label).toBe('Automatic pause');

        settings = await loadFor('es-ES');
        expect(settings.localizedServerSetting('auto_pause').label).toBe('Pausa automática');
        expect(settings.localizedVisibility('lan')).toBe('Red local (LAN)');
        expect(settings.localizedServerSetting('future_setting', 'Backend help')).toEqual({
            label: 'Future Setting',
            description: 'Backend help',
        });
    });
});
