# DOSSIÊ MESTRE — MOTOR AUTÔNOMO DE DESCOBERTA E INTELIGÊNCIA TÉCNICA

**Arquivo de preservação do projeto e da conversa · 30 de outubro de 2026**  
**Nome de trabalho:** Network Intelligence Engine  
**Componentes conceituais:** Git Eye · Web Eye · Autonomous Discovery Engine · Biblioteca Permanente · Motor de Verificação Cruzada  
**Situação:** especificação conceitual e pesquisa comparativa; **nenhuma implementação da plataforma foi afirmada como concluída**.

> **LEIA ANTES DE APAGAR O CHAT:** este material é uma **consolidação detalhada + registro cronológico reconstruído + cópias integrais dos quatro arquivos anexados**. Não é uma exportação nativa, byte a byte, da conversa. A íntegra exata de todas as mensagens, incluindo eventuais saídas de ferramentas e formatação original, só pode ser garantida por uma exportação oficial do histórico da conta, se disponível. Guarde o ZIP e, se a fidelidade literal de cada mensagem for indispensável, exporte os dados antes de excluir a conversa. Informações dos outros relatórios foram preservadas como **alegações das fontes**, não convertidas automaticamente em fatos verificados.

---

## 1. Resumo executivo e veredito final

O projeto propõe uma plataforma auto-hospedável para **pesquisar conteúdos técnicos difíceis de localizar** em repositórios Git, páginas web, fóruns, PDFs, manuais, sites históricos, bases de documentação e índices especializados, com possibilidade de incluir determinadas fontes `.onion` públicas e metadados públicos de BitTorrent quando houver finalidade legítima.

Em lugar de um único mecanismo que tente copiar a internet, o sistema combinará **buscas externas de largo alcance com um acervo próprio acumulativo, qualificado e pesquisável**. O usuário definirá **o que quer descobrir**; a plataforma deverá descobrir automaticamente **onde procurar**, seguir referências relevantes, registrar origens e verificar compatibilidade técnica.

**Veredito acordado durante a conversa:**

- A ambição é **tecnicamente plausível**, mas não existe evidência de uma aplicação open source única, pronta e completa que entregue toda a combinação com qualidade comprovada.
- **Não é necessário criar um buscador ou modelo de IA do zero.** Desenvolveremos a integração e o conhecimento específico sobre redes sobre ferramentas existentes.
- O traço mais original não é o crawler recursivo, pois essa abordagem já tem precedentes, mas a **especialização multivendor**, a **correlação técnica por equipamento/firmware** e a **memória contínua com evidências**.
- Não se deve anunciar que a plataforma pesquisa “literalmente a web inteira”, que é “a primeira do mundo” ou que uma informação foi comprovada só porque apareceu em três sites.
- O teste do produto deve medir **quantos problemas técnicos difíceis ele resolve com fontes corretas**, não a quantidade bruta de terabytes ou links armazenados.
- Deve começar com um **MVP enxuto**, escalar conforme medições e evitar redundância desnecessária entre softwares.

### 1.1. Frase central do projeto

**“Uma investigação técnica autônoma que percorre trilhas de evidências, encontra materiais raros em diferentes fontes, verifica aplicabilidade por equipamento e versão, constrói um acervo permanente e explica tudo com referências verificáveis.”**

### 1.2. O que o projeto não é

Não é um vírus, botnet, ferramenta de invasão, programa para contornar autenticação ou sistema que tenha acesso universal a dados privados. A semelhança com “vírus”, “raízes”, “água” ou “eletricidade” é **exclusivamente a expansão recursiva de caminhos a partir de referências públicas permitidas**.

---

## 2. Origem da ideia: o que o usuário pediu

O pedido inicial foi localizar **sistemas open source** disponíveis no GitHub, GitLab e outras fontes que conseguissem pesquisar **milhões ou bilhões de informações de repositórios** e outro conjunto de mecanismos que conseguisse pesquisar a **web amplamente**, incluindo Google, Bing, DuckDuckGo, fóruns, PDFs, documentos, sites, índices próprios, redes Tor e metadados ou conteúdos distribuídos por torrent.

O exemplo prático era um buscador excepcionalmente profundo para engenharia de redes e provedores de internet (ISPs):

- **Fabricantes:** Huawei, Datacom, Cisco, ZTE, VSOL, FiberHome, Intelbras, Digistar, TP-Link, Furukawa, MikroTik, SonicWall, Hillstone, entre outros.
- **Equipamentos:** switches, roteadores, OLTs, ONUs, firewalls, BNG/BRAS e concentradores PPPoE; especialmente Huawei NetEngine NE8000.
- **Tecnologias:** BGP, MPLS, PBR, RADIUS, PPPoE, CGNAT, GPON/XGS-PON, VLANs, redundância, roteamento, operação e interoperabilidade.
- **Casos difíceis:** redundância entre autenticadores BNG/BRAS, preservação de sessões, failover BGP, configurações raras de firmware e comandos CLI específicos de equipamento.
- **Formato desejado:** biblioteca crescente de gigabytes a milhares de gigabytes, organizada, pesquisável em linguagem natural e por comandos exatos, com download e preservação de fontes autorizadas.

O usuário acrescentou um princípio: encontrar a **mesma informação técnica em três fontes independentes** deve aumentar a confiança, sem ignorar que os três documentos podem ser cópias da mesma origem.

Depois acrescentou o requisito mais marcante: a plataforma **não deve depender de o usuário indicar sites**. Deve descobrir índices, pistas, nomes, citações, links e novas fontes, seguir cada caminho promissor e expandir as buscas em novas ramificações — **“como uma ramificação de uma árvore”, “uma eletricidade em uma madeira” e “como a água da chuva”.**

---

## 3. Objetivos de produto e indicadores de sucesso

### 3.1. Objetivos

