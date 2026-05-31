import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ChevronLeft, ChevronRight, ExternalLink, Newspaper, Search, X } from 'lucide-react';
import { api } from './api';
import { relativeTime } from './news-utils';

export default function NewsExplorer() {
    const [draft, setDraft] = useState({ q: '', stock: '', source: '', officialOnly: false });
    const [filters, setFilters] = useState({ page: 1, pageSize: 12 });
    const [selected, setSelected] = useState('');
    const news = useQuery({ queryKey: ['news-explorer', filters], queryFn: () => api.news(filters) });
    const detail = useQuery({ queryKey: ['news-detail', selected], queryFn: () => api.newsDetail(selected), enabled: Boolean(selected) });
    const impact = useQuery({ queryKey: ['news-impact', selected], queryFn: () => api.newsImpact(selected), enabled: Boolean(selected), retry: false });
    const meta = news.data?.meta ?? {};

    function apply(event) {
        event.preventDefault();
        setFilters({ ...draft, page: 1, pageSize: 12 });
    }

    return <section className="pb-20 lg:pb-0">
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-4">
        <div><p className="text-xs uppercase tracking-[.18em] text-positive font-semibold">Market news</p><h1 className="text-3xl font-semibold mt-2">News explorer</h1><p className="text-sm text-slate-400 mt-2">Browse recent market stories and see what they may mean for related stocks.</p></div>
        <div className="rounded-lg border border-line px-3 py-2 text-xs text-slate-400">News from established publishers</div>
      </div>

      <form onSubmit={apply} className="news-filter-form surface rounded-2xl p-4 mt-6 grid md:grid-cols-[1.6fr_.7fr_.7fr_.5fr_auto] gap-3">
        <label className="relative"><Search size={15} className="absolute left-3 top-3 text-slate-500"/><input aria-label="Search news" value={draft.q} onChange={event => setDraft({ ...draft, q: event.target.value })} placeholder="Search headlines or summaries" className="w-full rounded-lg border border-line bg-white/[.025] py-2.5 pl-9 pr-3 text-sm"/></label>
        <input aria-label="Filter by stock" value={draft.stock} onChange={event => setDraft({ ...draft, stock: event.target.value.toUpperCase() })} placeholder="Stock (INFY)" className="rounded-lg border border-line bg-white/[.025] px-3 py-2.5 text-sm"/>
        <input aria-label="Filter by publisher" value={draft.source} onChange={event => setDraft({ ...draft, source: event.target.value })} placeholder="Publisher" className="rounded-lg border border-line bg-white/[.025] px-3 py-2.5 text-sm"/>
        <label className="flex items-center gap-2 rounded-lg border border-line px-3 py-2 text-xs text-slate-400"><input type="checkbox" checked={draft.officialOnly} onChange={event => setDraft({ ...draft, officialOnly: event.target.checked })} className="accent-emerald-400"/>Official only</label>
        <button className="rounded-lg bg-positive px-4 py-2.5 text-sm font-semibold text-ink">Apply</button>
      </form>

      {news.isLoading && <div className="surface rounded-2xl p-10 mt-5 text-center text-sm text-slate-500 animate-pulse">Loading the latest news…</div>}
      {news.error && <div role="alert" className="rounded-xl border border-negative/30 bg-negative/10 p-4 mt-5 text-sm text-negative">{news.error.message}</div>}
      {!news.isLoading && !news.error && <div className="grid lg:grid-cols-2 gap-4 mt-5">
        {(news.data?.data ?? []).map(article => <button key={article.id} onClick={() => setSelected(article.id)} className="surface rounded-2xl p-5 text-left hover:border-positive/30 transition-colors">
          <div className="flex flex-wrap items-center gap-2 text-[10px] text-slate-500"><span className="text-positive">{relativeTime(article.publishedAt)}</span><span>·</span><span>{article.sourceName}</span>{article.official && <span className="rounded border border-positive/30 px-1.5 text-positive">OFFICIAL</span>}</div>
          <h2 className="font-semibold leading-6 mt-3">{article.title}</h2>
          <p className="text-xs text-slate-400 leading-5 mt-2 line-clamp-3">{article.body || 'A summary is not available for this story.'}</p>
          <div className="flex flex-wrap gap-2 mt-4">{article.symbols?.map(symbol => <span key={symbol} className="rounded-md bg-cyan-400/10 px-2 py-1 text-[10px] text-cyan-300">{symbol}</span>)}{article.duplicateCount > 1 && <span className="rounded-md bg-white/5 px-2 py-1 text-[10px] text-slate-400">Covered by {article.duplicateCount} sources</span>}<span className="ml-auto text-[10px] text-slate-500">{article.language?.toUpperCase()}</span></div>
        </button>)}
        {!news.data?.data?.length && <div className="surface rounded-2xl p-12 text-center lg:col-span-2"><Newspaper className="mx-auto text-slate-600"/><p className="mt-3 text-sm">No news matches these filters</p><p className="text-xs text-slate-500 mt-1">Try a different stock, publisher or search term.</p></div>}
      </div>}

      <div className="flex items-center justify-between mt-5 text-xs text-slate-500"><span>{meta.total ?? 0} stories</span><div className="flex items-center gap-2"><button disabled={(filters.page ?? 1) <= 1} onClick={() => setFilters({ ...filters, page: filters.page - 1 })} className="rounded-lg border border-line p-2 disabled:opacity-30" aria-label="Previous page"><ChevronLeft size={15}/></button><span>Page {meta.page ?? 1} of {Math.max(meta.totalPages ?? 1, 1)}</span><button disabled={(filters.page ?? 1) >= (meta.totalPages ?? 1)} onClick={() => setFilters({ ...filters, page: filters.page + 1 })} className="rounded-lg border border-line p-2 disabled:opacity-30" aria-label="Next page"><ChevronRight size={15}/></button></div></div>

      {selected && <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm p-4 sm:p-10" onMouseDown={event => event.target === event.currentTarget && setSelected('')}>
        <article role="dialog" aria-modal="true" aria-label="News story details" className="surface max-w-3xl max-h-[90vh] overflow-y-auto scrollbar mx-auto rounded-2xl p-6 sm:p-8">
          <div className="flex items-start justify-between gap-4"><div><p className="text-xs text-positive">{detail.data?.data?.sourceName}</p><h2 className="text-2xl font-semibold leading-8 mt-2">{detail.data?.data?.title ?? 'Loading story…'}</h2></div><button onClick={() => setSelected('')} className="text-slate-500" aria-label="Close story"><X size={20}/></button></div>
          {detail.error && <p role="alert" className="mt-6 text-negative">{detail.error.message}</p>}
          {detail.data?.data && <><div className="flex flex-wrap gap-2 mt-5 text-[10px] text-slate-500"><span>{new Date(detail.data.data.publishedAt).toLocaleString('en-IN', { timeZone: 'Asia/Kolkata' })} IST</span>{detail.data.data.duplicateCount > 1 && <><span>·</span><span>Covered by {detail.data.data.duplicateCount} sources</span></>}</div><p className="mt-6 text-sm leading-7 text-slate-300 whitespace-pre-wrap">{detail.data.data.body || 'A summary is not available for this story. Open the original article to read more.'}</p>{impact.data?.data && <ImpactPanel impact={impact.data.data}/>} {!impact.isLoading && !impact.data && <div className="rounded-xl border border-line bg-white/[.02] p-4 mt-5 text-xs text-slate-500">The stock impact lists are not ready yet. Please check again shortly.</div>}<a href={detail.data.data.url} target="_blank" rel="noreferrer" className="mt-5 inline-flex items-center gap-2 text-sm text-positive">Read the original article <ExternalLink size={14}/></a><p className="text-[10px] text-slate-600 mt-6">Stock impact directions are estimates based on available news, not financial advice.</p></>}
        </article>
      </div>}
    </section>;
}

