# NETWORK INTELLIGENCE ENGINE (NIE)
## Plano Mestre V3 — Visão integral, capacidades, arquitetura, ecossistema open source e excelência técnica

**Data de consolidação:** 2 de novembro de 2026  
**Natureza:** documento único de visão de produto e arquitetura-alvo, com diagnóstico de aderência da implementação existente.  
**Base histórica:** `ARQUIVO_MESTRE.md` (30/10/2026), Plano Mestre V2, Adendo Mestre V2.1, catálogo histórico de 59 registros, documentação do repositório e auditorias críticas até a V6.  
**Base do diagnóstico de código:** último pacote inspecionado, **`nie-main (4).zip`**, acompanhado da `AUDITORIA_CRITICA_NIE_V6_2026-11-02.md`. O pacote não contém o histórico `.git`. Alterações posteriores não estão avaliadas aqui.  
**Situação:** projeto em desenvolvimento. A visão abaixo descreve **o produto desejado**; as seções de diagnóstico distinguem explicitamente o que já existe do que ainda é meta.  
**Regra de precedência:** este documento consolida a visão e resolve propostas arquiteturais conflitantes para fins de planejamento. Não apaga o histórico, não reclassifica integrações experimentais como concluídas e não dispensa conferir o código e as licenças nas versões realmente instaladas.

### Acompanhamento de execução — atualização posterior à auditoria V6

**Legenda:** ✅ executado no escopo descrito; 🟡 parcialmente executado; ⬜ pendente de implementação ou de comprovação. Uma marca verde confirma somente o item e o teste mencionados, não o aceite integral de um marco. As descrições da V6 abaixo permanecem como histórico.

| Situação | Item executado ou pendente | Comprovação e limite |
|---|---|---|
| ✅ | Avaliação automática e busca derivada de lacunas | Fluxo testado com banco e fonte HTTP controlada; planejamento e suficiência continuam limitados por regras. |
| ✅ | Histórico e controle de rodadas | Decisões, consultas e motivos de parada registrados e exibidos na campanha. |
| ✅ | Geração de lease e proteção contra worker substituído | Testes com PostgreSQL 18 para renovação, expiração, retomada e cancelamento de requisição. |
| ✅ | Extrações versionadas e trechos históricos | Migração de dados legados, reprocessamento e preservação dos IDs de trechos testados em PostgreSQL 18. |
| ✅ | Download do original com verificação de integridade | API autenticável verifica o SHA-256 antes de entregar ou reprocessar o original. |
| ✅ | Histórico de claims e persistência de contradições | Revisões, evidências e contradições preservadas e testadas com banco real. |
| ✅ | OCR opcional para PDF escaneado | Testado com PDF sintético sem camada de texto, Tesseract e Poppler; ainda falta avaliar manuais reais e isolar o parser. |
| 🟡 | Revisita e mudança de documento | Hash, ETag e 304 ficam registrados; a mudança gera alerta e pode reabrir a avaliação. Faltam diff textual, intervalo operacional e ensaio público. |
| ✅ | Backup e restauração conjunta com Restic | Ensaio isolado em PostgreSQL 18 restaurou catálogo e objetos, com verificação de conteúdo e hashes; faltam metas e ensaios operacionais de escala. |
| ✅ | Testes com PostgreSQL 18.6 | Store, API, worker e backup/restauração testados em instância isolada; implantação Docker completa ainda não validada. |
| 🟡 | Qualidade factual e pesquisa autônoma | Regressões adversariais V6 passam. Subperguntas determinísticas entram no ciclo quando não há lacuna de claim; a cobertura por termos não certifica resposta. Faltam avaliação factual abrangente, LLM no planejamento e campanhas públicas completas. |
| ⬜ | Aceite integral dos marcos A–F | Permanecem pendentes os critérios de cada marco e as capacidades não marcadas acima. |

Detalhes, testes e limitações: [registro de execução](EXECUCAO_PLANO_V3.md) e [backup e restauração](BACKUP_E_RESTAURACAO.md).

---

# PARTE I — O PRODUTO QUE QUEREMOS CONSTRUIR

## 1. Definição fundamental: o que é o NIE?

O **Network Intelligence Engine (NIE)** é uma plataforma **auto-hospedável, modular, extensível, multifuente e orientada a evidências** para investigação autônoma na internet e em bases de conhecimento próprias. Recebe um problema, assunto ou hipótese; encontra lugares relevantes para pesquisar sem depender de uma lista fixa de URLs; segue links, referências e menções; formula novas perguntas; coleta e preserva conteúdo autorizado; cruza fontes; identifica contradições; e produz resultados que qualquer pessoa possa conferir nas fontes originais.

O NIE **não deve ser apenas**:

- um mecanismo de metabusca que retorna links;
- um web scraper que baixa HTML;
- um chatbot que resume os dez primeiros resultados;
- um RAG que responde somente a partir de PDFs previamente enviados;
- uma coleção de projetos Docker sem integração coerente;
- um banco de terabytes de arquivos que ninguém consegue encontrar ou avaliar.

**Definição curta:** *um sistema que investiga autonomamente, constrói uma biblioteca acumulativa e transforma fontes heterogêneas em conhecimento rastreável, sem confundir descoberta, hipótese, documentação e comprovação operacional.*

**Princípio do usuário:** *“Quero definir o assunto e deixar o sistema descobrir sozinho onde estão as evidências.”*

A plataforma poderá pesquisar assuntos gerais — ciência, software, história, tecnologia, normas, indústria, segurança defensiva, documentação pública etc. — por meio de um núcleo genérico. **Redes e telecomunicações** são o primeiro domínio especializado e o principal conjunto inicial de testes, não uma restrição definitiva da arquitetura.

### 1.1 Missão

Reduzir o custo e a dificuldade de encontrar informações raras, dispersas, multilíngues, históricas ou contraditórias, preservando **origem, contexto, limites de validade e evidência auditável**.

### 1.2 Proposta de valor

1. **Descoberta além dos primeiros resultados:** utilizar buscadores externos, índices independentes, acervos históricos e expansão por links.
2. **Investigação que se aprofunda:** não parar na primeira resposta; perseguir referências, versões e lacunas quando isso agrega conhecimento.
3. **Memória que cresce:** reutilizar documentos e conexões já encontrados em novas pesquisas.
4. **Conhecimento com lastro:** cada alegação importante aponta para documento, versão, trecho, data e contexto.
5. **Controle real:** campanhas observáveis, orçamentos, pausa/retomada, autorização de fontes e interruptor de emergência.
6. **Independência tecnológica:** funcionamento local e substituição de buscadores, crawlers, modelos, parsers e índices sem perder os dados do NIE.
7. **Especialização técnica profunda:** para telecom, diferenciar comandos, funcionalidades, protocolos, modelo, sistema operacional, release, patch, licença e condições operacionais.

### 1.3 O que a plataforma nunca pode prometer

- Não terá acesso a **100% da internet**, a todos os repositórios Git, a todos os serviços `.onion` ou a conteúdo privado sem autorização.
- Não transformará automaticamente uma ocorrência textual em verdade factual.
- Não garantirá que um comando documentado funcione em qualquer firmware ou topologia.
- Não precisará copiar toda a web ou todo o Git público para fazer pesquisas extensas.
- Não contornará autenticação, proibições de acesso, CAPTCHA, restrições contratuais ou proteção de dados para ampliar cobertura.
- Não executará comandos CLI descobertos em equipamentos de produção como consequência de uma pesquisa.
- Não usará um grande número de serviços instalados como indicador de sucesso.

## 2. Como será a experiência ideal do usuário

A experiência inicial deve ser simples: **uma caixa de pesquisa, objetivos claros e controles visuais avançados quando necessários**. O usuário poderá informar apenas um assunto; filtros, URLs-semente e versões específicas serão opcionais.

### 2.1 Modos de uso

| Modo | O que a pessoa faz | O que o NIE entrega |
|---|---|---|
| **Busca rápida** | Digita uma pergunta, expressão exata, mensagem de erro ou comando CLI | Resultados unificados de acervo, web, Git e fontes habilitadas, com filtros e origens |
| **Pesquisa profunda** | Define um problema complexo | Plano de subperguntas, coleta em etapas, comparação de evidências, lacunas e dossiê citável |
| **Investigação autônoma** | Autoriza um objetivo e orçamento | Várias rodadas reais de descoberta → análise → novas buscas → reavaliação, até critério de parada |
| **Monitoramento contínuo** | Seleciona assuntos/fontes para acompanhar | Revisitas, mudanças documentais, novos manuais/releases, alertas e histórico de alterações |
| **Biblioteca permanente** | Consulta tudo que já foi autorizado a preservar | Busca exata, textual, metadados, semântica opcional, documentos e versões antigas |
| **Comparador** | Seleciona versões, produtos, soluções ou hipóteses | Diferenças documentadas, condições de aplicabilidade, conflitos e ausências de evidência |
| **Grafo de descobertas** | Explora como o sistema encontrou as fontes | Árvore/grafo de links, menções, citações, repositórios e dependências entre fontes |
| **Auditoria de evidências** | Clica em uma afirmação do relatório | Documento exato, trecho, origem, independência, suporte/refutação, limites e estado de validação |
| **Administração e políticas** | Configura limites e permissões | Fila, fontes, custos, workers, modelos de IA, retenção, restauração, logs e incidentes |

### 2.2 O que deverá aparecer no painel de uma campanha

- Pergunta original preservada sem alterações silenciosas; objetivo da pesquisa e restrições conhecidas.
- Subperguntas em execução, concluídas, descartadas ou em revisão.
- Fontes externas consultadas; cobertura por tipo, idioma, domínio, fabricante e modelo.
- URL-semente e **proveniência de cada novo caminho**; motivo pelo qual o caminho foi priorizado.
- Estado de crawling: aguardando, processando, bloqueado por política, erro transitório, coletado, deduplicado ou descartado.
- Links e menções encontrados em cada página, README, PDF, discussão ou índice.
- Volume baixado, volume único armazenado, orçamento reservado, tempo, requisições, tokens e custo de modelos.
- Evidências favoráveis, contrárias, indiretas, não verificadas, cópias e grupos de origem.
- Documentos originais abríveis, versões, trechos com destaque e relações com cada conclusão.
- Pausar, retomar, cancelar, mudar prioridades, limitar fontes e revisar decisões sem editar código.
- Dossiê exportável com **o que sabemos, o que é hipótese e o que permanece desconhecido**.

### 2.3 Quatro cenários ilustrativos

**A — Telecom: Huawei NE8000 como BNG/BRAS.** O usuário investiga redundância PPPoE usando BGP e RADIUS. O NIE separa disponibilidade de roteamento, autenticação RADIUS, estado do BNG e preservação de sessões PPPoE; procura versões VRP, manuais, casos de fórum, RFCs e configurações; distingue recursos correlatos que não provam a solução completa.

**B — Configuração rara de OLT.** O usuário informa fabricante, modelo, firmware e tarefa de provisionamento. O sistema encontra manual e configuração de release compatível; rejeita comandos que só aparecem em outra família; aponta dependências, permissões e necessidade de teste autorizado.

**C — Documento desaparecido.** O usuário procura um manual antigo cujo URL mudou. O NIE encontra referências em fóruns e READMEs, consulta Wayback e outros índices, compara versões e registra que a cópia histórica pode estar incompleta ou não ter licença de redistribuição.

**D — Assunto fora de telecom.** O usuário pesquisa a evolução histórica de uma biblioteca open source, uma norma industrial ou um tema científico. O mesmo motor genérico encontra repositórios, artigos e documentos, acompanha citações e gera relatório sem depender de vocabulário fixo de redes.

## 3. O diferencial: a árvore de descobertas que realmente cresce

A ramificação de links **já existe em softwares maduros**. Nosso produto não deve reinventar isso. Sua contribuição própria é conectar caminhos heterogêneos, controlar o que merece aprofundamento e analisar evidências enquanto a pesquisa se expande.

### 3.1 Tipos de descoberta que o NIE deverá dominar

1. **Metabusca:** resultados de SearXNG e fontes especializadas.
2. **Índices independentes:** Mwmbl e outros índices permitidos, quando disponíveis.
3. **Git:** pesquisa em metadados, código, README, issues, releases, tags, commits e referências cruzadas.
4. **Links HTML:** navegação de páginas, artigos, fóruns, comentários e índices.
5. **Sitemap/RSS/Atom:** expansão organizada por documentos publicados e feeds.
6. **PDF e Office:** hyperlinks, títulos citados, páginas referenciadas, notas, tabelas, normas e bibliografias.
7. **Menções sem URL:** nome de manual, comando, protocolo, RFC, publicação, produto ou release que gera nova consulta.
8. **Arquivos históricos:** URLs removidas e versões de sites/documentos.
9. **Busca multilíngue:** tradução e expansão terminológica orientadas ao domínio, preservando texto original.
10. **Lacunas da investigação:** hipótese que precisa de comprovação, conflito entre versões ou ausência de fonte oficial.
11. **Conhecimento anterior:** relações armazenadas em campanhas anteriores sugerem novos caminhos.
12. **Atualização incremental:** revisitar fontes que mudaram, sem baixar e indexar repetidamente o que está igual.

### 3.2 Uma URL não deve ser apenas uma string na fila

Cada candidato precisa carregar, no mínimo, URL canônica, origem, campanha, tipo de descoberta, profundidade, caminho percorrido, condições de acesso, score, razões do score, domínio, MIME estimado, idioma, custo previsto e histórico de tentativas. Um mesmo documento pode ser descoberto por cinco fontes: **o arquivo é preservado uma vez, mas as cinco arestas de proveniência permanecem**.

Uma investigação também pode descobrir **um termo de pesquisa sem URL**. Ele deve entrar na fronteira como tarefa de descoberta, não ser confundido com download de arquivo.