1. Receber consultas simples ou investigações complexas em português e operar também com termos em inglês, espanhol e, se pertinente, chinês.
2. Pesquisar em **fontes federadas**, tanto existentes na internet quanto em arquivos previamente coletados.
3. Descobrir **novas fontes automaticamente**, inclusive por referências não clicáveis e nomes de protocolos/documentos.
4. Extrair texto de páginas, fóruns e documentos, preservar os originais quando permitido e manter histórico das versões.
5. Detectar e indexar **comandos CLI, exemplos de configuração, mensagens de erro, procedimentos, pré-requisitos e notas de compatibilidade**.
6. Relacionar cada descoberta ao **fabricante, produto, família, revisão, sistema operacional, firmware, release e contexto operacional**.
7. Corroborar afirmações por fontes independentes, detectar cópias, contradições e evidências insuficientes.
8. Produzir respostas e dossiês com **trechos citáveis, página, URL, data, origem, versão e estado de validação**.
9. Atualizar o acervo por **coleta incremental e campanhas agendadas**, preservando deduplicação e limites operacionais.
10. Ser **auto-hospedável, modular, auditável** e evolutivo.

### 3.2. Critérios de sucesso

- Proporção de perguntas com **pelo menos uma fonte técnica relevante**, preferencialmente oficial.
- Precisão de **fabricante/modelo/firmware**, evitando misturar comandos incompatíveis.
- Correção da correspondência **comando → finalidade → modo operacional → pré-requisitos**.
- Reprodutibilidade da resposta: o leitor consegue localizar a seção original.
- Tempo e custo da pesquisa comparados com busca manual.
- Taxa de **fontes inéditas úteis** descobertas por expansão automática.
- Taxa de respostas sem evidência que o sistema corretamente **se recusa a afirmar**.
- Volume útil preservado vs. páginas repetidas/irrelevantes; custo de armazenamento.

**Importante:** latências, percentuais e tamanhos anunciados em relatórios externos são estimativas ou alegações que exigem ensaios reais; não são metas garantidas.

---

## 4. Arquitetura-alvo

```text
                       USUÁRIO / INTERFACE ÚNICA
                      pergunta, CLI, filtros, campanha
                                    │
                                    ▼
                  ORQUESTRADOR DE INVESTIGAÇÃO
           planejamento, subperguntas, fronteira de URLs,
           prioridades, descobertas, verificação cruzada
                                    │
                    ┌───────────────┴───────────────┐
                    ▼                               ▼
              MOTOR GIT EYE                    MOTOR WEB EYE
      GitHub/GitLab/Gitea/Forgejo         SearXNG, Mwmbl, APIs,
        Software Heritage, Zoekt         Common Crawl, Wayback,
         código/configs/README            sites, fóruns, RSS,
                                       sitemap, Tor público opcional
                    │                               │
                    └───────────────┬───────────────┘
                                    ▼
                    COLETA + EXTRAÇÃO SEGURA
               Scrapy / Crawl4AI / Tika / Docling
                  HTML/PDF/Office/configs/CLI
                                    │
                                    ▼
                 NORMALIZAÇÃO + DEDUPLICAÇÃO
          proveniência, hash, idiomas, versões, classificação,
             redaction de segredos, confiança e licenças
                                    │
            ┌───────────────────────┼────────────────────────┐
            ▼                       ▼                        ▼
    ARQUIVOS ORIGINAIS          POSTGRESQL               OPENSEARCH
     MinIO/NAS/S3           catálogo / grafo /        BM25 + filtros +
        versionados          filas / histórico         busca vetorial
            │                       │                        │
            └───────────────────────┼────────────────────────┘
                                    ▼
                         BUSCA HÍBRIDA + RAG
                   recuperação exata e semântica,
                   fontes/versões/contradições
                                    │
                                    ▼
                       DOSSIÊ COM EVIDÊNCIAS
                 texto, CLI, tabelas, citações, riscos,
                     compatibilidade e dúvidas
                                    │
                                    └─────────────┐
                                                  ▼
                                       NOVAS PISTAS / FILA
                                           (ciclo controlado)
```

### 4.1. Separação fundamental

- **Descoberta:** encontrar referências potenciais, URLs, nomes e repositórios.
- **Coleta:** obter legitimamente o conteúdo de fontes escolhidas.
- **Arquivamento:** preservar uma cópia do material, segundo licença e permissões.
- **Indexação:** criar estrutura de busca exata, estruturada e semântica.
- **Investigação:** avaliar e correlacionar documentos encontrados.
- **Resposta:** apresentar conclusão, condições de validade e citações.

Nenhuma ferramenta individual resolve necessariamente todas essas etapas.

---

## 5. Git Eye — mecanismo de pesquisa em código

**Objetivo:** localizar comandos, configurações, scripts, templates de automação, READMEs, exemplos de firmware e artefatos textuais em repositórios publicamente acessíveis ou autorizados.

**Fontes de descoberta:** APIs de busca GitHub/GitLab; projetos públicos em Gitea/Forgejo/Bitbucket e outras forjas; catálogos como ecosyste.ms; Software Heritage como arquivo histórico; eventualmente buscas de código públicas como grep.app ou Sourcegraph quando oferecidas e permitidas.

**Índice próprio:** Zoekt (motor especializado em código e texto, com substring/regex), alimentado por repositórios selecionados que a plataforma efetivamente baixou e indexou. OpenGrok é complementar para navegação por símbolos, referências e histórico; não é obrigatório no início.

**Não confundir:** Software Heritage preserva código-fonte em escala enorme, mas **não oferece automaticamente um grep instantâneo em todos os arquivos**. GH Archive contém principalmente **eventos públicos do GitHub** e não uma cópia integral pesquisável de todas as linhas de código. Zoekt precisa de conteúdo local ou pipeline de indexação preparado; não pesquisa literalmente todos os repositórios públicos sem aquisição/indexação.

**Consultas exemplares:**

```text
"display bgp peer" "NE8000"
"radius-server" "pppoe" "huawei"
"/routing/bgp/connection" "RouterOS 7"
"ip policy route-map" "SonicWall"
"interface gpon" "OLT"
"DmOS" "BGP" "route-policy"
```

