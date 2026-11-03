# Execução do Plano Mestre V3

Referência de visão: [Plano Mestre V3](PLANO_MESTRE_NIE_V3_VISAO_ESCOPO_E_ARQUITETURA.md).
Registro de implementação e validação em 2 de novembro de 2026. O diagnóstico V6
do plano permanece como registro histórico; este arquivo descreve avanços posteriores.

## Entrega: ciclo automático limitado por regras

Ao terminar a coleta, a própria fila agenda um job `investigate`. Ele recupera
passagens, avalia claims, persiste evidências e propõe buscas para lacunas ou
contradições. Novas buscas alimentam a fila existente. Ao esvaziá-la novamente,
outra avaliação considera os documentos disponíveis naquele momento.

Decisão e enfileiramento são publicados em uma transação. Pausa, lease vencido
ou tentativa substituída impedem essa publicação. O bloqueio da campanha impede
duas avaliações simultâneas serem agendadas pelo reaper e pelos workers.
Tentativas repetidas não duplicam rodadas publicadas nem consultas.

As fontes derivam dos provedores que já possuem tarefas de descoberta na campanha
e continuam disponíveis no worker. Uma campanha apenas com seeds não adquire
automaticamente acesso a todos os provedores configurados no servidor.

Paradas registradas:

- `no_detected_gaps`: as regras não identificaram outra lacuna nos claims avaliados.
- `no_new_queries`: as consultas propostas já foram enfileiradas anteriormente.
- `no_discovery_provider`: nenhum provedor selecionado está disponível.
- `budget_exhausted`: teto de URLs, bytes armazenados ou pesquisas atingido.
- `round_limit`: limite de avaliações automáticas atingido.
- `evaluation_failed`: avaliação esgotou suas tentativas; detalhes na fila de jobs.

`NIE_AUTOMATIC_RESEARCH` é habilitado por padrão. `NIE_MAX_RESEARCH_ROUNDS` tem
padrão 3, inclui a avaliação final e é limitado ao intervalo de 0 a 10. Zero
desativa novas avaliações automáticas. Jobs já persistidos permanecem na fila.
O limite é da implantação; limites individuais por campanha ainda são evolução.

Histórico: `GET /api/campaigns/{id}/rounds` e seção **Rodadas automáticas** na UI.
Cada busca gerada carrega no payload o identificador do job de pesquisa e sua
rodada; motivos e consultas propostas ficam em `research_rounds`. A migração
`0008_research_rounds.sql` adiciona a tabela e o tipo de job sem alterar originais.

## Evidência de verificação

- Testes Go de unidade/regressão, incluindo os adversariais V6.
- Testes com PostgreSQL 16 real no WSL: migração, agendamento concorrente,
  pausa, lease expirado, tentativa antiga, idempotência e limites.
- Integração worker + PostgreSQL + descoberta HTTP controlada: uma evidência
  sem modelo/firmware produz uma busca por compatibilidade; o resultado vira
  tarefa de coleta; a inclusão controlada de uma evidência com escopo completo
  altera a decisão da próxima avaliação.
- Build do frontend Svelte/Vite.

O teste controlado registra documentos diretamente no store; não comprova o
download público, a extração de um manual real ou todo o percurso pelo browser.
Não há aqui validação operacional de PostgreSQL 18, browser isolado, qualidade de
fontes externas, carga contínua, nem o benchmark real das 20 questões.

## Entrega posterior: subperguntas no ciclo automático

A avaliação continua determinística e não chama LLM. `PlanSubquestions` passa a participar do ciclo. As subperguntas são gravadas em `research_questions` (migração `0012_research_questions.sql`) e expostas em `GET /api/campaigns/{id}/questions`, com a seção **Subperguntas** na campanha.

A rodada só envia uma subpergunta à descoberta quando os claims avaliados não produzem outra lacuna. Assim a busca por versão ou contradição não é adiada por uma lista fixa. Estados:

- `open`: ainda não encaminhada.
- `queued`: já existe consulta de descoberta com o mesmo texto normalizado.
- `addressed`: os termos distintivos da subpergunta, fora os da pergunta original, aparecem todos no mesmo trecho. Exige pelo menos dois termos. Isso não certifica que a subpergunta foi respondida.

Paradas novas, além das já registradas:

- `open_subquestions`: não havia lacuna de claim; a rodada enfileirou subperguntas abertas.
- `subquestions_searched`: as subperguntas já tinham sido enviadas e não surgiu lacuna nova.
- `subquestions_addressed`: a heurística de termos marcou todas as subperguntas; continua sendo heurística.

O ensaio com fonte HTTP controlada mostra a primeira rodada saindo da lacuna de release e a segunda, depois de um trecho com modelo e firmware, enfileirando as subperguntas. Não há campanha pública, planejamento por LLM, nem as 20 questões de referência.

## Entrega posterior: revisita e mudança de hash

