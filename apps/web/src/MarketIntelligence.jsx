import { useQuery } from '@tanstack/react-query';
import { ArrowDownRight, ArrowUpRight, CalendarDays, Layers3 } from 'lucide-react';
import { api } from './api';
import { relativeTime } from './news-utils';

function Change({ value = 0 }) {
    const positive = value >= 0;
    return <span className={`inline-flex items-center text-xs font-mono ${positive ? 'text-positive' : 'text-negative'}`}>{positive ? <ArrowUpRight size={13}/> : <ArrowDownRight size={13}/>} {Math.abs(value).toFixed(2)}%</span>;
}

export default function MarketIntelligence({ onSelectStock }) {
    const overview = useQuery({ queryKey: ['overview'], queryFn: api.overview });
    const sectors = useQuery({ queryKey: ['market-sectors'], queryFn: api.sectors });
    const movers = useQuery({ queryKey: ['market-movers'], queryFn: api.movers });
    const events = useQuery({ queryKey: ['market-events'], queryFn: () => api.events({}) });
    const error = overview.error || sectors.error || movers.error || events.error;
    const breadth = overview.data?.data?.breadth ?? {};
    return <div>
        <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4"><div><p className="text-xs uppercase tracking-[.18em] text-positive font-semibold">Indian markets</p><h1 className="text-3xl font-semibold mt-2">Market</h1><p className="text-sm text-slate-400 mt-2">See today’s market direction, sector performance, largest moves and important events.</p></div><div className="rounded-lg border border-warning/20 bg-warning/[.06] px-3 py-2 text-xs text-warning">Prices may be delayed</div></div>
        {error && <div role="alert" className="mt-5 rounded-xl border border-negative/30 bg-negative/10 p-4 text-sm text-negative">{error.message}</div>}
        <section className="grid sm:grid-cols-3 gap-3 mt-6">{[['Advances', breadth.advances ?? 0, true], ['Declines', breadth.declines ?? 0], ['Unchanged', breadth.unchanged ?? 0]].map(([label, value, good]) => <div key={label} className="surface rounded-2xl p-5"><p className="text-[10px] uppercase tracking-wider text-slate-500">{label}</p><p className={`font-mono text-2xl mt-2 ${good ? 'text-positive' : ''}`}>{value}</p></div>)}</section>
        <div className="grid xl:grid-cols-[1.15fr_.85fr] gap-5 mt-5">
            <section className="surface rounded-2xl p-5"><div className="flex items-center gap-2"><Layers3 size={16} className="text-positive"/><h2 className="font-semibold">Sector performance</h2></div><div className="mt-4 divide-y divide-line">{(sectors.data?.data ?? []).map(sector => <div key={sector.sector} className="grid grid-cols-[1fr_auto_auto] gap-4 py-3 items-center"><div><p className="text-sm">{sector.sector}</p><p className="text-[10px] text-slate-500">{sector.companyCount} companies · {sector.advances} up · {sector.declines} down</p></div><Change value={sector.averageChangePercent}/><span className="text-[10px] text-slate-500">mood {Number(sector.averageSentiment).toFixed(1)}</span></div>)}{!sectors.data?.data?.length && <p className="py-8 text-sm text-slate-500">Sector information is not available yet.</p>}</div></section>
            <section className="surface rounded-2xl p-5"><h2 className="font-semibold">Largest moves</h2><p className="text-xs text-slate-500 mt-1">Stocks with the biggest price changes today</p><div className="mt-4 divide-y divide-line">{(movers.data?.data ?? []).map(item => <button key={item.symbol} onClick={() => onSelectStock?.(item.symbol)} className="w-full flex items-center gap-3 py-3 text-left hover:text-white"><div className="min-w-0 flex-1"><p className="text-sm font-medium">{item.symbol}</p><p className="text-[10px] text-slate-500 truncate">{item.companyName} · {item.source}</p></div><div className="text-right"><Change value={item.changePercent}/><p className="text-[10px] text-slate-500 mt-1">₹{Number(item.lastPrice).toLocaleString('en-IN')}</p></div></button>)}{!movers.data?.data?.length && <p className="py-8 text-sm text-slate-500">Price changes are not available yet.</p>}</div></section>
        </div>
        <section className="surface rounded-2xl p-5 mt-5"><div className="flex items-center gap-2"><CalendarDays size={16} className="text-positive"/><div><h2 className="font-semibold">Important market events</h2><p className="text-xs text-slate-500 mt-1">Recent announcements and developments that may affect the market.</p></div></div><div className="grid lg:grid-cols-2 gap-3 mt-5">{(events.data?.data ?? []).map(event => <a key={event.id} href={event.sourceUrl} target="_blank" rel="noreferrer" className="rounded-xl border border-line p-4 hover:border-positive/30"><div className="flex gap-2 text-[10px] text-slate-500"><span className="text-positive">{relativeTime(event.eventAt)}</span><span>·</span><span>{event.source}</span>{event.official && <span className="text-positive">OFFICIAL</span>}</div><h3 className="text-sm mt-2">{event.title}</h3><p className="text-[10px] uppercase tracking-wider text-slate-500 mt-3">{event.symbol || 'Market'} · {event.category.replaceAll('_', ' ')}</p></a>)}{!events.data?.data?.length && <p className="text-sm text-slate-500 lg:col-span-2 py-6">No important market events are available yet.</p>}</div></section>
    </div>;
}
