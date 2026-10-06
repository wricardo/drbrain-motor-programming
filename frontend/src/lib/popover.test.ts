import { describe, expect, it } from 'vitest';
import { placePopover } from './popover';

const viewport = { width: 1000, height: 800 };
const size = { width: 200, height: 300 };

describe('placePopover', () => {
	it('opens below the anchor, left-aligned', () => {
		expect(placePopover({ left: 100, top: 100, width: 48, height: 48 }, size, viewport)).toEqual({ left: 100, top: 154 });
	});

	it('flips above when there is no room below', () => {
		expect(placePopover({ left: 100, top: 600, width: 48, height: 48 }, size, viewport)).toEqual({ left: 100, top: 294 });
	});

	it('stays below when it fits neither way', () => {
		const tall = { width: 200, height: 700 };
		expect(placePopover({ left: 100, top: 300, width: 48, height: 48 }, tall, viewport).top).toBe(354);
	});

	it('clamps inside the viewport horizontally', () => {
		expect(placePopover({ left: 950, top: 100, width: 48, height: 48 }, size, viewport).left).toBe(792);
		expect(placePopover({ left: -20, top: 100, width: 48, height: 48 }, size, viewport).left).toBe(8);
	});
});
