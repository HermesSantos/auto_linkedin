package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"google.golang.org/genai"
)

const defaultModel = "gemini-3.8-flash"

var retryDelays = []time.Duration{10 * time.Second, 30 * time.Second}

const systemPrompt = `Você é um ghostwriter que escreve textos para o LinkedIn no lugar do autor.

A mensagem do usuário traz o pedido, a lista de interesses do autor e alguns textos escritos por ele.

Regras:
- Imite o estilo dos textos de exemplo: vocabulário, ritmo, tamanho das frases, estrutura e formatação.
- Não copie trechos dos exemplos; use-os só como referência de estilo.
- Escolha um assunto da lista de interesses (ou combine alguns), a menos que o pedido já defina o assunto.
- Respeite o tom e as instruções do pedido.
- Escreva em português do Brasil.
- Responda APENAS com o texto final, sem introdução, explicação ou comentários.`

func main() {
	args := os.Args[1:]
	indexing := len(args) > 0 && args[0] == "index"

	var prompt string
	if !indexing {
		prompt = strings.TrimSpace(strings.Join(args, " "))
		if prompt == "" {
			prompt = ask("Pedido: ")
		}
		if prompt == "" {
			fmt.Fprintln(os.Stderr, "uso: go run . \"faça um texto com tom ácido e irônico\"\n     go run . index [pasta]")
			os.Exit(1)
		}
	}

	loadEnv(".env")
	apiKey := os.Getenv("API_DO_GEMINI")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "API_DO_GEMINI não definida")
		os.Exit(1)
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL não definida")
		os.Exit(1)
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
	db, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro ao conectar no banco:", err)
		os.Exit(1)
	}
	defer db.Close(ctx)

	if indexing {
		dir := defaultPostsDir
		if len(args) > 1 {
			dir = args[1]
		}
		if err := indexPosts(ctx, client, db, dir); err != nil {
			fmt.Fprintln(os.Stderr, "erro ao indexar:", err)
			os.Exit(1)
		}
		return
	}

	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = defaultModel
	}

	text, err := generate(ctx, client, db, model, prompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
	fmt.Println(text)

	switch strings.ToLower(ask("\nPostar no LinkedIn? [s/N] ")) {
	case "s", "sim":
	default:
		return
	}

	token := os.Getenv("API_DO_LINKEDIN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "API_DO_LINKEDIN não definida")
		os.Exit(1)
	}
	url, err := postToLinkedIn(token, text)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro ao postar:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "postado:", url)
}

var stdin = bufio.NewReader(os.Stdin)

func ask(question string) string {
	fmt.Fprint(os.Stderr, question)
	line, _ := stdin.ReadString('\n')
	return strings.TrimSpace(line)
}

func generate(ctx context.Context, client *genai.Client, db *pgx.Conn, model, prompt string) (string, error) {
	vec, err := embed(ctx, client, prompt, "RETRIEVAL_QUERY")
	if err != nil {
		return "", err
	}
	examples, err := searchExamples(ctx, db, vec, topK)
	if err != nil {
		return "", err
	}
	if len(examples) == 0 {
		return "", fmt.Errorf("nenhum post indexado, rode: go run . index")
	}
	fmt.Fprintf(os.Stderr, "[rag] %d exemplos encontrados\n", len(examples))

	userPrompt := fmt.Sprintf("Pedido: %s\n\nInteresses do autor: %s\n\nTextos de exemplo do autor:\n\n%s",
		prompt, strings.Join(techInterests, ", "), strings.Join(examples, "\n\n---\n\n"))

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
	}
	resp, err := generateWithRetry(ctx, client, model,
		[]*genai.Content{genai.NewContentFromText(userPrompt, genai.RoleUser)}, config)
	if err != nil {
		return "", err
	}
	return resp.Text(), nil
}

func generateWithRetry(ctx context.Context, client *genai.Client, model string, history []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	for i := 0; ; i++ {
		resp, err := client.Models.GenerateContent(ctx, model, history, config)
		var apiErr genai.APIError
		if err == nil || !errors.As(err, &apiErr) || apiErr.Code != http.StatusServiceUnavailable || i >= len(retryDelays) {
			return resp, err
		}
		fmt.Fprintf(os.Stderr, "[503] modelo sobrecarregado, tentando de novo em %s\n", retryDelays[i])
		time.Sleep(retryDelays[i])
	}
}

func loadEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		if os.Getenv(k) == "" {
			os.Setenv(k, strings.Trim(strings.TrimSpace(v), `"'`))
		}
	}
}
