// @vitest-environment jsdom
import React, {useState} from 'react';
import {createRoot} from 'react-dom/client';
import {act} from 'react-dom/test-utils';
import {afterEach, beforeAll, beforeEach, describe, expect, it, vi} from 'vitest';

const mapApi = vi.hoisted(() => ({
    defaults: vi.fn(), create: vi.fn(), preview: vi.fn(),
    listPresets: vi.fn(), savePreset: vi.fn(), deletePreset: vi.fn(),
}));

vi.mock('../../api/resources/mapGenerator', () => ({default: {
    defaults: mapApi.defaults,
    create: mapApi.create,
    preview: mapApi.preview,
    presets: {list: mapApi.listPresets, save: mapApi.savePreset, delete: mapApi.deletePreset},
}}));

import GameSettings, {PercentageField} from './GameSettings';

beforeAll(() => { globalThis.IS_REACT_ACT_ENVIRONMENT = true; });
beforeEach(() => {
    vi.clearAllMocks();
    mapApi.defaults.mockResolvedValue({
        native_presets: ['default'],
        map_gen_settings: {width: 0, height: 0, starting_area: 1, peaceful_mode: false, autoplace_controls: {}},
        map_settings: {difficulty_settings: {technology_price_multiplier: 1}, pollution: {enabled: true}, enemy_evolution: {enabled: true}, enemy_expansion: {enabled: true}},
    });
    mapApi.listPresets.mockResolvedValue([]);
});
afterEach(() => vi.useRealTimers());

const setNativeValue = (element, value) => {
    const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set;
    setter.call(element, value);
    element.dispatchEvent(new Event('input', {bubbles: true}));
};

describe('PercentageField', () => {
    it('keeps the editable percentage and slider synchronized', () => {
        const container = document.createElement('div');
        const Harness = () => {
            const [value, setValue] = useState(1);
            return <PercentageField label="Frequency" value={value} onChange={setValue}/>;
        };
        act(() => createRoot(container).render(<Harness/>));
        const slider = container.querySelector('input[type="range"]');
        const number = container.querySelector('input[type="number"]');

        expect(slider.value).toBe('100');
        expect(number.value).toBe('100');
        act(() => setNativeValue(number, '250'));
        expect(slider.value).toBe('250');
        expect(number.value).toBe('250');
        act(() => setNativeValue(slider, '375'));
        expect(number.value).toBe('375');
    });
});

describe('live map preview', () => {
    it('debounces changes and cancels the obsolete preview request', async () => {
        vi.useFakeTimers();
        mapApi.preview.mockImplementation(() => new Promise(() => {}));
        const container = document.createElement('div');
        const root = createRoot(container);
        await act(async () => {
            root.render(<GameSettings serverStatus={{running: false}}/>);
            await Promise.resolve();
        });
        await act(async () => { await Promise.resolve(); });

        expect(container.querySelector('[role="status"]')).not.toBeNull();
        await act(async () => { vi.advanceTimersByTime(650); });
        expect(mapApi.preview).toHaveBeenCalledTimes(1);
        const firstSignal = mapApi.preview.mock.calls[0][1];
        expect(firstSignal.aborted).toBe(false);

        const width = container.querySelectorAll('input[type="number"]')[1];
        act(() => setNativeValue(width, '128'));
        expect(firstSignal.aborted).toBe(true);
        await act(async () => { vi.advanceTimersByTime(649); });
        expect(mapApi.preview).toHaveBeenCalledTimes(1);
        await act(async () => { vi.advanceTimersByTime(1); });
        expect(mapApi.preview).toHaveBeenCalledTimes(2);
        expect(mapApi.preview.mock.calls[1][0].map_gen_settings.width).toBe(128);

        act(() => root.unmount());
    });
});
