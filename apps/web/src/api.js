/**
 * @typedef {Object} Security
 * @property {string} id
 * @property {string} nseSymbol
 * @property {string} bseCode
 * @property {string} isin
 * @property {string} companyName
 * @property {string} sector
 * @property {string} industry
 * @property {number} price
 * @property {number} changePercent
 * @property {string} signal
 * @property {number} newsCount
 * @property {string} asOf
 * @property {string} source
 * @property {boolean} [alertsPaused]
 */

let accessToken = '';
let refreshPromise;
let sessionExpired = () => {};

export const setAccessToken = (value) => { accessToken = value || ''; };
export const setSessionExpiredHandler = (handler) => { sessionExpired = handler; };

async function responseBody(response) {
    if (response.status === 204)
        return undefined;
    return response.json().catch(() => ({}));
}

async function refreshAccessToken() {
    if (!refreshPromise) {
        refreshPromise = fetch('/api/v1/auth/refresh', {
            method: 'POST',
            credentials: 'include',
            headers: { Accept: 'application/json' }
        }).then(async (response) => {
            const body = await responseBody(response);
            if (!response.ok)
                throw new Error(body?.error ?? 'Unable to restore session');
            setAccessToken(body.accessToken);
            return body;
        }).finally(() => { refreshPromise = undefined; });
    }
    return refreshPromise;
}

async function authorizedFetch(path, init = {}, retry = true) {
    const headers = new Headers(init.headers);
    headers.set('Accept', headers.get('Accept') || 'application/json');
    if (init.body)
        headers.set('Content-Type', 'application/json');
    if (accessToken)
        headers.set('Authorization', `Bearer ${accessToken}`);
    let response = await fetch(`/api/v1${path}`, { ...init, headers, credentials: 'include' });
    if (response.status === 401 && retry && !path.startsWith('/auth/')) {
        try {
            await refreshAccessToken();
            response = await authorizedFetch(path, init, false);
        }
        catch (error) {
            setAccessToken('');
            sessionExpired();
            throw error;
        }
    }
    return response;
}

/** @template T @param {string} path @param {RequestInit} [init] @returns {Promise<T>} */
async function request(path, init = {}) {
    const response = await authorizedFetch(path, init);
    const body = await responseBody(response);
    if (!response.ok)
        throw new Error(body?.error ?? `Request failed (${response.status})`);
    return body;
}

const query = (values = {}) => {
    const params = new URLSearchParams();
    Object.entries(values).forEach(([key, value]) => {
        if (value !== '' && value !== undefined && value !== null && value !== false)
            params.set(key, String(value));
    });
    const encoded = params.toString();
    return encoded ? `?${encoded}` : '';
};

function parseEventBlock(block) {
    let event = 'message';
    const data = [];
    for (const line of block.split('\n')) {
        if (line.startsWith('event:'))
            event = line.slice(6).trim();
        else if (line.startsWith('data:'))
            data.push(line.slice(5).trim());
    }
    if (!data.length)
        return null;
    const text = data.join('\n');
    try { return { event, data: JSON.parse(text) }; }
    catch { return { event, data: text }; }
}

async function consumeStream(signal, onEvent) {
    const response = await authorizedFetch('/stream', { signal, headers: { Accept: 'text/event-stream' } });
    if (!response.ok || !response.body)
        throw new Error(`Live updates unavailable (${response.status})`);
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    while (!signal.aborted) {
        const { value, done } = await reader.read();
        if (done)
            return;
        buffer += decoder.decode(value, { stream: true }).replaceAll('\r\n', '\n');
        let boundary;
        while ((boundary = buffer.indexOf('\n\n')) >= 0) {
            const parsed = parseEventBlock(buffer.slice(0, boundary));
            buffer = buffer.slice(boundary + 2);
            if (parsed)
                onEvent(parsed);
        }
    }
}

function subscribe(onEvent) {
    const controller = new AbortController();
    let retryMs = 1000;
    const run = async () => {
        while (!controller.signal.aborted) {
            try {
                await consumeStream(controller.signal, onEvent);
                retryMs = 1000;
            }
            catch (error) {
                if (controller.signal.aborted)
                    return;
                onEvent({ event: 'connection-error', data: { message: error.message } });
            }
            await new Promise(resolve => setTimeout(resolve, retryMs));
            retryMs = Math.min(retryMs * 2, 15_000);
        }
    };
    run();
    return () => controller.abort();
}

export const api = {
    restore: async () => { await refreshAccessToken(); return request('/auth/me'); },
    logout: () => request('/auth/logout', { method: 'POST' }),
    overview: () => request('/market/overview'),
	sectors: () => request('/market/sectors'),
	movers: () => request('/market/movers'),
	events: (filters) => request(`/events${query(filters)}`),
    search: (q) => request(`/stocks/search?q=${encodeURIComponent(q)}`),
	stock: (symbol) => request(`/stocks/${encodeURIComponent(symbol)}`),
	stockQuote: (symbol) => request(`/stocks/${encodeURIComponent(symbol)}/quote`),
	stockFundamentals: (symbol) => request(`/stocks/${encodeURIComponent(symbol)}/fundamentals`),
	stockPeers: (symbol) => request(`/stocks/${encodeURIComponent(symbol)}/peers`),
	stockRiskFlags: (symbol) => request(`/stocks/${encodeURIComponent(symbol)}/risk-flags`),
    watchlist: () => request('/watchlist'),
    addWatchlist: (symbol) => request('/watchlist', { method: 'POST', body: JSON.stringify({ symbol }) }),
    removeWatchlist: (symbol) => request(`/watchlist/${encodeURIComponent(symbol)}`, { method: 'DELETE' }),
    pauseWatchlist: (symbol, alertsPaused) => request(`/watchlist/${encodeURIComponent(symbol)}`, { method: 'PATCH', body: JSON.stringify({ alertsPaused }) }),
    alerts: (limit = 50) => request(`/alerts?limit=${encodeURIComponent(limit)}`),
    markAlertRead: (id) => request(`/alerts/${encodeURIComponent(id)}/read`, { method: 'PATCH' }),
    alertRules: () => request('/alert-rules'),
    createAlertRule: (rule) => request('/alert-rules', { method: 'POST', body: JSON.stringify(rule) }),
    updateAlertRule: (id, patch) => request(`/alert-rules/${encodeURIComponent(id)}`, { method: 'PATCH', body: JSON.stringify(patch) }),
    deleteAlertRule: (id) => request(`/alert-rules/${encodeURIComponent(id)}`, { method: 'DELETE' }),
    briefings: (kind = 'daily') => request(`/briefings?kind=${encodeURIComponent(kind)}`),
    evaluationReport: (asOf) => request(`/evaluation/report${query({ asOf })}`),
    login: (email, password) => request('/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) }),
    register: (email, password, displayName) => request('/auth/register', { method: 'POST', body: JSON.stringify({ email, password, displayName }) }),
    news: (filters) => request(`/news${query(filters)}`),
    newsDetail: (id) => request(`/news/${encodeURIComponent(id)}`),
    newsAnalysis: (id) => request(`/news/${encodeURIComponent(id)}/analysis`),
    stockNews: (symbol, filters) => request(`/stocks/${encodeURIComponent(symbol)}/news${query(filters)}`),
    stockSignals: (symbol) => request(`/stocks/${encodeURIComponent(symbol)}/signals`),
    sourceHealth: () => request('/system/source-health'),
    subscribe
};
