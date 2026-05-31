import { fireEvent, render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import NewsExplorer from './NewsExplorer';
import { api } from './api';

vi.mock('./api', () => ({
    api: {
        news: vi.fn(),
        newsDetail: vi.fn(),
        newsImpact: vi.fn()
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
const impact = {
    ready: true,
    marketImpact: [{ symbol: 'ONGC', companyName: 'Oil and Natural Gas Corporation', direction: 'bearish', explanation: 'Higher costs may pressure margins.', horizon: 'short term' }],
    portfolioImpact: [{ symbol: 'RELIANCE', companyName: 'Reliance Industries', direction: 'bullish', explanation: 'The development may support this holding.', horizon: 'medium term' }]
};

function renderExplorer() {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    return render(<QueryClientProvider client={client}><NewsExplorer /></QueryClientProvider>);
}

describe('NewsExplorer', () => {
    beforeEach(() => {
        api.news.mockResolvedValue({ data: [article], meta: { page: 1, pageSize: 12, total: 1, totalPages: 1, syntheticOnly: true } });
        api.newsDetail.mockResolvedValue({ data: article, meta: {} });
        api.newsImpact.mockResolvedValue({ data: impact });
    });

    it('opens a story with separate market and portfolio impact lists', async () => {
        renderExplorer();
        expect(await screen.findByText('Synthetic fixture announcement')).toBeInTheDocument();
        fireEvent.click(screen.getByText('Synthetic fixture announcement'));
        const sourceLink = await screen.findByRole('link', { name: /Read the original article/ });
        expect(api.newsDetail).toHaveBeenCalledWith(article.id);
        expect(await screen.findByText('Other affected stocks')).toBeInTheDocument();
        expect(screen.getByText('Your portfolio trend')).toBeInTheDocument();
        expect(screen.getByText('ONGC')).toBeInTheDocument();
        expect(screen.getAllByText('RELIANCE')).toHaveLength(2);
        expect(screen.queryByText(/confidence/i)).not.toBeInTheDocument();
        expect(sourceLink).toHaveAttribute('href', article.url);
    });
});
