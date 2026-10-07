import { describe, expect, it } from 'vitest';
import { globalKeyAction } from './globalKeys';

function ev(key: string, target: Element | null = document.body, mods: Partial<KeyboardEvent> = {}) {
	return { key, target, ctrlKey: false, metaKey: false, altKey: false, ...mods } as KeyboardEvent;
}

describe('globalKeyAction', () => {
	it('maps R to reset and ? to shortcuts on the page', () => {
		expect(globalKeyAction(ev('r'))).toBe('reset');
		expect(globalKeyAction(ev('R'))).toBe('reset');
		expect(globalKeyAction(ev('?'))).toBe('shortcuts');
		expect(globalKeyAction(ev('x'))).toBeNull();
	});

	it('ignores keys in program slots, form fields and with modifiers', () => {
		const slot = document.createElement('button');
		slot.dataset.slot = '-1:0';
		const input = document.createElement('input');
		expect(globalKeyAction(ev('r', slot))).toBeNull();
		expect(globalKeyAction(ev('r', input))).toBeNull();
		expect(globalKeyAction(ev('r', document.body, { metaKey: true }))).toBeNull();
	});
});
