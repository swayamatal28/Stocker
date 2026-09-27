import { useQuery } from '@tanstack/react-query';
import { AlertTriangle, CheckCircle2, Clock3, ShieldCheck } from 'lucide-react';
import { api } from './api';

export default function EvaluationCenter() {
    const report = useQuery({ queryKey: ['evaluation-report'], queryFn: () => api.evaluationReport() });
    const data = report.data?.data;
    const meta = report.data?.meta;
    const cards = [
        ['Candidates', data?.candidateSignals ?? 0, ShieldCheck, 'All signals at or before the evaluation time'],
        ['Evaluated', data?.evaluatedSignals ?? 0, CheckCircle2, 'Signals with a mature, timestamp-safe outcome'],
        ['Pending', data?.pendingSignals ?? 0, Clock3, 'Awaiting their horizon or a valid exit quote'],
        ['Leakage rejected', data?.leakageViolations?.length ?? 0, AlertTriangle, 'Features unavailable at the original decision time']
    ];
    return <div>
        <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4"><div><p className="text-xs uppercase tracking-[.18em] text-positive font-semibold">Phase 6 assurance</p><h1 className="text-3xl font-semibold mt-2">Signal evaluation</h1><p className="text-sm text-slate-400 mt-2">Historical outcomes reconstructed at event time, with explicit look-ahead leakage rejection.</p></div>{data && <div className={`rounded-lg border px-3 py-2 text-xs ${meta?.lookAheadSafe ? 'border-positive/20 text-positive' : 'border-warning/20 text-warning'}`}>{meta?.lookAheadSafe ? 'No leakage detected' : 'Leakage findings require review'}</div>}</div>
        {report.isLoading && <div className="surface rounded-2xl p-8 mt-6 text-sm text-slate-500">Reconstructing historical outcomes…</div>}
        {report.error && <div role="alert" className="mt-6 rounded-xl border border-negative/30 bg-negative/10 p-4 text-sm text-negative">{report.error.message}</div>}
        {data && <>
            <div className="grid sm:grid-cols-2 xl:grid-cols-4 gap-3 mt-6">{cards.map(([label, value, Icon, detail]) => <section key={label} className="surface rounded-2xl p-5"><div className="flex items-center gap-2 text-slate-400"><Icon size={16}/><span className="text-xs uppercase tracking-wider">{label}</span></div><p className="font-mono text-3xl mt-3">{value}</p><p className="text-[11px] leading-5 text-slate-500 mt-2">{detail}</p></section>)}</div>
            <section className="surface rounded-2xl mt-5 overflow-hidden"><div className="p-5"><h2 className="font-semibold">Accuracy and positive precision</h2><p className="text-xs text-slate-500 mt-1">Reported separately by confidence band, category, sector and horizon. Zero means the slice has no positive predictions.</p></div><div className="overflow-x-auto"><table className="w-full text-sm"><thead className="border-y border-line text-[10px] uppercase tracking-wider text-slate-500"><tr><th className="text-left p-3 pl-5">Dimension</th><th className="text-left p-3">Slice</th><th className="text-right p-3">N</th><th className="text-right p-3">Accuracy</th><th className="text-right p-3 pr-5">Precision</th></tr></thead><tbody>{(data.slices ?? []).map(slice => <tr key={`${slice.dimension}-${slice.value}`} className="border-b border-line/60"><td className="p-3 pl-5 text-slate-500">{slice.dimension}</td><td className="p-3">{slice.value}</td><td className="p-3 text-right font-mono">{slice.evaluated}</td><td className="p-3 text-right font-mono">{slice.accuracy.toFixed(2)}%</td><td className="p-3 pr-5 text-right font-mono">{slice.precision.toFixed(2)}%</td></tr>)}</tbody></table>{!data.slices?.length && <p className="p-8 text-center text-sm text-slate-500">Insufficient matured history for sliced metrics. Outcomes appear after each signal horizon has elapsed and a timestamp-safe quote exists.</p>}</div></section>
            {(data.leakageViolations?.length ?? 0) > 0 && <section className="surface rounded-2xl border-warning/20 p-5 mt-5"><h2 className="font-semibold text-warning">Leakage findings</h2><div className="mt-3 space-y-2">{data.leakageViolations.map(item => <div key={`${item.signalId}-${item.field}`} className="rounded-xl border border-line p-3 text-xs"><p>{item.field} became available after decision time</p><p className="text-slate-500 mt-1">Signal {item.signalId} · available {new Date(item.availableAt).toLocaleString('en-IN')}</p></div>)}</div></section>}
            <p className="mt-4 text-[10px] text-slate-600">Evaluation version {data.version} · as of {new Date(data.asOf).toLocaleString('en-IN')}. Historical performance does not guarantee future results.</p>
        </>}
    </div>;
}
