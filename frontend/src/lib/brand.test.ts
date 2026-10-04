import { describe, expect, it } from 'vitest';
import { GAME, SITE, TITLE, pageTitle } from './brand';

describe('brand', () => {
	it('composes the full name and page titles', () => {
		expect(TITLE).toBe(`${SITE} - ${GAME}`);
		expect(pageTitle()).toBe('Dr Brain - Motor Programming');
		expect(pageTitle('Maps')).toBe('Maps — Dr Brain - Motor Programming');
	});
});
