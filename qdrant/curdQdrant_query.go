package qdrant

import (
	"context"
	"log"

	"github.com/qdrant/go-client/qdrant"
)

// QueryBaseVector
/**
 * 查询向量  这里是根据向量去查询最相似的向量 只要Id 和 score
 * @param ctx 上下文
 * @param documentName 集合名称
 * @param vector 查询向量
 * @param topK 查询数量
 * @return
 */
func QueryBaseVector(ctx context.Context, documentName string, vector []float32, topK int) ([]*qdrant.ScoredPoint, error) {
	limit := uint64(topK)
	scoredPoints, err := QdrantClient.Query(ctx, &qdrant.QueryPoints{
		CollectionName: documentName,
		Query:          qdrant.NewQuery(vector...),
		Limit:          &limit,
		WithVectors:    qdrant.NewWithVectors(false),
		WithPayload:    qdrant.NewWithPayload(false),
	})
	if err != nil {
		log.Println("QueryBaseVector error:", err)
		return nil, err
	}
	return scoredPoints, nil
}

//QueryEmbeddingInQdrant
/**
 * 查询向量  这里是根据向量去查询最相似的向量(在qdrant中进行内积) 但是需要在qdrant中设置embedding_model
 * @param ctx 上下文
 * @param documentName 集合名称
 * @param vector 嵌入向量
 * @param topK
 * @return
 */
func QueryEmbeddingInQdrant(ctx context.Context, documentName, text, model string, topK int) ([]*qdrant.ScoredPoint, error) {
	limit := uint64(topK)
	scoredPoints, err := QdrantClient.Query(ctx, &qdrant.QueryPoints{
		CollectionName: documentName,
		Query:          qdrant.NewQueryDocument(&qdrant.Document{Text: text, Model: model}),
		Limit:          &limit,
		WithVectors:    qdrant.NewWithVectors(false),
		WithPayload:    qdrant.NewWithPayload(false),
	})
	if err != nil {
		log.Println("QueryBaseVector error:", err)
		return nil, err
	}
	return scoredPoints, nil
}

func QueryFilterData(ctx context.Context, documentName string, vector []float32, filter *qdrant.Filter, topK int) []*qdrant.ScoredPoint {
	topValue := uint64(topK)
	scoredPoints, err := QdrantClient.Query(ctx, &qdrant.QueryPoints{
		CollectionName: documentName,
		Query:          qdrant.NewQuery(vector...),
		Limit:          &topValue,
		Filter:         filter,
		WithVectors:    qdrant.NewWithVectors(false),
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		log.Println("QueryBaseVector error:", err)
		return nil
	}
	log.Println("QueryFilterData:", scoredPoints)
	return scoredPoints
}