A sintaxe precisa de busca de cada motor será gerada conforme seu dialeto; os exemplos acima são **expressões de investigação**, não comandos de configuração validados.

---

## 6. Web Eye — mecanismo de descoberta web

**Objetivo:** alcançar documentação que uma busca comercial comum não prioriza: fóruns pequenos, arquivos históricos, posts antigos, PDFs e apresentações, listas de discussão, blogs técnicos, páginas em outros idiomas e portais de fabricantes.

**Fontes e papéis:**

- **SearXNG:** metabusca. Consulta múltiplos motores, mas não mantém uma cópia da web.
- **Mwmbl:** iniciativa open source de buscador com **índice próprio**. Complementa a descoberta, sem alegar cobertura comprovada de bilhões de páginas.
- **Common Crawl:** coleção pública massiva de capturas da web; índice de URLs e registros WARC/WAT/WET, não um mecanismo pronto de busca lexical instantânea em todo o conteúdo.
- **Wayback Machine / CDX:** localizar cópias de URLs ou sites antigos quando há capturas disponíveis.
- **Scrapy:** rastreamento direcionado, extração de links, sitemaps, filas e regras.
- **Crawl4AI:** extrair páginas e conteúdo preparado para IA, inclusive páginas modernas quando acessíveis.
- **Apache Nutch / Apache StormCrawler / Heritrix:** candidatos de expansão para crawling maior, distribuído ou preservação arquivística; **não instalar todos por padrão**.
- **YaCy:** alternativa P2P com crawler/índice próprio para experimentar, não componente obrigatório.
- **Ahmia:** pesquisas/crawling de serviços `.onion` públicos **conhecidos**, sem garantia de enumerar toda a rede Tor.
- **bitmagnet:** discovery e busca de **metadados BitTorrent/DHT**, não leitura universal do conteúdo de arquivos distribuídos.

**Fóruns previstos, quando acessíveis e permitidos:** Stack Overflow, Network Engineering e Server Fault (Stack Exchange), Viva o Linux, fórum MikroTik, Cisco Community, fóruns Huawei e de provedores, discussões públicas do GitHub, comunidades relevantes e conteúdos preservados no Wayback.

O sistema poderá **ler discussões, respostas, comentários, links e material referenciado**, dentro das condições de acesso e coleta de cada plataforma.

---

## 7. Autonomous Discovery Engine — a “água encontrando caminhos”

Este é o requisito que o usuário explicitou mais enfaticamente: a plataforma deve **descobrir novas fontes sem exigir que cada site seja indicado manualmente**.

### 7.1. Caminhos de descoberta

1. **Consultas amplas** a mecanismos externos e índices preexistentes.
2. **Links HTML** encontrados em artigos, páginas e respostas de fóruns.
3. **Referências em PDF, HTML e README** (URLs, autores, títulos, RFCs e nomes de documentação).
4. **Menções sem hyperlink:** termos e nomes de tecnologias/documentos geram novas buscas por título, modelo ou protocolo.
5. **Sitemap, RSS e Atom:** feeds e caminhos sinalizados pelos sites.
6. **Citações cruzadas:** vídeos, artigos, slides, repositórios e comentários que indicam mais material.
7. **Histórico:** URLs mortas levam à busca de capturas em arquivos históricos.
8. **Novas hipóteses:** a IA percebe lacunas e formula pesquisas derivadas.
9. **Atualizações:** revisitação de fontes importantes por alteração conhecida, feed ou agenda.

### 7.2. Fronteira de descoberta

Cada nova pista entra em uma fila contendo, idealmente:

`source_url`, `discovered_from`, `discovery_type`, `topic`, `vendor`, `model`, `firmware`, `language`, `candidate_score`, `depth`, `crawl_status`, `first_seen`, `last_seen`, `next_check`, `robots_policy`, `license`, `content_hash` e `parent_document_id`.

**Prioridade:** relevância para o tema, novidade, probabilidade de evidência útil, reputação e independência da fonte, compatibilidade de equipamento/versão e custo estimado de aquisição.

**Segurança:** limites por domínio e por campanha, taxa de requisições, profundidade, tamanho de resposta, timeout, filas de quarentena, detecção de traps de crawling, deduplicação por URL canônica e por hash, recusa de áreas privadas/não permitidas. Não usar rotação de identidade para contornar proibições.

### 7.3. O crescimento não deve ser “tudo indiscriminadamente”

A metáfora da árvore é a **expansão em grafo**. Uma página pode apontar para 20 links, que apontam para centenas de outros. Sem filtros, seriam milhões de páginas inúteis, repetidas ou armadilhas. O sistema deve priorizar **descoberta orientada ao assunto** (*focused crawling*), com orçamentos por tema/domínio, políticas de recursão e possibilidade de interromper ramificações pouco promissoras.

### 7.4. Exemplo de ramificação

1. Pergunta sobre redundância PPPoE no Huawei NE8000.
2. Busca encontra uma discussão sobre BNG HA.
3. Essa discussão cita determinado manual VRP.
4. O manual cita mecanismos de backup de assinantes e uma versão específica.
5. A plataforma procura esse mecanismo e encontra apresentações, scripts, outros fóruns e históricos.
6. Compara as versões e guarda documentos/citações relevantes.
7. Uma descoberta alimenta novas buscas, sem repetir o que já foi indexado.

Esse comportamento **não deve fazer downloads indiscriminados, acessar contas ou executar scripts**.

---

## 8. Verificação cruzada — “três fontes” não são sempre três evidências

O usuário propôs: **se a mesma informação aparece em três locais distintos, sua chance de ser verdadeira é maior**. Isso é um princípio útil desde que as fontes tragam **evidência independente e equivalente**, com escopo técnico comparável.

### 8.1. Classes de evidência sugeridas

