# Backup e restauração do NIE

O comando `nie-maint` coordena um snapshot PostgreSQL com os objetos originais e
derivados referenciados pelo catálogo. Restic cuida do repositório criptografado.
As operações foram ensaiadas com banco real e repositório Restic local isolado.

## Preparação

Compile `go build -o bin/nie-maint ./cmd/nie-maint`. Instale Restic, `pg_dump` e
`pg_restore` da versão principal do servidor PostgreSQL. Os três executáveis
precisam estar no `PATH`. O filesystem do acervo deve permitir hard links.

Crie um arquivo de senha do Restic protegido e inicialize o repositório:

```bash
export RESTIC_REPOSITORY=/caminho/fora-do-acervo/repositorio-restic
export RESTIC_PASSWORD_FILE=/caminho/protegido/senha-restic
restic init
```

O repositório deve ter seu próprio armazenamento e cópia externa conforme a
política de retenção da operação. O comando não configura retenção nem apaga
snapshots automaticamente.

## Criar backup

```bash
export NIE_DATABASE_URL='postgres://usuario:senha@localhost:5432/nie?sslmode=disable'
export NIE_DATA_DIR=/caminho/acervo
./bin/nie-maint backup -config config/searxng/settings.yml,.env
```

O resultado informa `snapshot_id`. A configuração é opcional; os arquivos
escolhidos, inclusive segredos presentes neles, entram no repositório criptografado.
A senha do repositório deve permanecer recuperável fora desse próprio backup.

A manutenção exporta um snapshot transacional. `pg_dump`, as somas das tabelas e
a seleção de arquivos usam esse mesmo estado do banco. Cada original e derivado
é validado por SHA-256 antes de entrar no staging. Hard links permitem montar o
conjunto sem copiar todo o corpus; o dump precisa de espaço temporário no acervo.
Migrações do NIE e captura do snapshot compartilham o lock de migração.

O manifesto contém versão do servidor, contagem e SHA-256 do conteúdo de cada
tabela, caminhos, tamanhos e hashes dos arquivos. Um objeto ausente ou adulterado
interrompe o backup. Os snapshots incluem `claim_revisions` e `extraction_runs`.

## Restaurar em destinos novos

Crie um banco vazio dedicado e selecione um diretório que ainda não existe:

```bash
export NIE_RESTORE_DATABASE_URL='postgres://usuario:senha@localhost:5432/nie_restaurado?sslmode=disable'
export NIE_RESTORE_DATA_DIR=/caminho/nie-restaurado
./bin/nie-maint restore ID_EXPLICITO_DO_SNAPSHOT
```

O comando recusa diretório existente e banco com tabelas. Recupera o snapshot
para staging, verifica os arquivos, restaura o dump numa transação e compara
contagens e somas de todas as tabelas. Somente após essas verificações publica
o diretório de dados. Configurações aparecem em `recovered-config/` para revisão
e aplicação deliberada; não são executadas ou sobrepostas automaticamente.

Se a verificação posterior ao `pg_restore` falhar, o banco isolado pode continuar
preenchido para diagnóstico. O diretório final não é publicado. Reinicie o ensaio
com outro banco vazio. O serviço NIE deve continuar apontando para os destinos
originais até a recuperação ser aceita.

Antes de operar no novo ambiente, revise configurações e fontes, mantenha os
workers pausados enquanto verifica a biblioteca e só então habilite a execução
de tarefas pendentes que foram preservadas pelo snapshot.

## Ensaio automatizado

`internal/backup/backup_integration_test.go` exige **dois bancos isolados e vazios**,
Restic e clientes PostgreSQL instalados:

```bash
export NIE_BACKUP_TEST_DATABASE_URL='postgres://.../nie_backup_test?sslmode=disable'
export NIE_RESTORE_TEST_DATABASE_URL='postgres://.../nie_restore_test?sslmode=disable'
go test ./internal/backup -run TestResticBackupRestoreIntegration -v
```

O ensaio cria duas extrações, citações, uma contradição, revisões de claim e uma
aresta de descoberta. Modifica a origem após o backup e demonstra que o restore
recupera o estado anterior. Confere originais e derivado por hash, roda
`restic check --read-data`, recusa sobrescrever banco preenchido e rejeita um
novo backup após adulteração de um original. Os bancos de teste devem ser
recriados vazios antes de repetir esse teste.

## Limites atuais

- O ensaio local não determina RPO, RTO ou comportamento com vários TB.
- As somas por tabela exigem uma leitura completa dos metadados/textos e podem
  precisar de outra estratégia para grandes corpora.
- O staging usa caminhos temporários por execução; o Restic deduplica conteúdo,
  mas o custo de metadados e duração ainda precisa de benchmark.
- O procedimento não implementa PITR/WAL, rotação de credenciais, retenção,
  agendamento nem recuperação automática de desastre.
- A compatibilidade com backends remotos e filesystems sem hard links requer
  ensaio próprio antes de operação.

Referências de implementação: [snapshot do pg_dump](https://www.postgresql.org/docs/current/app-pgdump.html),
[backup do Restic](https://restic.readthedocs.io/en/stable/040_backup.html) e
[restauração do Restic](https://restic.readthedocs.io/en/stable/050_restore.html).