function ImpactPanel({ impact }) {
    const pendingMessage = 'This list is still being prepared.';
    return <><section className="grid md:grid-cols-2 gap-4 mt-6"><ImpactList title="Other affected stocks" subtitle="Related stocks outside your portfolio" items={impact.marketImpact} emptyMessage={impact.ready ? undefined : pendingMessage}/><ImpactList title="Your portfolio trend" subtitle="Only holdings affected by this story" items={impact.portfolioImpact} emptyMessage={impact.ready ? undefined : pendingMessage}/></section>{!impact.ready && <p className="text-[10px] text-slate-600 mt-3">Stock links will appear here when the story review is ready.</p>}</>;
}

function ImpactList({ title, subtitle, items = [], emptyMessage }) {
    return <div className="rounded-xl border border-line bg-white/[.02] p-5"><h3 className="font-semibold">{title}</h3><p className="text-[10px] text-slate-500 mt-1">{subtitle}</p><div className="space-y-3 mt-4">{items.map(item => <article key={item.symbol} className="rounded-lg border border-line p-3"><div className="flex items-center justify-between gap-3"><div><p className="text-sm font-medium">{item.symbol}</p>{item.companyName && <p className="text-[10px] text-slate-500 mt-0.5">{item.companyName}</p>}</div><Direction value={item.direction}/></div><p className="text-xs leading-5 text-slate-400 mt-3">{item.explanation}</p>{item.horizon && <p className="text-[10px] text-slate-600 mt-2">Expected period: {item.horizon}</p>}</article>)}{!items.length && <p className="text-xs leading-5 text-slate-500 py-4">{emptyMessage || 'No stocks in this group are currently linked to the story.'}</p>}</div></div>;
}

function Direction({ value }) {
    const label = value === 'bullish' ? 'Bullish' : value === 'bearish' ? 'Bearish' : 'Mixed';
    const style = value === 'bullish' ? 'bg-positive/10 text-positive' : value === 'bearish' ? 'bg-negative/10 text-negative' : 'bg-warning/10 text-warning';
    return <span className={`rounded-full px-2 py-1 text-[10px] font-medium ${style}`}>{label}</span>;
}
