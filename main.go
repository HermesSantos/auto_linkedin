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

	"google.golang.org/genai"
)

const (
	defaultModel = "gemini-3.8-flash"
	maxRequests  = 3
)

var retryDelays = []time.Duration{10 * time.Second, 30 * time.Second}

const systemPrompt = `Você é um ghostwriter que escreve textos para o LinkedIn no lugar do autor.

Antes de escrever, chame AS DUAS tools (get_writing_example e get_interests) JUNTAS, em paralelo, na sua primeira resposta. Não chame nenhuma tool depois disso.

Regras:
- Imite o estilo do texto de exemplo: vocabulário, ritmo, tamanho das frases, estrutura e formatação.
- Escolha um assunto da lista de interesses (ou combine alguns), a menos que o pedido já defina o assunto.
- Respeite o tom e as instruções do pedido.
- Escreva em português do Brasil.
- Responda APENAS com o texto final, sem introdução, explicação ou comentários.`

func main() {
	prompt := strings.TrimSpace(strings.Join(os.Args[1:], " "))
	if prompt == "" {
		fmt.Fprint(os.Stderr, "Pedido: ")
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		prompt = strings.TrimSpace(line)
	}
	if prompt == "" {
		fmt.Fprintln(os.Stderr, `uso: go run . "faça um texto com tom ácido e irônico"`)
		os.Exit(1)
	}

	loadEnv(".env")
	apiKey := os.Getenv("API_DO_GEMINI")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "API_DO_GEMINI não definida")
		os.Exit(1)
	}
	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = defaultModel
	}

	text, err := generate(context.Background(), apiKey, model, prompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
	fmt.Println(text)
}

func generate(ctx context.Context, apiKey, model, prompt string) (string, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", err
	}

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
		Tools:             tools,
	}
	history := []*genai.Content{genai.NewContentFromText(prompt, genai.RoleUser)}

	for range maxRequests {
		resp, err := generateWithRetry(ctx, client, model, history, config)
		if err != nil {
			return "", err
		}

		calls := resp.FunctionCalls()
		if len(calls) == 0 {
			return resp.Text(), nil
		}

		history = append(history, resp.Candidates[0].Content)
		var parts []*genai.Part
		for _, fc := range calls {
			fmt.Fprintf(os.Stderr, "[tool] %s\n", fc.Name)
			parts = append(parts, &genai.Part{FunctionResponse: &genai.FunctionResponse{
				ID:       fc.ID,
				Name:     fc.Name,
				Response: callTool(fc),
			}})
		}
		history = append(history, genai.NewContentFromParts(parts, genai.RoleUser))
	}

	return "", fmt.Errorf("limite de %d requests atingido sem resposta final", maxRequests)
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
