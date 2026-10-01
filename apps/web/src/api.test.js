import { beforeEach, describe, expect, it, vi } from 'vitest';
import { api, setAccessToken } from './api';

describe('API session recovery', () => {
    beforeEach(() => {
        setAccessToken('');
        localStorage.clear();
        vi.restoreAllMocks();
    });

    it('uses one refresh request for concurrent 401 responses', async () => {
        let refreshes = 0;
        vi.stubGlobal('fetch', vi.fn(async (url, init = {}) => {
            if (url === '/api/v1/auth/refresh') {
                refreshes++;
                return new Response(JSON.stringify({ accessToken: 'fresh-token' }), { status: 200, headers: { 'Content-Type': 'application/json' } });
            }
            const authorization = new Headers(init.headers).get('Authorization');
            if (authorization !== 'Bearer fresh-token')
                return new Response(JSON.stringify({ error: 'expired' }), { status: 401, headers: { 'Content-Type': 'application/json' } });
            return new Response(JSON.stringify({ data: [] }), { status: 200, headers: { 'Content-Type': 'application/json' } });
        }));

        await Promise.all([api.portfolio(), api.portfolio()]);

        expect(refreshes).toBe(1);
    });

    it('keeps access tokens out of persistent browser storage', () => {
        setAccessToken('sensitive-token');
        expect(localStorage.getItem('stocker_access')).toBeNull();
    });

    it('reconnects a failed live stream with backoff', async () => {
        vi.useFakeTimers();
        const fetchMock = vi.fn().mockResolvedValue(new Response('', { status: 503 }));
        vi.stubGlobal('fetch', fetchMock);
        const onEvent = vi.fn();
        const stop = api.subscribe(onEvent);
        await Promise.resolve();
        await Promise.resolve();
        expect(fetchMock).toHaveBeenCalledTimes(1);
        await vi.advanceTimersByTimeAsync(1000);
        expect(fetchMock.mock.calls.length).toBeGreaterThanOrEqual(2);
        expect(onEvent).toHaveBeenCalledWith(expect.objectContaining({ event: 'connection-error' }));
        stop();
        await vi.runOnlyPendingTimersAsync();
        vi.useRealTimers();
    });
});
