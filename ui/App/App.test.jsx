// @vitest-environment jsdom
import React from 'react';
import {createRoot} from 'react-dom/client';
import {act} from 'react-dom/test-utils';
import {afterEach, beforeAll, beforeEach, describe, expect, it, vi} from 'vitest';

const dependencies = vi.hoisted(() => ({
    handlers: {},
    userStatus: vi.fn(),
    userRefresh: vi.fn(),
    serverStatus: vi.fn(),
    versions: vi.fn(),
    saves: vi.fn(),
}));

vi.mock('../api/resources/user', () => ({default: {
    status: dependencies.userStatus,
    refresh: dependencies.userRefresh,
    logout: vi.fn(),
}}));
vi.mock('../api/resources/server', () => ({default: {
    status: dependencies.serverStatus,
    versions: dependencies.versions,
    start: vi.fn(), stop: vi.fn(), restart: vi.fn(), kill: vi.fn(), installVersion: vi.fn(),
}}));
vi.mock('../api/resources/saves', () => ({default: {list: dependencies.saves}}));
vi.mock('../api/socket', () => ({default: {
    on: vi.fn((name, handler) => { dependencies.handlers[name] = handler; }),
    off: vi.fn((name) => { delete dependencies.handlers[name]; }),
    emit: vi.fn(),
}}));

import App from './App';

beforeAll(() => { globalThis.IS_REACT_ACT_ENVIRONMENT = true; });
beforeEach(() => {
    const browserNotification = vi.fn();
    browserNotification.permission = 'granted';
    vi.stubGlobal('Notification', browserNotification);
    dependencies.handlers = {};
    dependencies.userStatus.mockResolvedValue({
        public_user_id: 'user-1', name: 'Admin', role: 'admin', can_manage: true,
        game_username: 'stupidll', server_admin: true, preferences: {language: 'pt-BR'},
    });
    dependencies.serverStatus.mockResolvedValue({
        running: false, state: 'stopped', fac_version: '2.0.72.0', bindip: '0.0.0.0', port: 34197,
    });
    dependencies.versions.mockResolvedValue({
        stable: '2.0.77', latest: '2.1.20', versions: ['2.1.20', '2.0.77', '2.0.72'],
    });
    dependencies.saves.mockResolvedValue([{name: 'Load Latest (world.zip)'}]);
});
afterEach(() => {
    vi.clearAllMocks();
    vi.unstubAllGlobals();
});

describe('App status updates', () => {
    it('keeps the active panel mounted and offers the installed Factorio version', async () => {
        const container = document.createElement('div');
        const root = createRoot(container);
        await act(async () => {
            root.render(<App/>);
            await Promise.resolve();
        });
        await act(async () => { await Promise.resolve(); });

        const originalForm = container.querySelector('form');
        const versionSelect = container.querySelector('select[aria-label="Versão desejada do Factorio"]');
        expect(originalForm).not.toBeNull();
        expect([...versionSelect.options].map(option => option.value)).toContain('2.0.72');
        expect(versionSelect.value).toBe('2.0.72');

        await act(async () => {
            dependencies.handlers.server_status(JSON.stringify({
                running: false, state: 'error', fac_version: '2.0.72.0', last_error: 'exit status 1',
            }));
        });

        expect(container.querySelector('form')).toBe(originalForm);
        expect(dependencies.saves).toHaveBeenCalledTimes(1);
        expect(dependencies.versions).toHaveBeenCalledTimes(1);
        expect(window.Notification).toHaveBeenCalledWith('Falha no servidor Factorio', expect.objectContaining({body: 'exit status 1'}));
        act(() => root.unmount());
    });
});

