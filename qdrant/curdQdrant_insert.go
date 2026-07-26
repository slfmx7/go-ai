package qdrant

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

// InsertSingleVector 插入单个向量
func InsertSingleVector(ctx context.Context, documentName string, vector []float32, payloads map[string]*qdrant.Value) error {
	_, err := QdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: documentName,
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewID(uuid.New().String()),
				Vectors: qdrant.NewVectors(vector...),
				Payload: payloads,
			},
		},
	})
	if err != nil {
		log.Println("insertSingleVector error:", err)
		return err
	}
	return nil
}

func InsertBatchVector(ctx context.Context, documentName string, points []*qdrant.PointStruct) error {
	_, err := QdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: documentName,
		Points:         points,
	})
	if err != nil {
		log.Println("InsertBatchVector error:", err)
	}
	return nil
}
