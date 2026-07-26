package Embedding

import (
	"ai-go/config"
	"context"
	"sort"

	"github.com/sashabaranov/go-openai"
)

func GetAiClient() *openai.Client {
	clientConfig := openai.DefaultConfig(config.ConfigInfo.AIConfig.ApiKey)
	clientConfig.BaseURL = config.ConfigInfo.AIConfig.BaseUrl
	client := openai.NewClientWithConfig(clientConfig)
	return client
}

func GetEmbedding(context context.Context, text []string) ([][]float32, error) {
	client := GetAiClient()
	embedding, err := client.CreateEmbeddings(
		context,
		openai.EmbeddingRequest{
			Input: text,
			Model: openai.EmbeddingModel(config.ConfigInfo.AIConfig.Model),
		},
	)
	if err != nil {
		return nil, err
	}
	sort.Slice(embedding.Data, func(i, j int) bool {
		return embedding.Data[i].Index < embedding.Data[j].Index
	})
	result := make([][]float32, 0, len(text))
	for _, data := range embedding.Data {
		result = append(result, data.Embedding)
	}
	return result, nil
}