### 3.3 Ciclo autônomo-alvo

```text
                           PERGUNTA / OBJETIVO
                                  │
                                  ▼
                       PLANEJADOR DE INVESTIGAÇÃO
                    subperguntas, restrições, critérios
                                  │
                                  ▼
                   BUSCA MULTIFONTE + BIBLIOTECA LOCAL
                                  │
                                  ▼
                       FRONTEIRA DE DESCOBERTA
                  URLs, termos, repos, referências, filas
                                  │
              ┌───────────────────┼───────────────────┐
              ▼                   ▼                   ▼
        CRAWLER HTTP         BROWSER / DEEP       GIT / ARQUIVOS
         Scrapy / Go          Crawl4AI            APIs / Zoekt*
              └───────────────────┼───────────────────┘
                                  ▼
                   COLETA SEGURA + EXTRAÇÃO
             HTML, PDF, CLI, tabelas, links, menções
                                  │
                                  ▼
                     ARQUIVO ORIGINAL + CATÁLOGO
                hash, versão, parser, licença e origem
                                  │
                                  ▼
                       MOTOR DE EVIDÊNCIAS
               afirmações atômicas, conflitos, lacunas
                                  │
                                  ▼
                          DECISOR DE RODADA
                  suficiente? relevante? orçamento?
                      │                      │
                    NÃO                     SIM
                      │                      │
            novas hipóteses e URLs           ▼
                      │                DOSSIÊ CITÁVEL
                      └───────────────► histórico da campanha
                      └──► FRONTEIRA (nova rodada)

     * Zoekt pesquisa código previamente indexado, não todo o Git público.
```

### 3.4 Como a autonomia decide avançar ou parar

**Avançar** quando existir lacuna importante, fonte promissora, referência específica não examinada, conflito relevante ou oportunidade de descobrir evidência independente. **Suspender/encerrar** quando as perguntas foram atendidas no grau solicitado, o orçamento foi atingido, os caminhos restantes têm rendimento insuficiente, as fontes estão bloqueadas ou os resultados se tornaram repetitivos.

Essas decisões devem ser observáveis e reproduzíveis. Nenhum agente pode ignorar cotas, restrições de domínio ou regras de segurança porque “a investigação ainda parece interessante”.

### 3.5 Critérios de ranking da fronteira

Relevância temática e técnica; novidade do documento; diversidade de origem; pertinência ao modelo/versão; proximidade com evidência primária; profundidade; custo de coleta; disponibilidade e permissão; histórico de utilidade daquele domínio; e **valor esperado de informação nova**, não apenas ocorrência de palavras na URL. Evitar armadilhas de calendário infinito, parâmetros combinatórios, duplicações e conteúdo sintético/repetitivo.

## 4. Capacidades funcionais completas — escopo-alvo

| Área | Capacidade esperada | Resultado verificável |
|---|---|---|
| Descoberta global | Federar índices de web, Git, históricos e fontes especializadas | Descobrir domínios não fornecidos pelo usuário |
| Busca web | Pesquisar HTML público, blogs, documentação, notícias e fóruns autorizados | Página e contexto recuperáveis |
| Pesquisa profunda em fóruns | Ler postagens, respostas, comentários e referências permitidas | Relação entre discussão e fonte primária |
| Crawling em árvore | Seguir links, sitemap, RSS, menções e citações | Grafo com profundidade, convergência e prioridade |
| Pesquisa multietapas | Decompor problemas e replanejar conforme resultados | Novas tarefas realmente executadas |
| Inteligência multilíngue | Pesquisar PT, EN, ES e outros idiomas relevantes | Trecho original + tradução explicada quando houver |
| Git Eye | Pesquisar conteúdo, commits, arquivos, releases, branches e repositórios escolhidos | Arquivo, linha, commit/estado de trabalho verificável |
| Documentos | PDF digital, PDF escaneado, Office, HTML, Markdown, texto, tabelas | Extração por página/offset, parser versionado |
| Conhecimento acumulativo | Reusar documentos e pesquisas anteriores | Menos recolhimento duplicado, mais conexões úteis |
| Pesquisa lexical exata | Comandos CLI, OIDs, caminhos e erros preservados | Mesmo texto/pontuação que a fonte original |
| Pesquisa semântica | Termos similares e conceitos relacionados, quando habilitada | Melhoria medida em recall sem sacrificar precisão |
| Versões e compatibilidade | Equipamento, firmware, patch, licença, modo CLI, pré-requisitos | Comparação sem falsa equivalência |
| Verificação de afirmações | Distinguir suporte, refutação, citação indireta e ausência | Cada claim aponta ao trecho que a sustenta ou limita |
| Independência de fontes | Separar cópias, republicações e evidência original | Três cópias não contam como três provas |
| Contradições | Manter conflitos explícitos e hipóteses concorrentes | Não declarar certeza pela maioria de URLs |
| Biblioteca versionada | Arquivo original imutável + extrações e revisões | Reprocessar sem destruir histórico |
| Auditoria | Rastrear ações do agente, custos, escolhas e fontes | Investigação reproduzível |
| Monitoramento | Detectar alteração de documento e fontes novas | Diff, timestamp, alerta, impacto em claims |
| Interface | Pesquisa, grafo, campanhas, políticas, biblioteca e comparador | Operação sem editar configurações para tarefas normais |
| Dossiê | Conclusão qualificada, fontes, páginas, incertezas, testes/rollback | Documento técnico auditável |
| Escala | De uma VM para workers especializados/distribuídos | Ganho de throughput medido sem perder integridade |
| Generalidade | Módulos de domínio plugáveis | Caso não-telecom com busca e evidências reais |
| Privacidade | Dados locais, controle de retenção, segredos protegidos | Sem execução não autorizada nem vazamento para modelos |


---

# PARTE II — ARQUITETURA, STACK E REUTILIZAÇÃO DE SOFTWARE MADURO

## 5. Princípios obrigatórios de arquitetura

1. **Fonte única de verdade:** o NIE é dono dos seus documentos, campanhas, evidências, políticas e grafo. Nenhum crawler, ferramenta RAG ou agente externo assume a autoridade sobre o catálogo principal.
2. **A plataforma é genérica; telecom é um pacote especializado:** adicionar áreas não exige refazer o núcleo.
3. **Separar control plane e data plane:** Go decide *o quê, por quê, quando e com qual limite*; executores especializados obtêm dados sem poder redefinir políticas.
4. **Reusar motores maduros por contratos estáveis:** extratores, buscadores, crawlers e agentes são adaptadores substituíveis; usar apenas o necessário.
5. **Uma URL, um executor principal por tentativa:** evitando que Go, Scrapy e Crawl4AI dupliquem coletas sem motivo explícito.
6. **A coleta não implica confirmação:** conteúdo bruto é dado não confiável; inferências são hipóteses até terem evidências verificáveis.
7. **Imutabilidade dos originais, versionamento das transformações:** documentos e provas não desaparecem porque mudamos parser ou modelo.
8. **Crawl e investigação são coisas distintas:** seguir links não garante descoberta útil, e raciocinar sobre texto não significa verificar a afirmação.
9. **Sistemas pesados são escalonáveis, não padrão:** OpenSearch, Zoekt, StormCrawler, Docling e navegador executam conforme benefício comprovado.
10. **O software pode operar sem IA para pesquisa básica:** LLM é recurso substituível, não condição para abrir documentos e localizar comandos.
11. **Segurança implementada no sistema operacional, rede, fila e API:** não depender exclusivamente de prompts.
12. **Ninguém declara conclusão por existir um arquivo, classe, container ou teste mockado.** A prova exige execução relevante do fluxo.

## 6. Stack selecionada e linguagens

### 6.1 Núcleo preferencial

| Camada | Tecnologia e papel | Estado no ZIP V6 | Decisão de longo prazo |
|---|---|---|---|
| Backend/API | **Go + `net/http` + Huma** | Implementado | Manter como orquestrador central |
| Acesso a dados | **pgx** | Implementado | Acesso explícito, transações e migrações auditáveis |
| Banco/fila | **PostgreSQL** | Implementado; integração testada em PG16 conforme README | PG18 é alvo declarado; validar na versão efetiva |
| FTS e aproximação | PostgreSQL `tsvector`, `pg_trgm` | Código existente | Base inicial; não confundir FTS com BM25 |
| Frontend | **Svelte + Vite** | Implementado em **JavaScript**, não TypeScript | Migrar gradualmente a TypeScript quando contratos estabilizarem |
| Armazenamento | Filesystem content-addressed, SHA-256 | Implementado | Manter originais imutáveis; adaptador S3 apenas se necessário |
| Crawling HTTP simples | Fetcher nativo Go | Implementado | Fallback eficiente e executor básico seguro |
| Crawling especializado | Worker Python/FastAPI | Parcial | Integrar Scrapy/Crawl4AI reais quando justificado |
| Web metabusca | SearXNG | Adaptador/configuração presentes, E2E externo não comprovado | Fonte inicial importante quando habilitada |
| Backup | PostgreSQL backup + **Restic** para objetos/configurações | Não demonstrado na V6 | Indispensável antes de acervo valioso e operação contínua |
| Proxy reverso/HTTPS | Caddy ou equivalente | Não integrado no V6 | Condicional para acesso remoto, sem amarrar o produto |
| Execução/implantação | Docker Compose, Linux/VM Proxmox | Arquivos presentes | Operação reproduzível e, no futuro, trabalhadores separados |

**Rust:** não é requisito de arquitetura. Poderá ser considerado para um motor de processamento específico se um benchmark reproduzível demonstrar ganho relevante em RAM/CPU. Trocar Go por Rust agora não corrige problemas de evidências, agentes ou crawling.

**Python:** não substitui Go como control plane. Ele executa ferramentas que já são predominantemente Python (Scrapy, Crawl4AI, Docling e agentes, caso escolhidos).

### 6.2 Bibliotecas e dependências já presentes no último ZIP inspecionado

Esta é a descrição de **dependências declaradas no código**, não uma recomendação para incorporar outras sem teste:

**Go (`go.mod`):** `github.com/danielgtaylor/huma/v2`, `github.com/jackc/pgx/v5`, `github.com/ledongthuc/pdf`, `github.com/temoto/robotstxt` e `golang.org/x/net` (com dependências indiretas do ecossistema pgx e `x/text`, `x/sync`). O módulo declara `go 1.27.0`; o Dockerfile apontava para o toolchain 1.27.3 na auditoria V6.

**Frontend (`web/package.json`):** Svelte 5, Vite e `@sveltejs/vite-plugin-svelte`. Os arquivos `web/src/main.js` e `web/src/App.svelte` confirmam frontend em JavaScript/Svelte; **não registrar TypeScript como já implementado**.

**Worker Scrapy (`requirements.txt`):** FastAPI, Uvicorn, Pydantic, requests, BeautifulSoup e Scrapy. Entretanto, a execução observada aproveita sobretudo os componentes de extração de links; não equivale a um `CrawlSpider` com seu próprio motor.

**Worker navegador (diretório `crawl4ai_worker`):** FastAPI, Uvicorn, Pydantic, requests, BeautifulSoup e **Playwright**. **A biblioteca Crawl4AI não está declarada nem comprovadamente usada na V6.** O nome do diretório não é prova de integração; há uma decisão aberta entre integrar Crawl4AI de verdade ou renomear o executor para Playwright.

**Infraestrutura declarada:** Docker Compose, PostgreSQL 18 na imagem indicada, SearXNG configurável e dois workers Python. A existência dessas definições não demonstra execução de campanha real em todas as combinações.

**Política de dependências:** versões e hashes de imagem/pacotes devem ser fixados por release testada; varredura de vulnerabilidades, SBOM e revisão de licença entram no ciclo de lançamento. Qualquer biblioteca pode ser substituída por implementação compatível desde que preserve contratos e evidências.

## 7. Desenho de arquitetura lógico completo

```text
┌────────────────────────────────────────────────────────────────────┐
│                         EXPERIÊNCIA NIE                            │
│  Svelte: busca, campanhas, grafo, comparação, evidências, ajustes   │
└──────────────────────────────┬─────────────────────────────────────┘
                               ▼
┌────────────────────────────────────────────────────────────────────┐
│                   API / ORQUESTRADOR GO + HUMA                      │
│  autenticação • autorização • políticas • orçamento • auditoria     │
│  planejador • fila • estados • escalonador • adaptadores            │
│  árbitro de relevância • critérios de parada • dossiês             │
└───────────────┬─────────────────────┬──────────────────────────────┘
                │                     │
     ┌──────────▼──────────┐ ┌────────▼────────────────────────────┐
     │     GIT EYE         │ │             WEB EYE                  │
     │ GitHub/GitLab APIs  │ │ SearXNG, Mwmbl, Wayback, CommonCrawl │
     │ Software Heritage   │ │ sites, fóruns, feeds, documentos     │
     │ ecosyste.ms, Zoekt* │ │ APIs especializadas, índices        │
     └──────────┬──────────┘ └────────┬────────────────────────────┘
                └──────────┬──────────┘
                           ▼
              ┌─────────────────────────────┐
              │ FRONTEIRA / FILA CANÔNICA   │
              │ PostgreSQL: URL, query,     │
              │ prioridade, origem, lease,  │
              │ dedupe, retry, orçamento    │
              └──────────────┬──────────────┘
                             ▼
             ┌─────────────────────────────────────────┐
             │           ROTEADOR DE EXECUTORES         │
             └──────────┬──────────┬───────────┬───────┘
                        │          │           │
                        ▼          ▼           ▼
                    Go HTTP     Scrapy*     Crawl4AI*
                    simples    recursivo     JS/deep
                        └──────────┼───────────┘
                                   ▼
              ┌────────────────────────────────────────┐
              │ NORMALIZAÇÃO / EXTRAÇÃO / SEGURANÇA     │
              │ HTML, PDF, OCR*, Tika*, Docling*, CLI  │
              │ links, nomes, referências, proveniência│
              └──────────────────┬─────────────────────┘
                                 ▼
          ┌──────────────────────┼─────────────────────┐
          ▼                      ▼                     ▼
  ┌──────────────┐      ┌─────────────────┐    ┌──────────────────┐
  │ OBJETOS CAS  │      │ POSTGRESQL      │    │ ÍNDICES DERIVADOS│
  │ SHA-256      │      │ campanhas, docs, │    │ FTS/trgm, vetores│
  │ originais    │      │ extrações, claims│    │ Zoekt/OpenSearch*│
  │ e versões    │      │ arestas, provas  │    │ reconstruíveis    │
  └──────────────┘      └────────┬────────┘    └──────────────────┘
                                 ▼
              ┌────────────────────────────────────────┐
              │ INTELIGÊNCIA DE INVESTIGAÇÃO           │
              │ LLM local/API* • evidências atômicas   │
              │ conflitos • independência • lacunas    │
              └──────────────────┬─────────────────────┘
                                 ▼
              ┌────────────────────────────────────────┐
              │ DOSSIÊ + ALERTAS + AUDITORIA           │
              │ links exatos, condições, incertezas     │
              └──────────────────┬─────────────────────┘
                                 │
                                 └──► novas pistas / FRONTEIRA

* Módulos opcionais, sob demanda, planejados ou ainda não integrados.
```