Cada gravação de documento atualiza `source_observations` com a URL normalizada, o hash, ETag, `Last-Modified`, a primeira vez vista e a última verificação. Se o hash da mesma URL muda, o original anterior permanece, `document_diffs` registra os dois hashes e um alerta avisa a campanha. A migração é `0013_source_observations.sql`. O histórico sai em `GET /api/campaigns/{id}/changes`.

Uma resposta 304, com `If-None-Match` ou `If-Modified-Since`, só atualiza a última verificação. O job `revisit` faz essa leitura condicional. Ele só entra na fila quando `NIE_REVISIT_INTERVAL` é maior que zero; o padrão é desligado para não impedir o encerramento da campanha. O agendador então reabre campanhas concluídas, nunca as pausadas. Se a mudança ocorre com pesquisa automática ligada, uma avaliação `document_changed` é enfileirada enquanto ainda houver rodadas disponíveis.

Isso não mede diff textual de manual, não reavalia só os claims citados e não foi ensaiado com servidor público. A detecção é por SHA-256.

## Entrega posterior: GitLab como fonte de README

`NIE_GITLAB_ENABLED` ou `NIE_GITLAB_TOKEN` ligam o adaptador. A base padrão é `https://gitlab.com` (`NIE_GITLAB_URL` troca a instância). A busca usa `GET /api/v4/projects` e enfileira o README bruto em `/-/raw/<branch>/README.md`. O job é `discover_gitlab` (migração `0014_gitlab_discovery.sql`). Não baixa o repositório, não lê arquivos além do README e não cobre GitLab.com inteiro.

## Entrega posterior: adaptadores sob demanda

Ficaram ligados ao núcleo, cada um só quando a variável correspondente existe:

- Common Crawl CDX (`NIE_COMMONCRAWL_ENABLED`): busca por domínio ou URL, job `discover_commoncrawl`.
- Zoekt (`NIE_ZOEKT_URL`): `GET /api/search/zoekt` num índice já construído.
- OpenSearch (`NIE_OPENSEARCH_URL`): `GET /api/search/opensearch`, sem substituir o PostgreSQL.
- Tika e Docling (`NIE_TIKA_COMMAND`, `NIE_DOCLING_COMMAND`): o worker chama o comando se a extração básica falhar. A saída padrão vira texto.
- Arquivo remoto em um ref: `GET /api/git/remote-file` no GitHub ou no GitLab.
- ArchiveBox (`NIE_ARCHIVEBOX_DIR`): `POST /api/archive` recusa URL privada antes de executar.
- GPT Researcher (`NIE_GPT_RESEARCHER_URL`): `POST /api/research/external-report` guarda o texto como relatório externo, não como claim.
- pgvector (`NIE_PGVECTOR`): tenta criar a extensão e `passage_embeddings`. Se o pacote não estiver no PostgreSQL, a busca lexical segue.
- Caddy: `deploy/Caddyfile` e o serviço Compose no perfil `edge`.
- Mudança de documento: o alerta inclui um resumo das linhas que só existem numa das extrações.

Nenhum desses caminhos foi ensaiado contra os serviços públicos reais nesta máquina. StormCrawler, Nutch, Heritrix, RAGFlow e Onyx continuam de fora: o plano os trata como alternativa de escala ou segunda fonte de verdade, e não há medição que os justifique.

## O que ainda falta para o produto integral

### Encerramento da última tarefa e pausa solicitada

A migração `0011_job_lease_generation.sql` adiciona uma geração monotônica por
reivindicação de job, independente do número de tentativas (que pode diminuir
quando há throttle). Renovações, conclusão, repetição, descarte e adiamento do
worker exigem a geração vigente e um lease ainda válido. A publicação de uma
rodada também confere essa geração.

Durante o processamento, o worker renova o lease a cada 30s e cancela a execução
se perder a propriedade. As tentativas de renovação têm timeout e o encerramento
do processamento desliga o heartbeat. A liberação de reservas de bytes também
tem prazo limitado.

Os ensaios com PostgreSQL 18.6 confirmaram que o proprietário anterior não pode
alterar um job após deferimento e nova reivindicação com o mesmo número de
tentativa; um lease vencido não permite conclusão; e a substituição do worker
cancela a requisição HTTP em andamento sem afetar o novo proprietário.

Missão pausada por solicitação do usuário após finalizar essa tarefa. O objetivo
integral continua pendente; nenhuma conclusão de 100% foi atribuída ao projeto.

### Avanços posteriores: biblioteca e preservação

Migrações `0009` e `0010` acrescentam extrações versionadas e revisões de claims.
O reprocessamento produz novos trechos e mantém os IDs históricos; a busca usa
somente a extração ativa. Resultados incluem `extraction_run_id`. Offsets novos
referem-se aos bytes UTF-8 do texto da extração e são gravados somente quando o
trecho corresponde exatamente; informações ausentes nos registros antigos
continuam desconhecidas. Repetir uma extração antiga não rebaixa a versão ativa.

