'use strict';
// Kanjō Studio — frontend (vanilla JS, sans dépendance, sans réseau externe). Il parle à
// l'API JSON locale servie par `kanjo studio`. Une couche fine ; le cœur métier est en Go.

const token = document.querySelector('meta[name="kanjo-token"]').content;
const api = (path, opts = {}) => {
  opts.headers = Object.assign({ 'X-Kanjo-Token': token }, opts.headers || {});
  return fetch(path, opts);
};

const state = { docs: [], selected: -1, filter: null };

const sealCls = { ok: '', warning: 'warn', error: 'err', none: 'none' };
// Libellés français des verdicts. Le statut est toujours doublé d'un libellé français et d'un
// aria-label ; jamais porté par la couleur ou l'icône seule (§12.2, §12.10).
const verdictLabel = { ok: 'conforme', warning: 'sous réserve', error: 'non conforme', none: 'non validé' };

// Icônes (style Lucide : trait, sans remplissage). Le badge colore l'icône via currentColor.
const ICON = {
  ok: '<svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="9"/><path d="m8.5 12 2.5 2.5 4.5-5"/></svg>',
  warning: '<svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3 2.5 20h19z"/><path d="M12 10v4"/><path d="M12 17h.01"/></svg>',
  error: '<svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="9"/><path d="m9 9 6 6M15 9l-6 6"/></svg>',
  none: '<svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="9" stroke-dasharray="3 3"/></svg>',
  sun: '<path d="M12 7a5 5 0 1 0 0 10 5 5 0 0 0 0-10z"/><path d="M12 2v2M12 20v2M2 12h2M20 12h2M5 5l1.4 1.4M17.6 17.6 19 19M19 5l-1.4 1.4M6.4 17.6 5 19"/>',
  moon: '<path d="M20 14A8 8 0 0 1 10 4a7 7 0 1 0 10 10z"/>',
};

function sealHTML(v, extra = '') {
  const lbl = verdictLabel[v] || 'inconnu';
  return `<span class="seal ${extra} ${sealCls[v]}" role="img" aria-label="${lbl}" title="${lbl}">${ICON[v] || ICON.none}</span>`;
}

// ---- Formatage (français) ----
function money(a) {
  if (!a || a.currency === undefined) return '—';
  const neg = a.value < 0;
  let s = Math.abs(a.value).toString().padStart(a.scale + 1, '0');
  const i = a.scale ? s.slice(0, -a.scale) : s;
  const f = a.scale ? s.slice(-a.scale) : '';
  const grouped = i.replace(/\B(?=(\d{3})+(?!\d))/g, ' ');
  const sym = { EUR: '€', USD: '$', GBP: '£' }[a.currency] || a.currency;
  return (neg ? '-' : '') + grouped + (f ? ',' + f : '') + ' ' + sym;
}
const isoDate = d => d ? `${String(d.year).padStart(4,'0')}-${String(d.month).padStart(2,'0')}-${String(d.day).padStart(2,'0')}` : '';

// ---- Version / règles ----
api('/api/version').then(r => r.json()).then(v => {
  document.getElementById('ver').textContent = `v${v.tool} · schéma ${v.schema}`;
  document.getElementById('rules').textContent = `jeu de règles ${v.rules}`;
});

// ---- Thème clair / sombre ----
const themeBtn = document.getElementById('theme');
const themeIcon = document.getElementById('theme-icon');
function applyThemeIcon() {
  // En thème clair on propose la lune (passer en sombre), et inversement.
  themeIcon.innerHTML = document.documentElement.dataset.theme === 'hiru' ? ICON.moon : ICON.sun;
}
document.documentElement.dataset.theme = localStorage.getItem('kanjo-theme') || 'hiru';
applyThemeIcon();
themeBtn.onclick = () => {
  const t = document.documentElement.dataset.theme === 'hiru' ? 'yoru' : 'hiru';
  document.documentElement.dataset.theme = t;
  localStorage.setItem('kanjo-theme', t);
  applyThemeIcon();
};

// ---- Navigation ----
function show(screen) {
  document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
  document.getElementById('screen-' + screen).classList.add('active');
  document.querySelectorAll('#rail button').forEach(b =>
    b.classList.toggle('active', b.dataset.screen === screen));
  if (screen === 'sho') renderReport();
}
document.querySelectorAll('#rail button').forEach(b => b.onclick = () => show(b.dataset.screen));