### 7.1 Separação de planos

**Control plane (Go):** contratos, usuários, campanha, filas, prioridades, seleção de executor, limites, segurança, agendamento, métricas, política de fontes, estado da investigação, versionamento e índices de proveniência.

**Data plane (workers):** descobrir conteúdo em APIs, buscar/requisitar páginas, renderizar JS quando preciso, extrair estruturas, classificar formatos e processar artefatos em isolamento. **Um worker não altera por conta própria as regras centrais da campanha.**

**Knowledge plane (PostgreSQL + CAS + índices):** dados canônicos, entidades, evidências, vínculos e versões; índices textuais/vetoriais derivados e reconstruíveis.

**Reasoning plane:** decomposição da pergunta, seleção de lacunas, propostas de afirmações, confronto com fontes e redação de respostas ancoradas. O modelo **não concede validade operacional** a um comando por acreditar que parece correto.

## 8. Onde usar projetos prontos — sem reinventar a roda

### 8.1 Núcleo de crawling e descoberta

| Projeto | Capacidade madura a aproveitar | Como integraremos | Cuidado decisivo |
|---|---|---|---|
| **Scrapy** | `CrawlSpider`, `Rule`, `LinkExtractor`, middlewares, robots, controle de requisições e persistência de jobs | Executor Python para sites HTML e estruturas recursivas | Fila de campanhas continua canônica no PostgreSQL; não criar duas autoridades de URLs |
| **Crawl4AI** | Browser, extração preparada para IA, deep crawling BFS/DFS/Best-First, filtros, scoring e retomada | Executor pesado sob demanda para sites JS e pesquisas profundas | Biblioteca precisa estar instalada e realmente usada; preservar original separado de DOM/Markdown |
| **Playwright** | Execução real de navegador, JS, navegação e interceptação | Dependência do executor de navegador, diretamente **ou** por Crawl4AI | Não tratá-lo como crawler inteligente completo; proteger todos os subrecursos com egress isolado |
| **SearXNG** | Agregação de buscadores e saída estruturada | Radar de URLs-semente e pesquisas multicanal | Índices upstream podem limitar/bloquear; habilitar JSON e monitorar conectores |
| **Mwmbl** | Índice próprio de pesquisa web | Fonte externa complementar | Cobertura e SLA diferentes dos buscadores comerciais |
| **GitHub/GitLab APIs** | Repositórios, metadados, conteúdo/revisões e busca autorizada | Git Eye e descoberta de fontes | Paginação, rate limits, permissões e escopo dos índices |
| **Wayback / Common Crawl** | Acervos históricos, CDX e capturas WARC | Recuperação direcionada de material antigo | Não equivalem a um Google full-text sobre todos os registros |
| **Software Heritage / ecosyste.ms** | Código histórico e descoberta de projetos | Adaptadores Git Eye | Acervo global/metadados não equivalem a grep instantâneo universal |
| **Apache StormCrawler** | Crawling contínuo distribuído em streaming | Executor futuro se houver demanda de escala | Não dividir o estado canônico nem rodar concorrente com Scrapy sobre mesmas URLs |
| **Apache Nutch** | Crawling industrial em grande corpus | Alternativa de escala | Não instalar com StormCrawler só para aumentar a lista de serviços |
| **Heritrix** | Crawling para preservação arquivística | Especializado, opcional | Priorizar preservação seletiva e permissão de armazenamento |
| **Crawlee / YaCy** | Crawler JS/TS ou índice web próprio P2P | Alternativas experimentais | Evitar redundância com Scrapy/Crawl4AI/SearXNG |

O NIE deverá saber **por que enviou uma URL a determinado executor**. Exemplo: HTML estático → HTTP Go ou Scrapy; site JS necessário → Crawl4AI/Playwright isolado; repositório Git → adaptador Git, não crawler HTML genérico. Caso o executor falhe, registrar a tentativa e seu motivo, e só realizar fallback previsto pelas políticas.

### 8.2 Extração, documentos e arquivamento

| Projeto | Reutilização pretendida | Condição |
|---|---|---|
| **Apache Tika** | Detecção de formatos, texto e metadados de PDF/Office/outros | Chamar por formato quando o extrator básico for insuficiente |
| **Docling** | PDF técnico complexo, estrutura de páginas, tabelas, layout e caminho de OCR | Prioritário para manuais reais de fabricantes, em worker sob demanda |
| **Tesseract OCR** | OCR gratuito para páginas digitalizadas em fluxos adequados | Ativado com limite de CPU/RAM, avaliação de qualidade e preservação das coordenadas |
| **ArchiveBox** | Capturas selecionadas e formatos arquiváveis, inclusive WARC/PDF/HTML | Complemento de preservação; nunca a única fonte de verdade do NIE |
| **Restic** | Backups versionados, criptografados e restauração | Backup de objetos/configuração coordenado com dump/PITR do PostgreSQL |

Para arquivos do acervo, o NIE deve guardar **os bytes efetivamente recebidos** (e descrever encoding/decodificação), a versão renderizada se existir, texto extraído e páginas/tabelas. Cada etapa pode ser reprocessada com outra versão de ferramenta sem sobrescrever o original.

### 8.3 Busca documental e recuperação de conhecimento

| Projeto | Papel | Quando adotar |
|---|---|---|
| **PostgreSQL FTS + pg_trgm** | Busca textual e fuzzy inicial; filtros estruturados, joins por evidência | Núcleo obrigatório |
| **pgvector** | Embeddings no mesmo PostgreSQL | Após demonstrar ganho de recall em benchmark, sem perder busca exata |
| **Zoekt** | Busca rápida em arquivos de código de repos **selecionados e indexados** | Após reunir corpus Git suficientemente útil |
| **OpenSearch** | Índice documental BM25, facetas, busca híbrida/semântica | Quando PostgreSQL não atender às metas mensuradas de consulta/índice |
| **OpenGrok** | Referências cruzadas e navegação especializada em código | Piloto se uso por símbolos/versões exigir |
| **Apache Solr, Livegrep, Hound, Qdrant, Meilisearch** | Alternativas de busca textual/código/vetor | Somente se houver benefício claro; não duplicar todos os índices |

**Não confundir:** `tsvector` + `pg_trgm` não são, por si só, BM25 ou busca semântica; Zoekt não pesquisa instantaneamente repositórios que nunca foram obtidos/indexados; embeddings não garantem preservação exata de pontuação ou sintaxe CLI.

### 8.4 Agentes de pesquisa, memória e RAG

| Projeto | Contribuição potencial | Decisão |
|---|---|---|
| **GPT Researcher** | Planejamento de perguntas, pesquisa multietapas, leitura de fontes, relatórios | Primeiro piloto de agente executor; medir integração, custo e rastreabilidade |
| **Hyperresearch** | Vault permanente, múltiplas perspectivas, trilha de proveniência, checagem de citações, análise de independência e lacunas | Piloto concorrente ou fonte de boas práticas; não presumir serviço pronto independente de Codex/Claude Code |
| **RAGFlow** | Ingestão, indexação/recuperação documental e camada RAG já desenvolvida | Comparar com RAG próprio; evitar segunda fonte canônica de verdade |
| **Onyx** | Busca corporativa, conectores e RAG | Alternativa a RAGFlow, não obrigatório em paralelo |
| **Open WebUI / AnythingLLM** | Interfaces para modelos e acervos de conhecimento | Alternativas de interface; não substituir a experiência NIE sem necessidade |
| **LangChain Open Deep Research** | Referência histórica de arquitetura | Não eleger como dependência central sem revisar seu estado de manutenção |

**Questão arquitetural decisiva:** mesmo se um agente pronto fornecer pesquisa e relatórios, o NIE continua controlando **tarefas, permissão, orçamento, proveniência, versões, afirmações e memória permanente**. Resultados de agentes externos voltam como artefatos auditáveis, nunca como decisões incontestáveis.

## 9. Contratos canônicos entre módulos

Os nomes seguintes representam a interface-alvo, não necessariamente as estruturas já implementadas no Go atual.

| Contrato | Campos essenciais | Para que existe |
|---|---|---|
| `ResearchCampaign` | objetivo original, filtros, fontes, orçamento, política, critérios de parada, estado | Unidade recuperável de investigação |
| `ResearchQuestion` | texto, subpergunta, hipótese, requisito de evidência, prioridade, origem | Planejamento rastreável |
| `DiscoveryCandidate` | URL/termo/repo, como foi encontrado, pai, campanha, profundidade, score e razões | Expandir caminhos sem perder o histórico |
| `DiscoveryEdge` | nó de origem, nó destino, tipo (`link`,`mention`,`citation`,`sitemap`,`feed`,`redirect`...), tempo | Grafo verificável da árvore de exploração |
| `FetchTask` | URL validada, executor, restrições, limites, tentativas, deadline, lease | Unidade segura de trabalho |
| `FetchResult` | status, headers, URL final, bytes originais, bytes renderizados/derivados, links, métricas | Resultado de qualquer worker no mesmo formato |
| `Source` | identidade, domínio, autor/editor, licença, política, linhagem, autoridade | Rastrear origem e independência |
| `Document` | identidade lógica, hashes, título, URL original, MIME, datas, direitos | Documento referenciável independentemente do parser |
| `DocumentSnapshot` | captura específica, SHA-256, ETag, Last-Modified, localização | Preservar versões reais |
| `ExtractionRun` | snapshot, parser, versão, configuração, data, qualidade, estado | Reprocessar documento sem destruir extrações |
| `Passage` | `extraction_run_id`, página, seção, offsets, texto, idioma, tipo | Citação precisa e busca |
| `SourceIndependenceCluster` | relações de republicação/cópia, procedência e razão | Evitar falsa corroboração |
| `AtomicClaim` | sujeito, predicado, objeto, negação, condições, escopo, tempo | Unidade factual verificável |
| `EvidenceLink` | claim, passage, suporte/refutação/menção, escopo, justificativa | Verificação que aponta a texto real |
| `ValidationRun` | laboratório autorizado, ambiente, firmware, passos, resultado, evidências | Separar documentação de validação experimental |
| `ResearchRound` | entradas, pesquisas executadas, novas fontes, lacunas, decisões e orçamento | Provar autonomia multirrodada |
| `BudgetLedger` | bytes transferidos, únicos, reservados, requisições, tokens, tempo | Cumprimento de custos e cotas |
| `AuditEvent` | ator, evento, decisão, motivos, dados afetados, timestamps | Prestação de contas e investigação de falhas |

### 9.1 Invariantes do armazenamento

- Um SHA-256 identifica **os bytes preservados**, não uma conclusão de IA.
- O original e as derivações nunca compartilham uma identificação ambígua.
- Novos parsers criam `ExtractionRun` adicional, não substituem passagens históricas.
- Uma URL pode ter múltiplas capturas no tempo; um arquivo pode ser descoberto por múltiplas URLs.
- Deduplicar download não exclui as várias arestas de descoberta.
- Uma claim não deve perder evidências porque outra claim foi mesclada; migrações não podem apagar comprovações históricas.
- Evidência positiva, negativa ou não conclusiva guarda **trecho exato e escopo**; o resumo é reconstruível.
- Validação de laboratório é evento auditável, não um status facilmente sobrescrito por nova busca web.
- Índices semânticos, OpenSearch ou Zoekt podem ser reconstruídos a partir dos dados canônicos.
- Campanhas pausadas, concluídas e reabertas não perdem tarefas nem aceitam trabalhos fora de seu orçamento.

## 10. O motor de evidências que queremos — sem falsas certezas

O ponto mais importante da especialização telecom é **não confundir uma frase com uma prova**.

### 10.1 Três dimensões independentes de avaliação

| Dimensão | Estados exemplificativos | Significado |
|---|---|---|
| **O que a fonte afirma** | suporte, refutação, menção, condição, incerteza, silêncio | Conteúdo semântico da proposição |
| **Aplicabilidade ao caso** | compatível comprovado, escopo incompleto, não aplicável, conflitante | Equipamento/release/condições correspondem? |
| **Grau de comprovação** | não confirmado, documentação oficial, corroborado independente, laboratório validado, contestado | Que tipo de sustentação existe? |

