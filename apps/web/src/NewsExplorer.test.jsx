import { fireEvent, render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import NewsExplorer from './NewsExplorer';
import { api } from './api';

vi.mock('./api', () => ({
    api: {
        news: vi.fn(),
        newsDetail: vi.fn()
    }
}));

const article = {
    id: '68d3a7000000000000000001',
    clusterId: 'cluster-1',
    title: 'Synthetic fixture announcement',
    body: 'Project-owned retained evidence text.',
    sourceId: 'mock-exchange',
    sourceName: 'STOCKER fixture',
    sourceType: 'mock',
    url: 'https://example.invalid/item',
    language: 'en',
    attribution: 'Synthetic STOCKER fixture',
    licence: 'Project-owned',
    parserVersion: 'mock-v2',
    publishedAt: new Date().toISOString(),
    retrievedAt: new Date().toISOString(),
    symbols: ['RELIANCE'],
    sectors: ['Energy'],
    official: false,
    synthetic: true,
    duplicateCount: 1
};

function renderExplorer() {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    return render(<QueryClientProvider client={client}><NewsExplorer /></QueryClientProvider>);
}

describe('NewsExplorer', () => {
    beforeEach(() => {
        api.news.mockResolvedValue({ data: [article], meta: { page: 1, pageSize: 12, total: 1, totalPages: 1, syntheticOnly: true } });
        api.newsDetail.mockResolvedValue({ data: article, meta: {} });
    });

    it('labels synthetic evidence and opens attributed detail', async () => {
        renderExplorer();
        expect(await screen.findByText('Synthetic fixture announcement')).toBeInTheDocument();
        expect(screen.getAllByText('SYNTHETIC').length).toBeGreaterThan(0);
        fireEvent.click(screen.getByText('Synthetic fixture announcement'));
        const sourceLink = await screen.findByRole('link', { name: /Open original source/ });
        expect(api.newsDetail).toHaveBeenCalledWith(article.id);
        expect(screen.getByText((_, element) => element?.textContent === `Attribution: ${article.attribution}`)).toBeInTheDocument();
        expect(sourceLink).toHaveAttribute('href', article.url);
    });
});
