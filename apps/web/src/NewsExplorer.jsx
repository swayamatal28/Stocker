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
    const meta = news.data?.meta ?? {};

    function apply(event) {
        event.preventDefault();
        setFilters({ ...draft, page: 1, pageSize: 12 });
    }

    return <section className="pb-20 lg:pb-0">
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-4">
        <div><p className="text-xs uppercase tracking-[.18em] text-positive font-semibold">Phase 2 evidence feed</p><h1 className="text-3xl font-semibold mt-2">News explorer</h1><p className="text-sm text-slate-400 mt-2">Attributed source items, deduplicated into story clusters. No AI conclusions are added here.</p></div>
        <div className={`rounded-lg border px-3 py-2 text-xs ${meta.syntheticOnly ? 'border-warning/25 bg-warning/[.06] text-warning' : 'border-line text-slate-400'}`}>{meta.syntheticOnly ? 'Synthetic fixtures only' : 'Source-backed collection'}</div>
      </div>

      <form onSubmit={apply} className="surface rounded-2xl p-4 mt-6 grid md:grid-cols-[1.6fr_.7fr_.7fr_.5fr_auto] gap-3">
        <label className="relative"><Search size={15} className="absolute left-3 top-3 text-slate-500"/><input aria-label="Search collected news" value={draft.q} onChange={event => setDraft({ ...draft, q: event.target.value })} placeholder="Search title or retained text" className="w-full rounded-lg border border-line bg-white/[.025] py-2.5 pl-9 pr-3 text-sm"/></label>
        <input aria-label="Filter by stock" value={draft.stock} onChange={event => setDraft({ ...draft, stock: event.target.value.toUpperCase() })} placeholder="Stock (INFY)" className="rounded-lg border border-line bg-white/[.025] px-3 py-2.5 text-sm"/>
        <input aria-label="Filter by source" value={draft.source} onChange={event => setDraft({ ...draft, source: event.target.value })} placeholder="Source ID" className="rounded-lg border border-line bg-white/[.025] px-3 py-2.5 text-sm"/>
        <label className="flex items-center gap-2 rounded-lg border border-line px-3 py-2 text-xs text-slate-400"><input type="checkbox" checked={draft.officialOnly} onChange={event => setDraft({ ...draft, officialOnly: event.target.checked })} className="accent-emerald-400"/>Official only</label>
        <button className="rounded-lg bg-positive px-4 py-2.5 text-sm font-semibold text-ink">Apply</button>
      </form>

      {news.isLoading && <div className="surface rounded-2xl p-10 mt-5 text-center text-sm text-slate-500 animate-pulse">Loading collected evidence…</div>}
      {news.error && <div role="alert" className="rounded-xl border border-negative/30 bg-negative/10 p-4 mt-5 text-sm text-negative">{news.error.message}</div>}
      {!news.isLoading && !news.error && <div className="grid lg:grid-cols-2 gap-4 mt-5">
        {(news.data?.data ?? []).map(article => <button key={article.id} onClick={() => setSelected(article.id)} className="surface rounded-2xl p-5 text-left hover:border-positive/30 transition-colors">
          <div className="flex flex-wrap items-center gap-2 text-[10px] text-slate-500"><span className="text-positive">{relativeTime(article.publishedAt)}</span><span>·</span><span>{article.sourceName}</span>{article.synthetic && <span className="rounded border border-warning/30 px-1.5 text-warning">SYNTHETIC</span>}{article.official && <span className="rounded border border-positive/30 px-1.5 text-positive">OFFICIAL</span>}</div>
          <h2 className="font-semibold leading-6 mt-3">{article.title}</h2>
          <p className="text-xs text-slate-400 leading-5 mt-2 line-clamp-3">{article.body || 'The source did not provide retainable summary text.'}</p>
          <div className="flex flex-wrap gap-2 mt-4">{article.symbols?.map(symbol => <span key={symbol} className="rounded-md bg-cyan-400/10 px-2 py-1 text-[10px] text-cyan-300">{symbol}</span>)}{article.duplicateCount > 1 && <span className="rounded-md bg-white/5 px-2 py-1 text-[10px] text-slate-400">{article.duplicateCount} clustered copies</span>}<span className="ml-auto text-[10px] text-slate-500">{article.language?.toUpperCase()}</span></div>
        </button>)}
        {!news.data?.data?.length && <div className="surface rounded-2xl p-12 text-center lg:col-span-2"><Newspaper className="mx-auto text-slate-600"/><p className="mt-3 text-sm">No news matches these filters</p><p className="text-xs text-slate-500 mt-1">Only collected, policy-approved items appear here.</p></div>}
      </div>}

      <div className="flex items-center justify-between mt-5 text-xs text-slate-500"><span>{meta.total ?? 0} story clusters</span><div className="flex items-center gap-2"><button disabled={(filters.page ?? 1) <= 1} onClick={() => setFilters({ ...filters, page: filters.page - 1 })} className="rounded-lg border border-line p-2 disabled:opacity-30" aria-label="Previous page"><ChevronLeft size={15}/></button><span>Page {meta.page ?? 1} of {Math.max(meta.totalPages ?? 1, 1)}</span><button disabled={(filters.page ?? 1) >= (meta.totalPages ?? 1)} onClick={() => setFilters({ ...filters, page: filters.page + 1 })} className="rounded-lg border border-line p-2 disabled:opacity-30" aria-label="Next page"><ChevronRight size={15}/></button></div></div>

      {selected && <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm p-4 sm:p-10" onMouseDown={event => event.target === event.currentTarget && setSelected('')}>
        <article role="dialog" aria-modal="true" aria-label="News evidence detail" className="surface max-w-3xl max-h-[90vh] overflow-y-auto scrollbar mx-auto rounded-2xl p-6 sm:p-8">
          <div className="flex items-start justify-between gap-4"><div><p className="text-xs text-positive">{detail.data?.data?.sourceName}</p><h2 className="text-2xl font-semibold leading-8 mt-2">{detail.data?.data?.title ?? 'Loading evidence…'}</h2></div><button onClick={() => setSelected('')} className="text-slate-500" aria-label="Close evidence"><X size={20}/></button></div>
          {detail.error && <p role="alert" className="mt-6 text-negative">{detail.error.message}</p>}
          {detail.data?.data && <><div className="flex flex-wrap gap-2 mt-5 text-[10px] text-slate-500"><span>{new Date(detail.data.data.publishedAt).toLocaleString('en-IN', { timeZone: 'Asia/Kolkata' })} IST</span><span>·</span><span>Parser {detail.data.data.parserVersion}</span><span>·</span><span>{detail.data.data.duplicateCount} item cluster</span></div><p className="mt-6 text-sm leading-7 text-slate-300 whitespace-pre-wrap">{detail.data.data.body || 'No retained body text is permitted or available for this item.'}</p><div className="rounded-xl border border-line bg-white/[.02] p-4 mt-6 text-xs leading-5 text-slate-400"><p><span className="text-slate-200">Attribution:</span> {detail.data.data.attribution}</p><p className="mt-1"><span className="text-slate-200">Licence:</span> {detail.data.data.licence}</p></div><a href={detail.data.data.url} target="_blank" rel="noreferrer" className="mt-5 inline-flex items-center gap-2 text-sm text-positive">Open original source <ExternalLink size={14}/></a><p className="text-[10px] text-slate-600 mt-6">Collected evidence only. AI analysis and investment signals are separate later-stage outputs. Informational research, not financial advice.</p></>}
        </article>
      </div>}
    </section>;
}
