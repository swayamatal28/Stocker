import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, expect, it, vi } from 'vitest';
import { api } from './api';
import { SignalPill } from './App';
import PortfolioPanel from './Portfolio';
describe('SignalPill', () => { it('communicates sentiment with text, not colour alone', () => { render(<SignalPill label="Positive setup"/>); expect(screen.getByText('Positive setup')).toBeInTheDocument(); }); });

describe('PortfolioPanel', () => {
    it('updates alerts and removes a holding', async () => {
        vi.spyOn(api, 'portfolioNews').mockResolvedValue({ data: [] });
        const update = vi.spyOn(api, 'updatePortfolio').mockResolvedValue({ data: {} });
        const remove = vi.spyOn(api, 'removePortfolio').mockResolvedValue(undefined);
        const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
        const item = { id: '1', nseSymbol: 'RELIANCE', companyName: 'Reliance Industries', price: 100, quantity: 10, averageBuyPrice: 90, currentValue: 1000, pnl: 100, pnlPercent: 11.11, dayPnl: 10, detailsComplete: true, alertsPaused: false };
        render(<QueryClientProvider client={client}><PortfolioPanel data={[item]} summary={{ investedValue: 900, currentValue: 1000, pnl: 100, pnlPercent: 11.11, dayPnl: 10 }} loading={false} onAdd={() => {}}/></QueryClientProvider>);

        fireEvent.click(screen.getByRole('button', { name: 'Pause alerts for RELIANCE' }));
        await waitFor(() => expect(update).toHaveBeenCalledWith('RELIANCE', { alertsPaused: true }));
        fireEvent.click(screen.getByRole('button', { name: 'Remove RELIANCE from portfolio' }));
        await waitFor(() => expect(remove).toHaveBeenCalledWith('RELIANCE'));
    });
});
