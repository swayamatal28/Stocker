import { useQuery } from '@tanstack/react-query';
import { AlertTriangle, ExternalLink, Sparkles, X } from 'lucide-react';
import { api } from './api';
import { relativeTime } from './news-utils';

const money = value => Number(value || 0).toLocaleString('en-IN', { maximumFractionDigits: 2 });
const formatFundamental = metric => {
    const value = Number(metric.value || 0);
    if (metric.unit === 'INR' && Math.abs(value) >= 10_000_000)
        return `₹${money(value / 10_000_000)} Cr`;
    if (metric.unit === 'INR')
        return `₹${money(value)}`;
    return `${money(value)} ${metric.unit || ''}`.trim();
};
const researchTone = { positive: 'text-positive border-positive/25', neutral: 'text-cyan-300 border-cyan-400/25', caution: 'text-warning border-warning/25' };

export default function StockDetail({ symbol, onClose }) {
    const detail = useQuery({ queryKey: ['stock-detail', symbol], queryFn: () => api.stock(symbol), enabled: Boolean(symbol) });
    const news = useQuery({ queryKey: ['stock-news', symbol], queryFn: () => api.stockNews(symbol, { pageSize: 10 }), enabled: Boolean(symbol) });
    const advice = useQuery({ queryKey: ['portfolio-ai-advice', symbol], queryFn: () => api.portfolioAIAdvice(symbol), enabled: Boolean(symbol), retry: false });
    const item = detail.data?.data;
    const quote = item?.quote;
    return <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm p-3 sm:p-8" onMouseDown={event => event.target === event.currentTarget && onClose()}><section role="dialog" aria-modal="true" aria-label={`${symbol} market intelligence`} className="surface max-w-5xl mx-auto h-full rounded-2xl overflow-y-auto scrollbar p-5 sm:p-7">
        <div className="flex items-start justify-between gap-4"><div><p className="text-xs uppercase tracking-[.18em] text-positive">Stock intelligence</p><h2 className="text-2xl font-semibold mt-2">{item?.security?.companyName ?? symbol}</h2><p className="text-xs text-slate-500 mt-1">NSE: {item?.security?.nseSymbol ?? symbol}{item?.security?.bseCode ? ` · BSE: ${item.security.bseCode}` : ''}{item?.security?.isin ? ` · ${item.security.isin}` : ''}</p></div><button onClick={onClose} aria-label="Close stock detail" className="text-slate-500 hover:text-white"><X size={20}/></button></div>
        {detail.isLoading && <p className="py-16 text-center text-sm text-slate-500 animate-pulse">Loading the latest stock information…</p>}
        {detail.error && <div role="alert" className="mt-5 rounded-xl bg-negative/10 p-4 text-sm text-negative">{detail.error.message}</div>}
        {item && <>
            <section className="grid sm:grid-cols-2 lg:grid-cols-4 gap-3 mt-6"><Metric label="Last price" value={quote ? `₹${money(quote.lastPrice)}` : 'Unavailable'}/><Metric label="Day range" value={quote ? `₹${money(quote.dayLow)} – ₹${money(quote.dayHigh)}` : 'Unavailable'}/><Metric label="52-week range" value={quote ? `₹${money(quote.yearLow)} – ₹${money(quote.yearHigh)}` : 'Unavailable'}/><Metric label="Volume" value={quote ? money(quote.volume) : 'Unavailable'}/></section>
            {quote && <div className="mt-3 rounded-xl border border-line bg-white/[.02] p-4 text-xs"><p><span className="text-slate-500">Price source:</span> {quote.source}{quote.isDelayed ? ' · may be delayed' : ''}</p><p className="text-slate-500 mt-1">Price at {new Date(quote.asOf).toLocaleString('en-IN', { timeZone: 'Asia/Kolkata' })} IST</p></div>}
            <div className="grid lg:grid-cols-2 gap-5 mt-5"><section className="rounded-2xl border border-line p-5"><h3 className="font-semibold">Company fundamentals</h3><p className="text-xs text-slate-500 mt-1">Key financial figures for the latest available period.</p><div className="mt-4 divide-y divide-line">{item.fundamentals?.metrics?.map(metric => <div key={metric.key} className="flex justify-between gap-4 py-3"><div><p className="text-sm">{metric.label}</p><p className="text-[10px] text-slate-500">{metric.period}{metric.basis ? ` · ${metric.basis}` : ''}</p></div><p className="font-mono text-sm">{formatFundamental(metric)}</p></div>)}{!item.fundamentals?.metrics?.length && <p className="text-sm text-slate-500 py-6">Company financial figures are not available yet.</p>}</div></section><section className="rounded-2xl border border-line p-5"><h3 className="font-semibold">Things to watch</h3><div className="mt-4 space-y-3">{item.riskFlags?.map(flag => <div key={flag.code} className="rounded-xl bg-white/[.025] p-4 flex gap-3"><AlertTriangle size={16} className={flag.severity === 'high' ? 'text-negative' : 'text-warning'}/><div><p className="text-sm">{flag.title}</p><p className="text-xs leading-5 text-slate-500 mt-1">{flag.explanation}</p></div></div>)}{!item.riskFlags?.length && <p className="text-sm text-slate-500 py-6">No notable concerns were found in the available information.</p>}</div></section></div>
            <ResearchPanel research={item.research}/>
            <AIAdvicePanel result={advice.data?.data} loading={advice.isLoading} error={advice.error}/>
            <section className="rounded-2xl border border-line p-5 mt-5"><h3 className="font-semibold">Similar companies</h3><div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-3 mt-4">{item.peers?.map(peer => <div key={peer.symbol} className="rounded-xl bg-white/[.025] p-4"><p className="text-sm font-medium">{peer.symbol}</p><p className="text-[10px] text-slate-500 truncate">{peer.companyName}</p><p className="font-mono mt-3">₹{money(peer.lastPrice)}</p><p className={`text-xs mt-1 ${peer.changePercent >= 0 ? 'text-positive' : 'text-negative'}`}>{Number(peer.changePercent).toFixed(2)}% · P/E {money(peer.peRatio)}</p></div>)}{!item.peers?.length && <p className="text-sm text-slate-500">Similar-company information is not available yet.</p>}</div></section>
            <section className="rounded-2xl border border-line p-5 mt-5"><h3 className="font-semibold">Recent stock news</h3><p className="text-xs text-slate-500 mt-1">Up to 10 recent stories about this company.</p><div className="grid lg:grid-cols-2 gap-3 mt-4">{(news.data?.data ?? []).slice(0, 10).map(article => <a key={article.id} href={article.url} target="_blank" rel="noreferrer" className="rounded-xl bg-white/[.025] p-4 hover:bg-white/[.04]"><div className="flex items-center gap-2 text-[10px] text-slate-500"><span className="text-positive">{relativeTime(article.publishedAt)}</span><span>·</span><span>{article.sourceName}</span><ExternalLink size={11}/></div><p className="text-sm mt-2">{article.title}</p></a>)}{!news.isLoading && !news.data?.data?.length && <p className="text-sm text-slate-500 lg:col-span-2 py-6">No recent news has been found for {symbol}.</p>}</div></section>
            <p className="text-[10px] text-slate-600 mt-6">Prices may be delayed. Stock outlooks and concerns are for information only, not financial advice.</p>
        </>}
    </section></div>;
}

