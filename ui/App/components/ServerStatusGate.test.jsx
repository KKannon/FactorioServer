// @vitest-environment jsdom
import React from 'react';
import {createRoot} from 'react-dom/client';
import {act} from 'react-dom/test-utils';
import {MemoryRouter, Route, Routes} from 'react-router-dom';
import {beforeAll, describe, expect, it, vi} from 'vitest';
import ServerStatusGate from './ServerStatusGate';

beforeAll(() => { globalThis.IS_REACT_ACT_ENVIRONMENT = true; });

const renderGate = props => {
    const container = document.createElement('div');
    act(() => createRoot(container).render(
        <MemoryRouter>
            <Routes>
                <Route element={<ServerStatusGate {...props}/> }>
                    <Route index element={<div>Protected server controls</div>}/>
                </Route>
            </Routes>
        </MemoryRouter>
    ));
    return container;
};

describe('ServerStatusGate', () => {
    it('keeps status-dependent views unmounted while status is loading', () => {
        const container = renderGate({status: null, loading: false, error: null});
        expect(container.textContent).toContain('Loading server status...');
        expect(container.textContent).not.toContain('Protected server controls');
    });

    it('shows a recoverable error when the status request fails', () => {
        const onRetry = vi.fn();
        const container = renderGate({status: null, loading: false, error: 'Could not load status.', onRetry});
        expect(container.textContent).toContain('Could not load status.');
        act(() => container.querySelector('button').click());
        expect(onRetry).toHaveBeenCalledOnce();
    });

    it('renders the protected view after status is available', () => {
        const container = renderGate({status: {running: false}, loading: false, error: null});
        expect(container.textContent).toContain('Protected server controls');
    });
});
