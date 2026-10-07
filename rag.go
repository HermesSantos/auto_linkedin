package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
	"google.golang.org/genai"
)

const (
	defaultEmbeddingModel = "gemini-embedding-001"
	embeddingDims         = 768
	defaultPostsDir       = "posts"
	topK                  = 3
)

func embed(ctx context.Context, client *genai.Client, text, taskType string) ([]float32, error) {
	model := os.Getenv("EMBEDDING_MODEL")
	if model == "" {
		model = defaultEmbeddingModel
	}
	dims := int32(embeddingDims)

	resp, err := client.Models.EmbedContent(ctx, model,
		[]*genai.Content{genai.NewContentFromText(text, genai.RoleUser)},
		&genai.EmbedContentConfig{TaskType: taskType, OutputDimensionality: &dims})
	if err != nil {
		return nil, err
	}
	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("resposta de embedding vazia")
	}
	return resp.Embeddings[0].Values, nil
}

func indexPosts(ctx context.Context, client *genai.Client, db *pgx.Conn, dir string) error {
	paths, err := filepath.Glob(filepath.Join(dir, "*.txt"))
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("nenhum .txt em %s", dir)
	}

	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := strings.TrimSpace(string(b))
		if content == "" {
			continue
		}

		vec, err := embed(ctx, client, content, "RETRIEVAL_DOCUMENT")
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}

		source := filepath.Base(path)
		_, err = db.Exec(ctx, `
			INSERT INTO posts (source, content, embedding) VALUES ($1, $2, $3)
			ON CONFLICT (source) DO UPDATE SET content = EXCLUDED.content, embedding = EXCLUDED.embedding`,
			source, content, pgvector.NewVector(vec))
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		fmt.Fprintf(os.Stderr, "[index] %s\n", source)
	}
	return nil
}

func searchExamples(ctx context.Context, db *pgx.Conn, vec []float32, k int) ([]string, error) {
	rows, err := db.Query(ctx,
		`SELECT content FROM posts ORDER BY embedding <=> $1 LIMIT $2`,
		pgvector.NewVector(vec), k)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
