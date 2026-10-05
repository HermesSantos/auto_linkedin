# auto_linkedin

CLI em Go que escreve posts para o LinkedIn usando o Gemini e, se você quiser, já publica.

O modelo imita o seu estilo de escrita a partir de um texto de exemplo (`example_text.txt`) e escolhe o assunto numa lista de interesses técnicos (definida em `tools.go`). Depois de gerar o texto, o programa pergunta se deve postar no seu perfil.

## Requisitos

- Go 1.27+
- Chave da API do Gemini
- Token de acesso do LinkedIn (com os escopos `openid profile w_member_social`), só se for publicar

## Configuração

Crie um `.env` na raiz do projeto:

```env
API_DO_GEMINI=sua_chave_do_gemini
API_DO_LINKEDIN=seu_token_do_linkedin
```

Variáveis opcionais:

- `GEMINI_MODEL`: modelo usado (padrão `gemini-3.8-flash`)
- `LINKEDIN_PERSON_URN`: seu URN (`urn:li:person:SEU_ID`), se o token não tiver o escopo `openid profile`
- `LINKEDIN_VERSION`: versão da API do LinkedIn (padrão `202609`)

Coloque um texto seu em `example_text.txt` para o modelo copiar o estilo.

## Como rodar

```bash
go run . "faça um texto com tom ácido e irônico"
```

Sem argumento, o programa pede o pedido no terminal. O texto gerado aparece na tela e o programa pergunta `Postar no LinkedIn? [s/N]`. Responda `s` para publicar.
