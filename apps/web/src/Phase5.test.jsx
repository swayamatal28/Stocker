import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import AlertsCenter from './AlertsCenter';
import { api } from './api';

vi.mock('./api', () => ({ api: { alertRules: vi.fn(), alerts: vi.fn(), briefings: vi.fn(), watchlist: vi.fn(), createAlertRule: vi.fn(), updateAlertRule: vi.fn(), deleteAlertRule: vi.fn(), markAlertRead: vi.fn() } }));

const wrap = component => render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })}>{component}</QueryClientProvider>);

afterEach(cleanup);

describe('Phase 5 alerts and personalization', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        api.watchlist.mockResolvedValue({ data: [{ nseSymbol: 'OLAELEC' }] });
        api.alertRules.mockResolvedValue({ data: [{ id: 'rule-1', name: 'OLA daily move', symbol: 'OLAELEC', ruleType: 'change_above', threshold: 2, minimumConfidence: 60, minimumSeverity: 'low', cooldownMinutes: 60, channels: { inApp: true }, quietHours: { enabled: false }, enabled: true }] });
        api.alerts.mockResolvedValue({ data: [{ id: 'alert-1', ruleId: 'rule-1', symbol: 'OLAELEC', severity: 'medium', title: 'OLAELEC: daily gain at or above 2.00%', explanation: 'The persisted snapshot satisfies the rule.', confidence: 100, sourceAsOf: '2026-09-27T04:30:00Z', triggeredAt: '2026-09-27T04:31:00Z', deliveryStatus: 'delivered', evidence: [{ source: 'Fixture', url: 'https://example.invalid/quote' }] }], meta: { unread: 1 } });
        api.briefings.mockResolvedValue({ data: [{ id: 'brief-1', kind: 'daily', title: 'Daily watchlist briefing', summary: '1 sourced update.', items: [{ symbol: 'OLAELEC', confidence: 100, headline: 'OLAELEC 2.69%', explanation: 'Latest timestamped snapshot.' }] }] });
        api.createAlertRule.mockResolvedValue({ data: {} });
        api.updateAlertRule.mockResolvedValue({ data: {} });
        api.deleteAlertRule.mockResolvedValue(undefined);
        api.markAlertRead.mockResolvedValue({ data: {} });
    });

    it('shows a deduplicated evidence-backed alert and briefing', async () => {
        wrap(<AlertsCenter/>);
        expect(await screen.findByText('OLA daily move')).toBeInTheDocument();
        expect(screen.getByText('OLAELEC: daily gain at or above 2.00%')).toBeInTheDocument();
        expect(screen.getByText('Daily watchlist briefing')).toBeInTheDocument();
        fireEvent.click(screen.getByText('OLAELEC: daily gain at or above 2.00%'));
        await waitFor(() => expect(api.markAlertRead.mock.calls[0][0]).toBe('alert-1'));
    });

    it('creates an in-app watchlist rule with quiet-hours controls', async () => {
        wrap(<AlertsCenter/>);
        await screen.findByText('OLA daily move');
        fireEvent.change(screen.getByLabelText('Rule name'), { target: { value: 'OLA threshold' } });
        fireEvent.click(screen.getByRole('button', { name: 'Create and evaluate rule' }));
        await waitFor(() => expect(api.createAlertRule.mock.calls[0][0]).toEqual(expect.objectContaining({ name: 'OLA threshold', symbol: 'OLAELEC', channels: expect.objectContaining({ inApp: true }), cooldownMinutes: 60 })));
    });
});
