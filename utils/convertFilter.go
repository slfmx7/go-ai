package utils

import "github.com/qdrant/go-client/qdrant"

func ConvertFilterCondition() *qdrant.Filter {
	gteV := float64(500)
	lteV := float64(1600)
	// 创建一个过滤条件 品牌必须是Nike或者Adidas 必须是一手 或者价格在500-1500之间
	return &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "brand",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keywords{
								Keywords: &qdrant.RepeatedStrings{
									Strings: []string{"Nike", "Adidas"},
								},
							},
						},
					},
				},
			},
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "condition",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: "notUsed",
							},
						},
					},
				},
			},
		},
		Should: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "price",
						Range: &qdrant.Range{
							Gte: &gteV,
							Lte: &lteV,
						},
					},
				},
			},
		},
	}
}
