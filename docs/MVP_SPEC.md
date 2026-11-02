# MVP — Network Intelligence Engine

**Versão:** 0.1  
**Data:** 02/11/2026  
**Status:** especificação de implementação do vertical slice  
**Fonte arquitetural:** `PLANO_MESTRE_NETWORK_INTELLIGENCE_ENGINE_V2.md`

## 1. Objetivo do MVP

Provar, com código real e reproduzível, o fluxo:

```text
campanha → descoberta/seed → coleta segura → extração → SHA-256
        → PostgreSQL (documento, versão, trechos) → busca exata/FTS
        → citação com URL, página/ordinal e hash → novas pistas
```

O MVP **não** implementa IA, grafo visual completo, comparador técnico, OCR,
claims/evidências formais nem execução de comandos em equipamentos.

## 2. Escopo funcional

1. **Campanhas** com pergunta, seeds opcionais, orçamento e pausa/retomada.
2. **Descoberta externa** por SearXNG (opcional) e GitHub (opcional, mediante token).
3. **Crawler** HTTP(S) com fila persistente, profundidade máxima, cotas por campanha,
   limite por domínio, robots.txt e proteção SSRF/DNS rebinding.
4. **Extração** de HTML, texto e PDFs digitais; links internos viram novas pistas.
5. **Armazenamento** content-addressed por SHA-256, com escrita atômica.
6. **Busca** exata (case-insensitive), lexical (`tsvector`/`ts_rank_cd`) e
   aproximada (`pg_trgm`), com fusão e citação.
7. **Biblioteca** de documentos e visualizador de trechos por página/ordinal.
8. **Painel Svelte** para busca, campanhas e biblioteca.

## 3. Não objetivos explícitos

- Clonar a web inteira, GitHub inteiro ou operar crawling distribuído.
- Executar JavaScript de páginas, contornar anti-bot, paywall ou autenticação.
- Indexar conteúdo privado, Tor ou torrents.
- Gerar respostas por IA ou declarar validação operacional.
- Multiusuário/RBAC completo (a API aceita token único opcional).

## 4. Decisões técnicas

| Área | Decisão |
|---|---|
| Backend | Go 1.27, `net/http` + Huma v2 (`humago`) |
| Banco | PostgreSQL 18, `pg_trgm`, FTS `simple` |
| Fila | tabela `jobs` + `FOR UPDATE SKIP LOCKED` + lease |
| Storage | filesystem `objects/sha256/aa/bb/<hash>` |
| Frontend | Svelte 5 + Vite, build estático embutido no binário |
| Extração HTML | `golang.org/x/net/html` + `charset` |
| Extração PDF | `github.com/ledongthuc/pdf` (PDFs digitais; sem OCR) |
| Descoberta | seeds + links + SearXNG (opcional) + GitHub (opcional) |
| IA | ausente nesta fase |

## 5. Limites padrão por campanha

| Parâmetro | Padrão |
|---|---:|
| Profundidade | 3 |
| URLs | 200 |
| Domínios | 20 |
| Bytes totais | 1 GiB |
| Intervalo mínimo por domínio | 10 s |
| HTML por recurso | 5 MiB |
| PDF por recurso | 50 MiB |
| Retries | 3 |

Todos configuráveis na UI; variáveis de ambiente apenas para implantação.

## 6. Modelo de dados mínimo

- `campaigns`: pergunta, status, seeds, orçamento.
- `jobs`: fronteira de descoberta, lease, tentativas, payload de descoberta.
- `documents`: conteúdo único por SHA-256 + texto extraído.
- `document_versions`: ocorrência URL/conteúdo/campanha, ETag, data.
- `passages`: trechos citáveis com `tsvector` gerado e índice trigram.
- `discovery_edges`: proveniência de cada pista.
- `domain_state`: throttle compartilhado entre workers.

## 7. Critérios de aceite

1. `docker compose up` sobe PostgreSQL + aplicação sem passos manuais.
2. Criar campanha com uma seed HTTP local de teste; o worker coleta, extrai,
   armazena e permite buscar um comando/termo exato.
3. Um link encontrado na página gera novo job com `discovered_from` e profundidade.
4. Buscar um comando com símbolos (`display bgp peer`, `ip route 10.0.0.0/8`)
   retorna o trecho correto e o documento original.
5. Reiniciar o processo não perde a fila; jobs em `running` voltam a ser elegíveis
   após o lease.
6. URLs privadas/loopback/metadata são bloqueadas e cobertas por teste unitário.
7. `robots.txt` proibindo o caminho impede a coleta.
8. SHA-256 do arquivo armazenado confere com o hash exibido na API.
9. `go test ./...` e `npm run build` passam.
10. Nenhum conteúdo externo é executado.

## 8. Fases seguintes (fora deste MVP)

- Claims/evidências/origin clusters e detecção de cópias.
- Comparador por fabricante/modelo/firmware.
- Grafo visual e observabilidade avançada.
- OCR/PDF complexo em worker isolado.
- IA local/cloud com contratos validados.