1. **Não confirmado:** menção única ou conteúdo sem sustentação suficiente.
2. **Corroborado por fontes independentes:** resultados convergentes com proveniência diferente.
3. **Documentado oficialmente:** manual/fonte do fabricante indica suporte no modelo/release específico.
4. **Validado em laboratório:** execução reproduzida em ambiente autorizado e compatível.
5. **Conflitante ou não aplicável:** evidências divergentes, versão errada ou condição não satisfeita.

Essas classes **não substituem avaliação de qualidade**. Um fórum pode compartilhar uma configuração funcional, mas não provar aplicabilidade a outro firmware; um manual pode documentar um comando não habilitado na licença utilizada.

### 8.2. Independência de fontes

- Três páginas com mesmo texto copiado de um blog = **um grupo de origem**, não três confirmações independentes.
- Um manual oficial, um caso de suporte técnico independente e um teste de laboratório reproduzível = conjunto mais forte.
- Contar citações e **proveniência de origem**, não apenas URLs.
- Comparar **modelo, release de software, patch, feature, modo CLI, pré-requisitos e topologia**.
- Explicitar quando a evidência mostra somente “comando documentado” e não “solução completa comprovada”.

### 8.3. Exemplo crítico de redes

**Redundância BGP**, **failover de servidores RADIUS**, **alta disponibilidade do BNG/BRAS** e **continuidade/sincronização de sessões PPPoE** são assuntos relacionados, mas **não equivalentes**. Uma solução que mantém conectividade L3 não necessariamente preserva sessões de assinantes. O sistema precisa distinguir isso antes de propor comandos.

### 8.4. Saída da verificação

Uma afirmação deve trazer: resumo; evidências favoráveis/contrárias; URLs; seções ou páginas; origem comum detectada; fabricante; modelo; versão; risco de incompatibilidade; grau de validação; testes pendentes. Nenhuma ferramenta de IA deve dizer que um recurso funciona em produção sem a devida base.

---

## 9. Processamento de conteúdo e biblioteca permanente

### 9.1. Pipeline sugerido

`Descoberta → aquisição permitida → identificação de formato → extração → normalização → deduplicação → metadados → classificação técnica → busca lexical/vetorial → evidências citáveis → revisão incremental`.

- **Apache Tika:** texto e metadados em múltiplos formatos.
- **Docling:** estrutura de PDFs, tabelas e manuais (comparar com Tika na prática; podem operar como caminhos complementares).
- **ArchiveBox:** arquivamento de páginas selecionadas e seus artefatos quando permitido.
- **PostgreSQL:** catálogo e relacionamentos; trilha da coleta, versões e estado das evidências.
- **Object storage (MinIO/S3 compatível/NAS):** arquivos originais, extratos, capturas e versões.
- **OpenSearch:** busca textual BM25, filtros e possível busca híbrida/vetorial.
- **Zoekt:** índice especializado de texto/código em repositórios.
- **RAGFlow ou implementação própria:** camada opcional de RAG documental; decidir após protótipo.
- **GPT Researcher ou orquestrador próprio:** candidato para investigação; testar integração, custos e reprodutibilidade.

### 9.2. Preservação

Não basta salvar somente embeddings ou respostas produzidas por IA. Preservar **arquivo original (se permitido), hash SHA-256, URL, timestamp, parser utilizado, versão do extrator e número da página/seção**, além do texto estruturado. Documentos com mesma informação podem se relacionar sem duplicar o arquivo físico.

### 9.3. Estrutura física orientativa

```text
/acervo/
  originais/
    github/
    gitlab/
    manuais-pdf/
    paginas-web/
    arquivos-historicos/
  extraidos/
    markdown/
    texto/
    tabelas/
    blocos-cli/
  capturas/
    warc/
    html/
  metadados/
    docs/
    releases/
    proveniencia/
  indices/
    opensearch/
    zoekt/
  campanhas/
    bgp/
    mpls/
    pbr/
    pppoe-bng/
    gpon/
```

Fabricante, modelo, versão e protocolo devem ser **campos de catálogo e facetas de pesquisa**, e não apenas pastas que obriguem a duplicar documentos.

### 9.4. Entidades do domínio

- `Source`: URL/forja/domínio/tipo de origem/política.
- `Document`: ID, hash, título, idioma, formato, licença, originais e revisões.
- `Citation`: documento, página/seção, trecho, afirmação suportada.
- `Vendor`, `ProductFamily`, `DeviceModel`, `OperatingSystem`, `FirmwareRelease`.
- `Command`: sintaxe canônica, contexto, tipo (consulta/configuração), argumentos, feature.
- `Feature`, `Protocol`, `Scenario`, `Topology`, `Prerequisite`.
- `Claim`, `Evidence`, `Contradiction`, `IndependenceCluster`, `ValidationRun`.
- `DiscoveryEdge`: de onde veio cada link/menção/documento.
- `ResearchCampaign`: pergunta, sementes, orçamento, critérios, resultados e revisões.

---

## 10. Ferramentas: decisão e comparações

As ferramentas listadas são **candidatas ou fontes**, não dependências todas aprovadas. Licenças, documentação e manutenção podem mudar; confirmar versões antes da instalação.

