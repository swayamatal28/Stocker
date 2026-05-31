import { fireEvent, render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import MarketIntelligence from './MarketIntelligence';
import StockDetail from './StockDetail';
import { api } from './api';

vi.mock('./api', () => ({ api: { overview: vi.fn(), sectors: vi.fn(), movers: vi.fn(), events: vi.fn(), stock: vi.fn(), stockNews: vi.fn(), portfolioAIAdvice: vi.fn() } }));

const wrap = component => render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{component}</QueryClientProvider>);

describe('Phase 4 market intelligence', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        api.overview.mockResolvedValue({ data: { breadth: { advances: 4, declines: 2, unchanged: 1 } } });
        api.sectors.mockResolvedValue({ data: [{ sector: 'Consumer Discretionary', companyCount: 2, advances: 1, declines: 1, averageChangePercent: 1.2, averageSentiment: 4 }] });
        api.movers.mockResolvedValue({ data: [{ symbol: 'OLAELEC', companyName: 'Ola Electric Mobility Limited', source: 'Fixture', lastPrice: 42, changePercent: 2.69 }] });
        api.events.mockResolvedValue({ data: [] });
    });

    it('shows market breadth and opens a mover', async () => {
        const onSelect = vi.fn();
        wrap(<MarketIntelligence onSelectStock={onSelect}/>);
        expect(await screen.findByText('Consumer Discretionary')).toBeInTheDocument();
        fireEvent.click(await screen.findByText('OLAELEC'));
        expect(onSelect).toHaveBeenCalledWith('OLAELEC');
    });

    it('shows quote provenance, fundamental basis, risks and bounded stock news', async () => {
        api.stock.mockResolvedValue({ data: {
            security: { nseSymbol: 'OLAELEC', bseCode: '544225', isin: 'INE0LXG01040', companyName: 'Ola Electric Mobility Limited' },
            quote: { lastPrice: 42, dayLow: 40.8, dayHigh: 42.7, yearLow: 30.75, yearHigh: 102.5, volume: 1000, source: 'Fixture', isDelayed: true, synthetic: true, asOf: '2026-09-27T04:30:00Z', retrievedAt: '2026-09-27T04:45:00Z' },
            fundamentals: { metrics: [{ key: 'pe_ratio', label: 'P/E ratio', value: 0, unit: 'x', period: 'TTM', basis: 'trailing' }] },
            peers: [], riskFlags: [{ code: 'stale', severity: 'medium', title: 'Price may be delayed', explanation: 'Check the displayed time before relying on it.' }]
        } });
        api.stockNews.mockResolvedValue({ data: Array.from({ length: 10 }, (_, index) => ({ id: String(index), url: `https://example.invalid/${index}`, publishedAt: '2026-09-27T04:30:00Z', sourceName: 'Publisher', title: `OLA update ${index}` })) });
        api.portfolioAIAdvice.mockResolvedValue({ data: { available: false, message: 'AI advice not available. Add an approved AI API key and endpoint to enable it.' } });
        wrap(<StockDetail symbol="OLAELEC" onClose={() => {}}/>);
        expect(await screen.findByText('Ola Electric Mobility Limited')).toBeInTheDocument();
        expect(screen.getByText('Price source:').parentElement).toHaveTextContent('Fixture · may be delayed');
        expect(screen.getByText('TTM · trailing')).toBeInTheDocument();
        expect(screen.getByText('Price may be delayed')).toBeInTheDocument();
        expect(await screen.findByText(/AI advice not available/)).toBeInTheDocument();
        expect(await screen.findAllByText(/OLA update/)).toHaveLength(10);
    });
});