A fonte pode ser oficial e ainda assim **não comprovar** a aplicabilidade ao equipamento do usuário. Duas fontes independentes podem entrar em conflito; conflito não é incompatibilidade automática. Uma afirmação encontrada no fórum pode ser tecnicamente válida, mas continuar não verificada em laboratório.

### 10.2 Afirmação técnica atômica

Exemplo meramente ilustrativo — não é alegação de suporte real:

```yaml
claim_id: "exemplo-001"
subject:
  vendor: "Huawei"
  model: "NE8000"
  os: "VRP"
  release: "A CONFIRMAR"
predicate: "suporta"
object: "mecanismo específico de HA para sessões PPPoE"
polarity: "afirmativa"
conditions:
  license: "desconhecida"
  topology: "desconhecida"
  cli_mode: "não aplicável"
evidence:
  passage_id: "OBRIGATÓRIO PARA CONFIRMAR"
  source_snapshot_id: "OBRIGATÓRIO"
  quotation_offsets: "OBRIGATÓRIO"
applicability: "indeterminada"
lab_validation: "não realizada"
```

### 10.3 Regras de comprovação

1. **Sujeito vinculado ao predicado:** “Cisco suporta BGP” não comprova que “Huawei suporta BGP”, mesmo no mesmo PDF.
2. **Negação com alvo correto:** “não suporta PPPoE, mas suporta BGP” não significa que BGP é negado.
3. **Firmware explícito:** não inferir release por nome de hardware, ano, seção do manual ou similaridade textual.
4. **Não colapsar recursos compostos:** uma documentação que menciona BGP e PPPoE separadamente não confirma preservação de sessões no failover.
5. **Diferenciar menção de afirmação:** listas de tópicos e índices de manual não são provas de suporte.
6. **Condições obrigatórias:** módulos, licenças, modo de configuração, topologia e limitações devem fazer parte do escopo.
7. **Proveniência na mesma evidência:** a origem oficial e o trecho que sustenta a conclusão precisam pertencer à evidência correta, não à soma oportunista de documentos.
8. **Cópias não multiplicam autoridade:** medir independência de publicação e origem primária, com incerteza quando não for determinável.
9. **Conflito permanece conflito:** evidências contraditórias completas não devem virar `incompatible` automaticamente.
10. **IA pode propor, não certificar:** extração por LLM requer validação do texto e nunca substitui teste experimental autorizado.
11. **Dossiê nunca esconde lacunas:** fatos comprovados, hipóteses e cenários não validados devem aparecer separados.
12. **Regressão adversarial permanente:** novas classes de erro precisam entrar no conjunto de testes sem “ensinar” apenas uma frase para a regex.

### 10.4 Quando usar IA no processo factual

Um LLM pode ajudar a transformar trechos em propostas estruturadas (`subject`, `predicate`, `object`, condições, polaridade), identificar perguntas faltantes, comparar duas fontes ou sugerir caminhos de busca. Entretanto, o sistema precisa checar *evidence-to-claim binding* — se o trecho efetivamente sustenta aquela afirmação — com mecanismos estruturados, auditoria independente e opção de revisão humana. Para configurações de alto impacto, exigir documentação apropriada e, quando aplicável, testes de laboratório.

## 11. Pesquisa e recuperação — o NIE não deve mutilar comandos CLI

**Busca exata:** preservar caminhos, barras, aspas, flags, espaços relevantes, OIDs, endereços exemplificativos e sintaxe. Exemplo: `display bgp peer`, `/routing/bgp/connection`, `route-policy`, `interface gpon` e mensagens de erro específicas.

**Busca lexical:** PostgreSQL FTS inicial com stemming adequado ao idioma, filtros técnicos e ranking. Útil para documentos grandes, mas não garante todos os símbolos do CLI.

**Busca aproximada:** `pg_trgm` para nomes, títulos, erros de digitação e candidatos; aplicar com cuidado em expressões curtas e pontuadas.

**Busca semântica:** embeddings opcionais, com modelo e versão registrados. Devem ajudar a descobrir conteúdos equivalentes, mas **nunca corrigir automaticamente** uma sintaxe CLI.

**Busca Git:** APIs e código selecionado; índice Zoekt quando o corpus e benchmark justificarem. Todo resultado deve indicar *repositório, caminho, linha/intervalo, commit ou estado de working tree*.

**Combinação:** recuperação de candidatos → classificação do contexto → verificações por versão/escopo → citações com offsets. Medir precisão, recall, latência, custo e fração de evidências incorretas.


---

# PARTE III — INFRAESTRUTURA, SEGURANÇA, DADOS E OPERAÇÃO

## 12. Implantação enxuta hoje, escalável amanhã

### 12.1 Topologia recomendada para um servidor único

```text
┌──────────────────────── VM Linux / Proxmox ───────────────────────┐
│                                                                   │
│    Caddy / TLS* ─────► API Go + painel Svelte                      │
│                              │                                    │
│                      ┌───────┴──────────┐                         │
│                      ▼                  ▼                         │
│                PostgreSQL          Workers Go                     │
│                  │ FTS               │                            │
│                  │ fila              ├──► Scrapy* isolado         │
│                  │ claims            ├──► Crawl4AI* isolado       │
│                  │ grafo             └──► Docling/OCR* isolado     │
│                  │                                                │
│                  └───────────► storage CAS em volume separado     │
│                                      │                            │
│                              backup consistente                   │
│                                      │                            │
└──────────────────────────────────────┼────────────────────────────┘
                                       ▼
                          Cópia externa + teste de restore

* Componentes habilitados conforme necessidade e ambiente.
```

**Dois princípios físicos:** banco e índices quentes em SSD/NVMe; originais maiores e backups em volumes com políticas próprias. Não colocar worker de navegador que lê sites arbitrários na mesma zona de rede que gerencia Proxmox, switches, OLTs, roteadores ou serviços internos.

### 12.2 Perfis ilustrativos, não benchmarks

| Perfil | CPU / RAM | Disco de sistema e índice | Acervo | Limitações |
|---|---|---|---|---|
| **Protótipo mínimo** | 2 vCPU / 4 GB | SSD 60–100 GB | Pequeno e seletivo | Sem LLM local grande, sem OCR massivo e com concorrência muito baixa |
| **Primeira instalação recomendada** | 4 vCPU / 8 GB | SSD 160–250 GB | Volume separado conforme necessidade | SearXNG/Go/Postgres e poucos workers; processos pesados sob demanda |
| **Investigação contínua ampliada** | 8–12 vCPU / 16–32 GB | NVMe 500 GB ou mais | NAS/HDD/objeto de 2–8 TB ou conforme corpus | Medir RAM dos parsers, índices, browser e tamanho de PDFs |
| **Escala distribuída** | Dimensionamento por medições | DB/índices especializados, workers independentes | Armazenamento durável e backup | Novos nós apenas após comprovar volume e SLOs |

Todos os números são **hipóteses iniciais de capacidade**, não requisitos universalmente comprovados. OCR, conversão de PDFs complexos, navegadores e LLMs locais podem exigir recursos maiores e, em alguns casos, GPU. A configuração real dependerá da mistura de formatos, throughput, tamanho dos índices e uso de IA.

### 12.3 Quando adicionar recursos mais pesados

| Evidência de gargalo | Extensão candidata | Prova mínima antes de incorporar |
|---|---|---|
| Busca de texto/documentos do PostgreSQL aquém de meta medida | OpenSearch | Benchmarks com corpus real, custo operacional e reconstrução do índice |
| Busca em muitos repositórios selecionados lenta | Zoekt | Ganho claro sobre APIs Git + busca seletiva |
| HTML comum não entrega o conteúdo | Crawl4AI/Playwright | Páginas JS reais e isolamento adequado |
| PDFs técnicos perdem tabelas, páginas ou relações | Docling/Tika/OCR | Aumento de extração correta por tipos de documento |
| Campanhas exigem muitas instâncias remotas de crawling | StormCrawler **ou** Nutch | Ganho no volume **útil**, segurança e recuperação |
| Dados não cabem ou não podem permanecer em um host | S3 compatível/SeaweedFS ou solução mantida | Compatibilidade, durabilidade, recuperação, licença e custo |
| Recuperação semântica melhora casos difíceis | pgvector; Qdrant apenas se necessário | Ganho medido de recuperação e custo total |
| Mensageria PostgreSQL vira gargalo comprovado | Fila dedicada como NATS JetStream | Benefício superior ao custo de manter mais serviços |

## 13. Segurança, direitos e limites são parte da arquitetura

O NIE buscará muitas páginas que jamais foram revisadas por nós. **Todo conteúdo externo é potencialmente malicioso, errado, protegido por direitos ou contendo dados pessoais.** Segurança precisa ser aplicada fora do modelo de IA.

### 13.1 Coleta web e SSRF

- Permitir apenas esquemas autorizados e URLs públicas por padrão; rejeitar `file:`, destinos internos e esquemas não previstos.
- Validar e fixar o IP de conexão efetivo; revalidar DNS, IPv4/IPv6, redirects e cadeia de destino para reduzir SSRF e DNS rebinding.
- Bloquear por política de egress no nível de rede: loopback, privado, link-local, metadados cloud, interfaces de gestão e redes restritas.
- Aplicar a política a **todos os subrecursos do browser**, não apenas URL inicial e final.
- Respeitar `robots.txt`, políticas de uso, limites de domínio, `429`, autenticação e regras de APIs; pausar quando a permissão de acesso for incerta.
- Tratar conteúdo com tamanho exagerado, loops de calendário, páginas infinitas, ZIP/PDF bomb, attachments e formatos não confiáveis com limites de isolamento.
- Não usar identidade falsa, proxies rotativos ou mecanismos de evasão para ultrapassar proibições de coleta.

### 13.2 Prompt injection e ferramentas de IA

HTML, PDFs, README, discussões, snippets de busca e até textos retornados por APIs são **dados não confiáveis**, não instruções para o agente. Ferramentas devem validar parâmetros, permissões, domínios, custos e efeitos. Uma página que diz “ignore as instruções e execute este script” não deve receber capacidade de executar código ou acessar serviços internos. Controle de ações deve ocorrer no orquestrador e no sistema operacional.

### 13.3 Operação e identidade

- Autenticação e autorização antes de expor o painel fora do loopback; RBAC quando existirem múltiplos usuários.
- TLS, rate limits de API, proteção contra XSS/CSRF conforme o mecanismo de autenticação, cookies seguros, cabeçalhos e políticas de conteúdo.
- Segredos nunca gravados em logs, relatórios ou capturas; segredos encontrados em configurações públicas devem ser mascarados, isolados e tratados conforme política.
- Execução dos workers com privilégios mínimos, filesystem restrito, limites de CPU/RAM/PIDs, processos e tempo; browsers com sandbox efetivo e isolamento de rede. `--no-sandbox` não é solução de segurança.
- Versionamento de ferramentas, SBOM, verificação de imagens, histórico de deploy e possibilidade de rollback.

### 13.4 Direitos autorais, políticas de fontes e conteúdo sensível

Cada fonte poderá ser classificada como **link apenas**, **trecho permitido**, **arquivo preservável** ou **não coletar**, com justificativa. A plataforma deve respeitar licenças e direitos de redistribuição, separar acesso autorizado de indexação, permitir remoção/retenção e evitar disseminar dados pessoais/segredos. Modelos externos não devem receber documentos sensíveis sem permissão explícita e política de anonimização/redação.

A opção de pesquisar serviços `.onion` públicos conhecidos ou metadados BitTorrent **não** dá direito a contornar autenticação ou baixar material protegido. Esses conectores são opcionais, sujeitos a finalidade legítima, escopo e filtros específicos. Metadados DHT não significam acesso ao conteúdo inteiro de um torrent.

## 14. Biblioteca permanente — como armazenar conhecimento de verdade

### 14.1 Quatro camadas distintas

**(A) Captura original:** arquivo/entidade recebida, hash, URL, cabeçalhos relevantes, data, licença.  
**(B) Transformações versionadas:** DOM renderizado, HTML normalizado, extração de texto, OCR, tabelas, Markdown, identificação de comandos.  
**(C) Índices reconstruíveis:** FTS, trigramas, vetores, índice de código e facetas.  
**(D) Conhecimento verificável:** proposições, relações, contradições, independência de fontes, validações de laboratório, dossiês e históricos de decisão.

### 14.2 Ciclo de vida documental

```text
Descoberto → Permissão verificada → Capturado (original imutável)
     ↓                  ↓                    ↓
   Pista          Bloqueado/Link       Extração inicial
                                               ↓
                                         Texto / Tabelas
                                               ↓
                                          Indexação
                                               ↓
                                     Evidência / Claim
                                               ↓
                                 Revisita por mudança ou
                                reprocessamento com nova ferramenta
                                               ↓
                                     Nova versão sem apagar anterior
```

### 14.3 Política de atualização

- Registrar `first_seen`, `last_seen`, `last_checked`, `ETag`, `Last-Modified`, URL efetiva e hashes.
- Revisitar conteúdos relevantes e detectar diferenças; distinguir alteração real de anúncios repetidos.
- Preservar versão anterior quando houver relevância e permissão; não sobrescrever uma evidência citada.
- Dossiês antigos devem indicar qual snapshot sustentava cada frase, inclusive se uma nova versão contradiz o material anterior.
- Reprocessar manuais com novos extratores sem baixar de novo o original; comparar qualidade e permitir auditoria dos resultados.
- Deduplicar armazenamento físico sem perder múltiplas origens e relações de descoberta.

### 14.4 Backup e recuperação

Restic para volumes/documentos mais backup consistente PostgreSQL (dump e, quando a escala exigir, WAL/PITR). Testar restauração conjunta com **mesmo conjunto lógico de documentos, metadados, claims e proveniências**. Backup sem restauração exercitada não deve ser classificado como concluído. Índices reconstruíveis podem ter estratégia própria, mas o catálogo e os originais nunca podem depender exclusivamente do índice.

