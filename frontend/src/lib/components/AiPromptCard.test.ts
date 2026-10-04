import { cleanup, fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import AiPromptCard from './AiPromptCard.svelte';

afterEach(() => {
	cleanup();
	vi.restoreAllMocks();
});

describe('AiPromptCard', () => {
	it('copies the exact prompt and confirms', async () => {
		const writeText = vi.fn().mockResolvedValue(undefined);
		Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true });
		render(AiPromptCard, { prompt: 'PROMPT TEXT', blurb: 'b' });
		await fireEvent.click(screen.getByTestId('ai-prompt-copy'));
		expect(writeText).toHaveBeenCalledWith('PROMPT TEXT');
		expect(await screen.findByText('Copied!')).toBeTruthy();
	});

	it('falls back to showing the selected prompt when the clipboard is unavailable', async () => {
		Object.defineProperty(navigator, 'clipboard', {
			value: { writeText: vi.fn().mockRejectedValue(new Error('denied')) },
			configurable: true
		});
		render(AiPromptCard, { prompt: 'PROMPT TEXT', blurb: 'b' });
		await fireEvent.click(screen.getByTestId('ai-prompt-copy'));
		expect(await screen.findByRole('alert')).toBeTruthy();
		const area = screen.getByLabelText('Prompt for an AI') as HTMLTextAreaElement;
		expect(area.value).toBe('PROMPT TEXT');
		expect(screen.queryByText('Copied!')).toBeNull();
	});
});