// ---- Dépôt / upload ----
const drop = document.getElementById('drop');
['dragover', 'dragenter'].forEach(e => drop.addEventListener(e, ev => { ev.preventDefault(); drop.classList.add('over'); }));
['dragleave', 'drop'].forEach(e => drop.addEventListener(e, ev => { ev.preventDefault(); drop.classList.remove('over'); }));
drop.addEventListener('drop', ev => handleFiles(ev.dataTransfer.files));
document.getElementById('file').addEventListener('change', ev => handleFiles(ev.target.files));

function handleFiles(files) {
  const list = Array.from(files);
  if (!list.length) return;
  Promise.all(list.map(inspectFile)).then(() => { show('ken'); renderDocList(); });
}

function inspectFile(file) {
  return file.arrayBuffer()
    .then(buf => api('/api/inspect', {
      method: 'POST',
      headers: { 'Content-Type': 'application/octet-stream', 'X-Kanjo-Filename': file.name },
      body: buf,
    }))
    .then(r => r.json())
    .then(res => {
      state.docs.push({
        name: file.name,
        verdict: res.error ? 'error' : (res.verdict || 'none'),
        format: res.format || '', findings: res.findings || [],
        document: res.document || null, error: res.error || '',
      });
      updateCounters();
    });
}

// ---- Compteurs (barre d'état) ----
function updateCounters() {
  const c = { ok: 0, warning: 0, error: 0 };
  state.docs.forEach(d => { c[d.verdict] = (c[d.verdict] || 0) + 1; });
  document.getElementById('c-ok').textContent = c.ok;
  document.getElementById('c-warn').textContent = c.warning;
  document.getElementById('c-err').textContent = c.error;
}
document.querySelectorAll('footer .count').forEach(el => el.onclick = () => {
  state.filter = state.filter === el.dataset.f ? null : el.dataset.f;
  renderDocList();
});

// ---- Liste des documents (inspecteur) ----
function renderDocList() {
  const tb = document.getElementById('doclist');
  tb.innerHTML = '';
  state.docs.forEach((d, i) => {
    if (state.filter && d.verdict !== state.filter) return;
    const tr = document.createElement('tr');
    if (i === state.selected) tr.className = 'sel';
    tr.innerHTML = `<td>${sealHTML(d.verdict, 'sm')}</td>` +
      `<td>${escapeHtml(d.name)}</td>` +
      `<td>${verdictLabel[d.verdict]}</td>` +
      `<td class="k">${d.format}</td>`;
    tr.onclick = () => { state.selected = i; renderDocList(); renderDetail(d); };
    tb.appendChild(tr);
  });
}

function renderDetail(d) {
  const el = document.getElementById('detail');
  if (d.error) { el.innerHTML = `<div class="finding err">${escapeHtml(d.error)}</div>`; return; }
  const doc = d.document || {};
  const t = doc.totals || {};
  const terms = [
    ['BT-1', 'Numéro', doc.id],
    ['BT-2', 'Émission', isoDate(doc.issueDate)],
    ['BT-3', 'Type', doc.typeCode],
    ['BT-5', 'Devise', doc.currencyCode],
    ['BT-27', 'Vendeur', doc.seller && doc.seller.name],
    ['BT-31', 'TVA vendeur', doc.seller && doc.seller.vatId],
    ['BT-44', 'Acheteur', doc.buyer && doc.buyer.name],
    ['BT-109', 'Total HT', money(t.taxExclusiveAmount)],
    ['BT-110', 'Total TVA', money(t.taxAmount)],
    ['BT-112', 'Total TTC', money(t.taxInclusiveAmount)],
    ['BT-115', 'Net à payer', money(t.duePayableAmount)],
  ].filter(r => r[2]);

  let html = `<div style="display:flex;align-items:center;gap:12px">` +
    sealHTML(d.verdict, 'stamping') +
    `<div><h2 style="font-size:var(--text-xl)">${escapeHtml(doc.id || d.name)}</h2>` +
    `<div class="k">${verdictLabel[d.verdict]} · ${escapeHtml((doc.seller && doc.seller.name) || '')} → ${escapeHtml((doc.buyer && doc.buyer.name) || '')}</div></div></div>`;
  html += `<div style="margin-top:16px"><div class="k">TERMES</div>`;
  terms.forEach(r => { html += `<div class="termrow"><span>${r[1]} : <b>${escapeHtml(String(r[2]))}</b></span><span class="bt">${r[0]}</span></div>`; });
  html += `</div>`;
  if (d.findings.length) {
    html += `<div style="margin-top:16px"><div class="k">ANOMALIES</div>`;
    d.findings.forEach(f => {
      const cls = f.severity === 'error' || f.severity === 'fatal' ? 'err' : f.severity === 'warning' ? 'warn' : '';
      html += `<div class="finding ${cls}"><span class="rid">${f.ruleId}</span> ${escapeHtml(f.message)}` +
        (f.expected ? `<div class="k">attendu ${escapeHtml(f.expected)} · trouvé ${escapeHtml(f.actual || '')}</div>` : '') + `</div>`;
    });
    html += `</div>`;
  } else {
    html += `<div class="finding" style="border-left-color:var(--koke);margin-top:16px">Aucune anomalie.</div>`;
  }
  el.innerHTML = html;
  // Réanimer le sceau une seule fois (§12.5).
  const s = el.querySelector('.seal.stamping');
  if (s) s.addEventListener('animationend', () => s.classList.remove('stamping'), { once: true });
}