| Projeto | Papel correto | Decisão do projeto |
|---|---|---|
| **Zoekt** | Pesquisa em código já indexado | **Núcleo Git** |
| **SearXNG** | Metabusca sobre motores externos | **Núcleo web** |
| **Mwmbl** | Índice independente da web | **Fonte integrada** |
| **GitHub/GitLab APIs** | Descoberta/busca/metadata de repositórios | **Fonte integrada** |
| **Software Heritage** | Arquivo e descoberta de código histórico | Fonte externa, não clone mundial |
| **ecosyste.ms** | Descoberta e metadados de repositórios | Complementar |
| **Common Crawl** | Arquivos públicos e índice de URLs | Fonte externa sob demanda |
| **Wayback/CDX** | Capturas históricas | Fonte externa sob demanda |
| **Scrapy** | Crawler programável | **Coleta principal** |
| **Crawl4AI** | Extrator web orientado a IA | **Coleta/extrator principal** |
| **Tika** | Parser de formatos variados | Principal conforme formato |
| **Docling** | Parsing estruturado de PDFs/tabelas | Principal conforme formato |
| **ArchiveBox** | Preservação de páginas | Complementar; não único storage |
| **PostgreSQL** | Catálogo, fluxo e proveniência | **Núcleo** |
| **MinIO/NAS** | Armazenamento de objetos/arquivos | **Núcleo** |
| **OpenSearch** | Pesquisa textual/híbrida | **Núcleo documental** |
| **GPT Researcher** | Investigação multietapas com IA | Candidato a coordenador |
| **RAGFlow** | Ingestão documental e RAG | Candidato a interface, comparar |
| **Onyx** | Pesquisa corporativa e conectores/RAG | Alternativa, não instalar junto por padrão |
| **Apache Nutch** | Crawl web massivo | Fase posterior |
| **StormCrawler** | Rastreamento contínuo distribuído | Fase posterior / experimento |
| **Heritrix** | Crawling de qualidade arquivística | Alternativa futura |
| **YaCy** | Crawler e índice P2P | Opcional, experimento |
| **OpenGrok** | Navegação avançada em código | Complemento ao Zoekt |
| **Livegrep/Hound** | Busca local regex em código | Alternativas ao Zoekt |
| **Sourcebot** | Interface/busca Git baseada em Zoekt | Cautela sobre mudanças de licença |
| **Sourcegraph** | Busca de código mais ampla | Não pressupor edição comunitária gratuita |
| **GH Archive** | Eventos públicos do GitHub | Descoberta, não texto integral |
| **bitmagnet** | Metadados BitTorrent/DHT | Fase opcional |
| **Ahmia** | Busca/crawling de onion público conhecido | Fase opcional |
| **Paperless-ngx** | Biblioteca documental e OCR | Alternativa para gestão PDF, não exigido |
| **Diffbot** | Crawler/knowledge graph comercial | Precedente tecnológico, não núcleo livre |
| **Hyperresearch** | Pesquisa com acervo permanente e ligações | Candidato a estudo, verificar implementação |

### 10.1. Correções relevantes preservadas dos relatórios

1. **GH Archive ≠ código Git completo.** O BigQuery sobre eventos não possibilita grep direto em todos os arquivos de código.
2. **Software Heritage ≠ pesquisador universal de strings pronto.** Métricas de nós de grafo não equivalem a número de arquivos pesquisáveis; números de volume precisam da definição e data de cada dataset.
3. **Common Crawl ≠ Google full-text pronto.** Índices de URL/WARC não são a mesma coisa que pesquisa textual arbitrária em todos os registros.
4. **SearXNG ter muitos conectores ≠ todos funcionam em qualquer momento.** Rate limits, CAPTCHAs e bloqueios podem impedir consulta a certas fontes.
5. **Mwmbl “bilhões de páginas” não ficou comprovado** nos relatórios. Manter como índice complementar, com tamanho atual a verificar.
6. **Licenciamento importa:** Elasticsearch não deve ser simplificado como “distribuição padrão Apache 2.0”; Sourcebot e Sourcegraph têm avisos de mudança de distribuição/licença nos materiais. Validar versão e repositório antes da escolha.
7. **Instalação real:** exemplos antigos de Nutch com `inject` incompleto, `solrindex` deprecado/removido e `./OpenGrok index` não são receitas reproduzíveis. Não reutilizar sem checar documentação oficial.
8. **Zoekt: RAM/índice depende de corpus e workload.** Percentuais e fatores “3,5×” apresentados não devem ser tratados como garantias universais; planejar benchmark.
9. **Nutch não é requisito inicial.** Crawlers direcionados bastam para aprender quais fontes rendem evidência útil.
10. **Open Deep Research da LangChain arquivado:** os relatórios mencionaram arquivamento em agosto de 2026; verificar estado antes de basear projeto novo nele.
11. **Não inferir equivalência de tecnologias.** BGP, VRRP, RADIUS failover, Mesh e PPPoE session sync possuem semânticas e limites distintos conforme equipamento.
12. **Não confundir índice local com busca global.** Zoekt e OpenSearch trabalham no conteúdo que for efetivamente indexado neles.

Essas correções são **registro da discussão e dos relatórios**; detalhes de data e licença não foram revalidados como parte deste arquivamento.

---

## 11. Os precedentes: por que a ideia geral não é inédita

O usuário perguntou duas vezes se o projeto seria **“o primeiro do mundo”** e depois solicitou os precedentes concretos. A conclusão da conversa foi **não alegar ineditismo absoluto**.

| Precedente | Função semelhante | Diferença / limite frente ao projeto |
|---|---|---|
| **Diffbot** | Crawler recursivo, índice web e grafo de conhecimento | Plataforma comercial; não nichada como motor aberto de engenharia multivendor |
| **Hyperresearch** (Jordan Gibbs) | Pesquisa, ligação entre documentos, biblioteca permanente e verificação de fontes | Necessária validação de maturidade, implementação e cobertura de coleta |
| **GPT Researcher** | Investigações e subperguntas automatizadas, relatórios com citações | Agente de pesquisa não é por si só crawler universal + acervo técnico especializado |
| **YaCy** | Busca/crawling descentralizado e índice próprio | Não é agente especializado que valida compatibilidade técnica |
| **Apache StormCrawler** | Crawling distribuído e contínuo, expansão por links | Infraestrutura de coleta, não a inteligência técnica fim a fim |
| **Mwmbl** | Buscador com índice próprio e coleta comunitária | Não é a plataforma de dossiês e validação por firmware proposta |
| **Onyx** | Pesquisa corporativa, conectores, RAG e agentes | Não é, isoladamente, um rastreador recursivo universal |
| **Autonomous Research Agent** (projeto experimental citado) | Declara investigação contínua e criação de hipóteses | Proposta experimental, maturidade não comprovada |
| **RAGFlow** | Ingestão e recuperação documental avançada | Não substitui descoberta global e controle de evidências multivendor |
| **Juniper Marvis AI** | Assistência e diagnóstico de redes no ecossistema Juniper | Especializado no próprio fabricante/ecossistema |
| **Cisco AI Canvas** | Correlaciona evidências e ajuda investigações de rede | Produto comercial com escopo e integrações próprios |