O worker preserva o DOM renderizado como objeto separado do original e registra
o hash da entrada da extração. Os parsers nativos informam nome e versão.

A API oferece `GET /api/documents/{id}/original`, histórico em `/extractions`,
consulta `/extractions/{run_id}` e `POST /reprocess`. O original é verificado por
hash antes do download ou reprocessamento. A interface permite baixar o original,
consultar extrações históricas e reextrair com o parser nativo. O acesso continua
submetido ao token da API.

Reavaliações de claims arquivam as afirmações, evidências e contradições anteriores
em `claim_revisions`. Contradições agora são persistidas com os IDs reais das
evidências e recarregadas pela API. Histórico:
`GET /api/campaigns/{id}/claim-revisions`.

`nie-maint` cria backup coordenado via PostgreSQL e Restic e restaura em destinos
novos. O ensaio recuperou todas as tabelas com as mesmas somas, incluindo originais,
derivados, extrações, claims e contradições. Procedimento e limites em
[BACKUP_E_RESTAURACAO.md](BACKUP_E_RESTAURACAO.md). Isso não determina ainda RPO/RTO,
escala de produção ou política de retenção.

### Validação em PostgreSQL 18 e OCR real

Os testes de store, API, worker e backup/restauração também passaram em uma
instância isolada de PostgreSQL **18.6** no WSL. Isso inclui migração de passagens
legadas, concorrência de extrações/claims, retomada de rodadas, referências a
trechos históricos, download autenticado, adulteração de CAS e comparação integral
do catálogo restaurado. A implantação Docker completa continua sem ensaio.

O caminho OCR usa Poppler para converter cada página do PDF em imagem e Tesseract
para reconhecer texto. Um PDF sintético **sem camada de texto** produziu o texto
esperado em testes reais do extrator, da API de reprocessamento e do worker com
coleta controlada. Os testes preservam página e hash do original e demonstram
que repetir a mesma extração não duplica versões.

Ativação: `NIE_OCR_ENABLED=true`; ferramentas Tesseract, `pdfinfo` e `pdftoppm` no
PATH. `NIE_OCR_MAX_PAGES` tem padrão 10 e teto 50; documentos além do teto são
recusados, sem tratar uma leitura parcial como completa. `NIE_OCR_LANGUAGE` tem
padrão `eng`; outros idiomas exigem os respectivos dados instalados.
`NIE_OCR_TIMEOUT` tem padrão e teto de 45s na API/worker. O conversor limita a
maior dimensão a 1800 pixels no caminho padrão e usa uma página temporária de
cada vez. A saída textual tem teto e o processo respeita cancelamento.

`POST /api/documents/{id}/reprocess?parser=ocr` preserva a extração como versão
adicional. O worker tenta OCR após falha de extração básica de PDF quando o recurso
está habilitado. A interface oferece o botão apenas se a instalação dispõe das
ferramentas. Há um target Docker opcional `runtime-ocr`, ainda sem build validado
neste ambiente. Timeouts, subprocessos e limites de rasterização **não equivalem
a sandbox de parser nem a avaliação de qualidade de OCR em manuais reais**.

| Marco do plano | Trabalho necessário para aceite integral |
|---|---|
| A — Pesquisador funcional e seguro | Campanha pública autorizada reproduzível, PG18, segurança de todos os executores e recuperação operacional |
| B — Investigação verificável | Proposições e escopos mais completos, suficiência que não dependa só de termos no trecho, testes reais de telecom e temas gerais |
| C — Biblioteca madura | Diff textual, Docling/Tika sob demanda, revisita com intervalo de produção e reavaliação só dos claims citados |
| D — Git Eye | Conteúdo remoto e GitLab, revisões verificáveis, corpus seletivo e avaliação de Zoekt |
| E — Operação contínua | Políticas por campanha, controle visual dos limites, alertas confiáveis, observabilidade e ensaio de vários dias |
| F — Escala | Linha de base de carga, medições e adoção de índices/workers distribuídos apenas quando justificados |

As regras usam estados e lacunas dos claims e, quando isso não gera consulta,
enviam as subperguntas determinísticas ainda abertas. Ainda podem repetir uma
formulação, deixar de perceber uma lacuna ou recuperar somente parte dos trechos
relevantes. `no_detected_gaps`, `subquestions_searched` e `subquestions_addressed`
são decisões heurísticas, não certificação de verdade nem comprovação de cobertura.
O controle de bytes ainda herda a contabilização por documentos preservados do
store; tráfego total, repetição de downloads e reservas exigem ledger próprio.

O plano integral não está concluído. Cada marco será fechado por evidências de
aceite e riscos conhecidos, sem atribuir percentual de conclusão não mensurado.