// ---- Rapport groupé par règle ----
function renderReport() {
  const groups = {};
  state.docs.forEach(d => d.findings.forEach(f => {
    (groups[f.ruleId] = groups[f.ruleId] || { rule: f, docs: [] }).docs.push(d.name);
  }));
  const ids = Object.keys(groups).sort((a, b) => {
    const ea = groups[a].rule.severity === 'error', eb = groups[b].rule.severity === 'error';
    return ea === eb ? a.localeCompare(b) : (ea ? -1 : 1);
  });
  const el = document.getElementById('report');
  if (!ids.length) { el.innerHTML = `<div class="finding" style="border-left-color:var(--koke)">Aucune anomalie sur ${state.docs.length} document(s).</div>`; return; }
  el.innerHTML = ids.map(id => {
    const g = groups[id];
    const cls = g.rule.severity === 'error' ? 'err' : g.rule.severity === 'warning' ? 'warn' : '';
    return `<div class="finding ${cls}"><span class="rid">${id}</span> <span class="k">${g.docs.length} document(s) concerné(s)</span>` +
      `<div>${escapeHtml(g.rule.message)}</div><div class="k">${g.docs.map(escapeHtml).join(' ')}</div></div>`;
  }).join('');
}

// ---- Palette de commandes ⌘K (§12.10) ----
const actions = [
  { label: 'Aller à Accueil', run: () => show('genkan') },
  { label: 'Aller à Inspecteur', run: () => show('ken') },
  { label: 'Aller au Rapport', run: () => show('sho') },
  { label: 'Basculer le thème clair / sombre', run: () => themeBtn.click() },
  { label: 'Ouvrir un fichier', run: () => document.getElementById('file').click() },
];
let palette = null;
function openPalette() {
  if (palette) return;
  palette = document.createElement('div');
  palette.className = 'veil';
  palette.innerHTML = `<div class="modal palette"><input placeholder="Commande…" id="pq"><div id="pl"></div></div>`;
  document.body.appendChild(palette);
  const q = palette.querySelector('#pq'), pl = palette.querySelector('#pl');
  let sel = 0, filtered = actions.slice();
  const render = () => {
    pl.innerHTML = filtered.map((a, i) => `<div class="item ${i === sel ? 'on' : ''}">${escapeHtml(a.label)}</div>`).join('');
    pl.querySelectorAll('.item').forEach((it, i) => it.onclick = () => { filtered[i].run(); closePalette(); });
  };
  q.oninput = () => { filtered = actions.filter(a => a.label.toLowerCase().includes(q.value.toLowerCase())); sel = 0; render(); };
  q.onkeydown = e => {
    if (e.key === 'ArrowDown') { sel = Math.min(sel + 1, filtered.length - 1); render(); }
    else if (e.key === 'ArrowUp') { sel = Math.max(sel - 1, 0); render(); }
    else if (e.key === 'Enter' && filtered[sel]) { filtered[sel].run(); closePalette(); }
    else if (e.key === 'Escape') closePalette();
  };
  palette.onclick = e => { if (e.target === palette) closePalette(); };
  render(); q.focus();
}
function closePalette() { if (palette) { palette.remove(); palette = null; } }
document.addEventListener('keydown', e => {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); openPalette(); }
});

function escapeHtml(s) {
  return String(s).replace(/[&<>"]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
}
