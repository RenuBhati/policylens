import React, { useEffect, useRef, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { ArrowDownToLine, ArrowRight, ArrowUpRight, BookOpen, Check, CheckCircle2, ChevronRight, CircleHelp, Code2, ExternalLink, FileCode2, Fingerprint, FolderCheck, GitBranch, Layers3, Loader2, LockKeyhole, Search, Send, Shield, ShieldCheck, Sparkles, Terminal, XCircle } from 'lucide-react';
import './styles.css';

type Policy = { id: string; title: string; category: string; summary: string; rule: string; rationale: string; scope: string; field: string; fix: string; question: string; definition: string; source: { url: string; sha256: string } };
type Status = { engine_available: boolean; engine_version: string; answer_mode: string; model: string; policy_count: number; provenance: { revision: string } };
type CheckResult = { engine: string; revision: string; resource: string; latency_ms: number; summary: { pass: number; fail: number; skipped: number; error: number }; results: { policy_id: string; title: string; status: string; message: string; field: string; fix: string; source_url: string }[] };
type Answer = { answer: string; mode: string; abstained: boolean; latency_ms: number; citations: string[]; evidence: { id: string; policy_id: string; title: string; text: string; url: string; score: number }[] };
type View = 'explore' | 'check' | 'ask';
const icons = [Shield, Fingerprint, LockKeyhole, GitBranch, FolderCheck, Layers3];

async function api<T>(path: string, body?: unknown): Promise<T> {
  const response = await fetch(`/api/${path}`, body === undefined ? undefined : { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  const data = await response.json();
  if (!response.ok) throw new Error(data.error || `Request failed (${response.status})`);
  return data as T;
}

function App() {
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [status, setStatus] = useState<Status>();
  const [examples, setExamples] = useState({ failing: '', passing: '' });
  const [selected, setSelected] = useState('disallow-privileged-containers');
  const [view, setView] = useState<View>('explore');
  const [filter, setFilter] = useState('');
  const [loadError, setLoadError] = useState('');
  const [manifest, setManifest] = useState('');
  const [sample, setSample] = useState('failing');
  const [checkResult, setCheckResult] = useState<CheckResult>();
  const [checkError, setCheckError] = useState('');
  const [checking, setChecking] = useState(false);
  const [checkedManifest, setCheckedManifest] = useState('');
  const [question, setQuestion] = useState('');
  const [answer, setAnswer] = useState<Answer>();
  const [askError, setAskError] = useState('');
  const [asking, setAsking] = useState(false);
  const [scope, setScope] = useState('collection');
  const [answerQuestion, setAnswerQuestion] = useState('');
  const gutter = useRef<HTMLPreElement>(null);
  const current = policies.find(p => p.id === selected) ?? policies[0];

  useEffect(() => {
    Promise.all([api<Policy[]>('policies'), api<Status>('status'), api<typeof examples>('examples')])
      .then(([ps, st, ex]) => { setPolicies(ps); setStatus(st); setExamples(ex); setManifest(ex.failing); })
      .catch(e => setLoadError(e.message));
  }, []);

  function loadSample(which: 'failing' | 'passing') { setManifest(examples[which]); setSample(which); setCheckResult(undefined); setCheckError(''); }
  async function runCheck() {
    const submitted = manifest;
    setChecking(true); setCheckError(''); setCheckResult(undefined);
    try { const result = await api<CheckResult>('check', { manifest: submitted }); setCheckResult(result); setCheckedManifest(submitted); }
    catch (e) { setCheckError((e as Error).message); }
    finally { setChecking(false); }
  }
  async function ask(event?: React.FormEvent, supplied?: string) {
    event?.preventDefault(); const q = supplied ?? question; if (!q.trim() || asking) return;
    setQuestion(q); setAsking(true); setAskError(''); setAnswer(undefined); setAnswerQuestion(q);
    try { setAnswer(await api<Answer>('ask', { question: q, policy_id: scope === 'selected' ? selected : '' })); }
    catch (e) { setAskError((e as Error).message); }
    finally { setAsking(false); }
  }
  function downloadManifest() {
    const url = URL.createObjectURL(new Blob([manifest], { type: 'text/yaml' })); const a = document.createElement('a'); a.href = url; a.download = 'policylens-pod.yaml'; a.click(); URL.revokeObjectURL(url);
  }
  function changePolicy(id: string) { setSelected(id); setView('explore'); }

  return <div className="app-shell">
    <aside className="sidebar" aria-label="Policy collection">
      <a className="brand" href="/" aria-label="PolicyLens home"><span className="brand-mark"><ShieldCheck size={23}/></span><span>Policy<span className="brand-light">Lens</span><small>OPEN POLICY EXPLORER</small></span></a>
      <div className="collection-label">YOUR COLLECTION <span>01</span></div>
      <div className="collection-card"><span className="kube-icon">⎈</span><div><strong>Kubernetes</strong><small>Security & best practices</small></div><ChevronRight size={16}/></div>
      <label className="search-box"><Search size={16}/><span className="sr-only">Search policies</span><input value={filter} onChange={e => setFilter(e.target.value)} placeholder="Find a policy…"/></label>
      <div className="sidebar-section-label">POLICIES <span>{policies.length.toString().padStart(2, '0')}</span></div>
      <nav className="policy-nav" aria-label="Choose a policy">
        {policies.map((p, index) => ({ p, index })).filter(({ p }) => `${p.title} ${p.summary} ${p.field}`.toLowerCase().includes(filter.toLowerCase())).map(({ p, index }) => {
          const Icon = icons[index] ?? Shield;
          return <button key={p.id} className={selected === p.id ? 'policy-link active' : 'policy-link'} onClick={() => changePolicy(p.id)} aria-current={selected === p.id ? 'true' : undefined}><Icon size={17}/><span>{p.title}</span><ChevronRight size={14}/></button>;
        })}
        {policies.length > 0 && !policies.some(p => `${p.title} ${p.summary} ${p.field}`.toLowerCase().includes(filter.toLowerCase())) && <p className="no-policies">No matching policies.</p>}
      </nav>
      <div className="sidebar-bottom"><div className="public-note"><BookOpen size={20}/><strong>Public rules. Open sources.</strong><p>Explore real policies from the Kyverno community.</p><a href="https://github.com/kyverno/policies" target="_blank" rel="noreferrer">View the collection <ArrowUpRight size={14}/></a></div><span className="sidebar-footer"><span className="tiny-dot"/> Sample workspace <span>v0.1</span></span></div>
    </aside>

    <main>
      <header className="topbar"><div className="breadcrumbs"><span>Public collections</span><ChevronRight size={14}/><strong>Kubernetes</strong></div><div className="topbar-right"><span className="engine-indicator"><span className={`tiny-dot ${status && !status.engine_available ? 'amber' : ''}`}/>{status ? status.engine_available ? 'Kyverno ready' : 'Engine unavailable' : 'Connecting…'}</span><a href="https://github.com/kyverno/policies" target="_blank" rel="noreferrer" className="source-top">Source library <ExternalLink size={14}/></a></div></header>
      <div className="content">
        <div className="page-heading"><div><div className="eyebrow"><span/> SECURITY, MADE UNDERSTANDABLE</div><h1>Understand the rule.<br/><span>Check the configuration.</span></h1><p>A clearer way to explore Kubernetes policies — with real sources<br className="desktop-break"/> and checks you can reproduce.</p></div><div className="pack-stamp"><ShieldCheck size={25}/><strong>6 public policies</strong><span>One versioned collection</span><small>Kyverno {status?.engine_version ?? '1.19.1'}</small></div></div>
        <nav className="view-tabs" aria-label="Workspace views">
          <button className={view === 'explore' ? 'selected' : ''} onClick={() => setView('explore')} aria-current={view === 'explore' ? 'page' : undefined}><BookOpen size={17}/> Explore policies</button>
          <button className={view === 'check' ? 'selected' : ''} onClick={() => setView('check')} aria-current={view === 'check' ? 'page' : undefined}><Code2 size={17}/> Check a Pod</button>
          <button className={view === 'ask' ? 'selected' : ''} onClick={() => setView('ask')} aria-current={view === 'ask' ? 'page' : undefined}><Sparkles size={17}/> Ask a question</button>
          <span className="tabs-meta"><GitBranch size={13}/> {status?.provenance.revision.slice(0, 7) ?? 'versioned'} <span className="tabs-meta-label">source revision</span></span>
        </nav>
        {loadError && <div className="error-box" role="alert">Could not load PolicyLens: {loadError}. Check that the Go service is running.</div>}
        {!current && !loadError && <div className="empty-state"><Loader2 className="spin"/> Loading public policies…</div>}

        {view === 'explore' && current && <>
          <div className="explore-grid">
            <article className="panel policy-detail"><div className="panel-top"><span className={`category ${current.category.includes('convention') ? 'blue' : ''}`}>{current.category}</span><span className="muted small">{(policies.indexOf(current) + 1).toString().padStart(2, '0')} / 06</span></div><h2>{current.title}</h2><p className="detail-summary">{current.summary}</p><div className="field-chip"><Code2 size={15}/><code>{current.field}</code></div><div className="detail-section"><h3><CheckCircle2 size={17}/> What the rule checks</h3><p>{current.rule}</p></div><div className="detail-section"><h3><CircleHelp size={17}/> Why it matters</h3><p>{current.rationale}</p></div><div className="scope-note"><Fingerprint size={18}/><div><strong>Know the scope</strong><p>{current.scope}</p></div></div><footer className="source-footer"><span>Explanation authored for PolicyLens</span><a href={current.source.url} target="_blank" rel="noreferrer">Original policy <ArrowUpRight size={14}/></a></footer></article>
            <aside className="explore-right"><div className="panel quickstart"><span className="card-icon"><Terminal size={22}/></span><div className="eyebrow plain">FROM RULE TO REALITY</div><h2>See it in action.</h2><p>Start with a Pod that breaks these rules. Then check the corrected version.</p><div className="mini-example"><span><FileCode2 size={14}/> payment-api.yaml</span><pre><code>{'securityContext:\n  privileged: true\n  runAsNonRoot: false\n  allowPrivilegeEscalation: true'}</code></pre><div><XCircle size={14}/> A useful place to start</div></div><button className="primary full" onClick={() => { loadSample('failing'); setView('check'); }}>Try the sample check <ArrowRight size={17}/></button><small>Local evaluation · no cluster required</small></div><div className="panel source-card"><div><GitBranch size={18}/><h3>Trace every rule</h3></div><p>The original policy is pinned to a Git revision. Your check uses that same definition.</p><a href={current.source.url} target="_blank" rel="noreferrer">Inspect the source <ArrowUpRight size={15}/></a></div></aside>
          </div>
          <details className="panel definition-panel"><summary><FileCode2 size={17}/> Original Kyverno definition <span>YAML <ChevronRight size={15}/></span></summary><pre><code>{current.definition}</code></pre><p>Unmodified upstream file · SHA-256 <code>{current.source.sha256}</code></p></details>
        </>}

        {view === 'check' && <section className="check-workspace" aria-label="Pod configuration checker">
          <div className="section-heading"><div><h2>Check a Pod configuration</h2><p>Evaluate one Pod against all six policies in this collection.</p></div><span className="category">Offline Kyverno check</span></div>
          <div className="checker-grid"><div className="panel editor-panel"><div className="editor-toolbar"><span><FileCode2 size={16}/> pod.yaml</span><button className="icon-button" onClick={downloadManifest} aria-label="Download Pod manifest"><ArrowDownToLine size={16}/></button></div><div className="sample-switch"><button className={sample === 'failing' ? 'active' : ''} onClick={() => loadSample('failing')}>Failing example</button><button className={sample === 'passing' ? 'active' : ''} onClick={() => loadSample('passing')}>Corrected example</button><span>or edit below</span></div><label className="sr-only" htmlFor="manifest">Pod YAML configuration</label><div className="code-editor"><pre ref={gutter} aria-hidden="true">{manifest.split('\n').map((_, i) => i + 1).join('\n')}</pre><textarea id="manifest" value={manifest} onChange={e => { setManifest(e.target.value); setSample('custom'); }} onScroll={e => { if (gutter.current) gutter.current.scrollTop = e.currentTarget.scrollTop; }} spellCheck={false} autoCapitalize="off" autoCorrect="off"/></div><div className="editor-bottom"><span>v1 Pod only · max 64 KiB</span><button className="primary" onClick={() => void runCheck()} disabled={checking || !manifest.trim() || !status?.engine_available}>{checking ? <Loader2 className="spin" size={17}/> : <ShieldCheck size={17}/>} {checking ? 'Checking…' : 'Run policy check'}</button></div></div>
            <div className="results-column" aria-live="polite" aria-busy={checking}>
              {!status?.engine_available && <div className="error-box" role="alert">The pinned Kyverno engine is unavailable. Run <code>./scripts/install-kyverno.sh</code> and restart the service.</div>}
              {checking && <div className="panel result-placeholder"><Loader2 className="spin" size={30}/><h3>Evaluating your Pod</h3><p>Kyverno is checking the fixed policy collection.</p></div>}
              {checkError && <div className="error-box" role="alert">{checkError}</div>}
              {!checking && !checkResult && !checkError && <div className="panel result-placeholder"><span className="placeholder-icon"><FolderCheck size={28}/></span><h3>Evidence, before assumptions.</h3><p>Run a check to see which policies pass, which fail, and the exact engine messages.</p><div className="result-legend"><span><span className="tiny-dot"/> Pass</span><span><span className="tiny-dot red"/> Fail</span><span><span className="tiny-dot amber"/> Skipped / error</span></div></div>}
              {checkResult && <><div className={`panel result-summary ${checkResult.summary.fail || checkResult.summary.error || checkResult.summary.skipped ? 'has-failures' : 'all-pass'}`}>{checkResult.summary.fail || checkResult.summary.error || checkResult.summary.skipped ? <XCircle size={24}/> : <CheckCircle2 size={24}/>}<div><strong>{checkResult.summary.pass === 6 ? 'All six policies pass' : `${checkResult.summary.fail} failed · ${checkResult.summary.pass} passed`}</strong><small>{checkResult.engine} · {(checkResult.latency_ms / 1000).toFixed(2)}s{checkResult.summary.error > 0 ? ` · ${checkResult.summary.error} errors` : ''}{checkResult.summary.skipped > 0 ? ` · ${checkResult.summary.skipped} skipped` : ''}</small></div></div>{checkedManifest !== manifest && <div className="stale-note">You have edited the Pod. Run another check to update these results.</div>}<div className="panel results-list">{checkResult.results.map(r => <details className="result-item" key={r.policy_id} open={r.status !== 'pass'}><summary>{r.status === 'pass' ? <CheckCircle2 size={17} className="pass-icon"/> : <XCircle size={17} className="fail-icon"/>}<span>{r.title}</span><span className={`result-status ${r.status}`}>{r.status}</span></summary><div className="result-detail"><p>{r.message}</p><code>{r.field}</code>{r.status === 'fail' && <p className="fix-note"><strong>Suggested change</strong>{r.fix}</p>}<a href={r.source_url} target="_blank" rel="noreferrer">View checked policy <ArrowUpRight size={13}/></a></div></details>)}</div><button className="text-button" onClick={() => { setView('ask'); setQuestion('What does runAsNonRoot do?'); }}>Explore the explanation <ArrowRight size={15}/></button></>}
            </div></div><div className="workspace-note"><Shield size={16}/><p>This checks the selected pack, not complete Kubernetes security or live admission. The bundled upstream policies use audit actions. Example images are not pulled or deployed.</p></div>
        </section>}

        {view === 'ask' && current && <section className="ask-workspace"><div className="section-heading"><div><h2>Ask the policy collection</h2><p>Explore an explanation with the supporting passages beside it.</p></div><span className="category blue">{status?.answer_mode === 'ollama' ? `LLM · ${status.model}` : 'Source excerpts'}</span></div><div className="panel question-panel"><div className="ask-icon"><Sparkles size={23}/></div><h3>A question is a good place to start.</h3><p>{status?.answer_mode === 'ollama' ? 'The model answers from retrieved policy explanations. Inspect the evidence to assess its answer.' : 'This key-free mode returns relevant source-backed explanations. Enable Ollama for generated answers.'}</p><form onSubmit={e => void ask(e)}><label className="sr-only" htmlFor="question">Question about the policies</label><div className="question-input"><input id="question" value={question} onChange={e => setQuestion(e.target.value)} maxLength={1000} placeholder="Which registries are allowed in this demo?"/><button className="primary" disabled={asking || !question.trim()} type="submit">{asking ? <Loader2 size={17} className="spin"/> : <Send size={17}/>} {asking ? 'Finding sources…' : 'Ask sources'}</button></div><label className="scope-select">Search in <select value={scope} onChange={e => setScope(e.target.value)}><option value="collection">All six policies</option><option value="selected">Selected policy: {current.title}</option></select></label></form><div className="suggested-questions">{[current.question, 'Which image registries are allowed?', 'Why is the latest image tag rejected?'].filter((q, i, arr) => arr.indexOf(q) === i).map(q => <button key={q} disabled={asking} onClick={() => void ask(undefined, q)}>{q}<ArrowUpRight size={13}/></button>)}</div></div>
          <div aria-live="polite" aria-busy={asking}>{askError && <div className="error-box" role="alert">{askError}</div>}{answer && <div className="answer-grid"><article className="panel answer-panel"><div className="panel-top"><span className="category">{answer.abstained ? 'No supporting passage' : answer.mode === 'ollama' ? 'Generated explanation' : 'Retrieved explanations'}</span><span className="small muted">{answer.latency_ms} ms</span></div><h3>{answerQuestion}</h3><div className="answer-text">{answer.answer.split('\n\n').map((p, i) => <p key={i}>{p}</p>)}</div><small>{answer.mode === 'ollama' ? 'References are validated against retrieved passage IDs. Review whether the claims follow the evidence.' : 'Authored PolicyLens explanations retrieved from the versioned collection; no LLM generation.'}</small></article><aside className="panel evidence-panel"><h3><BookOpen size={17}/> Supporting passages</h3>{answer.evidence.length === 0 && <p>No evidence was retrieved from this collection.</p>}{answer.evidence.map((e, i) => <div className="evidence-item" key={e.id}><span className="evidence-number">{i + 1}</span><div><strong>{e.title}</strong><p>{e.text}</p><a href={e.url} target="_blank" rel="noreferrer">Original policy <ArrowUpRight size={13}/></a><small>{answer.citations.includes(e.id) ? 'Cited' : 'Retrieved, not cited'} · {e.id.split(':')[1]}</small></div></div>)}</aside></div>}</div><div className="workspace-note"><CircleHelp size={16}/><p>The collection covers six Kubernetes rules. It does not contain Citi policies, VPN permissions or a complete security standard.</p></div></section>}
        <footer className="page-footer"><span><Check size={13}/> Versioned sources <span>·</span> Reproducible checks <span>·</span> Clear explanations</span><a href="https://kyverno.io" target="_blank" rel="noreferrer">Powered by Kyverno <ArrowUpRight size={13}/></a></footer>
      </div>
    </main>
  </div>;
}

createRoot(document.getElementById('root')!).render(<React.StrictMode><App/></React.StrictMode>);
