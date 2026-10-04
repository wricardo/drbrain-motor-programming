import { describe, expect, it } from 'vitest';
import { nextAngle } from './grid';

describe('nextAngle', () => {
	it('takes the shortest turn across the 0/360 seam', () => {
		expect(nextAngle(270, 'UP')).toBe(360);
		expect(nextAngle(0, 'LEFT')).toBe(-90);
		expect(nextAngle(360, 'LEFT')).toBe(270);
	});

	it('keeps the angle when facing does not change', () => {
		expect(nextAngle(450, 'RIGHT')).toBe(450);
	});

	it('turns a half-turn consistently', () => {
		expect(Math.abs(nextAngle(0, 'DOWN'))).toBe(180);
	});
});