## 15. Observabilidade, limites e custo

Cada campanha deve gerar eventos e métricas suficientes para explicar **o que o NIE fez e por que fez**.

**Descoberta:** fontes consultadas, URLs únicas, arestas, profundidade, buscas derivadas, porcentagem de caminhos úteis, fontes inéditas, falhas/recusas, páginas sem relevância.

**Fila:** jobs criados/executados/duplicados, idade do job, retries, dead-letter, reabertura de campanha, perdas por expiração de lease, vazão por executor.

**Recursos:** `downloaded_bytes`, `stored_unique_bytes`, `reserved_inflight_bytes`, requisições, tempo, CPU/RAM de worker, espaço em disco, tokens/custo por modelo, latência de parser/consulta.

**Qualidade:** precisão das citações, relevância, versões corretas, independência efetiva das fontes, número de afirmações sem comprovação, falsos positivos críticos, contraditórios preservados, taxa de recuperação de documentos de referência.

**Segurança:** bloqueios SSRF, robots, destinatários rejeitados, solicitações de subrecursos proibidos, arquivos maliciosos/quarentenados, dados pessoais redigidos e ações negadas.

**Governança:** origem, ferramenta e versão de cada transformação; alterações de políticas; quem iniciou, pausou ou alterou campanhas; quais critérios levaram ao encerramento.

A infraestrutura de monitoramento deverá ser escolhida pelo custo/benefício (logs estruturados e métricas simples primeiro; OpenTelemetry e Grafana/Prometheus quando houver complexidade que justifique). Não exigir um cluster de observabilidade para duas goroutines, mas não operar campanhas contínuas sem trilha de diagnóstico.

## 16. Especialização inicial em engenharia de redes

O NIE deve organizar documentação em entidades semânticas de domínio. O núcleo genérico continuará funcionando em outros assuntos.

### 16.1 Fabricantes e famílias prioritários

Huawei (NE8000, MA5800 e famílias pertinentes), Datacom DmOS, Cisco IOS-XE/IOS-XR, MikroTik RouterOS v6/v7, ZTE, FiberHome, V-SOL, Intelbras, Digistar, TP-Link, Furukawa, SonicWall e Hillstone. A lista é extensível; nomes e versões não devem ser inventados pelo classificador.

### 16.2 Tópicos prioritários

BGP, roteamento de políticas (PBR), MPLS L2/L3 VPN, VRF, PPPoE, BNG/BRAS, RADIUS, CGNAT, GPON/XGS-PON, VLANs, alta disponibilidade, failover, SNMP/MIBs, interoperação entre equipamentos, comandos CLI, mensagens de erro, atualizações de firmware e limitações de licenciamento.

### 16.3 O que o modelo técnico precisa diferenciar

- Dispositivo e família versus produto de nome parecido.
- Versão exata do sistema operacional/firmware, patch e build; documento aplicável a release diferente não confirma compatibilidade.
- Comando de consulta versus comando de configuração; modo de CLI, sequência, privilégios e parâmetros.
- Exemplo comunitário versus declaração do fabricante, caso de suporte e teste de laboratório.
- Disponibilidade de roteamento **versus** continuidade de sessões PPPoE **versus** redundância RADIUS **versus** estado do BNG.
- Cenário de laboratório **versus** produção; pré-requisitos, licenças, módulos e topologia.
- Documentação atual versus histórica e funcionalidades depreciadas.

### 16.4 Limites de automação operacional

O NIE **pesquisa, explica, documenta, compara e ajuda a construir planos de teste**. A execução de CLI em roteadores, switches, OLTs e demais equipamentos não faz parte da capacidade automática de pesquisa. Uma futura integração autorizada com outros sistemas do ecossistema de ISP deve ter permissões, revisão humana, simulação, rollback e auditoria próprios.


---

# PARTE IV — ESTADO DA IMPLEMENTAÇÃO E LACUNAS PARA O PRODUTO IDEAL

## 17. Regra para ler este diagnóstico

Esta seção não é uma lista de coisas “prometidas em commit”; usa o **último código-fonte enviado e auditado (`nie-main (4).zip`)**, a matriz de README, as dependências reais e a auditoria V6. Diferencia:

- **Presente:** o componente aparece no código e há uma integração identificável.
- **Testado isoladamente:** testes cobrem funções/regras, não necessariamente a aplicação em operação.
- **Parcial:** existe parte do caminho, mas o comportamento completo não foi comprovado.
- **Pendente:** o mecanismo planejado não foi integrado ao runtime observado.
- **Não comprovado E2E:** nenhuma campanha real completa foi validada nesta auditoria.

Um item pode estar **presente + testado em unidade**, mas **não comprovado E2E**. As notas **7,5/10 engenharia e 4,5/10 visão** são avaliações qualitativas da V6, **não medições automatizadas** nem comparação de commits posteriores.

## 18. O que já existe e merece ser aproveitado

| Área | Situação no código V6 | Qualidade da comprovação / ressalva |
|---|---|---|
| API e núcleo Go + Huma | Presente | Estrutura real e aproveitável |
| Campanhas, pausa e retomada | Presente | Fluxos e testes parciais; campanha externa E2E não registrada |
| PostgreSQL + pgx + migrações | Presente | Testes de integração relatados com PostgreSQL 16; alvo PG18 ainda sem prova completa na auditoria |
| Fila `SKIP LOCKED`, leases, retries | Presente | Testes de integração e cenários locais; verificar carga e interrupções reais |
| Coleta HTTP nativa em Go | Presente | Limites, políticas e testes locais evoluíram |
| SSRF do fetcher Go | Controles presentes | Verificação do IP de conexão e redirects; não generalizar a todos os browsers |
| `robots.txt` | Presente | Cobertura local e alterações recentes; testes de falhas de rede e redirects relevantes |
| Descoberta por links HTML | Presente | Ramificação básica; não é toda a investigação autônoma |
| Sitemap, RSS/Atom | Presente | Extração e tipos de aresta testados em unidade/integração |
| Menções sem hyperlink | Presente | Vocabulário genérico e telecom; qualidade da descoberta precisa de teste real |
| Descoberta via SearXNG | Adaptador presente | Uso de motores externos real ainda não demonstrado integralmente |
| Descoberta GitHub/Mwmbl/Software Heritage/Wayback | Adaptadores presentes | Amplitude e resultados reais variam; testar limites/acesso |
| CAS por SHA-256 | Presente | Bom alicerce; falta evolução plena de extrações versionadas |
| Busca exata, FTS e trigramas | Presente | Não equivale ainda a recuperação semântica/vetorial nem BM25 |
| Interface Svelte | Presente | Código em JavaScript; grafo/controles existem em diferentes níveis de maturidade |
| Claims e evidências estruturadas | Presente | Progresso real, mas avaliação factual reprova nos adversariais recentes |
| Clusters de origem | Presente | Fingerprints/agrupamento; **não prova independência autoral ou factual completa** |
| Retomada de campanhas e dedupe de consultas | Presente | Melhoria comprovada na versão precedente; fluxo autônomo ainda requer gatilho |
| Orçamento por tarefa | Presente e aprimorado | Reserva atômica; diferença entre bytes transferidos e arquivos únicos ainda existe |
| PDF digital básico | Presente | OCR/Docling/Tika ainda não integrados operacionalmente |
| Dossiê e relatório | Presente | Estrutura e avisos existem; conteúdo limitado pela qualidade dos claims |
| Agendamento | Presente em parte | Mudança real de documento e alertas confiáveis não comprovados |

**Ponto de equilíbrio:** o código atual não é um protótipo vazio e não deve ser descartado. Mas o núcleo de pesquisa profunda e a qualidade das conclusões ainda precisam alcançar a maturidade da infraestrutura.

## 19. O que ainda falta ao NIE para cumprir o escopo original

### 19.1 Autonomia completa

🟡 **Avanço posterior à V6:** ✅ rodadas automáticas limitadas, histórico e subperguntas determinísticas no ciclo, testados com fontes controladas. A autonomia completa, o planejamento por LLM e a suficiência medida nos casos reais permanecem pendentes.

**Estado V6:** existe `PlanAdaptiveLeads()` e a rota `/investigate` consegue introduzir novas consultas na fila, mas a passagem automática de uma rodada concluída para outra ainda não é o fluxo padrão. `PlanSubquestions()` com LLM não está ligado ao ciclo produtivo normal. Faltam critérios formais de suficiência, planejamento iterativo real, decisão de prosseguir e reavaliação automática após novas evidências.

**Estado-alvo:** uma campanha recebe um objetivo, divide o problema, executa pesquisas iniciais, avalia o resultado, identifica uma pergunta ainda em aberto e **inicia automaticamente** outra rodada. O sistema deixa rastros do que mudou e encerra com motivo transparente. A investigação deve continuar segura mesmo quando IA local ou API externa estiver indisponível, degradando para regras documentadas.

### 19.2 Evidências confiáveis (bloqueador central)

🟡 **Avanço posterior à V6:** ✅ regressões adversariais V6 passam; contradições e revisões de claims são persistidas. A correção factual abrangente ainda exige avaliação e aceite.

**Estado V6:** vários defeitos antigos foram corrigidos, mas a auditoria V6 reproduziu **quatro falhas adversariais**: suporte de um fabricante atribuído a outro; negação de BGP atribuída a OSPF; negação de PPPoE atribuída a BGP; e conflito explícito transformado em incompatibilidade. Eram frases sintéticas criadas para testar o algoritmo, **não declarações factuais sobre equipamentos**.

**Estado-alvo:** relações sujeito-predicado-objeto atômicas, polaridade correta por recurso, escopo aplicado ao **mesmo trecho** e conflito preservado. Um conjunto de testes positivos, negativos, multilíngues e adversariais deve demonstrar que o NIE **não promove informação sem comprovação**.

### 19.3 Git Eye profundo

**Estado V6:** descoberta de repositórios e alguns READMEs via GitHub, além de busca local com melhor rastreabilidade de commit. **Não há integração operacional comprovada com Zoekt, busca abrangente no conteúdo Git remoto ou GitLab.**

**Estado-alvo:** fontes Git externas + recuperação seletiva de repositórios relevantes + indexação de arquivos, tags e versões, com caminho, linha, hash de commit e contexto. Zoekt entra quando houver corpus próprio e avaliação de ganho; nunca deve receber a missão impossível de “buscar todo o GitHub” sem possuir o corpus.

### 19.4 Crawlers maduros realmente aproveitados

**Estado V6:** o worker “Scrapy” aproveita primitivas da biblioteca, mas não um spider completo com motor Scrapy; o worker “Crawl4AI” usa Playwright direto, sem importar Crawl4AI. O navegador real não passou por validação integral E2E nem por isolamento de egress demonstrado. A escolha de executor é em grande parte heurística por URL.

**Estado-alvo:** integrar a funcionalidade madura comprovadamente necessária ou nomear corretamente o que já existe. Scrapy para páginas e navegação HTTP recursiva; Crawl4AI para renderização, deep crawl, scoring e JS quando gerar benefício; um único orquestrador canônico controlando ambos. Navegador somente com isolamento de rede testado.

### 19.5 Documentos complexos

🟡 **Avanço posterior à V6:** ✅ OCR opcional com Poppler/Tesseract e extrações versionadas foram testados com PDF escaneado sintético. Docling, Tika, qualidade em manuais reais e isolamento do parser continuam pendentes.

**Estado V6:** leitura de PDFs digitais e referências textuais; arquivo sem texto pode ser preservado para processamento posterior. **Não há Docling, Apache Tika nem OCR completos no pipeline**, e a API atual de reprocessamento é limitada quando o documento já tem passagens.

**Estado-alvo:** extração seletiva de layout, tabelas, páginas, legendas, notas, URLs de anotações embutidas, OCR e bibliografia, com parser versionado, confiança da extração e preservação do original. Reprocessamento deve produzir `ExtractionRun` novo, não substituir a extração anterior.

### 19.6 Busca híbrida completa

**Estado V6:** exata + FTS + trigramas, adequadas para um MVP lexical, mas **sem pgvector/embeddings operacionais demonstrados**. Não se deve chamar essa combinação de “busca semântica completa”.

**Estado-alvo:** suporte opcional a busca semântica, recuperação por código e reordenação de evidências com métricas comparativas. Busca exata em CLI continua prioritária mesmo se vetores forem adicionados.

### 19.7 Biblioteca permanente madura

🟡 **Avanço posterior à V6:** ✅ extrações e claims históricos, verificação de originais e restauração conjunta com Restic foram testados em ambiente isolado. Revisitas, detecção de mudanças e validação operacional em escala continuam pendentes.

**Estado V6:** armazenamento por hash, versões de ocorrência, trechos e clusters; sem histórico completo de extrações versionadas; migrações podem ter versões divergentes entre instalações; o controle de bytes precisa distinguir acervo de tráfego; não há backup/restauração comprovados.

**Estado-alvo:** captura íntegra, dedupe por conteúdo e origem, extrações versionadas, migrações imutáveis e reconciliáveis, logs de decisões, backup restaurável e revisitas que preservem versões antigas.

### 19.8 Qualidade e provas de funcionamento

🟡 **Avanço posterior à V6:** ✅ testes de store, API, worker e backup/restauração passaram em PostgreSQL 18.6 isolado. A campanha pública E2E, o browser isolado e o benchmark real continuam pendentes.

**Estado V6:** testes unitários e de integração limitados demonstram avanços importantes. A última auditoria executou **9/9 testes Python**, testes Go isolados existentes, e observou **4/4 novos testes adversariais falhando**. **Não houve execução verificada da suíte Go completa com toolchain exigido, PostgreSQL 18, navegador real ou campanha externa completa.**

