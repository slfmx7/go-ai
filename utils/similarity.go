package utils

import (
	"errors"
	"math"
)

// CosineSimilarity 余弦相似度 使用 a向量*b向量的模 除以 a向量的模 * b向量的模
// 也就是去算夹角 夹角值越小越相识
func CosineSimilarity(vecA, vecB []float32) (float32, error) {
	if len(vecA) != len(vecB) {
		return 0, errors.New("向量维度不一致")
	}
	if len(vecA) == 0 {
		return 0, errors.New("向量不能为空")
	}

	var dotProduct, normA, normB float64 // 使用 float64 防止 float32 精度丢失

	for i := 0; i < len(vecA); i++ {
		a := float64(vecA[i])
		b := float64(vecB[i])
		dotProduct += a * b
		normA += a * a
		normB += b * b
	}

	// 计算模长
	denominator := math.Sqrt(normA) * math.Sqrt(normB)

	// 防御性检查：防止除以 0（零向量情况）
	if denominator == 0 {
		return 0, errors.New("存在零向量，无法计算余弦相似度")
	}

	return float32(dotProduct / denominator), nil
}

// EuclideanDistance 计算两个向量的欧氏距离
// 返回值范围: [0, +∞)，越小越相似
func EuclideanDistance(vecA, vecB []float32) (float32, error) {
	if len(vecA) != len(vecB) {
		return 0, errors.New("向量维度不一致")
	}

	var sumOfSquares float64
	for i := 0; i < len(vecA); i++ {
		diff := float64(vecA[i]) - float64(vecB[i])
		sumOfSquares += diff * diff
	}

	return float32(math.Sqrt(sumOfSquares)), nil
}

// DotProduct 计算两个向量的点积
// 返回值越大越相似 (前提是向量已归一化)
// 这里的向量归一化的要意思就是每个点的模长度为1
func DotProduct(vecA, vecB []float32) (float32, error) {
	if len(vecA) != len(vecB) {
		return 0, errors.New("向量维度不一致")
	}

	var dotProduct float64
	for i := 0; i < len(vecA); i++ {
		dotProduct += float64(vecA[i]) * float64(vecB[i])
	}

	return float32(dotProduct), nil
}