**A inovação plausível:** integrar descoberta recursiva em múltiplas fontes, persistência, deduplicação de origem, extração estruturada de CLI e **compreensão verificável de fabricante + modelo + versão + cenário de rede**. Ainda assim, pesquisas de anterioridade e concorrência seriam necessárias antes de declarar qualquer exclusividade ou pioneirismo.

---

## 12. Exemplos de pesquisas técnicas que motivaram o projeto

### 12.1. Huawei NE8000 como BNG/BRAS

Pergunta: “Como fazer redundância entre dois NE8000 para assinantes PPPoE, usando servidores RADIUS e failover de rotas BGP, considerando preservação de sessões?”

Investigar separadamente:

- Alta disponibilidade do encaminhamento de pacotes (routing / BGP).
- Autenticação e contabilização RADIUS, timeout, failover e estado das sessões.
- Mecanismos específicos de HA de BNG/BRAS documentados no modelo/release.
- Sincronização/preservação de sessões PPPoE, quando de fato suportada.
- Licenças, limitações de hardware, timers, precondições de topologia.

**Não emitir uma receita única sem correspondência de modelo/VRP.**

### 12.2. MikroTik RouterOS 7

Investigar PPPoE concentrator, BGP/VRF, RADIUS, políticas de roteamento, VRRP e a distinção entre continuidade de tráfego e continuidade de sessões PPPoE. Separar exemplos v6 e v7.

### 12.3. OLTs e redes GPON/XGS-PON

Huawei, ZTE, FiberHome, VSOL, Datacom, Intelbras, Digistar, TP-Link, Furukawa: identificar família, release, sintaxe de perfil, provisionamento e compatibilidade de ONUs. Não transferir comandos entre gerações distintas de OLT.

### 12.4. PBR e NGFW

SonicWall e Hillstone: pesquisar documentação de Policy-Based Routing e coexistência com BGP/roteamento dinâmico, algoritmos de decisão, políticas e ordem de avaliação. Confrontar guia oficial e relatos operacionais.

### 12.5. Busca lexical exata

As expressões `display bgp peer`, `route-policy`, `interface gpon`, `/ppp secret`, MIB OIDs, nomes de parâmetros RADIUS e mensagens de erro são exemplos de conteúdo que exige busca **lexical** além de embeddings. RAG sem pesquisa exata pode trocar sintaxe e produzir resultados incorretos.

---

## 13. Experiência de uso ideal

O produto deve apresentar ao usuário uma **única caixa de pesquisa** com modos:

- **Busca rápida:** documentos, CLI, repositórios e fóruns com filtros.
- **Pesquisa profunda:** desmembrar um problema em subquestões, expandir links, comparar fontes e gerar relatório.
- **Monitoramento contínuo:** acompanhar fontes e assuntos que o usuário permitiu.
- **Biblioteca:** pesquisar somente o acervo local, com históricos e referências.
- **Comparador:** examinar diferenças entre versões de comandos, dois modelos ou duas hipóteses técnicas.
- **Auditoria de evidências:** ver quais URLs suportam cada afirmação e quais derivam da mesma origem.

Filtros úteis: fabricante, família de equipamentos, modelo, firmware, protocolo, idioma, documento oficial/comunitário, data, tipo de arquivo, confiabilidade, texto exato, regex e pesquisa semântica. Deve ficar visível a **origem** e o **estado de verificação**, sem vender “alta confiança” como certeza infalível.

### 13.1. Dossiê gerado

1. Pergunta e restrições interpretadas.
2. Resposta resumida com limitações.
3. Cenários/arquiteturas possíveis.
4. Comandos e trechos documentados, com versão e modo de execução.
5. Evidências pró/contra; cluster de fontes duplicadas.
6. Pré-requisitos e licenciamento relevante.
7. Plano de testes em laboratório, validação e rollback.
8. Fontes URLs, datas, PDFs com páginas/seções.
9. O que segue desconhecido.
10. Metadados e links do material preservado.

---

## 14. Riscos, limitações e regras de operação

### 14.1. Limitações incontornáveis

- Nenhum sistema conhece ou indexa **toda a internet**, cada repositório, cada fórum privado ou serviço onion não divulgado.
- Busca de documentos públicos pode esbarrar em autenticação, paywalls, remoções, `robots.txt`, APIs e termos.
- Fontes históricas não garantem arquivos preservados.
- Metadados de torrents não dão acesso automático ao conteúdo de arquivos e não legitimam downloads proibidos.
- A coleta deve respeitar direitos autorais e termos de uso. **Links e referências** podem ser preservados mesmo quando o arquivo integral não puder ser arquivado.
- Exemplo comunitário não garante que comando funcione em equipamento diferente.

### 14.2. Segurança do crawler

- Sem execução automática de scripts, firmwares, arquivos binários ou comandos CLI encontrados.
- Parsers e browsers em workers isolados, limites de CPU/RAM/disco/tempo e sem acesso à infraestrutura de produção.
- Inspeção de tipos de arquivo, limitação de downloads, checagem de origem e varredura de risco quando pertinente.
- Remoção/mascaramento de senhas, segredos, tokens, configs com dados pessoais e credenciais em material coletado.
- Sem violar autenticação, burlas técnicas, invasão, coleta de conteúdo privado ou redistribuição não autorizada.
- Política de retenção/remoção e trilha auditável de coleta.

### 14.3. Custo e escala

Uma biblioteca de **dezenas de GB a múltiplos TB** é uma ambição viável com dimensionamento adequado; um espelho integral do Git público ou da web é outra ordem de magnitude. Consumo de armazenamento depende de PDFs, repositórios, snapshots, extratos e índices; não afirmar fatores de compressão ou requisitos fixos sem medições. Preferir descoberta global **sem precisar copiar todos os dados**.

