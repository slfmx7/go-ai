package qdrant

import (
	"ai-go/config"
	"context"

	"github.com/qdrant/go-client/qdrant"
)

var QdrantClient *qdrant.Client

// 初始化qdrant 客户端
func init() {
	var err error
	QdrantClient, err = qdrant.NewClient(&qdrant.Config{
		Host: config.ConfigInfo.QdrantConfig.Host,
		Port: config.ConfigInfo.QdrantConfig.Port, // 使用的grpc端口 6334
	})
	if err != nil {
		panic(err)
	}
}

// CreateSingleCollection 创建集合 (单向量)
func CreateSingleCollection(ctx context.Context, documentName string) error {
	exists, err := QdrantClient.CollectionExists(ctx, documentName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	} else {
		err := QdrantClient.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: documentName,
			VectorsConfig: &qdrant.VectorsConfig{
				// 指定集合向量配置优先级高
				Config: &qdrant.VectorsConfig_Params{
					Params: &qdrant.VectorParams{
						Size:     uint64(config.ConfigInfo.QdrantConfig.Dimension), // 指定集合维度
						Distance: qdrant.Distance_Cosine,                           // 指定相似度计算算法
					},
				},
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateHybridCollection 创建集合 (混合向量)
func CreateHybridCollection(ctx context.Context, documentName string) {
	exists, err := QdrantClient.CollectionExists(ctx, documentName)
	if err != nil {
		panic(err)
	}
	if exists {
		return
	} else {
		err := QdrantClient.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: documentName,
			VectorsConfig: &qdrant.VectorsConfig{
				Config: &qdrant.VectorsConfig_ParamsMap{
					ParamsMap: &qdrant.VectorParamsMap{
						Map: map[string]*qdrant.VectorParams{
							"text_vector": {
								Size:       uint64(config.ConfigInfo.QdrantConfig.Dimension),
								Distance:   qdrant.Distance_Cosine,
								HnswConfig: &qdrant.HnswConfigDiff{}, // 指定集合向量配置优先级高
							},
							"image_vector": {
								Size:     uint64(config.ConfigInfo.QdrantConfig.Dimension),
								Distance: qdrant.Distance_Cosine,
							},
						},
					},
				},
			},
		})
		if err != nil {
			panic(err)
		}
	}
}

//
