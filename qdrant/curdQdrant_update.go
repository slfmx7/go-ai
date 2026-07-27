package qdrant

import (
	"context"

	"github.com/qdrant/go-client/qdrant"
)

// UpdateVectors 更新向量(仅仅更新向量，负载数据不会发生改变)
/**
 * @param ctx 上下文
 * @param documentName 集合名称
 * @param vectors 向量
 * @param point 点ID
 */
func UpdateVectors(ctx context.Context, documentName string, vectors []float32, point *qdrant.PointId) {
	_, err := QdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: documentName,
		Points: []*qdrant.PointStruct{
			{
				Id: &qdrant.PointId{
					PointIdOptions: point.PointIdOptions,
				},
				Vectors: qdrant.NewVectors(vectors...),
			},
		},
	})
	if err != nil {
		panic(err)
	}
}

// UpdateSetPayload 更新负载数据(仅仅更新负载数据，向量不会发生改变)
/**
 * @param ctx 上下文
 * @param documentName 集合名称
 * @param payloads 负载数据
 * @param point 点ID
 */

func UpdateSetPayload(ctx context.Context, documentName string, payloads map[string]*qdrant.Value, point *qdrant.PointId) {
	_, err := QdrantClient.SetPayload(ctx, &qdrant.SetPayloadPoints{
		CollectionName: documentName,
		PointsSelector: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{
						point,
					},
				},
			},
		},
		Payload: payloads,
	})
	if err != nil {
		panic(err)
	}
}

// UpdateSetPayloadNoPointId 更新负载数据(不通过点ID，而是通过过滤条件)
/**
 * @param ctx 上下文
 * @param documentName 集合名称
 * @param payloads 负载数据
 * @param filter 过滤条件
 */
func UpdateSetPayloadNoPointId(ctx context.Context, documentName string, payloads map[string]*qdrant.Value, filter *qdrant.Filter) {
	_, err := QdrantClient.SetPayload(ctx, &qdrant.SetPayloadPoints{
		CollectionName: documentName,
		PointsSelector: qdrant.NewPointsSelectorFilter(filter),
		Payload:        payloads,
	})
	if err != nil {
		panic(err)
	}
}

// UpdateOverwritePayload 覆盖更新负载数据(覆盖更新负载数据，向量不会发生改变)
/**
 * @param ctx 上下文
 * @param documentName 集合名称
 * @param payloads 负载数据
 * @param point 点ID
 */
func UpdateOverwritePayload(ctx context.Context, documentName string, payloads map[string]*qdrant.Value, point *qdrant.PointId) {
	_, err := QdrantClient.OverwritePayload(ctx, &qdrant.SetPayloadPoints{
		CollectionName: documentName,
		PointsSelector: qdrant.NewPointsSelector(point),
		Payload:        payloads,
	})
	if err != nil {
		panic(err)
	}
}

// UpdateDeletePayload 删除负载数据(删除负载数据-可以删除某些特定的负载数据，向量不会发生改变)
/**
 * @param ctx 上下文
 * @param documentName 集合名称
 * @param payloadKey 负载数据key
 * @param point 点ID
 */
func UpdateDeletePayload(ctx context.Context, documentName string, payloadKey []string, point *qdrant.PointId) {
	_, err := QdrantClient.DeletePayload(ctx, &qdrant.DeletePayloadPoints{
		CollectionName: documentName,
		PointsSelector: qdrant.NewPointsSelector(point),
		Keys:           payloadKey,
	})
	if err != nil {
		panic(err)
	}
}
