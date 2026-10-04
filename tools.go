package main

import (
	"fmt"
	"os"

	"google.golang.org/genai"
)

const exampleTextPath = "example_text.txt"

var techInterests = []string{
	"Inteligência Artificial",
	"LLMs",
	"RAG",
	"Embeddings",
	"Vector Databases",
	"pgvector",
	"AI Agents",
	"Multi-Agent Systems",
	"Reranking",
	"Context Engineering",
	"AI Engineering",
	"AI-assisted Software Development",

	"Arquitetura de Software",
	"System Design",
	"Sistemas Distribuídos",
	"Concorrência",
	"Paralelismo",
	"Escalabilidade",
	"Alta Disponibilidade",
	"Tolerância a Falhas",
	"Load Balancing",
	"Caching",
	"Filas e processamento assíncrono",
	"Event-Driven Architecture",
	"Arquitetura Orientada a Eventos",
	"Distributed Systems",
	"Performance",
	"Otimização",
	"Modelagem de Dados",
	"Banco de Dados",
	"Consistência e Transações",
	"Locks e Concorrência",
	"Observabilidade",

	"Engenharia de Software",
	"Design Patterns",
	"Clean Code",
	"Simplicidade de Software",
	"Manutenibilidade",
	"Legacy Modernization",
	"Refatoração",
	"Trade-offs arquiteturais",
	"Complexidade de Software",
	"Developer Experience",

	"Compiladores",
	"Linguagens de Programação",
	"AST",
	"Parsing",
	"Tree-sitter",
}

var tools = []*genai.Tool{{
	FunctionDeclarations: []*genai.FunctionDeclaration{
		{
			Name:        "get_writing_example",
			Description: "Retorna um texto escrito pelo autor, para imitar o estilo de escrita dele (vocabulário, ritmo, estrutura, formatação).",
		},
		{
			Name:        "get_interests",
			Description: "Retorna a lista de assuntos técnicos de interesse do autor. O texto deve tratar de um ou mais desses assuntos.",
		},
	},
}}

func callTool(fc *genai.FunctionCall) map[string]any {
	switch fc.Name {
	case "get_writing_example":
		b, err := os.ReadFile(exampleTextPath)
		if err != nil {
			return map[string]any{"error": err.Error()}
		}
		return map[string]any{"output": string(b)}
	case "get_interests":
		return map[string]any{"output": techInterests}
	default:
		return map[string]any{"error": fmt.Sprintf("tool desconhecida: %s", fc.Name)}
	}
}