---

## 15. Plano de desenvolvimento aprovado em princípio

### Fase 0 — Requisitos e prova de descoberta

- Definir 15 a 20 casos reais de redes difíceis de pesquisar manualmente.
- Montar inventário inicial pequeno de fontes oficiais e comunidades.
- Prototipar busca externa e índice local com fontes abertas permitidas.
- Registrar licenças e versões efetivamente testadas.

**Aceite:** localizar documentos relevantes e preservar URLs, páginas e versões corretamente.

### Fase 1 — MVP funcional de pesquisa

- API e interface única.
- SearXNG (e API Git), Zoekt para repositórios selecionados, Scrapy/Crawl4AI, Tika/Docling.
- PostgreSQL + arquivos originais + OpenSearch BM25, filtros e histórico.
- Deduplicação por hash/URL; ingestão manual e programada; indicadores de qualidade.

**Aceite:** pesquisa exata e navegação entre documentos originais; nenhuma dependência de agente para simples pesquisa.

### Fase 2 — Investigação multietapas e evidências

- Agente para gerar subperguntas e coordenar fontes.
- RAG híbrido com citações precisas.
- Extração/classificação de CLI por fabricante/modelo/versão.
- Comparação de evidências, clusters de cópia, contradições e estados de validação.
- Dossiê técnico exportável.

**Aceite:** respostas fundamentadas para casos difíceis sem misturar firmwares, com conflitos explicitados.

### Fase 3 — Exploração autônoma ramificada

- Fila de URL/referência, fontes seed opcionais, links e menções, relevância por campanha.
- Revisitas incrementais, descoberta de novos índices e fóruns, sitemap/RSS, históricos.
- Controles de profundidade, taxa, domínios, custo, deduplicação e bloqueio de crawl traps.

**Aceite:** novas fontes úteis descobertas automaticamente, com proveniência e baixo desperdício.

### Fase 4 — Escala

- Se houver demanda medida, testar Nutch/StormCrawler e workers distribuídos.
- Armazenamento de múltiplos TB, controle de retenção, backup, observabilidade.
- Fontes especializadas opcionais (Ahmia público, metadados BitTorrent, YaCy), sob justificativa concreta.

**Aceite:** throughput, custos e precisão monitorados; conteúdo irrelevante não domina os índices.

### Fase 5 — Integrações futuras

- Possível integração com o ecossistema NexoISP e seus módulos de operação de rede.
- Permitir consulta de configurações **somente em ambientes autorizados**; execução em produção é um produto separado que exigiria autorização, revisões e controles adicionais.

**Aceite:** camada de conhecimento desacoplada e segura; não executar CLI automaticamente em produção.

---

## 16. Benchmark inicial de 20 perguntas técnicas

Não são testes já executados; são **casos recomendados** para avaliação futura.

1. Huawei NE8000: diferença entre redundância BGP e continuidade de sessões PPPoE.
2. Huawei NE8000: documentação BNG hot backup por modelo e versão VRP.
3. Huawei NE8000: autenticação RADIUS redundante vs. contabilização.
4. Huawei NE8000: route-policy e controle de rotas em cenários multihomed.
5. MikroTik RouterOS 7: diferenças de BGP em relação à versão 6.
6. MikroTik como PPPoE concentrator: riscos de failover e statefulness.
7. MikroTik RouterOS 7: PBR com routing rules vs. mangle.
8. Cisco: route-maps e policy-based routing dependente de plataforma.
9. Datacom DmOS: BGP policies documentadas por release.
10. Datacom: configuração SNMP/MIBs por família de equipamento.
11. OLT Huawei MA5800: perfis de linha/serviço por versão.
12. OLT ZTE C300/C320 vs. C600: diferenças reais de CLI.
13. FiberHome: provisionamento GPON por firmware.
14. V-SOL: perfis de VLAN e ONU por modelo.
15. Intelbras/Digistar/TP-Link/Furukawa: diferenças de CLI para OLT/ONU.
16. SonicWall: PBR e políticas de roteamento por firmware.
17. Hillstone StoneOS: PBR associado a BGP e precedência de regras.
18. MPLS L3VPN: configuração típica e compatibilidade de BGP/VRF.
19. Comunidades BGP: cenários de traffic engineering e diferenças por fabricante.
20. Pesquisa histórica: encontrar versão arquivada de manual técnico que mudou de URL.

Para cada pergunta, medir: cobertura de fontes úteis; número de **fontes independentes**; comandos específicos corretos; compatibilidade por release; links funcionais; contradições detectadas; rastreabilidade; custo/tempo; documentos de referência humanamente verificados.

---

## 17. Decisões ainda abertas

1. **Agente:** adaptar GPT Researcher, Hyperresearch, RAGFlow ou implementar orquestração própria? Precisa de comparativo e protótipo.
2. **Biblioteca RAG:** RAGFlow ou pipeline próprio sobre OpenSearch? Evitar duplicação e lock-in.
3. **Crawling distribuído:** StormCrawler, Nutch ou ampliação gradual de Scrapy? Medir primeiro.
4. **Modelos de IA:** local, cloud ou híbrido? Política de custos, privacidade e desempenho.
5. **Licenças exatas:** revisar por versão antes de incorporar componentes, inclusive redistribuição comercial futura.
6. **Orçamento de coleta:** limites por fonte, domínio, hora, campanha, tamanho e tipos de arquivo.
7. **Política de arquivo integral:** respeitar direitos autorais/licenças e exclusão de conteúdo restrito.
8. **Metadados estruturados:** taxonomia de fabricantes/modelos/firmwares, revisão humana quando ambíguo.
9. **Hardware:** definir após benchmark de índices, PDFs, OCR e análise de IA.
10. **Nome final e posicionamento:** “Network Intelligence Engine”, “Git Eye” e “Web Eye” são nomes provisórios, não marcas registradas aqui.

