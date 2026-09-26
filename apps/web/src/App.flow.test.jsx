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

    it('registers, searches the security master, and adds a watchlist item', async () => {
        let watchlist = [];
        const security = { id: '507f1f77bcf86cd799439011', nseSymbol: 'RELIANCE', bseCode: '500325', isin: 'INE002A01018', companyName: 'Reliance Industries', sector: 'Energy', industry: 'Diversified', price: 1400, changePercent: 0.8, signal: 'Positive setup', newsCount: 1, asOf: new Date().toISOString(), source: 'Fixture', alertsPaused: false };
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
            if (url.startsWith('/api/v1/stocks/search'))
                return json({ data: [security] });
            if (url === '/api/v1/watchlist' && init.method === 'POST') {
                watchlist = [security];
                return json({ data: security }, 201);
            }
            if (url === '/api/v1/watchlist')
                return json({ data: watchlist, meta: { count: watchlist.length, limit: 10 } });
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
        fireEvent.click(screen.getByText(/Search stocks, news or ISIN/));
        const search = await screen.findByPlaceholderText('Company, NSE symbol, BSE code or ISIN');
        fireEvent.change(search, { target: { value: 'RELIANCE' } });
        const result = await screen.findByText('Reliance Industries');
        fireEvent.click(result.closest('button'));

        await waitFor(() => expect(screen.getByText('1 of 10 companies monitored')).toBeInTheDocument());
        expect(fetch).toHaveBeenCalledWith('/api/v1/watchlist', expect.objectContaining({ method: 'POST' }));
    });
});
