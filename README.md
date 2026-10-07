# auto_linkedin

CLI em Go que escreve posts para o LinkedIn usando o Gemini e, se você quiser, já publica.

O projeto usa RAG clássico. Seus posts antigos (pasta `posts/`) são indexados com embeddings do Gemini num Postgres com pgvector. A cada pedido, o programa busca os posts mais parecidos com o que você pediu e manda esses exemplos, junto com a lista de interesses técnicos (definida em `interests.go`), para o Gemini imitar o seu estilo. Depois de gerar o texto, o programa pergunta se deve postar no seu perfil.

## Requisitos

- Go 1.27+
- Docker (para o Postgres)
- Chave da API do Gemini
- Token de acesso do LinkedIn (com os escopos `openid profile w_member_social`), só se for publicar

## Banco de dados

Suba o Postgres (porta `5433` no host, volume `auto_linkedin_rag_pgdata`):

```bash
docker compose up -d
```

A imagem é o `postgres:18` oficial, sem pgvector. Instale a extensão no container e depois crie a tabela:

```bash
psql postgres://auto_linkedin:auto_linkedin@localhost:5433/auto_linkedin -f schema.sql
```

## Configuração

Crie um `.env` na raiz do projeto:

```env
API_DO_GEMINI=sua_chave_do_gemini
API_DO_LINKEDIN=seu_token_do_linkedin
DATABASE_URL=postgres://auto_linkedin:auto_linkedin@localhost:5433/auto_linkedin
```

Variáveis opcionais:

- `GEMINI_MODEL`: modelo usado para gerar o texto (padrão `gemini-3.8-flash`)
- `EMBEDDING_MODEL`: modelo de embedding (padrão `gemini-embedding-001`, com 768 dimensões)
- `LINKEDIN_PERSON_URN`: seu URN (`urn:li:person:SEU_ID`), se o token não tiver o escopo `openid profile`
- `LINKEDIN_VERSION`: versão da API do LinkedIn (padrão `202609`)

## Indexação

Coloque seus posts em `posts/`, um `.txt` por post, e indexe:

```bash
go run . index
```

Para usar outra pasta: `go run . index caminho/da/pasta`. O nome do arquivo é a chave no banco, então rodar de novo atualiza os posts que mudaram sem duplicar. Se trocar o `EMBEDDING_MODEL`, reindexe tudo.

## Como rodar

```bash
go run . "faça um texto com tom ácido e irônico"
```

Sem argumento, o programa pede o pedido no terminal. O texto gerado aparece na tela e o programa pergunta `Postar no LinkedIn? [s/N]`. Responda `s` para publicar.
