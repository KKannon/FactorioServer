// @vitest-environment jsdom
import React from 'react';
import {createRoot} from 'react-dom/client';
import {act} from 'react-dom/test-utils';
import {beforeAll, describe, expect, it} from 'vitest';
import Avatar from './Avatar';

beforeAll(() => { globalThis.IS_REACT_ACT_ENVIRONMENT = true; });

describe('Avatar', () => {
    it('uses initials when no versioned picture is available', () => {
        const container = document.createElement('div');
        act(() => createRoot(container).render(<Avatar user={{name: 'Test User'}}/>));
        expect(container.textContent).toBe('TU');
        expect(container.querySelector('img')).toBeNull();
    });

    it('renders the current picture URL', () => {
        const container = document.createElement('div');
        act(() => createRoot(container).render(<Avatar user={{name: 'Test User', picture: 'https://cdn.example/avatar.png?v=2'}}/>));
        expect(container.querySelector('img').src).toContain('avatar.png?v=2');
    });

    it('updates immediately when the versioned picture changes', () => {
        const container = document.createElement('div');
        const root = createRoot(container);
        act(() => root.render(<Avatar user={{name: 'Test User', picture: 'https://cdn.example/avatar.png?v=2'}}/>));
        act(() => root.render(<Avatar user={{name: 'Test User', picture: 'https://cdn.example/avatar.png?v=3'}}/>));
        expect(container.querySelector('img').src).toContain('avatar.png?v=3');
    });
});