---

## 18. Links e fontes mencionados na conversa

Esses endereços são **pontos de partida históricos citados ao longo das mensagens e arquivos**. Eles não são certificação de que uma versão, política ou funcionalidade permanece inalterada.

### 18.1. Busca Git / código

- Zoekt — https://github.com/sourcegraph/zoekt
- OpenGrok — https://github.com/oracle/opengrok
- Sourcegraph — https://sourcegraph.com/
- Sourcebot — https://github.com/sourcebot-dev/sourcebot
- Livegrep — https://github.com/livegrep/livegrep
- Hound — https://github.com/hound-search/hound
- Software Heritage — https://www.softwareheritage.org/
- Software Heritage Archive — https://archive.softwareheritage.org/
- ecosyste.ms — https://ecosyste.ms/
- grep.app — https://grep.app/
- GitHub Code Search — https://github.com/search?type=code
- GitHub REST Search — https://docs.github.com/en/rest/search/search
- GitLab Search — https://docs.gitlab.com/
- GH Archive — https://www.gharchive.org/
- World of Code — https://worldofcode.org/

### 18.2. Web e coleta

- SearXNG — https://github.com/searxng/searxng
- Mwmbl — https://github.com/mwmbl/mwmbl
- Mwmbl Search — https://mwmbl.org/
- Common Crawl — https://commoncrawl.org/
- Wayback Machine — https://web.archive.org/
- Scrapy — https://github.com/scrapy/scrapy
- Crawl4AI — https://github.com/unclecode/crawl4ai
- Apache Nutch — https://github.com/apache/nutch
- Apache StormCrawler — https://github.com/apache/stormcrawler
- Heritrix — https://github.com/internetarchive/heritrix3
- YaCy — https://yacy.net/
- Crawlee — https://github.com/apify/crawlee
- Ahmia crawler — https://github.com/ahmia/ahmia-crawler
- bitmagnet — https://github.com/bitmagnet-io/bitmagnet

### 18.3. Extração, índices e conhecimento

- Apache Tika — https://github.com/apache/tika
- Docling — https://github.com/docling-project/docling
- ArchiveBox — https://github.com/ArchiveBox/ArchiveBox
- OpenSearch — https://github.com/opensearch-project/OpenSearch
- Apache Solr — https://solr.apache.org/
- PostgreSQL — https://www.postgresql.org/
- MinIO — https://github.com/minio/minio
- Qdrant — https://github.com/qdrant/qdrant
- Paperless-ngx — https://github.com/paperless-ngx/paperless-ngx
- RAGFlow — https://github.com/infiniflow/ragflow
- Onyx — https://github.com/onyx-dot-app/onyx
- Open WebUI — https://github.com/open-webui/open-webui
- AnythingLLM — https://github.com/Mintplex-Labs/anything-llm
- GPT Researcher — https://github.com/assafelovic/gpt-researcher
- Hyperresearch — https://github.com/jordan-gibbs/hyperresearch
- LangChain Open Deep Research — https://github.com/langchain-ai/open_deep_research

### 18.4. Precedentes e documentação de redes

- Diffbot — https://www.diffbot.com/technology
- Juniper Marvis — https://www.juniper.net/
- Cisco AI — https://www.cisco.com/
- Cisco Community — https://community.cisco.com/
- MikroTik Forum — https://forum.mikrotik.com/
- Viva o Linux — https://www.vivaolinux.com.br/
- Stack Overflow — https://stackoverflow.com/
- Network Engineering Stack Exchange — https://networkengineering.stackexchange.com/
- Server Fault — https://serverfault.com/
- RFC Editor — https://www.rfc-editor.org/
- RIPE NCC — https://www.ripe.net/
- NANOG — https://nanog.org/
- FRRouting — https://frrouting.org/

---

## 19. Relação com os relatórios e materiais anteriores

Quatro arquivos colocados no chat foram preservados **integralmente e sem alteração** no diretório `originais/` do ZIP:

- `Texto colado.txt` — consolidações de outras IAs, correções de Sourcebot/GH Archive, arquitetura e acréscimo posterior sobre Mwmbl.
- `Markdown colado (2).md` — múltiplos relatórios sobre Git, web, Tor, BitTorrent, Nutch, OpenGrok, documentos e síntese de ferramentas.
- `Texto colado (3).txt` — relatório extenso sobre rastreamento distribuído, Zoekt, Nutch, TorBot/TorCrawl e bitmagnet, incluindo afirmações técnicas que precisam validação.
- `Texto colado (4).txt` — pesquisa textual detalhada sobre Software Heritage, Code Eye/Web Eye, Common Crawl, Ahmia, bitmagnet, GPT Researcher e limitações.

**Atenção a citações:** alguns desses materiais contêm marcadores `cite...`, que só funcionavam em outras conversas ou respostas de IA. Foram mantidos como texto original, **não são referências bibliográficas automaticamente verificadas neste pacote**. Endereços e números específicos devem ser checados antes de usar em decisões de produção ou divulgação.

---

## 20. Estado final e recomendação de continuidade

**Estado final em 30/10/2026:** o usuário queria preservar todo o planejamento antes de excluir o chat. A direção conceitual foi aprovada para prototipagem, mas **não houve entrega de implementação funcional do motor nem medição de benchmark**.

**Próxima ação recomendada:** iniciar um repositório privado com este arquivo mestre e os originais, criar a especificação técnica do MVP e testar uma mesma bateria de perguntas complexas usando (a) busca convencional, (b) busca Git + web federada, e (c) investigação com fontes locais e verificação cruzada. Somente após isso decidir escala, arquitetura de agentes e funcionalidades adicionais.

**Essência preservada:** “Quero definir o assunto e deixar o sistema descobrir sozinho onde estão as evidências. Quero que ele siga ramificações de links, menções, PDFs e códigos, cruze fontes independentes, entenda comandos de redes por fabricante/firmware e construa uma biblioteca técnica permanente, confiável e crescente.”
