const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

/**
 * Short relative time for activity columns: "just now", "5m ago", "3h ago",
 * "2d ago", then a date ("Oct 4", plus the year when it differs from now).
 * Invalid input is returned unchanged.
 */
export function relativeTime(iso: string, now: Date = new Date()): string {
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return iso;
	const sec = Math.max(0, Math.floor((now.getTime() - d.getTime()) / 1000));
	if (sec < 60) return 'just now';
	const min = Math.floor(sec / 60);
	if (min < 60) return `${min}m ago`;
	const hours = Math.floor(min / 60);
	if (hours < 24) return `${hours}h ago`;
	const days = Math.floor(hours / 24);
	if (days < 7) return `${days}d ago`;
	const date = `${MONTHS[d.getMonth()]} ${d.getDate()}`;
	return d.getFullYear() === now.getFullYear() ? date : `${date}, ${d.getFullYear()}`;
}

/** Absolute local time for title tooltips. */
export function absoluteTime(iso: string): string {
	const d = new Date(iso);
	return Number.isNaN(d.getTime()) ? iso : d.toLocaleString();
}
