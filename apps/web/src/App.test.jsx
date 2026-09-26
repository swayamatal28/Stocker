import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, expect, it, vi } from 'vitest';
import { api } from './api';
import { SignalPill, WatchlistPanel } from './App';
describe('SignalPill', () => { it('communicates sentiment with text, not colour alone', () => { render(<SignalPill label="Positive setup"/>); expect(screen.getByText('Positive setup')).toBeInTheDocument(); }); });

describe('WatchlistPanel', () => {
    it('pauses and removes a monitored company', async () => {
        const pause = vi.spyOn(api, 'pauseWatchlist').mockResolvedValue({ data: {} });
        const remove = vi.spyOn(api, 'removeWatchlist').mockResolvedValue(undefined);
        const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
        const item = { id: '1', nseSymbol: 'RELIANCE', companyName: 'Reliance Industries', price: 100, changePercent: 1, signal: 'Positive setup', source: 'Fixture', alertsPaused: false };
        render(<QueryClientProvider client={client}><WatchlistPanel data={[item]} loading={false} onAdd={() => {}}/></QueryClientProvider>);

        fireEvent.click(screen.getByRole('button', { name: 'Pause alerts for RELIANCE' }));
        await waitFor(() => expect(pause).toHaveBeenCalledWith('RELIANCE', true));
        fireEvent.click(screen.getByRole('button', { name: 'Remove RELIANCE from watchlist' }));
        await waitFor(() => expect(remove).toHaveBeenCalledWith('RELIANCE'));
    });
});
