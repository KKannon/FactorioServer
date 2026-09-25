// @vitest-environment jsdom
import React, {useState} from 'react';
import {createRoot} from 'react-dom/client';
import {act} from 'react-dom/test-utils';
import {beforeAll, describe, expect, it} from 'vitest';
import {PercentageField} from './GameSettings';

beforeAll(() => { globalThis.IS_REACT_ACT_ENVIRONMENT = true; });

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
