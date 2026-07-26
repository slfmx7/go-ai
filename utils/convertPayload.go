package utils

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

// ConvertPoints 转出批量数据为qdrant格式
// map[string] key为数据的唯一标识 value为负载数据已经向量vector
func ConvertPoints(datas map[string]map[string]interface{}) []*qdrant.PointStruct {
	points := make([]*qdrant.PointStruct, 0, len(datas))
	for _, dataValues := range datas {
		pointStruct := &qdrant.PointStruct{
			Id:      qdrant.NewID(uuid.New().String()),
			Payload: ConvertPayload(dataValues["payload"].(map[string]interface{})),
			Vectors: qdrant.NewVectors(dataValues["vector"].([]float32)...),
		}
		points = append(points, pointStruct)
	}
	return points
}
func ConvertPayload(data map[string]interface{}) map[string]*qdrant.Value {
	payloads := make(map[string]*qdrant.Value, len(data))
	for k, v := range data {
		payloads[k] = convertValue(v)
	}
	return payloads
}

func convertValue(v interface{}) *qdrant.Value {
	if v == nil {
		return &qdrant.Value{Kind: &qdrant.Value_NullValue{NullValue: 0}}
	}

	switch val := v.(type) {
	case string:
		return &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: val}}
	case int, int32, int64:
		// 注意：Qdrant 的 IntegerValue 是 int64
		return &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: toInt64(val)}}
	case float32, float64:
		return &qdrant.Value{Kind: &qdrant.Value_DoubleValue{DoubleValue: toFloat64(val)}}
	case bool:
		return &qdrant.Value{Kind: &qdrant.Value_BoolValue{BoolValue: val}}
	case []interface{}:
		listValues := make([]*qdrant.Value, 0, len(val))
		for _, item := range val {
			listValues = append(listValues, convertValue(item))
		}
		return &qdrant.Value{Kind: &qdrant.Value_ListValue{ListValue: &qdrant.ListValue{Values: listValues}}}
	default:
		return &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: fmt.Sprintf("%v", val)}}
	}
}

func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return val
	default:
		return 0
	}
}
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float32:
		return float64(val)
	case float64:
		return val
	default:
		return 0
	}
}
