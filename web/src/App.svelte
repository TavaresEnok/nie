<script>
  import { onMount } from 'svelte'

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
  let meta = $state({
    version: '',
    web_discovery: false,
    git_discovery: false,
    mwmbl_discovery: false,
    wayback_enabled: false,
    software_heritage_discovery: false,
    mentions_discovery: false,
    auth_required: false
  })
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

  // Fase 2 e 3 State
  let deepDossier = $state(null)
  let loadingDeep = $state(false)
  let claimsList = $state([])
  let graphData = $state({ nodes: [], edges: [] })
  let schedules = $state([])
  let alerts = $state([])
  let gitResults = $state([])
  let searchMode = $state('corpus') // 'corpus' or 'git'
  let newSchedule = $state({ name: '', query: '', seeds: '', interval_seconds: 86400 })

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
      use_git_discovery: false,
      use_mwmbl_discovery: true,
      use_swh_discovery: false
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
      if (searchMode === 'git') {
        const r = await api(`/api/search/git?q=${encodeURIComponent(query)}&limit=30`)
        gitResults = r.results || []
        searchInfo = { query: query.trim(), count: gitResults.length, mode: 'git' }
      } else {
        const suffix = filterCampaign ? `&campaign_id=${encodeURIComponent(filterCampaign)}` : ''
        const r = await api(`/api/search?q=${encodeURIComponent(query)}&limit=40${suffix}`)
        results = r.items || []
        searchInfo = { query: query.trim(), count: results.length, mode: 'corpus' }
      }
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
        use_git_discovery: !!form.use_git_discovery,
        use_mwmbl_discovery: !!form.use_mwmbl_discovery,
        use_swh_discovery: !!form.use_swh_discovery
      }
      const c = await api('/api/campaigns', { method: 'POST', body: JSON.stringify(payload) })
      notice = 'Campanha criada e enfileirada com sucesso.'
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

  // Fase 2: Investigação Profunda com IA / Regras
  async function runInvestigation(campaignID) {
    loadingDeep = true
    error = ''
    try {
      const data = await api(`/api/campaigns/${campaignID}/investigate`, { method: 'POST' })
      deepDossier = data
      claimsList = data.claims || []
      view = 'investigate'
      notice = 'Investigação profunda e cruzamento de evidências concluídos.'
    } catch (e) {
      error = e.message
    } finally {
      loadingDeep = false
    }
  }

  // Fase 3: Grafo Visual de Descoberta
  async function loadGraph(campaignID) {
    error = ''
    try {
      const data = await api(`/api/campaigns/${campaignID}/graph`)
      graphData = data
      selectedID = campaignID
      view = 'graph'
    } catch (e) {
      error = e.message
    }
  }

  // Fase 3: Agendamentos e Alertas
  async function openSchedules() {
    error = ''
    try {
      const [s, a] = await Promise.all([
        api('/api/schedules'),
        api('/api/alerts')
      ])
      schedules = s.schedules || []
      alerts = a.alerts || []
      view = 'schedules'
    } catch (e) {
      error = e.message
    }
  }

  async function saveSchedule() {
    if (!newSchedule.query || !newSchedule.name) return
    try {
      const seeds = newSchedule.seeds.split(/[\n,]+/).map((s) => s.trim()).filter(Boolean)
      await api('/api/schedules', {
        method: 'POST',
        body: JSON.stringify({
          name: newSchedule.name,
          query: newSchedule.query,
          seed_urls: seeds,
          interval_seconds: Number(newSchedule.interval_seconds) || 86400,
          max_depth: 3,
          max_urls: 150,
          max_domains: 15
        })
      })
      notice = 'Agendamento periódico criado.'
      newSchedule = { name: '', query: '', seeds: '', interval_seconds: 86400 }
      await openSchedules()
    } catch (e) {
      error = e.message
    }
  }

  async function ackAlert(id) {
    try {
      await api(`/api/alerts/${id}/ack`, { method: 'POST' })
      alerts = alerts.filter(a => a.id !== id)
      notice = 'Alerta confirmado.'
    } catch (e) {
      error = e.message
    }
  }

  async function copyDossierText(md) {
    try {
      await navigator.clipboard.writeText(md)
      notice = 'Dossiê em Markdown copiado com sucesso!'
    } catch {
      error = 'Não foi possível copiar.'
    }
  }

  function downloadDossierFile(md, id) {
    const blob = new Blob([md], { type: 'text/markdown;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `dossie-investigacao-${(id || 'campanha').slice(0, 8)}.md`
    a.click()
    URL.revokeObjectURL(url)
    notice = 'Arquivo de dossiê baixado.'
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
    <button class:active={view === 'investigate'} onclick={() => { if (selectedID) runInvestigation(selectedID); else view = 'investigate' }}>Investigação & Dossiês</button>
    <button class:active={view === 'graph'} onclick={() => { if (selectedID) loadGraph(selectedID); else view = 'graph' }}>Grafo de Descobertas</button>
    <button class:active={view === 'schedules'} onclick={openSchedules}>Autonomia & Agendamentos</button>
    <button class:active={view === 'library'} onclick={() => showLibrary(filterCampaign)}>Biblioteca</button>
  </nav>
  <div class="top-actions">
    <span class="version">v{meta.version || '0.2.1'}</span>
    <span class="cap" class:on={meta.web_discovery} title="SearXNG (Descoberta Web)">searxng</span>
    <span class="cap" class:on={meta.git_discovery} title="GitHub (Código)">git</span>
    <span class="cap" class:on={meta.mwmbl_discovery} title="Mwmbl (Índice Livre)">mwmbl</span>
    <span class="cap" class:on={meta.wayback_enabled} title="Wayback Machine (Preservação 404)">wayback</span>
    <span class="cap" class:on={meta.mentions_discovery} title="Descoberta Contínua por Menções">menções</span>
    <button class="ghost small" onclick={() => (showSettings = !showSettings)}>API</button>
  </div>
</header>

{#if showSettings}
  <section class="settings">
    <div>
      <strong>Acesso à API</strong>
      <p class="muted">Se o servidor exige token, informe-o abaixo. Ele fica salvo neste navegador.</p>
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

  <!-- TAB: BUSCA -->
  {#if view === 'search'}
    <section class="hero">
      <p class="eyebrow">Investigação técnica multivendor</p>
      <h1>Encontre a evidência,<br />não apenas o link.</h1>
      <p class="lede">Busca exata em comandos CLI, proveniência por URL, página, cluster de origem e SHA-256.</p>
      
      <div style="display:flex; justify-content:center; gap:0.5rem; margin-bottom:1rem;">
        <button class="ghost small" class:active={searchMode === 'corpus'} onclick={() => (searchMode = 'corpus')}>Corpus Documental (Web/PDF)</button>
        <button class="ghost small" class:active={searchMode === 'git'} onclick={() => (searchMode = 'git')}>Git / Configurações Locais</button>
      </div>

      <form class="searchbar" onsubmit={(e) => { e.preventDefault(); doSearch() }}>
        <input bind:value={query} placeholder={searchMode === 'git' ? 'Ex.: "router bgp" ou "policy-based"' : 'Ex.: "display bgp peer" NE8000 ou "routing filter"'} aria-label="Termo de busca" />
        <button class="primary" disabled={loading}>{loading ? 'Buscando…' : 'Buscar'}</button>
      </form>

      {#if searchMode === 'corpus'}
        <div class="vendor-chips">
          <span class="chip-label">Filtros Telecom:</span>
          {#each ['Huawei', 'Cisco', 'MikroTik', 'Datacom', 'ZTE', 'FiberHome', 'VSOL', 'SonicWall', 'BGP', 'PPPoE', 'RADIUS', 'PBR'] as tag}
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
      {/if}
    </section>

    {#if searchInfo}
      <div class="section-head">
        <h2>{searchInfo.count} resultado{searchInfo.count === 1 ? '' : 's'}</h2>
        <span class="muted">para “{searchInfo.query}” ({searchInfo.mode === 'git' ? 'Git local' : 'Corpus'})</span>
      </div>

      {#if searchInfo.mode === 'git'}
        <div class="results">
          {#each gitResults as gr}
            <article class="result">
              <div class="result-head">
                <strong>{gr.path}</strong>
                <span class="match exact">Git Local</span>
              </div>
              <pre class="snippet">{gr.snippet}</pre>
            </article>
          {/each}
        </div>
      {:else}
        {#if results.length === 0}
          <div class="empty">
            <strong>Nenhum trecho encontrado.</strong>
            <p>Crie uma campanha de investigação para coletar fontes ou ajuste o termo. A busca preserva pontuação de CLI.</p>
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
  {/if}

  <!-- TAB: CAMPANHAS -->
  {#if view === 'campaigns'}
    <div class="section-head">
      <div>
        <p class="eyebrow">Descoberta autônoma</p>
        <h1>Campanhas</h1>
      </div>
      <div class="head-actions">
        <button class="ghost" onclick={() => showLibrary('')}>Biblioteca</button>
        <button class="primary" onclick={() => (showForm = !showForm)}>{showForm ? 'Fechar' : 'Nova campanha'}</button>
      </div>
    </div>

    {#if showForm}
      <section class="panel form-panel">
        <h2>Nova campanha de investigação</h2>
        <form onsubmit={(e) => { e.preventDefault(); createCampaign() }}>
          <label class="wide">
            <span>Pergunta técnica / Problema de rede</span>
            <input bind:value={form.query} required placeholder="Ex.: redundância BGP BNG no Huawei NE8000 com failover de PPPoE" />
          </label>
          <label class="wide">
            <span>Seeds iniciais (uma URL por linha)</span>
            <textarea bind:value={form.seeds} rows="3" placeholder="https://support.huawei.com&#10;https://forum.huawei.com"></textarea>
          </label>
          <label><span>Profundidade (níveis)</span><input type="number" min="1" max="10" bind:value={form.max_depth} /></label>
          <label><span>Máx. URLs</span><input type="number" min="1" max="5000" bind:value={form.max_urls} /></label>
          <label><span>Máx. domínios</span><input type="number" min="1" max="500" bind:value={form.max_domains} /></label>
          <label><span>Máx. bytes</span><input type="number" min="1048576" step="1048576" bind:value={form.max_bytes} /></label>
          <label><span>Intervalo/domínio (s)</span><input type="number" min="1" max="3600" bind:value={form.min_domain_interval_seconds} /></label>
          <label class="check">
            <input type="checkbox" bind:checked={form.use_web_discovery} disabled={!meta.web_discovery} />
            <span>Descoberta web externa {meta.web_discovery ? '(SearXNG)' : '(desativada)'}</span>
          </label>
          <label class="check">
            <input type="checkbox" bind:checked={form.use_git_discovery} disabled={!meta.git_discovery} />
            <span>Descoberta Git {meta.git_discovery ? '(GitHub)' : '(desativada)'}</span>
          </label>
          <label class="check">
            <input type="checkbox" bind:checked={form.use_mwmbl_discovery} disabled={!meta.mwmbl_discovery} />
            <span>Índice livre independente {meta.mwmbl_discovery ? '(Mwmbl)' : '(desativado)'}</span>
          </label>
          <label class="check">
            <input type="checkbox" bind:checked={form.use_swh_discovery} disabled={!meta.software_heritage_discovery} />
            <span>Acervo de código histórico {meta.software_heritage_discovery ? '(Software Heritage)' : '(desativado)'}</span>
          </label>
          <div class="form-actions wide">
            <button class="primary" type="submit">Iniciar campanha</button>
            <button class="ghost" type="button" onclick={() => (showForm = false)}>Cancelar</button>
          </div>
        </form>
      </section>
    {/if}

    <div class="campaign-layout">
      <div class="campaign-list">
        {#if campaigns.length === 0}
          <div class="empty"><strong>Nenhuma campanha criada.</strong><p>Inicie a primeira para enfileirar seeds e expansão recursiva.</p></div>
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
          
          <div class="detail-actions" style="margin-top:1.2rem; display:flex; flex-wrap:wrap; gap:0.5rem;">
            {#if selected.status === 'running' || selected.status === 'queued'}
              <button class="ghost small" onclick={() => toggleCampaign(selected, 'pause')}>Pausar</button>
            {:else if selected.status === 'paused'}
              <button class="ghost small" onclick={() => toggleCampaign(selected, 'resume')}>Retomar</button>
            {/if}
            <button class="primary small" disabled={loadingDeep} onclick={() => runInvestigation(selected.id)}>
              {loadingDeep ? 'Investigando…' : '⚡ Investigação & Dossiê'}
            </button>
            <button class="ghost small" onclick={() => loadGraph(selected.id)}>Ver Grafo ◈</button>
            <button class="ghost small" onclick={() => showLibrary(selected.id)}>Documentos</button>
            <button class="ghost small" disabled={loadingDossier} onclick={() => openDossier(selected.id)}>
              {loadingDossier ? 'Gerando…' : 'Dossiê Rápido (.md)'}
            </button>
          </div>
          <p class="muted small" style="margin-top:0.8rem;">
            Limites: profundidade {selected.budget.max_depth} · {selected.budget.max_urls} URLs · {selected.budget.max_domains} domínios
          </p>
        </aside>
      {/if}
    </div>
  {/if}

  <!-- TAB: INVESTIGAÇÃO & DOSSIÊS (FASE 2) -->
  {#if view === 'investigate'}
    <div class="section-head">
      <div>
        <p class="eyebrow">Fase 2 — Evidências e Auditoria</p>
        <h1>Investigação Profunda & Dossiê Técnico</h1>
      </div>
      {#if deepDossier}
        <div class="head-actions">
          <button class="ghost" onclick={() => copyDossierText(deepDossier.markdown)}>Copiar Markdown</button>
          <button class="primary" onclick={() => downloadDossierFile(deepDossier.markdown, deepDossier.campaign.id)}>Baixar Dossiê (.md)</button>
        </div>
      {/if}
    </div>

    {#if !deepDossier}
      <div class="empty">
        <strong>Nenhum dossiê selecionado.</strong>
        <p>Selecione uma campanha e clique em "⚡ Investigação & Dossiê" para extrair claims e confrontar evidências.</p>
        <button class="primary small" onclick={() => (view = 'campaigns')}>Ir para campanhas</button>
      </div>
    {:else}
      <section class="panel" style="margin-bottom:1.5rem;">
        <h2>{deepDossier.campaign.query}</h2>
        <p class="lede">{deepDossier.summary}</p>
        
        {#if Object.keys(deepDossier.compatibility_matrix || {}).length > 0}
          <h3>Matriz de Fabricantes e Firmware</h3>
          <table class="compat-table">
            <thead>
              <tr><th>Fabricante</th><th>Sistema Operacional / Release</th></tr>
            </thead>
            <tbody>
              {#each Object.entries(deepDossier.compatibility_matrix) as [vendor, os]}
                <tr><td><strong>{vendor}</strong></td><td>{os}</td></tr>
              {/each}
            </tbody>
          </table>
        {/if}

        {#if (deepDossier.cli_commands || []).length > 0}
          <h3>Comandos CLI Canônicos Identificados</h3>
          <pre class="dossier-text" style="background:#05070a; border:1px solid var(--line); border-radius:var(--radius-sm); padding:1rem; color:#7ee787;">
{#each deepDossier.cli_commands as cmd}{cmd}
{/each}
          </pre>
        {/if}
      </section>

      <section>
        <h2>Claims Auditados e Origin Clusters</h2>
        <p class="muted">Três cópias do mesmo artigo contam como uma única origem para evitar falso consenso.</p>
        
        <div class="claims-grid">
          {#each claimsList as c}
            <div class="claim-card">
              <div class="claim-header">
                <span class="claim-title">{c.title}</span>
                <span class="claim-badge {c.status}">{c.status}</span>
              </div>
              <p style="margin:0 0 0.5rem; color:var(--text-dim);">{c.statement}</p>
              <div class="meta">
                <span>Aplicabilidade: <strong>{c.applicability}</strong></span>
                <span>·</span>
                <span>Validação Operacional: <strong>{c.operational_validation}</strong></span>
                {#if c.vendor_name}
                  <span>·</span><span>Fabricante: {c.vendor_name}</span>
                {/if}
              </div>
              <div class="evidence-list">
                <strong>Evidências Rastreadas ({c.evidences?.length || 0}):</strong>
                {#each c.evidences || [] as ev}
                  <div class="evidence-item">
                    <blockquote>"{ev.excerpt}"</blockquote>
                    <div class="meta">
                      <span class="stance-pill" style="color:var(--accent);">[{ev.stance.toUpperCase()}]</span>
                      {#if ev.is_official_doc}
                        <span style="color:var(--blue); font-weight:600;">DOC OFICIAL</span>
                      {/if}
                      {#if ev.page}<span>Pág. {ev.page}</span>{/if}
                      <span>{ev.notes}</span>
                    </div>
                  </div>
                {/each}
              </div>
            </div>
          {/each}
        </div>
      </section>

      <section class="panel" style="margin-top:1.5rem;">
        <h3>Procedimento de Rollback e Contingência</h3>
        <pre class="dossier-text" style="white-space:pre-wrap; background:transparent; padding:0;">{deepDossier.rollback_plan}</pre>
      </section>
    {/if}
  {/if}

  <!-- TAB: GRAFO DE DESCOBERTAS (FASE 3) -->
  {#if view === 'graph'}
    <div class="section-head">
      <div>
        <p class="eyebrow">Fase 3 — Autonomia e Ramificações</p>
        <h1>Grafo Visual de Descoberta ("Água & Eletricidade")</h1>
      </div>
      <div class="head-actions">
        <select bind:value={selectedID} onchange={() => loadGraph(selectedID)}>
          <option value="">Selecione uma campanha</option>
          {#each campaigns as c}
            <option value={c.id}>{c.query}</option>
          {/each}
        </select>
      </div>
    </div>

    {#if graphData.nodes.length === 0}
      <div class="empty">
        <strong>Nenhum grafo carregado.</strong>
        <p>Selecione uma campanha para inspecionar a árvore de ramificações e links seguidos.</p>
      </div>
    {:else}
      <div class="graph-container">
        <div class="panel">
          <div class="meta" style="margin-bottom:1rem;">
            <span>{graphData.nodes.length} nós descobertos</span>
            <span>·</span>
            <span>{graphData.edges.length} conexões e ramificações</span>
          </div>
          <div class="graph-tree">
            {#each graphData.nodes as node}
              <div class="tree-node depth-{node.depth}" class:mention-node={node.type === 'mention'}>
                <span class="depth">n{node.depth}</span>
                <span class="cap on" class:mention-cap={node.type === 'mention'}>{node.type === 'mention' ? 'menção técnica' : node.type}</span>
                <strong>{node.label}</strong>
                <span class="muted small">{node.domain}</span>
                <span class="hash">{node.id}</span>
              </div>
            {/each}
          </div>
        </div>
      </div>
    {/if}
  {/if}

  <!-- TAB: AUTONOMIA & AGENDAMENTOS (FASE 3) -->
  {#if view === 'schedules'}
    <div class="section-head">
      <div>
        <p class="eyebrow">Fase 3 — Autonomia Contínua</p>
        <h1>Agendamentos Recorrentes & Alertas de Documentação</h1>
      </div>
      <div class="head-actions">
        <button class="primary" onclick={openSchedules}>Atualizar</button>
      </div>
    </div>

    <section class="panel" style="margin-bottom:1.5rem;">
      <h2>Criar Agendamento Periódico</h2>
      <form onsubmit={(e) => { e.preventDefault(); saveSchedule() }} style="display:grid; grid-template-columns:1fr 1fr; gap:1rem;">
        <label>
          <span>Nome da Rotina</span>
          <input bind:value={newSchedule.name} required placeholder="Ex.: Monitor de BGP Huawei NE8000" />
        </label>
        <label>
          <span>Consulta Técnica</span>
          <input bind:value={newSchedule.query} required placeholder="Ex.: Huawei NE8000 VRP v8 BGP BNG" />
        </label>
        <label class="wide" style="grid-column:1 / -1;">
          <span>Seeds Iniciais (URLs)</span>
          <textarea bind:value={newSchedule.seeds} rows="2" placeholder="https://support.huawei.com"></textarea>
        </label>
        <label>
          <span>Intervalo de Execução</span>
          <select bind:value={newSchedule.interval_seconds}>
            <option value={3600}>A cada 1 hora</option>
            <option value={21600}>A cada 6 horas</option>
            <option value={86400}>Diariamente (24h)</option>
            <option value={604800}>Semanalmente</option>
          </select>
        </label>
        <div style="display:flex; align-items:flex-end;">
          <button class="primary" type="submit">Salvar Agendamento</button>
        </div>
      </form>
    </section>

    <h2>Alertas de Alterações e Mudanças de Firmware ({alerts.length})</h2>
    {#if alerts.length === 0}
      <p class="muted">Nenhum alerta pendente no momento.</p>
    {:else}
      <div class="alerts-panel">
        {#each alerts as a}
          <div class="alert-row {a.severity}">
            <div>
              <strong>{a.title}</strong>
              <p style="margin:0.2rem 0; color:var(--text-dim);">{a.message}</p>
              <span class="meta">{formatDate(a.created_at)} · Severidade: {a.severity}</span>
            </div>
            {#if !a.acknowledged}
              <button class="ghost small" onclick={() => ackAlert(a.id)}>Confirmar Ciência</button>
            {/if}
          </div>
        {/each}
      </div>
    {/if}

    <h2 style="margin-top:2rem;">Agendamentos Ativos ({schedules.length})</h2>
    <div class="schedules-list">
      {#each schedules as s}
        <div class="schedule-card">
          <div>
            <strong>{s.name}</strong>
            <p style="margin:0.4rem 0; color:var(--text-dim); font-size:0.9rem;">Query: "{s.query}"</p>
            <div class="meta">
              <span>Intervalo: {s.interval_seconds / 3600}h</span>
              <span>·</span>
              <span>Próximo disparo: {formatDate(s.next_run_at)}</span>
            </div>
          </div>
          <div style="margin-top:1rem;">
            <span class="status {s.enabled ? 'running' : 'paused'}">{s.enabled ? 'Ativo' : 'Desativado'}</span>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  <!-- TAB: BIBLIOTECA -->
  {#if view === 'library'}
    <div class="section-head">
      <div>
        <p class="eyebrow">Acervo imutável</p>
        <h1>Biblioteca de Documentos</h1>
      </div>
      <div class="head-actions">
        {#if filterCampaign}
          <button class="ghost" onclick={() => showLibrary('')}>Limpar filtro</button>
        {/if}
        <button class="ghost" onclick={() => (view = 'campaigns')}>Campanhas</button>
      </div>
    </div>
    {#if documents.length === 0}
      <div class="empty"><strong>Nenhum documento arquivado.</strong><p>O acervo é preenchido automaticamente pelas campanhas.</p></div>
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

<!-- MODAL: DOCUMENTO -->
{#if doc}
  <div class="modal-backdrop" role="presentation" onclick={closeDoc}>
    <div class="modal" role="dialog" aria-modal="true" aria-label="Documento" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()}>
      <div class="modal-head">
        <div>
          <p class="eyebrow">Documento original preservado</p>
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
            </div>
            <pre>{p.text}</pre>
          </article>
        {/each}
      </div>
    </div>
  </div>
{/if}

<!-- MODAL: DOSSIÊ RÁPIDO -->
{#if dossier}
  <div class="modal-backdrop" role="presentation" onclick={closeDossier}>
    <div class="modal" role="dialog" aria-modal="true" aria-label="Dossiê Técnico" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()}>
      <div class="modal-head">
        <div>
          <p class="eyebrow">Dossiê Técnico Rápido</p>
          <h2>{dossier.campaign.query}</h2>
          <div class="meta">
            {dossier.documents_count} documentos<span>·</span>
            {dossier.passages_count} trechos<span>·</span>
            {dossier.domains_count} domínios<span>·</span>
            {formatBytes(dossier.bytes_total)}
          </div>
        </div>
        <div style="display:flex; gap:0.5rem; align-items:center;">
          <button class="primary small" onclick={() => downloadDossierFile(dossier.markdown, dossier.campaign.id)}>Baixar (.md)</button>
          <button class="ghost small" onclick={() => copyDossierText(dossier.markdown)}>Copiar Markdown</button>
          <button class="ghost small" onclick={closeDossier}>Fechar</button>
        </div>
      </div>
      <pre class="dossier-text">{dossier.markdown}</pre>
    </div>
  </div>
{/if}

<footer>
  <span>NIE v{meta.version || '0.2.0'} · Network Intelligence Engine</span>
  <span class="muted">Plataforma autônoma de pesquisa técnica multivendor</span>
</footer>
