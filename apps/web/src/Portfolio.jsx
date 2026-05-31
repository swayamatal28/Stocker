import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { BellOff, BellRing, BriefcaseBusiness, ExternalLink, Pencil, Plus, Trash2 } from 'lucide-react';
import { api } from './api';
import { relativeTime } from './news-utils';

const money = value => Number(value || 0).toLocaleString('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 2 });
const tone = value => Number(value) >= 0 ? 'text-positive' : 'text-negative';

export default function PortfolioPanel({ data = [], summary = {}, loading, onAdd, onSelect }) {
    const client = useQueryClient();
    const [editing, setEditing] = useState('');
    const news = useQuery({ queryKey: ['portfolio-news'], queryFn: api.portfolioNews, enabled: data.length > 0 });
    const refresh = () => {
        client.invalidateQueries({ queryKey: ['portfolio'] });
        client.invalidateQueries({ queryKey: ['portfolio-news'] });
    };
    const update = useMutation({ mutationFn: ({ symbol, patch }) => api.updatePortfolio(symbol, patch), onSuccess: () => { setEditing(''); refresh(); } });
    const remove = useMutation({ mutationFn: symbol => api.removePortfolio(symbol), onSuccess: refresh });
    const error = update.error || remove.error;

    function save(event, symbol) {
        event.preventDefault();
        const form = new FormData(event.currentTarget);
        update.mutate({ symbol, patch: { quantity: Number(form.get('quantity')), averageBuyPrice: Number(form.get('averageBuyPrice')) } });
    }

    if (loading)
        return <div className="surface rounded-2xl p-8 text-sm text-slate-500 animate-pulse">Loading your portfolio…</div>;

    return <div className="space-y-5">
        <section className="grid sm:grid-cols-2 xl:grid-cols-4 gap-3">
            <Summary label="Amount invested" value={money(summary.investedValue)}/>
            <Summary label="Current value" value={money(summary.currentValue)}/>
            <Summary label="Overall profit / loss" value={money(summary.pnl)} detail={`${Number(summary.pnlPercent || 0).toFixed(2)}%`} valueClass={tone(summary.pnl)}/>
            <Summary label="Today’s profit / loss" value={money(summary.dayPnl)} valueClass={tone(summary.dayPnl)}/>
        </section>

        <section className="surface rounded-2xl overflow-hidden">
            <div className="p-5 flex items-center justify-between gap-4"><div><h2 className="font-semibold">Your holdings</h2><p className="text-xs text-slate-500 mt-1">{data.length} of 50 stocks · select a company to view its financial research</p></div><button onClick={onAdd} className="flex gap-2 items-center text-xs rounded-lg bg-positive text-ink font-semibold px-3 py-2"><Plus size={14}/>Add holding</button></div>
            {error && <p role="alert" className="mx-5 mb-4 rounded-lg bg-negative/10 p-3 text-xs text-negative">{error.message}</p>}
            {!data.length ? <div className="border-t border-line py-14 text-center"><BriefcaseBusiness className="mx-auto text-slate-600"/><p className="mt-3 text-sm">Your portfolio is empty</p><p className="text-xs text-slate-500 mt-1">Add a stock with its quantity and average buy price.</p></div> : <div className="table-shell"><table className="portfolio-table responsive-data-table w-full text-sm"><thead className="text-[10px] uppercase tracking-wider text-slate-500 border-y border-line"><tr><th className="text-left p-3 pl-5">Company</th><th className="text-right p-3">Quantity</th><th className="text-right p-3">Average price</th><th className="text-right p-3">Current price</th><th className="text-right p-3">Current value</th><th className="text-right p-3">Overall P&amp;L</th><th className="text-right p-3">Today</th><th className="p-3 pr-5"><span className="sr-only">Actions</span></th></tr></thead><tbody>{data.map(item => {
                const symbol = item.nseSymbol || item.bseCode;
                const busy = update.isPending || remove.isPending;
                if (editing === symbol)
                    return <tr key={item.id} className="portfolio-edit-row border-b border-line"><td colSpan="8" className="p-4"><form onSubmit={event => save(event, symbol)} className="grid sm:grid-cols-[1fr_1fr_auto_auto] items-end gap-3"><label className="text-xs text-slate-400"><span className="block mb-2">Quantity</span><input name="quantity" type="number" min="0.0001" step="0.0001" defaultValue={item.quantity || ''} required className="w-full rounded-lg border border-line bg-white/[.025] px-3 py-2 text-sm"/></label><label className="text-xs text-slate-400"><span className="block mb-2">Average buy price</span><input name="averageBuyPrice" type="number" min="0.01" step="0.01" defaultValue={item.averageBuyPrice || ''} required className="w-full rounded-lg border border-line bg-white/[.025] px-3 py-2 text-sm"/></label><button disabled={busy} className="rounded-lg bg-positive px-4 py-2 text-sm font-semibold text-ink">Save</button><button type="button" onClick={() => setEditing('')} className="rounded-lg border border-line px-4 py-2 text-sm">Cancel</button></form></td></tr>;
                return <tr key={item.id} className="border-b border-line/60 last:border-0 hover:bg-white/[.02]"><td data-label="Company" className="p-3 pl-5"><button onClick={() => onSelect?.(symbol)} className="text-left hover:text-positive"><p className="font-medium">{symbol}</p><p className="text-[11px] text-slate-500 max-w-44 truncate">{item.companyName}</p>{!item.detailsComplete && <span className="text-[10px] text-warning">Add quantity and average price</span>}</button></td><td data-label="Quantity" className="p-3 text-right font-mono">{item.detailsComplete ? Number(item.quantity).toLocaleString('en-IN') : '—'}</td><td data-label="Average price" className="p-3 text-right font-mono">{item.detailsComplete ? money(item.averageBuyPrice) : '—'}</td><td data-label="Current price" className="p-3 text-right"><p className="font-mono">{money(item.price)}</p><p className="text-[10px] text-slate-500">{item.asOf ? relativeTime(item.asOf) : 'time unavailable'}</p></td><td data-label="Current value" className="p-3 text-right font-mono">{item.detailsComplete ? money(item.currentValue) : '—'}</td><td data-label="Overall P&L" className={`p-3 text-right font-mono ${item.detailsComplete ? tone(item.pnl) : ''}`}>{item.detailsComplete ? <>{money(item.pnl)}<br/><span className="text-[10px]">{Number(item.pnlPercent).toFixed(2)}%</span></> : '—'}</td><td data-label="Today" className={`p-3 text-right font-mono ${item.detailsComplete ? tone(item.dayPnl) : ''}`}>{item.detailsComplete ? money(item.dayPnl) : '—'}</td><td data-label="Actions" className="p-3 pr-5"><div className="flex justify-end gap-2"><button disabled={busy} onClick={() => setEditing(symbol)} aria-label={`Edit ${symbol} holding`} className="rounded-lg border border-line p-2 text-slate-400 hover:text-white"><Pencil size={15}/></button><button disabled={busy} onClick={() => update.mutate({ symbol, patch: { alertsPaused: !item.alertsPaused } })} aria-label={`${item.alertsPaused ? 'Resume' : 'Pause'} alerts for ${symbol}`} className="rounded-lg border border-line p-2 text-slate-400 hover:text-white">{item.alertsPaused ? <BellRing size={15}/> : <BellOff size={15}/>}</button><button disabled={busy} onClick={() => remove.mutate(symbol)} aria-label={`Remove ${symbol} from portfolio`} className="rounded-lg border border-line p-2 text-slate-400 hover:text-negative"><Trash2 size={15}/></button></div></td></tr>;
            })}</tbody></table></div>}
        </section>

        {data.length > 0 && <section className="surface rounded-2xl p-5"><div><h2 className="font-semibold">News about your holdings</h2><p className="text-xs text-slate-500 mt-1">Recent stories linked to companies in your portfolio.</p></div><div className="grid lg:grid-cols-2 gap-3 mt-4">{(news.data?.data ?? []).slice(0, 10).map(article => <a key={article.id} href={article.url} target="_blank" rel="noreferrer" className="rounded-xl border border-line p-4 hover:border-positive/30"><div className="flex items-center gap-2 text-[10px] text-slate-500"><span className="text-positive">{relativeTime(article.publishedAt)}</span><span>·</span><span>{article.sourceName}</span><ExternalLink size={11}/></div><p className="text-sm mt-2 leading-5">{article.title}</p><div className="flex gap-2 mt-3">{article.symbols?.map(symbol => <span key={symbol} className="rounded bg-cyan-400/10 px-2 py-1 text-[10px] text-cyan-300">{symbol}</span>)}</div></a>)}{news.isLoading && <p className="text-sm text-slate-500 py-6">Loading portfolio news…</p>}{!news.isLoading && !news.data?.data?.length && <p className="text-sm text-slate-500 py-6">No recent stories have been linked to your holdings yet.</p>}</div></section>}
    </div>;
}

function Summary({ label, value, detail, valueClass = '' }) {
    return <div className="surface rounded-2xl p-5"><p className="text-[10px] uppercase tracking-wider text-slate-500">{label}</p><p className={`font-mono text-xl mt-2 ${valueClass}`}>{value}</p>{detail && <p className={`text-xs mt-1 ${valueClass}`}>{detail}</p>}</div>;
}
