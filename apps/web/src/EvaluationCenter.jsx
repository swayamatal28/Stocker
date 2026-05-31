import { useQuery } from '@tanstack/react-query';
import { AlertTriangle, CheckCircle2, Clock3, ShieldCheck } from 'lucide-react';
import { api } from './api';

export default function EvaluationCenter() {
    const report = useQuery({ queryKey: ['evaluation-report'], queryFn: () => api.evaluationReport() });
    const data = report.data?.data;
    const meta = report.data?.meta;
    const cards = [
        ['Total outlooks', data?.candidateSignals ?? 0, ShieldCheck, 'All past stock outlooks included in this review'],
        ['Results available', data?.evaluatedSignals ?? 0, CheckCircle2, 'Outlooks old enough to compare with what happened'],
        ['Waiting for results', data?.pendingSignals ?? 0, Clock3, 'Recent outlooks that need more time'],
        ['Excluded', data?.leakageViolations?.length ?? 0, AlertTriangle, 'Outlooks that could not be compared fairly']
    ];
    return <div>
        <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4"><div><p className="text-xs uppercase tracking-[.18em] text-positive font-semibold">Past performance</p><h1 className="text-3xl font-semibold mt-2">Outlook accuracy</h1><p className="text-sm text-slate-400 mt-2">See how earlier stock outlooks compared with later price movements.</p></div>{data && <div className={`rounded-lg border px-3 py-2 text-xs ${meta?.lookAheadSafe ? 'border-positive/20 text-positive' : 'border-warning/20 text-warning'}`}>{meta?.lookAheadSafe ? 'Fair comparison' : 'Some results were excluded'}</div>}</div>
        {report.isLoading && <div className="surface rounded-2xl p-8 mt-6 text-sm text-slate-500">Checking past results…</div>}
        {report.error && <div role="alert" className="mt-6 rounded-xl border border-negative/30 bg-negative/10 p-4 text-sm text-negative">{report.error.message}</div>}
        {data && <>
            <div className="grid sm:grid-cols-2 xl:grid-cols-4 gap-3 mt-6">{cards.map(([label, value, Icon, detail]) => <section key={label} className="surface rounded-2xl p-5"><div className="flex items-center gap-2 text-slate-400"><Icon size={16}/><span className="text-xs uppercase tracking-wider">{label}</span></div><p className="font-mono text-3xl mt-3">{value}</p><p className="text-[11px] leading-5 text-slate-500 mt-2">{detail}</p></section>)}</div>
            <section className="surface rounded-2xl mt-5 overflow-hidden"><div className="p-5"><h2 className="font-semibold">Results by group</h2><p className="text-xs text-slate-500 mt-1">Compare results by confidence, event type, sector and time period.</p></div><div className="table-shell"><table className="evaluation-table responsive-data-table w-full text-sm"><thead className="border-y border-line text-[10px] uppercase tracking-wider text-slate-500"><tr><th className="text-left p-3 pl-5">Group</th><th className="text-left p-3">Category</th><th className="text-right p-3">Outlooks</th><th className="text-right p-3">Correct</th><th className="text-right p-3 pr-5">Positive calls correct</th></tr></thead><tbody>{(data.slices ?? []).map(slice => <tr key={`${slice.dimension}-${slice.value}`} className="border-b border-line/60"><td data-label="Group" className="p-3 pl-5 text-slate-500">{slice.dimension}</td><td data-label="Category" className="p-3">{slice.value}</td><td data-label="Outlooks" className="p-3 text-right font-mono">{slice.evaluated}</td><td data-label="Correct" className="p-3 text-right font-mono">{slice.accuracy.toFixed(2)}%</td><td data-label="Positive calls correct" className="p-3 pr-5 text-right font-mono">{slice.precision.toFixed(2)}%</td></tr>)}</tbody></table>{!data.slices?.length && <p className="p-8 text-center text-sm text-slate-500">There is not enough history to show these results yet. More results will appear as earlier outlooks become old enough to review.</p>}</div></section>
            {(data.leakageViolations?.length ?? 0) > 0 && <section className="surface rounded-2xl border-warning/20 p-5 mt-5"><h2 className="font-semibold text-warning">Excluded comparisons</h2><p className="text-xs text-slate-500 mt-2">{data.leakageViolations.length} result{data.leakageViolations.length === 1 ? ' was' : 's were'} excluded because a fair comparison was not possible.</p></section>}
            <p className="mt-4 text-[10px] text-slate-600">Results updated {new Date(data.asOf).toLocaleString('en-IN')}. Past performance does not guarantee future results.</p>
        </>}
    </div>;
}