**Estado-alvo:** CI reproduzível, testes de contratos Go↔Python↔PostgreSQL, provas E2E com fontes permitidas, benchmarks de 20 problemas técnicos e casos fora de telecom, testes de migração/restore, varreduras de segurança, carga e falhas.

### 19.9 Operação contínua e interface

**Estado V6:** interface para campanhas, busca e visualização; agendamentos e algumas opções de controle. Falta observar e comprovar mudanças de documentos, progresso adaptativo de várias rodadas, inteligência de prioridade e motivos de cada decisão no grafo. Caddy/Restic e controles completos de backup ainda não foram demonstrados.

**Estado-alvo:** gestão visual das campanhas, grafo com proveniência, alertas por mudança relevante, diff de documentos/release, limites visuais, papéis/autorizações quando houver multiusuário, healthcheck e recuperação operacional.

## 20. O risco arquitetural que não podemos perder de vista

É possível construir **uma plataforma com muitos módulos, boas telas e dezenas de testes, mas sem a capacidade de investigar de verdade**. O NIE não será julgado pela quantidade de adapters nem pelo número bruto de URLs baixadas.

Seu diferencial deverá ser medido por um percurso que qualquer auditor possa repetir:

> Assunto inicial → fontes externas não indicadas → links/menções/citações → nova pergunta gerada a partir do que foi encontrado → nova busca realmente executada → documentos e versões preservados → claims que suportam/contestam o resultado → relatório reproduzível → novas revisitas quando houver mudança.

Se esse ciclo falhar, a visão de produto ainda não foi entregue, mesmo que cada serviço isolado passe nos testes.

---

# PARTE V — O QUE SIGNIFICA ATINGIR 10/10

## 21. 10/10 em qualidade de engenharia: dez critérios de excelência

**Nota máxima não significa ausência garantida de qualquer bug futuro.** Significa que o projeto satisfaz um padrão público e reproduzível de engenharia para seu escopo e risco, sem bloqueadores conhecidos de gravidade crítica. Nenhum item abaixo será aprovado somente por README ou teste com mock.

| # | Dimensão | Condição necessária para avaliação 10/10 |
|---|---|---|
| E1 | **Arquitetura e modularidade** | Contratos claros, módulos com responsabilidades não sobrepostas, adaptadores substituíveis e PostgreSQL como fonte de verdade |
| E2 | **Integridade e modelagem de dados** | Dados canônicos coerentes, snapshots e extrações versionados, migrações não destrutivas testadas em bases já existentes, índices reconstruíveis |
| E3 | **Segurança operacional** | Proteções SSRF/egress para todos os workers, navegador isolado, autenticação adequada, segredos protegidos, política de fontes e testes adversariais |
| E4 | **Testabilidade e reprodutibilidade** | Build determinístico, versões fixadas, CI com testes unitários, integração com PG18, contratos entre serviços e E2E real |
| E5 | **Correção do motor factual** | Nenhum falso “aplicável/validado” nos casos críticos de aceite; escopo, negação, origem e contradição vinculados corretamente às evidências |
| E6 | **Concorrência e resiliência** | Fila idempotente, leases, recuperação de falhas, cancelamento seguro, migrações concorrentes e orçamentos consistentes |
| E7 | **Eficiência mensurada** | CPU/RAM/latência e throughput conhecidos para perfis definidos; serviços pesados ativados por benefício real e sem gargalos ocultos |
| E8 | **Observabilidade e operação** | Logs estruturados, métricas, alertas, trilha por campanha, diagnósticos, healthchecks e gestão de incidentes |
| E9 | **Backup, restauração e upgrades** | Restauração testada de documentos + banco, política de retenção, rollback de releases e atualização segura a partir de versão antiga |
| E10 | **Manutenção e governança** | Dependências/licenças auditadas, documentação fiel ao código, ADRs, gestão de mudanças, testes contra regressões e processo de release |

### 21.1 Provas exigidas em engenharia

**P0 bloqueia nota máxima.** Reprovação de evidência aplicável a firmware incorreto, bypass de SSRF, perda de arquivos originais, alteração destrutiva de claims ou impossibilidade de recuperar banco não são defeitos cosméticos. Também impedem afirmar 10/10: CI verde só com mocks, dependências marcadas como “integradas” sem uso real, falhas de persistência ignoradas e ausência de restore comprovado.

### 21.2 Metas mensuráveis a pactuar, sem inventar benchmarks

A equipe deve definir, antes da validação: corpora representativos, tamanho do acervo, volume de consultas, perfis de hardware, carga de crawlers, SLO de busca, taxa de erro, disponibilidade, RPO/RTO e custo de modelos. **Não há, até a V6, medições suficientes para prometer valores universais de RAM, latência, recall ou milhões de páginas por dia.** Publicar metas só depois de executar uma linha de base com dados reais.

## 22. 10/10 na realização da visão: dez capacidades completas

| # | Capacidade | Demonstração de aceite para 10/10 |
|---|---|---|
| V1 | **Assunto sem URLs fornecidas** | Encontrar fontes úteis não indicadas diretamente em uma campanha de tema novo |
| V2 | **Expansão real em árvore** | Percorrer várias profundidades, registrar arestas e convergência, respeitar limites e evitar loops |
| V3 | **Autonomia adaptativa multirrodada** | Detectar lacuna, formular pergunta derivada, executar nova busca e reavaliar sem chamada manual a cada rodada |
| V4 | **Multifuente genuíno** | Combinar web, Git, PDF, históricos e fóruns permitidos, com diferentes mecanismos e falhas isoladas |
| V5 | **Biblioteca acumulativa** | Reutilizar material de campanhas anteriores e preservar versões, originais e descobertas sem perda |
| V6 | **Evidências corretas e auditáveis** | Cada conclusão significativa tem proposição, escopo, citação exata e estado de validação defensável |
| V7 | **Independência e contradição** | Cópias não contam como confirmações independentes; conflitos permanecem explícitos |
| V8 | **Domínio técnico profundo e generalidade** | Resolver casos difíceis de telecom sem misturar firmwares; executar também casos fora de telecom |
| V9 | **Pesquisa visual e controlável** | Usuário acompanha progresso, grafo, custos, motivos, fontes e pode alterar limites sem editar código |
| V10 | **Continuidade e evolução** | Revisitar fontes, detectar mudanças, atualizar conclusões e alertar sem corromper relatórios históricos |

**Nota máxima de visão não significa:** achar tudo na internet, inventar fatos ausentes nas fontes, ser infalível, baixar arquivos privados ou ganhar de qualquer buscador em qualquer consulta. Significa entregar a promessa definida **dentro de uma cobertura, infraestrutura e política de acesso explicitadas**.

## 23. Como comprovar as dez capacidades numa única jornada

**Teste integrado exemplar: Huawei NE8000 e continuidade de sessões PPPoE** (ou outro problema de mesma dificuldade com base de referência).

1. Iniciar sem URL-semente, com fontes permitidas e orçamento finito.
2. Descobrir manual/fórum/repositório externo e identificar sua origem.
3. Seguir ao menos uma referência de segundo nível e uma menção **sem link**.
4. Localizar um PDF ou README técnico com trecho, página/linha, versão e URL.
5. Formular uma subpergunta nova baseada em lacuna descoberta **e executá-la automaticamente**.
6. Recuperar resultados de Git ou arquivo histórico quando relevantes, não só notícias ou o primeiro buscador.
7. Separar BGP, RADIUS, HA do BNG e preservação PPPoE, sem inferir equivalências.
8. Preservar original, extração e documentos derivados sob identidades distintas e reconstruir a árvore de origem.
9. Apresentar evidências e contradições com escopo, evitando afirmar recurso não comprovado.
10. Pausar/reiniciar/retomar e emitir dossiê que permita reproduzir cada conclusão.

Executar ainda um teste equivalente **fora de telecomunicações**, para demonstrar que a inteligência da plataforma é genérica, não apenas uma lista de palavras de roteadores.

## 24. Benchmark de qualidade: o que medir antes de falar em excelência

O Plano Mestre original definiu **20 questões técnicas de telecom**. Preservamos a bateria e acrescentamos avaliações adversariais e gerais.

### 24.1 Vinte temas de referência preservados

| # | Tema | Falha que o teste deve detectar |
|---|---|---|
| 1 | Huawei NE8000: BGP versus continuidade de sessões PPPoE | Confundir encaminhamento com estado de assinante |
| 2 | Huawei NE8000: BNG hot backup por modelo/VRP | Inventar suporte por release |
| 3 | Huawei NE8000: RADIUS redundante versus accounting | Confundir autenticação com estado operacional |
| 4 | Huawei NE8000: route-policy multihomed | CLI/contexto inadequados |
| 5 | MikroTik RouterOS v6 versus v7: BGP | Mesclar sintaxe de versões |
| 6 | MikroTik PPPoE concentrator: failover e statefulness | Inferir preservação de sessão por VRRP |
| 7 | RouterOS 7: PBR com routing rules versus mangle | Misturar caminhos de configuração |
| 8 | Cisco: route-map/PBR por plataforma | Equivalência indevida entre famílias |
| 9 | Datacom DmOS: política BGP por release | Aplicar documentação de release diferente |
| 10 | Datacom: SNMP/MIB por família | MIB sem correspondência de hardware |
| 11 | Huawei MA5800: perfis GPON por versão | CLI de geração inadequada |
| 12 | ZTE C300/C320 versus C600 | Confundir gerações de OLT |
| 13 | FiberHome: provisionamento GPON por firmware | Comando descontextualizado |
| 14 | V-SOL: perfis VLAN/ONU por modelo | Aplicabilidade indevida |
| 15 | Intelbras, Digistar, TP-Link, Furukawa | Mesclar fabricantes e famílias |
| 16 | SonicWall: PBR por firmware | Regra ou ordem de avaliação errada |
| 17 | Hillstone StoneOS: PBR/BGP | Ignorar precedência de políticas |
| 18 | MPLS L3VPN e BGP/VRF | Exemplo sem contexto de protocolo |
| 19 | Comunidades BGP e engenharia de tráfego | Confundir efeito pretendido e provado |
| 20 | Manual técnico com URL movida ou removida | Perder versão histórica ou citar captura errada |

**Esses itens não são resultados comprovados**; são perguntas de avaliação a responder com fontes verificadas por especialistas.

### 24.2 Outros ensaios obrigatórios

- **Negação e escopo:** frases em PT/EN/ES, coordenação de recursos, outro fabricante na mesma página, outro firmware, fontes conflitantes e listas sem proposições.
- **Independência:** um mesmo manual publicado em três sites não gera três confirmações.
- **Corpus adversarial:** instruções maliciosas dentro de HTML/PDF/README, hashes adulterados, páginas com diferentes charset, arquivos truncados e PDF escaneado.
- **Crawling:** loops de calendário, milhões de parâmetros, redirects, robots 403/404/5xx, DNS rebinding, endereços internos, subrecursos do navegador e rate limits.
- **Resiliência:** queda de worker no meio do arquivo, lease expirado, campanha pausada, banco reiniciado, migração de schema e backup restaurado.
- **Busca:** expressão CLI com símbolos, termos exatos, consulta multilíngue e relatório de recall/precisão com documentos de referência.
- **Autonomia:** a segunda rodada precisa nascer dos documentos da primeira; não pode ser apenas uma lista estática de quatro perguntas.
- **Pesquisa geral:** no mínimo um conjunto representativo de assuntos não relacionados a telecom, com evidência de generalização.

### 24.3 Comparação justa

Comparar, em corpora e condições documentados: **(a) busca manual tradicional**, **(b) metabusca/índices sem investigação**, **(c) NIE com expansão autônoma**. Medir fonte útil descoberta, precisão de modelo/versão, capacidade de encontrar material raro, citações corretas, contradições detectadas, tempo de parede, custo de rede, tokens, armazenamento e revisitas.

**Falsas confirmações críticas não podem ser escondidas por uma média alta.** A avaliação precisa registrar erros por gravidade, e o relatório de teste deve indicar exatamente quais modelos, fontes e versões sustentam as conclusões.

## 25. Como evoluir de 7,5/10 e 4,5/10 para 10/10

| Eixo | O que já favorece a nota atual | O que falta para excelência |
|---|---|---|
| **Engenharia (7,5/10)** | Boa base Go/Postgres, fila persistente, API, storage CAS, testes locais, estrutura modular | Provas E2E, segurança plena do browser, claims confiáveis, migrações seguras, versionamento de extrações, backup/restore, observabilidade, benchmark de carga e CI reproduzível |
| **Visão (4,5/10)** | Busca e coleta, links recursivos, fontes adicionais, grafo e dossiês básicos | Planejamento adaptativo automático, múltiplas rodadas, Git Eye profundo, documentos complexos, evidência semântica de escopo, memória cumulativa madura e acompanhamento de mudanças |

Não é necessário desenvolver todas as expansões opcionais para conquistar 10/10. É necessário **comprovar todas as promessas assumidas para a versão avaliada**, com documentação e limites honestos. Se o objetivo da versão for o NIE completo descrito neste plano, a régua será consideravelmente mais alta que a do MVP.


---

# PARTE VI — EVOLUÇÃO DO PRODUTO SEM PERDER O ESCOPO

## 26. Marcos de maturidade do NIE

O roteiro de evolução não deve ser interpretado como “instalar todos os componentes de cada tabela”. O avanço é feito por **capacidades demonstradas**, mantendo a infraestrutura tão pequena quanto possível para cada etapa.

### Marco A — Pesquisador funcional e seguro

🟡 **Progresso:** ✅ fila com recuperação e lease protegido, testes com PostgreSQL 18 e download verificável do original. ⬜ Aceite completo: campanha pública E2E e segurança de todos os executores.

