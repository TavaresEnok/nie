<script>
  import { onMount } from 'svelte'

  const PAGE_SIZE = 50

  let view = $state('search')
  let query = $state('')
  let results = $state([])
  let searchInfo = $state(null)
  let loading = $state(false)
  let error = $state('')
  let notice = $state('')
  let campaigns = $state([])
  let documents = $state([])
  let edges = $state([])
  let meta = $state({ version: '', web_discovery: false, git_discovery: false, auth_required: false })
  let stats = $state(null)
  let selected = $state(null)
  let selectedID = $state('')
  let doc = $state(null)
  let dossier = $state(null)
  let loadingDossier = $state(false)
  let showForm = $state(false)
  let showSettings = $state(false)
  let token = $state(localStorage.getItem('nie_token') || '')
  let filterCampaign = $state('')
  let form = $state(defaultForm())

  function defaultForm() {
    return {
      query: '',
      seeds: '',
      max_depth: 3,
      max_urls: 200,
      max_domains: 20,
      max_bytes: 1073741824,
      min_domain_interval_seconds: 10,
      use_web_discovery: true,
      use_git_discovery: false
    }
  }

  async function api(path, opts = {}) {
    const headers = { ...(opts.headers || {}) }
    if (token) headers['X-API-Token'] = token
    if (opts.body && !headers['Content-Type']) headers['Content-Type'] = 'application/json'
    const res = await fetch(path, { ...opts, headers })
    const text = await res.text()
    let data = null
    try { data = text ? JSON.parse(text) : null } catch { data = null }
    if (!res.ok) {
      const message = data?.detail || data?.title || `HTTP ${res.status}`
      throw new Error(message)
    }
    return data
  }

  async function refresh() {
    try {
      const [c, d, m, s] = await Promise.all([
        api('/api/campaigns?limit=100'),
        api('/api/documents?limit=100'),
        api('/api/meta'),
        api('/api/stats')
      ])
      campaigns = c.items || []
      documents = d.items || []
      meta = m
      stats = s
      if (!meta.web_discovery) form.use_web_discovery = false
      if (!meta.git_discovery) form.use_git_discovery = false
      if (selectedID) await loadSelected(selectedID, false)
      error = ''
    } catch (e) {
      error = e.message
    }
  }

  async function refreshCampaigns() {
    try {
      const c = await api('/api/campaigns?limit=100')
      campaigns = c.items || []
      if (selectedID) await loadSelected(selectedID, false)
    } catch (e) {
      error = e.message
    }
  }

  async function loadDocuments(campaignID = '') {
    try {
      const r = await api('/api/documents?limit=100' + (campaignID ? `&campaign_id=${campaignID}` : ''))
      documents = r.items || []
    } catch (e) {
      error = e.message
    }
  }

  async function loadSelected(id, showError = true) {
    try {
      const [c, e] = await Promise.all([
        api(`/api/campaigns/${id}`),
        api(`/api/campaigns/${id}/edges?limit=1000`)
      ])
      selected = c
      edges = e.items || []
    } catch (e) {
      if (showError) error = e.message
    }
  }

  async function doSearch() {
    if (query.trim().length < 2) return
    loading = true
    error = ''
    try {
      const suffix = filterCampaign ? `&campaign_id=${encodeURIComponent(filterCampaign)}` : ''
      const r = await api(`/api/search?q=${encodeURIComponent(query)}&limit=40${suffix}`)
      results = r.items || []
      searchInfo = { query: query.trim(), count: results.length }
      view = 'search'
    } catch (e) {
      error = e.message
    } finally {
      loading = false
    }
  }

  async function createCampaign() {
    error = ''
    try {
      const seeds = form.seeds.split(/[\n,]+/).map((s) => s.trim()).filter(Boolean)
      const payload = {
        query: form.query.trim(),
        seed_urls: seeds,
        max_depth: Number(form.max_depth),
        max_urls: Number(form.max_urls),
        max_domains: Number(form.max_domains),
        max_bytes: Number(form.max_bytes),
        min_domain_interval_seconds: Number(form.min_domain_interval_seconds),
        use_web_discovery: !!form.use_web_discovery,
        use_git_discovery: !!form.use_git_discovery
      }
      const c = await api('/api/campaigns', { method: 'POST', body: JSON.stringify(payload) })
      notice = 'Campanha criada e enfileirada.'
      showForm = false
      form = defaultForm()
      if (!meta.web_discovery) form.use_web_discovery = false
      await refreshCampaigns()
      selected = c
      selectedID = c.id
      view = 'campaigns'
    } catch (e) {
      error = e.message
    }
  }

  async function toggleCampaign(c, action) {
    try {
      await api(`/api/campaigns/${c.id}/${action}`, { method: 'POST' })
      notice = action === 'pause' ? 'Campanha pausada.' : 'Campanha retomada.'
      await refreshCampaigns()
    } catch (e) {
      error = e.message
    }
  }

  async function openCampaign(c) {
    selectedID = c.id
    view = 'campaigns'
    await loadSelected(c.id)
  }

  async function openDoc(id) {
    try {
      doc = await api(`/api/documents/${id}`)
    } catch (e) {
      error = e.message
    }
  }

  function closeDoc() {
    doc = null
  }

  async function openDossier(campaignID) {
    loadingDossier = true
    error = ''
    try {
      dossier = await api(`/api/campaigns/${campaignID}/dossier`)
    } catch (e) {
      error = e.message
    } finally {
      loadingDossier = false
    }
  }

  function closeDossier() {
    dossier = null
  }

  async function copyDossier() {
    if (!dossier) return
    try {
      await navigator.clipboard.writeText(dossier.markdown)
      notice = 'Dossiê em Markdown copiado para a área de transferência.'
    } catch {
      error = 'Não foi possível copiar o dossiê.'
    }
  }

  function downloadDossier() {
    if (!dossier) return
    const blob = new Blob([dossier.markdown], { type: 'text/markdown;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `dossie-${(dossier.campaign.id || 'campanha').slice(0, 8)}.md`
    a.click()
    URL.revokeObjectURL(url)
    notice = 'Dossiê baixado com sucesso.'
  }

  function copyMermaid() {
    if (!edges || edges.length === 0) return
    const lines = ['graph TD']
    edges.forEach((e) => {
      let fromName = 'Seed'
      if (e.from_url) {
        try { fromName = new URL(e.from_url).hostname } catch { fromName = e.from_url.slice(0, 25) }
      }
      let toName = e.to_url
      try { toName = new URL(e.to_url).hostname } catch { toName = e.to_url.slice(0, 25) }
      lines.push(`  "${fromName}" -->|n${e.depth}: ${e.kind}| "${toName}"`)
    })
    navigator.clipboard.writeText(lines.join('\n'))
    notice = 'Diagrama Mermaid copiado para a área de transferência!'
  }

  async function copyCitation(p) {
    if (!doc) return
    const url = doc.versions?.[0]?.url || ''
    const text = `"${p.text}"\n\nFonte: ${url}\nSHA-256: ${doc.sha256}${p.page ? `\nPágina: ${p.page}` : ''}`
    try {
      await navigator.clipboard.writeText(text)
      notice = 'Citação copiada.'
    } catch {
      error = 'Não foi possível copiar para a área de transferência.'
    }
  }

  function saveToken() {
    localStorage.setItem('nie_token', token)
    showSettings = false
    notice = 'Token salvo neste navegador.'
    refresh()
  }

  function clearToken() {
    token = ''
    localStorage.removeItem('nie_token')
    notice = 'Token removido.'
    refresh()
  }

  function showLibrary(campaignID = '') {
    filterCampaign = campaignID
    view = 'library'
    loadDocuments(campaignID)
  }

  function formatBytes(n) {
    if (!n) return '0 B'
    const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
    let i = 0
    let v = Number(n)
    while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
    return `${v.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
  }

  function formatDate(value) {
    if (!value) return '—'
    return new Date(value).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })
  }

  function statusLabel(s) {
    return ({ queued: 'Na fila', running: 'Em execução', paused: 'Pausada', completed: 'Concluída', failed: 'Falhou' })[s] || s
  }

  function campaignTotal(c) {
    const s = c.stats || {}
    return (s.jobs_pending || 0) + (s.jobs_running || 0) + (s.jobs_done || 0) + (s.jobs_dead || 0)
  }

  function campaignProgress(c) {
    const total = campaignTotal(c)
    if (!total) return 0
    return Math.round(((c.stats.jobs_done || 0) / total) * 100)
  }

  function matchLabel(m) {
    return ({ exact: 'exata', fts: 'lexical', fuzzy: 'aproximada' })[m] || m
  }

  function passageLabel(p) {
    return p.page ? `página ${p.page} · trecho ${p.ordinal + 1}` : `trecho ${p.ordinal + 1}`
  }

  onMount(() => {
    refresh()
    const timer = setInterval(() => {
      if (campaigns.some((c) => c.status === 'queued' || c.status === 'running')) {
        refreshCampaigns()
      }
    }, 3000)
    return () => clearInterval(timer)
  })
</script>

<header class="topbar">
  <div class="brand" onclick={() => (view = 'search')} role="button" tabindex="0" onkeydown={(e) => e.key === 'Enter' && (view = 'search')}>
    <span class="logo">◈</span>
    <div>
      <strong>NIE</strong>
      <small>Network Intelligence Engine</small>
    </div>
  </div>
  <nav>
    <button class:active={view === 'search'} onclick={() => (view = 'search')}>Busca</button>
    <button class:active={view === 'campaigns'} onclick={() => (view = 'campaigns')}>Campanhas</button>
    <button class:active={view === 'library'} onclick={() => showLibrary(filterCampaign)}>Biblioteca</button>
  </nav>
  <div class="top-actions">
    <span class="version">v{meta.version || '0.1.0'}</span>
    <span class="cap" class:on={meta.web_discovery} title="SearXNG">web</span>
    <span class="cap" class:on={meta.git_discovery} title="GitHub">git</span>
    <button class="ghost small" onclick={() => (showSettings = !showSettings)}>API</button>
  </div>
</header>

{#if showSettings}
  <section class="settings">
    <div>
      <strong>Acesso à API</strong>
      <p class="muted">Se o servidor exige token, informe-o abaixo. Ele fica apenas neste navegador.</p>
    </div>
    <div class="settings-actions">
      <input type="password" bind:value={token} placeholder="NIE_API_TOKEN (opcional)" />
      <button class="primary small" onclick={saveToken}>Salvar</button>
      <button class="ghost small" onclick={clearToken}>Limpar</button>
    </div>
  </section>
{/if}

<main>
  {#if error}
    <div class="banner error" role="alert">
      <span>{error}</span>
      <button class="ghost small" onclick={() => (error = '')}>fechar</button>
    </div>
  {/if}
  {#if notice}
    <div class="banner notice">
      <span>{notice}</span>
      <button class="ghost small" onclick={() => (notice = '')}>fechar</button>
    </div>
  {/if}

  {#if view === 'search'}
    <section class="hero">
      <p class="eyebrow">Investigação técnica</p>
      <h1>Encontre a evidência,<br />não apenas o link.</h1>
      <p class="lede">Busca exata em comandos e trechos, proveniência por URL, página e hash SHA-256.</p>
      <form class="searchbar" onsubmit={(e) => { e.preventDefault(); doSearch() }}>
        <input bind:value={query} placeholder='Ex.: "display bgp peer" NE8000' aria-label="Termo de busca" />
        <button class="primary" disabled={loading}>{loading ? 'Buscando…' : 'Buscar'}</button>
      </form>
      <div class="vendor-chips">
        <span class="chip-label">Filtros Telecom:</span>
        {#each ['Huawei', 'Cisco', 'MikroTik', 'Datacom', 'ZTE', 'BGP', 'PPPoE', 'RADIUS'] as tag}
          <button
            type="button"
            class="chip"
            class:active={query.includes(tag)}
            onclick={() => {
              if (query.includes(tag)) {
                query = query.replace(tag, '').trim()
              } else {
                query = query ? `${query} ${tag}` : tag
              }
              doSearch()
            }}
          >
            {tag}
          </button>
        {/each}
      </div>
      <div class="filters">
        <select bind:value={filterCampaign} aria-label="Filtrar por campanha">
          <option value="">Todas as campanhas</option>
          {#each campaigns as c}
            <option value={c.id}>{c.query}</option>
          {/each}
        </select>
        {#if stats}
          <span class="muted">{stats.documents} documentos · {stats.passages} trechos · {formatBytes(stats.bytes)}</span>
        {/if}
      </div>
    </section>

    {#if searchInfo}
      <div class="section-head">
        <h2>{searchInfo.count} resultado{searchInfo.count === 1 ? '' : 's'}</h2>
        <span class="muted">para “{searchInfo.query}”</span>
      </div>
      {#if results.length === 0}
        <div class="empty">
          <strong>Nenhum trecho encontrado.</strong>
          <p>Crie uma campanha, colete fontes ou ajuste o termo. A busca exata preserva símbolos de CLI.</p>
          <button class="primary small" onclick={() => (view = 'campaigns')}>Ir para campanhas</button>
        </div>
      {:else}
        <div class="results">
          {#each results as r}
            <article class="result">
              <div class="result-head">
                <button class="linklike" onclick={() => openDoc(r.document_id)}>{r.title || r.url}</button>
                <span class:exact={r.match === 'exact'} class="match">{matchLabel(r.match)}</span>
              </div>
              <div class="meta">{r.domain}<span>·</span>{r.page ? `página ${r.page}` : `trecho ${r.ordinal + 1}`}<span>·</span>{formatDate(r.fetched_at)}</div>
              <p class="snippet">{@html r.snippet}</p>
              <div class="result-actions">
                <button class="ghost small" onclick={() => openDoc(r.document_id)}>Abrir documento</button>
                <a class="external" href={r.url} target="_blank" rel="noreferrer">Original ↗</a>
                <span class="hash" title={r.sha256}>{r.sha256.slice(0, 12)}…</span>
              </div>
            </article>
          {/each}
        </div>
      {/if}
    {/if}
  {/if}

  {#if view === 'campaigns'}
    <div class="section-head">
      <div>
        <p class="eyebrow">Descoberta</p>
        <h1>Campanhas</h1>
      </div>
      <div class="head-actions">
        <button class="ghost" onclick={() => showLibrary('')}>Biblioteca</button>
        <button class="primary" onclick={() => (showForm = !showForm)}>{showForm ? 'Fechar' : 'Nova campanha'}</button>
      </div>
    </div>

    {#if showForm}
      <section class="panel form-panel">
        <h2>Nova campanha</h2>
        <form onsubmit={(e) => { e.preventDefault(); createCampaign() }}>
          <label class="wide">
            <span>Pergunta técnica</span>
            <input bind:value={form.query} required placeholder="Ex.: redundância de sessões PPPoE entre NE8000 com failover BGP" />
          </label>
          <label class="wide">
            <span>Seeds (uma URL por linha)</span>
            <textarea bind:value={form.seeds} rows="3" placeholder="https://exemplo.org/documentacao"></textarea>
          </label>
          <label><span>Profundidade</span><input type="number" min="1" max="10" bind:value={form.max_depth} /></label>
          <label><span>Máx. URLs</span><input type="number" min="1" max="5000" bind:value={form.max_urls} /></label>
          <label><span>Máx. domínios</span><input type="number" min="1" max="500" bind:value={form.max_domains} /></label>
          <label><span>Máx. bytes</span><input type="number" min="1048576" step="1048576" bind:value={form.max_bytes} /></label>
          <label><span>Intervalo/domínio (s)</span><input type="number" min="1" max="3600" bind:value={form.min_domain_interval_seconds} /></label>
          <label class="check">
            <input type="checkbox" bind:checked={form.use_web_discovery} disabled={!meta.web_discovery} />
            <span>Descoberta web {meta.web_discovery ? '(SearXNG)' : '(não configurada)'}</span>
          </label>
          <label class="check">
            <input type="checkbox" bind:checked={form.use_git_discovery} disabled={!meta.git_discovery} />
            <span>Descoberta Git {meta.git_discovery ? '(GitHub)' : '(não configurada)'}</span>
          </label>
          <div class="form-actions wide">
            <button class="primary" type="submit">Criar campanha</button>
            <button class="ghost" type="button" onclick={() => (showForm = false)}>Cancelar</button>
          </div>
        </form>
      </section>
    {/if}

    <div class="campaign-layout">
      <div class="campaign-list">
        {#if campaigns.length === 0}
          <div class="empty"><strong>Nenhuma campanha ainda.</strong><p>Crie a primeira para enfileirar seeds e descoberta.</p></div>
        {/if}
        {#each campaigns as c}
          <button class="campaign-card" class:selected={selectedID === c.id} onclick={() => openCampaign(c)}>
            <div class="campaign-top">
              <span class="status {c.status}">{statusLabel(c.status)}</span>
              <span class="muted">{formatDate(c.created_at)}</span>
            </div>
            <strong>{c.query}</strong>
            <div class="campaign-stats">
              <span>{c.stats.documents} docs</span>
              <span>{c.stats.jobs_pending} na fila</span>
              <span>{c.stats.domains} domínios</span>
              <span>{formatBytes(c.stats.bytes)}</span>
            </div>
            <div class="progress"><span style={`width:${campaignProgress(c)}%`}></span></div>
          </button>
        {/each}
      </div>

      {#if selected}
        <aside class="panel campaign-detail">
          <div class="campaign-top">
            <span class="status {selected.status}">{statusLabel(selected.status)}</span>
            <span class="muted">{formatDate(selected.created_at)}</span>
          </div>
          <h2>{selected.query}</h2>
          <div class="stat-grid">
            <div><strong>{selected.stats.documents}</strong><span>documentos</span></div>
            <div><strong>{selected.stats.passages || 0}</strong><span>trechos</span></div>
            <div><strong>{selected.stats.jobs_done}</strong><span>concluídos</span></div>
            <div><strong>{selected.stats.jobs_pending}</strong><span>na fila</span></div>
            <div><strong>{selected.stats.jobs_dead}</strong><span>encerrados</span></div>
            <div><strong>{formatBytes(selected.stats.bytes)}</strong><span>baixados</span></div>
          </div>
          <div class="progress large"><span style={`width:${campaignProgress(selected)}%`}></span></div>
          <div class="detail-actions">
            {#if selected.status === 'running' || selected.status === 'queued'}
              <button class="ghost small" onclick={() => toggleCampaign(selected, 'pause')}>Pausar</button>
            {:else if selected.status === 'paused'}
              <button class="ghost small" onclick={() => toggleCampaign(selected, 'resume')}>Retomar</button>
            {/if}
            <button class="ghost small" onclick={() => showLibrary(selected.id)}>Ver documentos</button>
            <button class="ghost small" onclick={() => { filterCampaign = selected.id; query = selected.query; doSearch() }}>Buscar nesta campanha</button>
            <button class="primary small" disabled={loadingDossier} onclick={() => openDossier(selected.id)}>
              {loadingDossier ? 'Gerando Dossiê…' : 'Exportar Dossiê (.md)'}
            </button>
          </div>
          <p class="muted small">
            Limites: profundidade {selected.budget.max_depth} · {selected.budget.max_urls} URLs ·
            {selected.budget.max_domains} domínios · {formatBytes(selected.budget.max_bytes)}
          </p>
          <div style="display:flex; justify-content:space-between; align-items:center; margin-top:1.4rem;">
            <h3 style="margin:0;">Rastros de descoberta</h3>
            {#if edges.length > 0}
              <button class="ghost tiny" onclick={copyMermaid}>Copiar Grafo Mermaid</button>
            {/if}
          </div>
          {#if edges.length === 0}
            <p class="muted small">Nenhuma aresta registrada ainda.</p>
          {:else}
            <div class="edge-list">
              {#each edges.slice(0, 25) as e}
                <div class="edge">
                  <span class="depth">n{e.depth}</span>
                  <span class="edge-url" title={e.to_url}>{e.to_url}</span>
                  <span class="muted">{e.kind}</span>
                </div>
              {/each}
            </div>
          {/if}
        </aside>
      {/if}
    </div>
  {/if}

  {#if view === 'library'}
    <div class="section-head">
      <div>
        <p class="eyebrow">Acervo</p>
        <h1>Biblioteca</h1>
      </div>
      <div class="head-actions">
        {#if filterCampaign}
          <button class="ghost" onclick={() => showLibrary('')}>Limpar filtro</button>
        {/if}
        <button class="ghost" onclick={() => (view = 'campaigns')}>Campanhas</button>
      </div>
    </div>
    {#if documents.length === 0}
      <div class="empty"><strong>Nenhum documento coletado.</strong><p>O acervo aparece após uma campanha processar as primeiras URLs.</p></div>
    {:else}
      <div class="doc-list">
        {#each documents as d}
          <button class="doc-row" onclick={() => openDoc(d.id)}>
            <div>
              <strong>{d.title || d.url}</strong>
              <div class="meta">{d.domain}<span>·</span>{formatBytes(d.size)}<span>·</span>{formatDate(d.fetched_at)}</div>
            </div>
            <span class="mime">{d.mime.split(';')[0]}</span>
            <span class="hash" title={d.sha256}>{d.sha256.slice(0, 12)}…</span>
          </button>
        {/each}
      </div>
    {/if}
  {/if}
</main>

{#if doc}
  <div class="modal-backdrop" role="presentation" onclick={closeDoc}>
    <div class="modal" role="dialog" aria-modal="true" aria-label="Documento" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()}>
      <div class="modal-head">
        <div>
          <p class="eyebrow">Documento original</p>
          <h2>{doc.title || 'Sem título'}</h2>
          <div class="meta">
            {doc.mime.split(';')[0]}<span>·</span>{formatBytes(doc.size)}<span>·</span>
            {formatDate(doc.created_at)}
          </div>
        </div>
        <button class="ghost small" onclick={closeDoc}>Fechar</button>
      </div>
      <div class="doc-links">
        {#each doc.versions || [] as v}
          <a class="external" href={v.url} target="_blank" rel="noreferrer">{v.url} ↗</a>
        {/each}
        <span class="hash">SHA-256: {doc.sha256}</span>
      </div>
      <div class="passages">
        {#each doc.passages || [] as p}
          <article class="passage">
            <div class="passage-head">
              <span>{passageLabel(p)}</span>
              <button class="ghost tiny" onclick={() => copyCitation(p)}>Copiar citação</button>
            </div>
            <pre>{p.text}</pre>
          </article>
        {/each}
      </div>
    </div>
  </div>
{/if}

{#if dossier}
  <div class="modal-backdrop" role="presentation" onclick={closeDossier}>
    <div class="modal" role="dialog" aria-modal="true" aria-label="Dossiê Técnico" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()}>
      <div class="modal-head">
        <div>
          <p class="eyebrow">Dossiê de Investigação Técnica</p>
          <h2>{dossier.campaign.query}</h2>
          <div class="meta">
            {dossier.documents_count} documentos<span>·</span>
            {dossier.passages_count} trechos<span>·</span>
            {dossier.domains_count} domínios<span>·</span>
            {formatBytes(dossier.bytes_total)}
          </div>
        </div>
        <div style="display:flex; gap:0.5rem; align-items:center;">
          <button class="primary small" onclick={downloadDossier}>Baixar (.md)</button>
          <button class="ghost small" onclick={copyDossier}>Copiar Markdown</button>
          <button class="ghost small" onclick={closeDossier}>Fechar</button>
        </div>
      </div>
      <pre class="dossier-text">{dossier.markdown}</pre>
    </div>
  </div>
{/if}

<footer>
  <span>NIE v{meta.version || '0.1.0'} · MVP local</span>
  <span class="muted">conteúdo externo é tratado como dado não confiável</span>
</footer>
