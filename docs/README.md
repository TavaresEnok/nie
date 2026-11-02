# Documentação do Projeto — Network Intelligence Engine (NIE)

Catálogo central de todos os documentos técnicos, planos arquiteturais, pesquisas e registros históricos do projeto em formato Markdown (`.md`).

---

## 📌 1. Planos Mestres e Arquitetura

* **[PLANO_MESTRE_V2.md](file:///c:/Users/tavares/Documents/projetos/buscador/docs/PLANO_MESTRE_V2.md)** (ou [PLANO_MESTRE_NETWORK_INTELLIGENCE_ENGINE_V2.md](file:///c:/Users/tavares/Documents/projetos/buscador/docs/PLANO_MESTRE_NETWORK_INTELLIGENCE_ENGINE_V2.md)):
  * **Versão 2.0 (01/11/2026) — Arquitetura Oficial para Implementação.**
  * Foco: Escolher em vez de somar, monorepo Go + Huma + Svelte + PostgreSQL 18, armazenamento em filesystem content-addressed (SHA-256), proteção no transporte contra SSRF e DNS Rebinding, motor de evidências e roteiro em 4 fases.
* **[DIAGRAMA_ARQUITETURA_NIE_2_0.md](file:///c:/Users/tavares/Documents/projetos/buscador/docs/DIAGRAMA_ARQUITETURA_NIE_2_0.md)**:
  * Diagramas de fluxo de investigação, componentes lógicos e topologia física para implantação em servidor único (VM Proxmox).
* **[ARQUIVO_MESTRE_V1.md](file:///c:/Users/tavares/Documents/projetos/buscador/docs/ARQUIVO_MESTRE_V1.md)** (original: [ARQUIVO_MESTRE (1).md](file:///c:/Users/tavares/Documents/projetos/buscador/docs/ARQUIVO_MESTRE%20%281%29.md)):
  * **Versão 1.0 (30/10/2026) — Dossiê Conceitual Extenso.**
  * Contém a formulação inicial dos requisitos de telecom/ISPs, benchmark com as 20 perguntas técnicas, histórico do debate e integração dos 4 relatórios anexos.

---

## 📋 2. Guias de Transição e Continuidade

* **[CONTINUAR_NO_FUTURO.md](file:///c:/Users/tavares/Documents/projetos/buscador/docs/CONTINUAR_NO_FUTURO.md)**:
  * Prompt de retomada do projeto contendo as premissas, regras de ouro, lista de fabricantes prioritários e tarefas imediatas.
* **[LEIA_PRIMEIRO.md](file:///c:/Users/tavares/Documents/projetos/buscador/docs/LEIA_PRIMEIRO.md)**:
  * Guia introdutório sobre o pacote de preservação de dados e alertas sobre não confundir planos com código implementado.

---

## 🔬 3. Especificação Modular e Ferramentas

* **[01_DOSSIE_COMPLETO.md](file:///c:/Users/tavares/Documents/projetos/buscador/docs/01_DOSSIE_COMPLETO.md)**:
  * Especificação técnica isolada com o escopo, arquitetura, critérios de sucesso e detalhamento dos motores *Git Eye* e *Web Eye*.
* **[02_REGISTRO_CRONOLOGICO_DA_CONVERSA.md](file:///c:/Users/tavares/Documents/projetos/buscador/docs/02_REGISTRO_CRONOLOGICO_DA_CONVERSA.md)**:
  * Reconstituição cronológica e lógica dos debates que levaram à definição dos limites e da arquitetura do buscador.
* **[FONTES_E_PROJETOS.md](file:///c:/Users/tavares/Documents/projetos/buscador/docs/FONTES_E_PROJETOS.md)**:
  * Catálogo tabulado com dezenas de ferramentas, URLs e notas técnicas categorizadas (Zoekt, OpenGrok, SearXNG, Common Crawl, Crawl4AI, Scrapy, Tika, Docling, OpenSearch, etc.).

---

## 💬 4. Transcrições e Textos Originais

* **[Agente_central_de_investigacao.md](file:///c:/Users/tavares/Documents/projetos/buscador/docs/Agente_central_de_investigacao.md)**:
  * Transcrição completa e detalhada da conversa exploratória e interativa do projeto com pesquisas de precedentes (Diffbot, Juniper Marvis, Cisco AI Canvas, GPT Researcher).
* **Pasta [`originais/`](file:///c:/Users/tavares/Documents/projetos/buscador/docs/originais/)**:
  * [01_Texto_colado.md](file:///c:/Users/tavares/Documents/projetos/buscador/docs/originais/01_Texto_colado.md) — Pesquisa preliminar sobre motores Git, Deep Web, Nutch e Mwmbl.
  * [02_Markdown_colado.md](file:///c:/Users/tavares/Documents/projetos/buscador/docs/originais/02_Markdown_colado.md) — Propostas iniciais de ferramentas para o "Olho de Deus" e telecom.
  * [03_Texto_colado.md](file:///c:/Users/tavares/Documents/projetos/buscador/docs/originais/03_Texto_colado.md) — Avaliação de precedentes comerciais e open source.
  * [04_Texto_colado.md](file:///c:/Users/tavares/Documents/projetos/buscador/docs/originais/04_Texto_colado.md) — Respostas sobre viabilidade e arquitetura sugerida.