**Experiência:** criar campanha com ou sem seeds quando uma fonte externa estiver habilitada; descobrir URLs, seguir links, capturar HTML/PDF digital, guardar originais por hash, pesquisar trechos e recuperar relatórios com fontes.  
**Reuso dominante:** Go/Huma/PostgreSQL/Svelte + APIs externas + SearXNG + captura HTTP e Scrapy seletivo.  
**Aceite:** fluxo ponta a ponta reproduzível, segurança de rede básica, recuperação de fila, teste com banco real, documento acessível e citação verificável.

### Marco B — Investigação inteligente verificável

🟡 **Progresso:** ✅ rodada automática por lacunas, subperguntas determinísticas depois que as lacunas fecham, e registro de contradições testados em cenário controlado. ⬜ Aceite completo: casos reais avaliados e suficiência que não seja só heurística de termos.

**Experiência:** separação automática de subproblemas, comparação de fontes, claims atômicas, versões, incertezas, contradições e investigação baseada em lacunas.  
**Reuso dominante:** motores de IA/modelos conectáveis, componentes selecionados de GPT Researcher/Hyperresearch se pilotos comprovarem vantagem, extração Docling/Tika para os documentos que exigirem.  
**Aceite:** casos reais difíceis de telecom e casos gerais, com nenhuma falsa confirmação crítica no conjunto de aceite. A segunda rodada é executada automaticamente.

### Marco C — Biblioteca permanente madura

🟡 **Progresso:** ✅ extrações versionadas, OCR sintético, backup/restauração isolados e detecção de mudança por hash com alerta. ⬜ Aceite completo: diff textual, revisita em produção e reavaliação limitada aos claims citados.

**Experiência:** documentos versionados, arquivos históricos, revisitas, comparação entre versões, OCR, extrações reprocessáveis, cluster de fontes, busca semântica opcional, monitoramento de alterações e backups restauráveis.  
**Reuso dominante:** ArchiveBox seletivo, Tika/Docling/OCR, Restic e adaptadores Wayback/Common Crawl/Software Heritage.  
**Aceite:** uma fonte muda e a plataforma detecta diferença, preserva a versão anterior, reavalia afirmações afetadas e recupera o acervo em teste de restauração.

### Marco D — Git Eye especializado e busca avançada

🟡 **Progresso:** há busca Git local e descoberta parcial. ⬜ Aceite completo de conteúdo remoto, versões e indexação seletiva ainda pendente.

**Experiência:** pesquisar arquivos de configuração, código, READMEs, commits e versões em repositórios externos e no acervo selecionado, conectando resultados a documentação e afirmações.  
**Reuso dominante:** GitHub/GitLab/APIs, ecosyste.ms/Software Heritage, Zoekt quando o corpus justificar.  
**Aceite:** encontrar exemplos reais de configuração por arquivo, linha e revisão imutável; diferenciar working tree de commit; indexar somente corpus útil.

### Marco E — Autonomia contínua e operacionalização

🟡 **Progresso:** ✅ decisões de rodadas e proteção de lease registradas e testadas. ⬜ Aceite completo de operação contínua, alertas e controles permanece pendente.

**Experiência:** campanhas agendadas, replanejamento de várias rodadas, alertas, controle visual fino, orçamentos, observabilidade e manutenção segura.  
**Reuso dominante:** workers especializados, métricas, banco, restic e ferramenta de observabilidade proporcional ao uso.  
**Aceite:** dias de operação com campanhas controladas, reinicializações sem perda de estado, mudanças reais detectadas e qualidade sustentada.

### Marco F — Escala distribuída, apenas por demanda

⬜ **Aceite pendente:** ainda não há benchmark que comprove ganho de escala distribuída.

**Experiência:** maior corpus, múltiplos workers e nós, armazenamento de vários TB ou mais, buscas rápidas com índices especializados, políticas de qualidade e custo em escala.  
**Reuso candidato:** StormCrawler **ou** Nutch; OpenSearch/Zoekt; storage compatível com S3; novas fontes opcionais.  
**Aceite:** ganhos mensurados de qualidade/throughput que superem o custo adicional, sem índices duplicados e sem degradação da rastreabilidade.

## 27. Ordem de dependências entre capacidades

```text
Segurança + dados canônicos + contratos
                  │
                  ▼
Busca/descoberta/coleta E2E verificável
                  │
                  ▼
Extração adequada + citações rastreáveis
                  │
                  ▼
Proposições e evidências confiáveis
                  │
                  ▼
Replanejamento automático por lacunas
                  │
                  ▼
Memória acumulativa + revisitas + monitoramento
                  │
                  ▼
Índices/crawlers distribuídos quando necessário
```

É possível trabalhar em módulos paralelos, mas **não há sentido em declarar concluída uma investigação inteligente se suas fontes, dados ou evidências não forem confiáveis**. O valor do produto vem da integração end-to-end, não da sequência de versões no README.

## 28. O que fica fora do escopo principal até decisão explícita

- Cópia integral de toda a internet ou de todo o Git público.
- Indexação indiscriminada de milhões de links sem filtro de pertinência e política de acesso.
- Acesso a sites privados sem autorização, quebra de autenticação, paywall, CAPTCHA ou outros controles.
- Varredura massiva de serviços `.onion` ou DHT BitTorrent sem justificativa legítima e políticas específicas.
- Execução automática de comandos em equipamentos de produção.
- Treinar um modelo de linguagem do zero com o acervo.
- Substituir o PostgreSQL como fonte de verdade por uma plataforma RAG fechada ou índice descartável.
- Instalação permanente de todos os motores de crawling, agentes, bancos de busca e chatbots disponíveis.
- Reescrita de Scrapy, Crawl4AI, Zoekt, Tika, Docling, OpenSearch ou GPT Researcher apenas para evitar dependência — salvo medição/necessidade específica.
- Compromissos de throughput, latência, número de páginas ou desempenho de IA sem benchmarks do ambiente real.

## 29. Como reconhecer que o NIE está realizando o projeto que imaginamos

A experiência desejada é a seguinte:

> O usuário escreve um problema complexo e seleciona um nível de profundidade. O NIE compreende o objetivo, registra os requisitos, inicia pesquisas em diferentes fontes públicas, descobre um documento pouco conhecido e segue suas referências. Uma referência em PDF revela outro termo técnico, que vira uma busca nova; um repositório Git traz uma configuração que aponta para uma release diferente. O sistema verifica as versões, registra a origem de cada afirmação, percebe um conflito e busca a fonte primária. Não encontrando comprovação suficiente, responde explicitamente que aquela parte continua incerta. Os documentos ficam no acervo; o grafo mostra como foram encontrados; semanas depois, uma nova publicação desencadeia atualização e reavaliação, sem destruir o resultado histórico. Tudo isso opera dentro de limites, permissões e custos conhecidos.

Este comportamento — **descobrir, aprofundar, confrontar, preservar, aprender com pesquisas anteriores e informar incertezas** — é o coração do produto.

---

# PARTE VII — CATÁLOGO HISTÓRICO INTEGRAL DE SOFTWARES, FONTES E PRECEDENTES

## 30. Como interpretar o catálogo

A seguir está o inventário de **59 registros** do `MATRIZ_59_FONTES_E_PROJETOS_V2_1.csv`, originalmente compilado a partir do arquivo mestre e do backup histórico. O inventário foi preservado **sem transformar todos os itens em dependências**.

- **Software instalável:** pode ser usado como serviço/biblioteca, sujeito a licença e integração.
- **Fonte externa/API:** consultamos, mas não precisamos instalar nem copiar sua base completa.
- **Alternativa/precedente:** útil como referência, sem aprovação para executar no NIE.
- **Conteúdo/documentação:** fóruns e bases públicas, não “bibliotecas de software acopladas”.
- **Estado V2.1:** é a decisão **histórica de destino**, não uma declaração de que o item já foi integrado ao código V6.

Os registros abaixo preservam os nomes e URLs do catálogo. Alguns representam o **mesmo ecossistema** em interface e repositório, e suas licenças/condições de uso devem ser verificadas na versão exata adotada.

### 30.1 Git, código, repositórios e fontes de software

