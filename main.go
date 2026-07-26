package main

import (
	"ai-go/Embedding"
	"ai-go/config"
	"ai-go/qdrant"
	"ai-go/utils"
	"context"
	"fmt"
	"strconv"
)

func main() {
	queryDataFilter()
}
func queryDataFilter() {
	background := context.Background()
	text := "我要要一双价格在500-1000空军一号，并且是没使用过的"
	embedding, err := Embedding.GetEmbedding(background, []string{text})
	if err != nil {
		panic(err)
	}
	data := qdrant.QueryFilterData(background, "test-0726", embedding[0], utils.ConvertFilterCondition(), 3)
	for _, point := range data {
		fmt.Println("point:", point.Id.GetUuid(), point.Score, point.Payload)
	}
}
func queryDataInQdrantEmbedding() {
	background := context.Background()
	text := "世界真美丽，我喜欢世界！"
	points, err := qdrant.QueryEmbeddingInQdrant(background, "test-0725", text, config.ConfigInfo.QdrantConfig.EmbeddingModel, 3)
	if err != nil {
		panic(err)
	}
	for _, point := range points {
		fmt.Println("point:", point.Id.GetUuid(), point.Score)
	}
}

func queryData() {
	background := context.Background()
	text := "世界真美丽，我喜欢世界！"
	embedding, err := Embedding.GetEmbedding(background, []string{text})
	if err != nil {
		panic(err)
	}
	points, err := qdrant.QueryBaseVector(background, "test-0725", embedding[0], 3)
	if err != nil {
		panic(err)
	}
	for _, point := range points {
		fmt.Println("point:", point.Id.GetUuid(), point.Score, point.Payload)
	}
}
func insertShoppingData() {
	dbData := []map[string]interface{}{
		{
			"id":        1001,
			"brand":     "Nike",
			"price":     1001,
			"text":      "空军1号",
			"condition": "notUsed",
		},
		{
			"id":        1002,
			"brand":     "Nike",
			"price":     501,
			"text":      "空军1号",
			"condition": "used",
		},
		{
			"id":        1003,
			"brand":     "Nike",
			"price":     1500,
			"text":      "空军2号",
			"condition": "notUsed",
		},
		{
			"id":        1004,
			"brand":     "Nike",
			"price":     750,
			"text":      "空军2号",
			"condition": "used",
		},
		{
			"id":        1005,
			"brand":     "Adidas",
			"price":     1001,
			"text":      "椰子1号",
			"condition": "notUsed",
		},
		{
			"id":        1006,
			"brand":     "Adidas",
			"price":     5001,
			"text":      "椰子1号",
			"condition": "used",
		},
		{
			"id":        1007,
			"brand":     "Adidas",
			"price":     1500,
			"text":      "椰子2号",
			"condition": "notUsed",
		},
		{
			"id":        1008,
			"brand":     "Adidas",
			"price":     750,
			"text":      "椰子2号",
			"condition": "used",
		},
	}
	tempTexts := make([]string, 0, len(dbData))
	for _, data := range dbData {
		tempTexts = append(tempTexts, data["text"].(string))
	}
	ctx := context.Background()
	embedding, err := Embedding.GetEmbedding(ctx, tempTexts)
	if err != nil {
		panic(err)
	}
	tempData := make(map[string]map[string]interface{}, len(dbData))
	for i, data := range dbData {
		tempData[strconv.Itoa(data["id"].(int))] = map[string]interface{}{
			"payload": data,
			"vector":  embedding[i],
		}
	}
	points := utils.ConvertPoints(tempData)
	err2 := qdrant.CreateSingleCollection(ctx, "test-0726")
	if err2 != nil {
		panic(err2)
	}
	err = qdrant.InsertBatchVector(ctx, "test-0726", points)
	if err != nil {
		panic(err)
	}
}
func insertData() {
	dbData := []map[string]interface{}{
		{
			"id":   1001,
			"text": "hello world",
		},
		{
			"id":   1002,
			"text": "golang",
		},
		{
			"id":   1003,
			"text": "python",
		},
		{
			"id":   1004,
			"text": "java",
		},
	}
	tempTexts := make([]string, 0, len(dbData))
	for _, data := range dbData {
		tempTexts = append(tempTexts, data["text"].(string))
	}
	ctx := context.Background()
	embedding, err := Embedding.GetEmbedding(ctx, tempTexts)
	if err != nil {
		panic(err)
	}
	tempData := make(map[string]map[string]interface{}, len(dbData))
	for i, data := range dbData {
		tempData[strconv.Itoa(data["id"].(int))] = map[string]interface{}{
			"payload": data,
			"vector":  embedding[i],
		}
	}
	points := utils.ConvertPoints(tempData)
	err2 := qdrant.CreateSingleCollection(ctx, "test-0725")
	if err2 != nil {
		panic(err2)
	}
	err = qdrant.InsertBatchVector(ctx, "test-0725", points)
	if err != nil {
		panic(err)
	}
}
