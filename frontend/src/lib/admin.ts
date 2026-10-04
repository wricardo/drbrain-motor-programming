const STORAGE_KEY = 'drbrain-motor-programming.adminKey';

/** Returns the admin key stored for this browser tab, or "" when unset. */
export function getAdminKey(): string {
	if (typeof sessionStorage === 'undefined') return '';
	return sessionStorage.getItem(STORAGE_KEY) ?? '';
}

/** Stores the admin key in sessionStorage (cleared when the tab closes). Empty clears it. */
export function setAdminKey(key: string): void {
	if (typeof sessionStorage === 'undefined') return;
	if (key) sessionStorage.setItem(STORAGE_KEY, key);
	else sessionStorage.removeItem(STORAGE_KEY);
}