| # | Projeto ou fonte | Tipo no inventário | Destino decidido na V2.1 |
|---:|---|---|---|
| 1 | [Zoekt](https://github.com/sourcegraph/zoekt) | Motor de índice Git | Integrar em Git Eye de forma seletiva |
| 2 | [OpenGrok](https://github.com/oracle/opengrok) | Motor Git alternativo | Piloto opcional |
| 3 | [Sourcegraph](https://sourcegraph.com/) | Plataforma comercial/serviço | Referência externa |
| 4 | [Sourcebot](https://github.com/sourcebot-dev/sourcebot) | Ferramenta de busca Git | Não usar como núcleo aberto |
| 5 | [Livegrep](https://github.com/livegrep/livegrep) | Motor Git alternativo | Não instalar inicialmente |
| 6 | [Hound](https://github.com/hound-search/hound) | Motor Git alternativo | Não instalar inicialmente |
| 7 | [Software Heritage](https://www.softwareheritage.org/) | Acervo público externo | Integrar via adaptador |
| 8 | [Software Heritage Archive](https://archive.softwareheritage.org/) | Interface do acervo | Integrar via adaptador |
| 9 | [ecosyste.ms](https://ecosyste.ms/) | Fonte de metadados | Integrar via adaptador |
| 10 | [grep.app](https://grep.app/) | Busca de código pública | Avaliar API/limites |
| 11 | [GitHub Code Search](https://github.com/search?type=code) | Busca pública | Integrar como fonte |
| 12 | [GitHub REST Search](https://docs.github.com/en/rest/search/search) | API pública | Integrar como fonte |
| 13 | [GitLab Search](https://docs.gitlab.com/) | API pública | Integrar como fonte |
| 14 | [GH Archive](https://www.gharchive.org/) | Dataset de eventos | Opcional |
| 15 | [World of Code](https://worldofcode.org/) | Corpus de pesquisa | Estudar como fonte |

### 30.2 Web, motores, crawling, fontes históricas e redes opcionais

| # | Projeto ou fonte | Tipo no inventário | Destino decidido na V2.1 |
|---:|---|---|---|
| 16 | [SearXNG](https://github.com/searxng/searxng) | Metabusca | Integrar no MVP |
| 17 | [Mwmbl](https://github.com/mwmbl/mwmbl) | Índice web independente | Integrar como fonte |
| 18 | [Mwmbl Search](https://mwmbl.org/) | Interface de busca | Unificar com Mwmbl |
| 19 | [Common Crawl](https://commoncrawl.org/) | Corpus web histórico | Integrar sob demanda |
| 20 | [Wayback Machine](https://web.archive.org/) | Arquivo histórico | Integrar via adaptador |
| 21 | [Scrapy](https://github.com/scrapy/scrapy) | Crawler recursivo | Promover a motor principal de crawling |
| 22 | [Crawl4AI](https://github.com/unclecode/crawl4ai) | Crawler inteligente/extrator | Integrar sob demanda |
| 23 | [Apache Nutch](https://github.com/apache/nutch) | Crawler massivo | Escala futura |
| 24 | [Apache StormCrawler](https://github.com/apache/stormcrawler) | Crawler contínuo distribuído | Preservar como eixo de expansão |
| 25 | [Heritrix](https://github.com/internetarchive/heritrix3) | Crawler de arquivamento | Opcional especializado |
| 26 | [YaCy](https://yacy.net/) | Buscador com crawler próprio | Experimento opcional |
| 27 | [Crawlee](https://github.com/apify/crawlee) | Crawler JS/TS | Alternativa a Scrapy/Crawl4AI |
| 28 | [Ahmia crawler](https://github.com/ahmia/ahmia-crawler) | Crawler onion | Fase opcional restrita |
| 29 | [bitmagnet](https://github.com/bitmagnet-io/bitmagnet) | Índice de metadados DHT | Fase opcional restrita |

### 30.3 Documentos, índices, modelos, agentes e plataformas

| # | Projeto ou fonte | Tipo no inventário | Destino decidido na V2.1 |
|---:|---|---|---|
| 30 | [Apache Tika](https://github.com/apache/tika) | Parser universal | Integrar sob demanda |
| 31 | [Docling](https://github.com/docling-project/docling) | Parser de documentos | Integrar sob demanda |
| 32 | [ArchiveBox](https://github.com/ArchiveBox/ArchiveBox) | Arquivamento | Integrar seletivamente |
| 33 | [OpenSearch](https://github.com/opensearch-project/OpenSearch) | Índice documental | Escala conforme benchmark |
| 34 | [Apache Solr](https://solr.apache.org/) | Índice alternativo | Não instalar inicialmente |
| 35 | [PostgreSQL](https://www.postgresql.org/) | Banco transacional | Núcleo obrigatório |
| 36 | [MinIO](https://github.com/minio/minio) | Storage de objetos | Não escolher repositório arquivado para novo núcleo |
| 37 | [Qdrant](https://github.com/qdrant/qdrant) | Motor vetorial | Opcional sob benchmark |
| 38 | [Paperless-ngx](https://github.com/paperless-ngx/paperless-ngx) | Gestor documental | Não instalar inicialmente |
| 39 | [RAGFlow](https://github.com/infiniflow/ragflow) | RAG/plataforma | Piloto competitivo, não obrigatório |
| 40 | [Onyx](https://github.com/onyx-dot-app/onyx) | Pesquisa/RAG corporativo | Alternativa de RAGFlow |
| 41 | [Open WebUI](https://github.com/open-webui/open-webui) | Interface para LLM | Opcional |
| 42 | [AnythingLLM](https://github.com/Mintplex-Labs/anything-llm) | Interface de conhecimento | Opcional alternativo |
| 43 | [GPT Researcher](https://github.com/assafelovic/gpt-researcher) | Agente de deep research | Piloto para motor de investigação |
| 44 | [Hyperresearch](https://github.com/jordan-gibbs/hyperresearch) | Research harness e biblioteca | Piloto de evidência/qualidade |
| 45 | [LangChain Open Deep Research](https://github.com/langchain-ai/open_deep_research) | Projeto histórico | Somente referência de arquitetura |

### 30.4 Precedentes comerciais e fontes públicas de conteúdo

| # | Projeto ou fonte | Tipo no inventário | Destino decidido na V2.1 |
|---:|---|---|---|
| 46 | [Diffbot](https://www.diffbot.com/technology) | Plataforma comercial | Precedente, não núcleo open source |
| 47 | [Juniper Marvis](https://www.juniper.net/) | Produto comercial | Concorrência/benchmark |
| 48 | [Cisco AI](https://www.cisco.com/) | Produto comercial | Concorrência/benchmark |
| 49 | [Cisco Community](https://community.cisco.com/) | Fonte de conteúdo | Conector de fonte permitido |
| 50 | [MikroTik Forum](https://forum.mikrotik.com/) | Fonte de conteúdo | Conector de fonte permitido |
| 51 | [Viva o Linux](https://www.vivaolinux.com.br/) | Fonte de conteúdo | Conector de fonte permitido |
| 52 | [Stack Overflow](https://stackoverflow.com/) | Fonte de conteúdo | API/extração permitida |
| 53 | [Network Engineering Stack Exchange](https://networkengineering.stackexchange.com/) | Fonte de conteúdo | API/extração permitida |
| 54 | [Server Fault](https://serverfault.com/) | Fonte de conteúdo | API/extração permitida |
| 55 | [RFC Editor](https://www.rfc-editor.org/) | Fonte documental | Conector prioritário |
| 56 | [RIPE NCC](https://www.ripe.net/) | Fonte documental | Conector prioritário |
| 57 | [NANOG](https://nanog.org/) | Fonte documental | Conector prioritário |
| 58 | [FRRouting](https://frrouting.org/) | Projeto técnico/fonte | Conector prioritário |
| 59 | [Autonomous Research Agent (experimental)](https://github.com/florian-grassl/autonomous-research-agent) | Precedente experimental | Somente avaliar após identificar URL exata |


## 31. Tecnologias adicionais mencionadas historicamente ou introduzidas na arquitetura V2

O inventário de 59 itens **não esgota todos os nomes** mencionados nos vários relatórios. Estes são candidatos, dependências transversais ou exemplos que merecem registro para não desaparecer do histórico. **Nenhum deve ser instalado automaticamente por estar nesta tabela.**

| Tecnologia/fonte | Papel possível | Decisão para NIE |
|---|---|---|
| [Go](https://go.dev/) | Linguagem do núcleo | **Adotada no código** |
| [Huma](https://github.com/danielgtaylor/huma) | Contratos/HTTP/OpenAPI | **Adotada no código** |
| [pgx](https://github.com/jackc/pgx) | Driver/PostgreSQL Go | **Adotado no código** |
| [Svelte](https://svelte.dev/) + [Vite](https://vite.dev/) | Interface e build | **Adotados**; frontend em JavaScript hoje |
| [TypeScript](https://www.typescriptlang.org/) | Tipos/contratos do frontend | Alvo gradual, ainda não presente em `web/src` na V6 |
| [FastAPI](https://github.com/fastapi/fastapi), [Uvicorn](https://www.uvicorn.org/), [Pydantic](https://github.com/pydantic/pydantic) | RPC/HTTP dos workers Python | **Declarados no código** |
| [requests](https://github.com/psf/requests) + [BeautifulSoup](https://www.crummy.com/software/BeautifulSoup/) | HTTP/HTML dos workers | **Declarados e utilizados** |
| [Playwright](https://github.com/microsoft/playwright) | Navegador e renderização de JS | **Presente no worker**, não validado browser E2E na V6 |
| [Colly](https://github.com/gocolly/colly) | Alternativa crawler Go | Não é base obrigatória depois da V2.1; evitar reimplementar motor de crawling |
| [pgvector](https://github.com/pgvector/pgvector) | Vetores no PostgreSQL | Opcional após benchmark |
| [Tesseract OCR](https://github.com/tesseract-ocr/tesseract) | OCR local gratuito | Caminho opcional no pipeline de PDFs |
| [Restic](https://github.com/restic/restic) | Backup criptografado | Meta operacional antes de acervo valioso |
| [Caddy](https://github.com/caddyserver/caddy) | HTTPS e proxy reverso | Opcional para acesso remoto |
| [SeaweedFS](https://github.com/seaweedfs/seaweedfs) | Armazenamento de objetos / S3 | Candidato futuro, sujeito a avaliação |
| [NATS JetStream](https://docs.nats.io/nats-concepts/jetstream) | Fila dedicada | Só se PostgreSQL demonstrar gargalo de mensageria |
| [OpenTelemetry](https://opentelemetry.io/), [Prometheus](https://prometheus.io/) e [Grafana](https://grafana.com/) | Telemetria/observabilidade | Escalonar observabilidade conforme operação |
| [Meilisearch](https://github.com/meilisearch/meilisearch) | Índice textual/semântico alternativo | Backlog; não somar ao OpenSearch sem motivo |
| [Recoll](https://www.lesbonscomptes.com/recoll/) | Busca documental desktop/local | Alternativa de estudo, não núcleo |
| Open Semantic Search, `duct`, DocFinder, Sphinx Search e Gigablast | Alternativas históricas de pesquisa/indexação | Apenas referência; revalidar manutenção, API e adequação |
| Repowise | Busca de repositório com AST/semântica, conforme relato histórico | Candidato a pesquisa específica, **sem integração aprovada** |
| TorBot/TorCrawl/TorSpider/OnionSearch | Ferramentas de descoberta `.onion` citadas nos relatos | Opcional restrito; não aprovar alegações de afiliação/propriedade sem verificar |
| Jackett, Prowlarr, Torrent Search | Agregadores de metadados de indexadores | Backlog especializado; não equivalem a acesso legítimo ao conteúdo |
| SpiderFoot, GitHound | OSINT e descoberta especializada | Fora do núcleo de documentação; avaliar somente para caso autorizado específico |
| WARC/WAT/WET, RSS/Atom, sitemap, CDX | Formatos e índices de fontes históricas/web | **Protocolos/formato de integração**, não necessariamente serviços para instalar |

## 32. Decisão de acoplamento: o que teremos de fato rodando?

**Instalação de base desejada:** Go/Huma, Svelte/Vite, PostgreSQL, armazenamento CAS, fonte(s) de descoberta ativadas, worker HTTP seguro e backup/restauração. Os executores pesados e índices adicionais entram **sob demanda**.

**Extensão que mais aumenta a capacidade de descoberta sem reescrever motores:** Scrapy e Crawl4AI, em responsabilidades diferentes, com uma única fronteira PostgreSQL. A biblioteca Crawl4AI só conta como integração quando sua API real é chamada em pesquisas que justifiquem deep crawling ou JS.

**Extensão que mais fortalece o acervo documental:** Docling/Tika/OCR com versionamento de extrações e ArchiveBox seletivo quando útil.

**Extensão que mais fortalece Git Eye:** APIs Git e Zoekt sobre repositórios selecionados, mantendo URLs, arquivos, linhas e revisões verificáveis.

**Extensão que pode melhorar a investigação:** piloto de GPT Researcher e, separadamente, Hyperresearch; adotar somente se superarem o pipeline atual em qualidade e custo, sem criar uma segunda biblioteca canônica.

**Extensão de escala:** StormCrawler **ou** Nutch, nunca por obrigação. OpenSearch apenas quando a busca documental local justificar. YaCy, Ahmia, bitmagnet, Heritrix e índices alternativos permanecem opcionais.

---

# PARTE VIII — GOVERNANÇA DO DOCUMENTO E CRITÉRIOS DE VERDADE

## 33. Como usar este Plano Mestre sem reproduzir os erros anteriores

Este arquivo é a **referência de visão, capacidade e arquitetura**. Não é um relatório que declara tudo implementado, nem uma ordem para instalar todos os itens do catálogo.

As decisões devem distinguir permanentemente:

1. **Planejado:** necessário à visão ou candidato de evolução.
2. **Implementado:** existe código real e contrato operacional.
3. **Integrado:** participa de um fluxo completo, com entradas e saídas comprovadas.
4. **Testado:** passou por teste compatível com a afirmação feita (unidade, contrato, integração ou E2E).
5. **Validado operacionalmente:** houve ensaio real com fonte autorizada, segurança, limites e resultados revisáveis.

Um commit com o título “Crawl4AI integrado” não prova que o worker usa Crawl4AI. Um teste que injeta a resposta correta não demonstra que a pesquisa encontrou a fonte. Um hash sobre texto normalizado não prova fidelidade dos bytes originais. Três URLs não significam três evidências independentes. E quatro regras que passam não significam que um modelo compreende todas as negações possíveis.

## 34. Fontes documentais utilizadas para esta consolidação

### 34.1 Materiais internos do projeto

- `ARQUIVO_MESTRE.md` — histórico conceitual, decisões originais, anexos e links (preservação de 30/10/2026).
- `PLANO_MESTRE_NETWORK_INTELLIGENCE_ENGINE_V2.md` — base de stack leve, fontes de verdade, armazenamento, segurança, critérios e fases.
- `ADENDO_MESTRE_NIE_V2_1.md` — reintegração de Scrapy/Crawl4AI, escolha gradual de workers e catálogo histórico.
- `docs/arquitetura/MATRIZ_59_FONTES_E_PROJETOS_V2_1.csv` — **59 registros integralmente listados na Parte VII**.
- `nie-main (4).zip` — último estado de implementação analisado, sem diretório `.git`.
- `nie-main/README.md`, `nie-main/docs/DECISOES.md`, `nie-main/docs/MVP_SPEC.md`, `nie-main/go.mod`, `nie-main/web/package.json`, `nie-main/workers/*/requirements.txt` e arquivos Go/Python/SQL.
- `AUDITORIA_CRITICA_NIE_V6_2026-11-02.md` — último parecer técnico, quatro regressões adversariais e limites da validação.

**Nota de fidelidade:** parte dos arquivos históricos é compilação de outras IAs e contém alegações que não foram independentemente verificadas. Este documento preserva sua existência e papel na pesquisa, mas não converte números, licenças ou declarações de capacidade dessas fontes em verdades comprovadas. Ao integrar um projeto, verificar a versão e suas condições atuais.

### 34.2 Referências públicas úteis para validar decisões antes de implantação

- [Scrapy — documentação](https://docs.scrapy.org/en/latest/) — spiders, regras, middleware, crawling e retomada.
- [Crawl4AI — Deep Crawling](https://docs.crawl4ai.com/core/deep-crawling/) — BFS, Best-First, priorização e extração.
- [Apache StormCrawler](https://stormcrawler.apache.org/) — crawling contínuo distribuído.
- [Apache Nutch](https://nutch.apache.org/) — crawling em maior escala.
- [Zoekt](https://github.com/sourcegraph/zoekt) — busca textual/regex em código indexado.
- [Apache Tika](https://github.com/apache/tika) e [Docling](https://github.com/docling-project/docling) — extração de documentos.
- [ArchiveBox](https://github.com/ArchiveBox/ArchiveBox) — formatos e arquivamento seletivo.
- [GPT Researcher](https://github.com/assafelovic/gpt-researcher) e [Hyperresearch](https://github.com/jordan-gibbs/hyperresearch) — agentes/pipelines candidatos.
- [PostgreSQL](https://www.postgresql.org/docs/), [pg_trgm](https://www.postgresql.org/docs/current/pgtrgm.html) e [pgvector](https://github.com/pgvector/pgvector) — dados e busca.
- [SearXNG](https://docs.searxng.org/) — metabusca e integração externa.
- [RFC 9309](https://www.rfc-editor.org/rfc/rfc9309) — Robots Exclusion Protocol.
- [OWASP — prevenção de SSRF](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html) — referência para segurança de destinos.
- [Sourcebot: mudança de desenvolvimento a partir da v6 (08/10/2026)](https://www.sourcebot.dev/blog/sourcebot-is-now-closed-source) — cautela com dependência aberta de longo prazo.
- [MinIO: repositório público arquivado em 25/04/2026](https://github.com/minio/minio) — não defini-lo como novo núcleo obrigatório sem avaliar manutenção/alternativas.

### 34.3 Restrições da avaliação

As informações sobre o **estado da implementação** estão limitadas ao ZIP V6 e às verificações documentadas na última auditoria. Não há prova nesta consolidação de execução da aplicação inteira com Docker, PostgreSQL 18, browser JS e investigação multirrodada em fontes externas. Os testes novos da última auditoria apontaram falhas de escopo factual, mas isso não significa que todo o motor de busca ou a infraestrutura estejam inutilizáveis. Alterações feitas depois desse ZIP devem ser avaliadas separadamente.

## 35. Declaração de produto para nortear todas as evoluções

> **NIE é um investigador autônomo de conhecimento:** encontra fontes em múltiplos índices e na web aberta, percorre ramificações de links e referências, usa ferramentas open source maduras para coleta e extração, preserva documentos e versões, constrói relações, confronta proposições e gera conclusões auditáveis com incertezas explícitas. Funciona sobre um núcleo aberto e modular, inicia pequeno e escala por necessidade comprovada. Sua primeira especialidade é telecomunicações, mas sua arquitetura não pertence a um único domínio.

A decisão arquitetural central permanece: **não reinventar a roda; integrar, orquestrar, auditar e especializar**. O avanço até 10/10 virá da qualidade das investigações e das provas operacionais — não da contagem de softwares instalados, do tamanho do banco ou de relatórios que parecem confiantes.

---

**Fim do Plano Mestre V3 — visão, escopo, arquitetura, catálogo histórico completo, situação V6 e critérios de excelência.**