function Metric({ label, value }) { return <div className="rounded-xl border border-line bg-white/[.025] p-4"><p className="text-[10px] uppercase tracking-wider text-slate-500">{label}</p><p className="font-mono mt-2">{value}</p></div>; }

function ResearchPanel({ research }) {
    if (!research)
        return <section className="rounded-2xl border border-line p-5 mt-5"><h3 className="font-semibold">Financial health research</h3><p className="text-sm text-slate-500 mt-3">There is not enough current financial information to calculate the checks.</p></section>;
    return <section className="rounded-2xl border border-line p-5 mt-5"><div className="flex flex-col sm:flex-row sm:items-start justify-between gap-3"><div><h3 className="font-semibold">Financial health research</h3><p className="text-xs text-slate-500 mt-1">Clear financial checks using published figures. No machine learning is used.</p></div><div className="sm:text-right"><p className="text-lg font-mono">{research.score}/100</p><p className="text-xs text-slate-400">{research.label} · {research.coverage} checks</p></div></div><div className="grid md:grid-cols-2 gap-3 mt-5">{research.checks?.map(check => <article key={check.key} className={`rounded-xl border p-4 ${researchTone[check.status] || 'border-line text-slate-300'}`}><div className="flex items-start justify-between gap-3"><h4 className="text-sm font-medium text-white">{check.title}</h4><span className="font-mono text-sm whitespace-nowrap">{check.display}</span></div><p className="text-xs leading-5 text-slate-400 mt-3">{check.explanation}</p><p className="text-[10px] leading-4 text-slate-600 mt-3">How it is calculated: {check.formula}</p></article>)}</div><div className="mt-4 border-t border-line pt-4 text-[10px] leading-4 text-slate-500"><p>{research.disclaimer}</p>{research.source && <p className="mt-1">Financial information: {research.source}{research.asOf ? ` · updated ${new Date(research.asOf).toLocaleDateString('en-IN')}` : ''}</p>}</div></section>;
}

function AIAdvicePanel({ result, loading, error }) {
    const unavailable = error ? 'AI advice is temporarily unavailable.' : result?.message;
    return <section className="rounded-2xl border border-line p-5 mt-5"><div className="flex items-center gap-2"><Sparkles size={17} className="text-cyan-300"/><h3 className="font-semibold">AI portfolio advice</h3></div>{loading ? <p className="text-sm text-slate-500 mt-4 animate-pulse">Reviewing the latest company information…</p> : !result?.available ? <div className="rounded-xl bg-white/[.025] p-4 mt-4"><p className="text-sm text-slate-400">{unavailable || 'AI advice not available.'}</p></div> : <><div className="flex flex-col sm:flex-row sm:items-start justify-between gap-3 mt-4"><p className="text-sm leading-6 text-slate-300 max-w-3xl">{result.summary}</p><Direction value={result.outlook}/></div><div className="grid md:grid-cols-3 gap-3 mt-5"><AdviceList title="Strengths" items={result.strengths}/><AdviceList title="Concerns" items={result.concerns}/><AdviceList title="What to watch" items={result.whatToWatch}/></div><p className="text-[10px] leading-4 text-slate-600 mt-4">{result.disclaimer}</p></>}</section>;
}

function AdviceList({ title, items = [] }) {
    return <div className="rounded-xl border border-line p-4"><h4 className="text-xs font-semibold">{title}</h4><ul className="space-y-2 mt-3">{items.map(item => <li key={item} className="text-xs leading-5 text-slate-400">• {item}</li>)}{!items.length && <li className="text-xs text-slate-600">Nothing listed.</li>}</ul></div>;
}

function Direction({ value }) {
    const label = value === 'bullish' ? 'Bullish' : value === 'bearish' ? 'Bearish' : value === 'neutral' ? 'Neutral' : 'Mixed';
    const style = value === 'bullish' ? 'bg-positive/10 text-positive' : value === 'bearish' ? 'bg-negative/10 text-negative' : 'bg-warning/10 text-warning';
    return <span className={`self-start rounded-full px-3 py-1.5 text-xs font-medium ${style}`}>{label}</span>;
}
