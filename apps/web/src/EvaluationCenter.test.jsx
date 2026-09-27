import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, expect, it, vi } from 'vitest';
import { api } from './api';
import EvaluationCenter from './EvaluationCenter';

describe('EvaluationCenter', () => {
    it('shows event-time outcomes and leakage findings', async () => {
        vi.spyOn(api, 'evaluationReport').mockResolvedValue({ data: { version: 'event-time-v1', asOf: '2026-09-20T12:00:00Z', candidateSignals: 2, evaluatedSignals: 1, pendingSignals: 0, outcomes: [], leakageViolations: [{ signalId: 'leaked', field: 'signal_data', availableAt: '2026-09-01T10:01:00Z' }], slices: [{ dimension: 'horizon', value: 'short term', evaluated: 1, accuracy: 100, precision: 100 }] }, meta: { lookAheadSafe: false } });
        const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
        render(<QueryClientProvider client={client}><EvaluationCenter/></QueryClientProvider>);
        expect(await screen.findByText('Leakage findings require review')).toBeInTheDocument();
        expect(screen.getByText('short term')).toBeInTheDocument();
        expect(screen.getAllByText('100.00%')).toHaveLength(2);
    });
});
