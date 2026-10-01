import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import App from './App';
import { api, setAccessToken } from './api';
import { useAppStore } from './store';

const json = (body, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });

describe('Phase 1 onboarding flow', () => {
    beforeEach(() => {
        setAccessToken('');
        localStorage.clear();
        useAppStore.setState({ user: null, sessionReady: false, theme: 'dark' });
        vi.restoreAllMocks();
    });

    it('registers, searches the security master, and adds a portfolio holding', async () => {
        let portfolio = [];
        const security = { id: '507f1f77bcf86cd799439011', nseSymbol: 'RELIANCE', bseCode: '500325', isin: 'INE002A01018', companyName: 'Reliance Industries', sector: 'Energy', industry: 'Diversified', price: 1400, changePercent: 0.8, signal: 'Positive setup', newsCount: 1, asOf: new Date().toISOString(), source: 'Live provider', alertsPaused: false };
        vi.spyOn(api, 'subscribe').mockReturnValue(() => {});
        vi.stubGlobal('fetch', vi.fn(async (url, init = {}) => {
            if (url === '/api/v1/auth/refresh')
                return json({ error: 'refresh session missing' }, 401);
            if (url === '/api/v1/auth/register')
                return json({ accessToken: 'access', user: { id: '507f1f77bcf86cd799439012', email: 'phase1@example.com', displayName: 'Phase One', role: 'user' } }, 201);
            if (url === '/api/v1/market/overview')
                return json({ data: { indices: [], mood: { label: 'Balanced', score: 50 } } });
            if (url === '/api/v1/news')
                return json({ data: [] });
            if (url === '/api/v1/system/source-health')
                return json({ data: [] });
            if (url === '/api/v1/stocks/RELIANCE/signals')
                return json({ data: [] });
            if (url === '/api/v1/alerts?limit=50')
                return json({ data: [], meta: { unread: 0 } });
            if (url.startsWith('/api/v1/stocks/search'))
                return json({ data: [security] });
            if (url === '/api/v1/portfolio' && init.method === 'POST') {
                const holding = { ...security, quantity: 5, averageBuyPrice: 1200, currentValue: 7000, investedValue: 6000, pnl: 1000, pnlPercent: 16.67, dayPnl: 56, detailsComplete: true };
                portfolio = [holding];
                return json({ data: holding }, 201);
            }
            if (url === '/api/v1/portfolio')
                return json({ data: portfolio, meta: { count: portfolio.length, limit: 50, summary: { investedValue: 6000, currentValue: 7000, pnl: 1000, pnlPercent: 16.67, dayPnl: 56 } } });
            if (url === '/api/v1/portfolio/news')
                return json({ data: [] });
            if (url === '/api/v1/refresh' && init.method === 'POST')
                return json({ data: { attempted: 1, refreshed: 1, failed: 0, message: 'Everything is up to date.' } });
            throw new Error(`Unhandled request: ${init.method || 'GET'} ${url}`);
        }));
        const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
        render(<QueryClientProvider client={client}><App/></QueryClientProvider>);

        await screen.findByText('Create your workspace');
        fireEvent.change(screen.getByLabelText('Your name'), { target: { value: 'Phase One' } });
        fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'phase1@example.com' } });
        fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'strong-password-123' } });
        fireEvent.click(screen.getByRole('checkbox'));
        fireEvent.click(screen.getByRole('button', { name: 'Create account' }));

        await screen.findByText(/Good evening, Phase/);
        expect(document.querySelector('.sidebar')).toHaveClass('fixed');
        expect(document.querySelector('.dashboard-content')).toBeInTheDocument();
        const themeToggle = screen.getByRole('button', { name: 'Switch to Financial Newspaper light mode' });
        fireEvent.click(themeToggle);
        await waitFor(() => expect(document.documentElement).toHaveClass('light'));
        expect(document.documentElement).not.toHaveClass('dark');
        expect(document.documentElement.dataset.theme).toBe('light');
        expect(localStorage.getItem('stocker_theme')).toBe('light');
        expect(screen.getByRole('button', { name: 'Switch to Trading Terminal dark mode' })).toBeInTheDocument();
        fireEvent.click(screen.getByText(/Search stocks, news or ISIN/));
        const search = await screen.findByPlaceholderText('Company, NSE symbol, BSE code or ISIN');
        fireEvent.change(search, { target: { value: 'RELIANCE' } });
        const result = await screen.findByText('Reliance Industries');
        fireEvent.click(result.closest('button'));
        fireEvent.change(screen.getByLabelText('Quantity'), { target: { value: '5' } });
        fireEvent.change(screen.getByLabelText('Average buy price (₹)'), { target: { value: '1200' } });
        fireEvent.click(screen.getByRole('button', { name: 'Add to portfolio' }));

        await waitFor(() => expect(screen.getByText(/1 of 50 stocks/)).toBeInTheDocument());
        expect(fetch).toHaveBeenCalledWith('/api/v1/portfolio', expect.objectContaining({ method: 'POST' }));
        fireEvent.click(screen.getByRole('button', { name: 'Refresh everything' }));
        await screen.findByText('Everything is up to date.');
        expect(fetch).toHaveBeenCalledWith('/api/v1/refresh', expect.objectContaining({ method: 'POST' }));
    });
});
