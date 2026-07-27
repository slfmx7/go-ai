# Qdrant 向量数据库 Go 客户端完全指南

> 基于 Qdrant Go Client v1.9.0+ 编写  
> 最后更新：2026-07-26

---

## 目录

1. [核心概念](#一核心概念)
2. [API 演进与使用](#二api-演进与使用)
3. [向量类型详解](#三向量类型详解)
4. [过滤搜索深度解析](#四过滤搜索深度解析)
5. [底层原理：HNSW + Bitmap](#五底层原理hnsw--bitmap)
6. [性能优化最佳实践](#六性能优化最佳实践)
7. [常见错误与解决方案](#七常见错误与解决方案)
8. [数据结构参考](#八数据结构参考)
9. [代码模板速查](#九代码模板速查)

---

## 一、核心概念

### 1.1 向量相似度搜索 vs 标量过滤

| 特性 | 向量相似度搜索 | 标量过滤/排序 |
|------|----------------|---------------|
| **解决的问题** | "像不像"（语义相关） | "谁更大/更小/等于"（业务规则） |
| **核心逻辑** | 计算高维空间距离（余弦、欧氏、点积） | 精确匹配、范围查询、布尔逻辑 |
| **适用场景** | 语义搜索、RAG、以图搜图、推荐系统 | 价格区间、时间范围、品牌筛选、状态过滤 |
| **计算方式** | 向量距离计算 | 比较运算符（=, >, <, IN） |
| **典型 SQL** | 传统 SQL 无法实现 | `WHERE price BETWEEN 100 AND 500` |

### 1.2 混合检索（Hybrid Search）

**定义**：在满足业务条件的前提下，进行向量相似度搜索。

**核心优势**：
- ✅ 结果一定满足业务条件
- ✅ 结果数量稳定（不会因为过滤而变少）
- ✅ 性能依然高效（毫秒级）

**这是向量数据库区别于传统向量检索库（如 Faiss）的核心杀手锏。**

---

## 二、API 演进与使用

### 2.1 统一使用 `Query` API（废弃 `Search`）

Qdrant 从 1.7.0 版本开始引入统一的 Query API，Go 客户端 v1.9.0+ 已全面跟进。

**旧版（已废弃）**：
```go
client.Search(ctx, &qdrant.SearchPoints{
    Vector: []float32{0.1, 0.2, ...},  // 直接传向量
    // ...
})
```

**新版（推荐）**：
```go
client.Query(ctx, &qdrant.QueryPoints{
    Query: qdrant.NewQuery(vector...),  // 用工厂函数包裹
    // ...
})
```

### 2.2 `NewQuery()` 工厂函数的多态机制

Go 客户端通过接口 `isVectorInput_Variant()` 实现多态。底层有多个变体：

```go
// 底层接口定义（简化）
type VectorInput interface {
    isVectorInput_Variant()  // 标记方法
}

// 各种变体
type VectorInput_Dense struct { ... }       // 密集向量
type VectorInput_Sparse struct { ... }      // 稀疏向量
type VectorInput_Id struct { ... }          // 按 ID 查询
type VectorInput_Document struct { ... }    // 文档（服务端嵌入）
type VectorInput_Image struct { ... }       // 图片（服务端嵌入）
```

**你传入什么类型，工厂函数就自动选择哪个变体**：

| 传入数据 | 工厂函数 | 自动选择的变体 |
|----------|----------|----------------|
| `[]float32` | `qdrant.NewQuery(vector...)` | `VectorInput_Dense` |
| `SparseVector` | `qdrant.NewQuerySparse(...)` | `VectorInput_Sparse` |
| `Point ID` | `qdrant.NewQueryById(...)` | `VectorInput_Id` |
| `string` (文本) | `qdrant.NewQueryDocument(...)` | `VectorInput_Document` |

### 2.3 ⚠️ 黄金法则：永远使用客户端嵌入

**错误示例**（会导致报错）：
```go
// ❌ 错误：试图让 Qdrant 服务端帮你做嵌入
Query: qdrant.NewQueryDocument(&qdrant.Document{
    Text: "今天天气真好",
    Model: "all-MiniLM-L6-v2",
})
```

**报错信息**：
```
Inference error: Service internal error: 
InferenceService URL not configured - please provide valid address in config
```

**正确做法**：
```go
// ✅ 正确：在客户端生成向量
func QueryEmbedding(ctx context.Context, text string) (*qdrant.QueryPoints, error) {
    // 1. 在客户端生成向量（使用 OpenAI 或本地模型）
    vector, err := GenerateEmbedding(text, "text-embedding-3-small")
    if err != nil {
        return nil, err
    }
    
    // 2. 直接传向量给 Qdrant
    return &qdrant.QueryPoints{
        CollectionName: "documents",
        Query:          qdrant.NewQuery(vector...),  // 传 float32 数组
        Limit:          ptr.Uint64(10),
    }, nil
}

// 辅助函数：调用 OpenAI 生成嵌入
func GenerateEmbedding(text, model string) ([]float32, error) {
    client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
    resp, err := client.CreateEmbeddings(context.Background(), openai.EmbeddingRequest{
        Input: []string{text},
        Model: model,
    })
    if err != nil {
        return nil, err
    }
    
    // 转换 []float64 为 []float32
    embedding := make([]float32, len(resp.Data[0].Embedding))
    for i, v := range resp.Data[0].Embedding {
        embedding[i] = float32(v)
    }
    return embedding, nil
}
```

**为什么？**
- Qdrant 服务端嵌入需要额外部署模型服务（如 Hugging Face）
- 增加系统复杂度和故障点
- 性能不如客户端嵌入（网络传输文本 vs 传输向量）
- **99% 的生产环境应该使用客户端嵌入**

---

## 三、向量类型详解

### 3.1 稠密向量 (Dense Vector)

| 特性 | 描述 |
|------|------|
| **定义** | 向量中绝大多数元素都是非零实数（如 `0.12`, `-0.45`） |
| **维度** | 相对较低，通常 128 ~ 4096 维（如 OpenAI 的 `text-embedding-3-small` 是 1536 维） |
| **生成方式** | 深度学习模型（BERT、Word2Vec、OpenAI Embeddings） |
| **核心能力** | **语义理解**：懂同义词、上下文、跨语言匹配 |
| **失败案例** | 搜 "iPhone 15 Pro Max 256GB"，可能匹配到 "iPhone 14 Pro"（语义太像） |

**适用场景**：
- 语义搜索："如何修理漏水的管子" → 匹配 "水管破裂维修指南"
- RAG：从知识库找出与问题最相关的文本块
- 跨语言匹配：搜 "Apple" → 匹配 "苹果"

### 3.2 稀疏向量 (Sparse Vector)

| 特性 | 描述 |
|------|------|
| **定义** | 向量中绝大多数元素都是 0，只有极少数位置有非零值 |
| **维度** | 极高，通常等于词表大小（30,000 ~ 100,000+ 维） |
| **生成方式** | TF-IDF、BM25（传统）；SPLADE、BGE-M3（神经） |
| **核心能力** | **精确匹配**：懂关键词、专有名词、特定代码 |
| **失败案例** | 搜 "汽车"，绝对匹配不到 "轿车" 或 "vehicle"（词表里是不同维度） |

**适用场景**：
- 专有名词/型号："RTX 4090" 必须精确匹配
- 罕见词/特定代码：`Error Code: 0x80070005`
- 关键词检索：BM25 传统搜索

**Go 代码示例**：
```go
// 稀疏向量只传非零值的索引和值
sparseQuery := &qdrant.SparseVector{
    Indices: []uint32{10, 500, 1024},     // 非零元素的位置（词表索引）
    Values:  []float32{0.8, 0.5, 0.3},    // 对应位置的值（权重）
}

query := qdrant.NewQuerySparse(sparseQuery)
```

### 3.3 混合检索 (Hybrid Search)

**为什么需要混合检索？**
- 稠密向量：懂语义，但容易忽略细节
- 稀疏向量：懂精确匹配，但不懂变通
- **最佳实践**：两个一起用，取长补短

**Qdrant 实现（使用 Prefetch）**：
```go
client.Query(ctx, &qdrant.QueryPoints{
    CollectionName: "documents",
    Limit:          10,
    
    // 阶段 1：先用稀疏向量做精确关键词召回
    Prefetch: []*qdrant.Prefetch{
        {
            Query: qdrant.NewQuerySparse(sparseVector),
            Limit: 50,
        },
    },
    
    // 阶段 2：用稠密向量对这 50 个结果进行语义重排序
    Query: qdrant.NewQuery(denseVector...),
})
```

---

## 四、过滤搜索深度解析

### 4.1 核心逻辑：为什么不能"先搜后滤"？

**错误做法**：先搜出 Top 100，然后在代码里过滤。

**场景**：搜"红色的衣服"，要求"价格 < 100 元"。

1. 向量搜索找出 Top 10：
    - 结果 1-9：红色衣服，但价格都是 500 元（不符合）
    - 结果 10：蓝色裤子，价格 50 元（符合价格，不符合颜色）

2. 代码过滤后：只剩 1 个结果，而且可能不是用户最想要的（真正的"红色且便宜"的衣服可能排在第 15 名，被截断了）

**Qdrant 的做法**：在 HNSW 遍历图的时候，**实时检查**每个节点是否满足过滤条件。如果不满足，直接跳过，继续找下一个。

**结果**：保证返回的 Top-K 结果**既满足业务条件，又是语义最相似的**。

### 4.2 过滤器逻辑组合

| 字段 | 逻辑关系 | SQL 类比 | 含义 |
|------|----------|----------|------|
| **`Must`** | **AND** | `WHERE A AND B` | **必须满足**。所有条件都必须为真。 |
| **`Should`** | **OR** | `WHERE A OR B` | **应该满足**。满足其一即可（或作为加分项）。 |
| **`MustNot`** | **NOT** | `WHERE A != B` | **必须不满足**。排除符合条件的项。 |

**重要规则**：
- ✅ **所有硬性条件（价格、状态、品牌）必须放在 `Must` 中**
- ⚠️ `Should` 仅用于软性加分（如"VIP 商品优先展示"）
- ❌ **不要把价格范围等硬性条件放在 `Should` 中**

### 4.3 常用 Condition 类型

#### 1. 精确匹配 (`Match`)

**单值匹配**：
```go
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
}
```

**多值匹配（IN 查询）**：
```go
// ✅ Go 客户端中正确的多值匹配写法
{
    ConditionOneOf: &qdrant.Condition_Field{
        Field: &qdrant.FieldCondition{
            Key: "brand",
            Match: &qdrant.Match{
                MatchValue: &qdrant.Match_Keywords{  // 注意：Go 中叫 Match_Keywords
                    Keywords: &qdrant.RepeatedStrings{
                        Strings: []string{"Nike", "Adidas"},
                    },
                },
            },
        },
    },
}
```

> ⚠️ **注意**：Go 客户端中**没有** `MatchAny` 字段（Python 中有），多值匹配使用 `Match_Keywords`。

#### 2. 范围查询 (`Range`)

```go
{
    ConditionOneOf: &qdrant.Condition_Field{
        Field: &qdrant.FieldCondition{
            Key: "price",
            Range: &qdrant.Range{
                Gte: ptr.Float64(500),   // Greater Than or Equal
                Lte: ptr.Float64(1500),  // Less Than or Equal
            },
        },
    },
}
```

#### 3. 其他条件

- **`HasId`**：指定只搜索某些 Point ID
- **`IsEmpty`**：字段不存在或为空
- **`IsNull`**：字段值为 null
- **`Nested`**：嵌套对象查询（如 `address.city == "London"`）
- **`Geo`**：地理位置半径搜索

### 4.4 完整过滤条件示例

```go
func BuildFilter() *qdrant.Filter {
    gteV, lteV := float64(500), float64(1500)
    
    return &qdrant.Filter{
        Must: []*qdrant.Condition{
            // 1. 品牌必须是 Nike 或 Adidas
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
            // 2. 必须是一手（未使用）
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
            // 3. 价格在 500-1500 之间（必须在 Must 中！）
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
        // 硬性条件都在 Must 中，Should 留空
        Should:  []*qdrant.Condition{},
        MustNot: []*qdrant.Condition{},
    }
}
```

---

## 五、底层原理：HNSW + Bitmap

### 5.1 两大武器

#### 武器 1：HNSW（负责"找得准"）
- **作用**：在多维空间中快速定位与查询向量距离最近的点
- **数据结构**：图（Graph），节点是向量，边是连接关系
- **原理**：分层导航小世界图（Hierarchical Navigable Small World）
    - 顶层：稀疏的"高速公路"，快速跨越远距离
    - 底层：密集的"普通街道"，精确找到最近邻居

#### 武器 2：倒排索引 / Bitmap（负责"筛得快"）
- **作用**：快速判断某个 Point 是否满足特定标量条件
- **数据结构**：位图（Bitmap）

**什么是 Bitmap？**
假设 Collection 有 100 万个数据点，对于 `city="London"` 这个条件：
- Qdrant 维护一个长度为 100 万的**二进制字符串**
- 第 1 位是 `1`：ID 为 1 的点，城市是 London
- 第 2 位是 `0`：ID 为 2 的点，城市不是 London
- ...

**为什么用 Bitmap？**
CPU 处理二进制位运算（AND, OR, NOT）的速度是**纳秒级**的。判断 100 万个点里哪些是 London，只需要几毫秒。

### 5.2 协同工作流程（边搜边滤）

**步骤 1：生成过滤位图**
```
用户传入 Filter: { brand: ["Nike", "Adidas"], price: 500-1500 }
↓
Qdrant 查询倒排索引
↓
生成 Bitmap（100 万位，满足条件的为 1，不满足的为 0）
耗时：2-5ms
```

**步骤 2：HNSW 带着"滤镜"寻路**
```
HNSW 开始从顶层向底层遍历图
↓
走到节点 A
↓
HNSW 问 Bitmap："节点 A 满足过滤条件吗？"
↓
Bitmap 回答："0（不满足，比如是 Puma 品牌）"
↓
HNSW 动作：直接跳过节点 A，不计算距离，不加入候选集
↓
继续看下一个邻居...
```

**关键魔法**：HNSW 在准备把某个邻居节点加入"候选队列"**之前**，会先查一下 Filter Bitmap。如果不满足，**直接无视**。

**结果**：最终返回的 Top-K 结果，**天然就是经过过滤的**。

### 5.3 智能回退机制（Fallback）

**问题**：如果过滤条件极其苛刻怎么办？
- 例如：100 万个点里，只有 **10 个点** 满足条件（过滤率 99.99%）
- 此时 HNSW 在图上跳跃，发现周围邻居全被 Bitmap 拦截了（全是 0）
- HNSW 会陷入"找不到路"的困境

**Qdrant 的解决方案**：
```
1. 搜索前，选择性评估器（Selectivity Estimator）预估满足条件的数据量
↓
2. 如果满足条件的数据 > 1%：
   使用 HNSW + Bitmap 预过滤（走高速公路，遇到红灯就拐弯）
↓
3. 如果满足条件的数据 < 1%：
   果断放弃 HNSW，切换为 Exact Search（精确扫描）
   - 直接捞出满足条件的少量 ID
   - 暴力计算它们与查询向量的距离
   - 排序返回
   （比让 HNSW 瞎转悠要快得多，且 100% 准确）
```

### 5.4 生死前提：必须建立索引！

**默认情况**：Qdrant **不会** 为 Payload 字段建立索引。

**后果**：如果直接对未索引的字段进行过滤，Qdrant 会退化为**全表扫描**（Linear Scan），速度极慢（几百毫秒甚至秒级），完全丧失向量数据库的性能优势。

**解决方案**：
```go
// 1. 为 brand 建立关键词索引（用于 Match 查询）
client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
    CollectionName: "products",
    FieldName:      "brand",
    FieldType:      qdrant.FieldType_FieldTypeKeyword,
})

// 2. 为 price 建立浮点数索引（用于 Range 查询）
client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
    CollectionName: "products",
    FieldName:      "price",
    FieldType:      qdrant.FieldType_FieldTypeFloat,
})

// 3. 为 condition 建立关键词索引
client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
    CollectionName: "products",
    FieldName:      "condition",
    FieldType:      qdrant.FieldType_FieldTypeKeyword,
})
```

**务必在插入数据前或插入后执行一次索引创建！**

---

## 六、性能优化最佳实践

### 6.1 `WithPayload` 的性能陷阱

**默认行为**：Qdrant 默认 `with_payload = true`，会返回所有业务字段。

**性能灾难示例**：
- 假设 Payload 平均大小：5 KB（标题、摘要、作者、URL 等）
- Top-K = 50
- 网络传输量 = 50 × 5 KB = **250 KB**
- 后果：带宽打满，反序列化消耗大量 CPU

**优化方案**：设置为 `false`
```go
scoredPoints, err := client.Query(ctx, &qdrant.QueryPoints{
    CollectionName: "articles",
    Query:          qdrant.NewQuery(vector...),
    Limit:          ptr.Uint64(10),
    WithPayload:    qdrant.NewWithPayload(false),  // ✅ 关键优化
    WithVectors:    qdrant.NewWithVectors(false),  // 默认就是 false
})
```

**传输量**：50 × (ID + Score) ≈ **2 KB**（性能提升 **百倍**）

### 6.2 ID 桥接模式（生产环境标准架构）

```go
// 1. 向量库极速检索，只拿 ID 和 Score
searchRes, err := QdrantClient.Query(ctx, &qdrant.QueryPoints{
    CollectionName: "articles",
    Query:          qdrant.NewQuery(vector...),
    Limit:          ptr.Uint64(10),
    WithPayload:    qdrant.NewWithPayload(false),  // 拒绝冗余数据
})

// 2. 提取 ID
var ids []string
for _, point := range searchRes.GetResult() {
    ids = append(ids, point.GetId().GetUuid())
}

// 3. 去 MySQL/Redis 批量查询详细信息
// SELECT id, title, author, url, created_at FROM articles WHERE id IN (...)
articles, err := db.BatchGetArticles(ctx, ids)

// 4. 按 Qdrant 返回的相似度顺序组装结果
return assembleResults(articles, searchRes.GetResult())
```

**优势**：
- ✅ 向量数据库保持极简和极致性能
- ✅ 复杂业务逻辑留在成熟的关系型数据库中处理
- ✅ 数据一致性由 MySQL/Redis 保证

### 6.3 "连表查询"的三种替代方案

Qdrant **不支持** 传统 SQL 的复杂 JOIN，采用以下方案：

#### 方案 1：反范式化（首选，性能最高）
**核心思想**：在写入时，把需要"连表"的字段作为 Payload 一起存进去。

```go
// ❌ 不推荐：分开存
// Collection A (商品向量): { id: 1, vector: [...], payload: { name: "iPhone" } }
// Collection B (库存信息): { id: 1, payload: { stock: 50, price: 5999 } }

// ✅ 推荐：反范式化合并
{
    "id": 1,
    "vector": [0.1, 0.2, ...],
    "payload": {
        "name": "iPhone 15",
        "price": 5999,       // 原本在另一张表的数据
        "stock": 50,         // 原本在另一张表的数据
        "seller_id": 1001,
    },
}
```

**优点**：一次查询搞定，速度最快  
**缺点**：如果关联数据频繁更新，需要同步更新 Payload（使用 `SetPayload` API）

#### 方案 2：Qdrant 原生 `WithLookup`（轻量级 JOIN）
**适用场景**：数据实在太大，或关联关系复杂。

```go
// Collection A: products（存商品向量，payload 包含 seller_id）
// Collection B: sellers（存商家信息，无向量）

searchRes, err := client.Query(ctx, &qdrant.QueryPoints{
    CollectionName: "products",
    Query:          qdrant.NewQuery(vector...),
    Limit:          ptr.Uint64(5),
    WithPayload:    qdrant.NewWithPayload(true),
    
    // 👇 核心：类似 SQL 的 LEFT JOIN
    WithLookup: &qdrant.WithLookup{
        Collection: "sellers",  // 要去关联的集合
        WithPayload: &qdrant.WithPayloadSelector{
            Include: &qdrant.PayloadIncludeSelector{
                Fields: []string{"seller_name", "seller_rating"},
            },
        },
    },
})
```

**返回结果**中会多出 `lookup` 字段，包含对应 `seller_id` 在 `sellers` 集合中的信息。

**优点**：解耦商品和商家数据  
**缺点**：每次搜索增加一次内部 RPC 调用，性能略低于方案 1

#### 方案 3：应用层二次查询（最灵活，最常用）
**核心思想**：向量库只负责"找 ID"，业务代码去 MySQL/Redis 补全详情。

见上文 **ID 桥接模式**。

---

## 七、常见错误与解决方案

### 7.1 类型错误：`uint64` vs `*uint64`

**错误信息**：
```
cannot use uint64(topK) (type uint64) as type *uint64
```

**原因**：Protobuf 生成的可选字段是指针类型，用于区分"未设置"（nil）和"设为 0"。

**解决方案**：
```go
// 方法 1：声明变量后取地址
limit := uint64(topK)
QueryPoints{
    Limit: &limit,
}

// 方法 2：辅助函数（推荐）
func Ptr[T any](v T) *T {
    return &v
}

QueryPoints{
    Limit: Ptr(uint64(topK)),
}

// 方法 3：直接写辅助函数
func Uint64Ptr(v uint64) *uint64 {
    return &v
}

QueryPoints{
    Limit: Uint64Ptr(uint64(topK)),
}
```

### 7.2 服务端嵌入未配置

**错误信息**：
```
Inference error: Service internal error: 
InferenceService URL not configured
```

**原因**：误用 `NewQueryDocument`，但 Qdrant 服务端未配置嵌入服务。

**解决方案**：改用客户端嵌入（见 2.3 节）

### 7.3 过滤搜索极慢

**现象**：查询耗时几百毫秒甚至秒级。

**原因**：过滤字段没有建立 Payload Index，导致无法使用 Bitmap，触发全表扫描。

**解决方案**：
```go
// 务必为所有过滤字段建立索引
client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
    CollectionName: "products",
    FieldName:      "price",
    FieldType:      qdrant.FieldType_FieldTypeFloat,
})
```

### 7.4 搜索结果不符合预期

**现象**：
- 返回了不符合过滤条件的数据
- 或结果数量不足 Top-K

**原因**：错误地将硬性过滤条件（如价格范围）放入了 `Should` 数组。

**解决方案**：将所有硬性条件移入 `Must` 数组。

### 7.5 Go 中找不到 `MatchAny` 字段

**原因**：受到 Python 客户端文档误导。

**解决方案**：Go 客户端中多值匹配的正确字段是 `Match_Keywords`（见 4.3 节）。

---

## 八、数据结构参考

### 8.1 `ScoredPoint`（搜索结果单条记录）

```go
type ScoredPoint struct {
    // Point id（唯一标识）
    Id *PointId `json:"id,omitempty"`
    
    // Payload（业务数据）
    Payload map[string]*Value `json:"payload,omitempty"`
    
    // Similarity score（相似度得分）
    Score float32 `json:"score,omitempty"`
    
    // Last update operation applied to this point（版本号）
    Version uint64 `json:"version,omitempty"`
    
    // Vectors to search（向量数据，默认不返回）
    Vectors *VectorsOutput `json:"vectors,omitempty"`
    
    // Shard key（分片键，分布式场景使用）
    ShardKey *ShardKey `json:"shard_key,omitempty"`
    
    // Order by value（标量排序时使用）
    OrderValue *OrderValue `json:"order_value,omitempty"`
}
```

**核心字段说明**：

| 字段 | 含义 | 注意事项 |
|------|------|----------|
| **`Id`** | 数据唯一标识 | 可以是 uint64 或 UUID 字符串 |
| **`Score`** | 相似度得分 | 根据 Distance 设置，越大或越小越相似 |
| **`Payload`** | 业务数据 | 需用 `.GetStringValue()` 等方法安全解包 |
| **`Version`** | 版本号 | 用于乐观并发控制 |
| **`Vectors`** | 向量本身 | 默认不返回，除非设置 `WithVectors: true` |

### 8.2 安全提取 Payload 数据

```go
for _, point := range searchRes.GetResult() {
    // 1. 提取 ID
    var pointID string
    if point.GetId() != nil {
        pointID = point.GetId().GetUuid()  // 或 GetNum()
    }
    
    // 2. 提取 Score
    score := point.GetScore()
    
    // 3. 提取 Payload（重点：解包 *Value）
    payload := point.GetPayload()
    if payload != nil {
        // 字符串字段
        if titleVal, ok := payload["title"]; ok && titleVal != nil {
            title := titleVal.GetStringValue()
        }
        
        // 整数字段
        if priceVal, ok := payload["price"]; ok && priceVal != nil {
            price := priceVal.GetIntegerValue()
        }
        
        // 浮点数字段
        if scoreVal, ok := payload["rating"]; ok && scoreVal != nil {
            rating := scoreVal.GetDoubleValue()
        }
        
        // 布尔字段
        if activeVal, ok := payload["is_active"]; ok && activeVal != nil {
            isActive := activeVal.GetBoolValue()
        }
    }
}
```

---

## 九、代码模板速查

### 9.1 完整查询函数模板

```go
func QueryFilterData(
    ctx context.Context,
    documentName string,
    vector []float32,
    filter *qdrant.Filter,
    topK int,
) ([]*qdrant.ScoredPoint, error) {
    // 1. 构建查询请求
    limit := uint64(topK)
    
    scoredPoints, err := QdrantClient.Query(ctx, &qdrant.QueryPoints{
        CollectionName: documentName,
        Query:          qdrant.NewQuery(vector...),
        Limit:          &limit,
        Filter:         filter,
        WithVectors:    qdrant.NewWithVectors(false),  // 默认不返回向量
        WithPayload:    qdrant.NewWithPayload(false),  // 只返回 ID 和 Score
    })
    
    if err != nil {
        log.Printf("Query error: %v", err)
        return nil, err
    }
    
    return scoredPoints, nil
}
```

### 9.2 过滤条件构建模板

```go
func BuildProductFilter(brand, condition string, minPrice, maxPrice float64) *qdrant.Filter {
    return &qdrant.Filter{
        Must: []*qdrant.Condition{
            // 品牌（多值匹配）
            {
                ConditionOneOf: &qdrant.Condition_Field{
                    Field: &qdrant.FieldCondition{
                        Key: "brand",
                        Match: &qdrant.Match{
                            MatchValue: &qdrant.Match_Keywords{
                                Keywords: &qdrant.RepeatedStrings{
                                    Strings: []string{brand},
                                },
                            },
                        },
                    },
                },
            },
            // 状态（单值匹配）
            {
                ConditionOneOf: &qdrant.Condition_Field{
                    Field: &qdrant.FieldCondition{
                        Key: "condition",
                        Match: &qdrant.Match{
                            MatchValue: &qdrant.Match_Keyword{
                                Keyword: condition,
                            },
                        },
                    },
                },
            },
            // 价格范围
            {
                ConditionOneOf: &qdrant.Condition_Field{
                    Field: &qdrant.FieldCondition{
                        Key: "price",
                        Range: &qdrant.Range{
                            Gte: &minPrice,
                            Lte: &maxPrice,
                        },
                    },
                },
            },
        },
        Should:  []*qdrant.Condition{},
        MustNot: []*qdrant.Condition{},
    }
}
```

### 9.3 索引创建模板

```go
func CreateIndexes(ctx context.Context, client *qdrant.Client, collectionName string) error {
    indexes := []struct {
        FieldName string
        FieldType qdrant.FieldType
    }{
        {"brand", qdrant.FieldType_FieldTypeKeyword},
        {"condition", qdrant.FieldType_FieldTypeKeyword},
        {"price", qdrant.FieldType_FieldTypeFloat},
        {"created_at", qdrant.FieldType_FieldTypeInteger},
        {"tags", qdrant.FieldType_FieldTypeKeyword},
    }
    
    for _, idx := range indexes {
        _, err := client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
            CollectionName: collectionName,
            FieldName:      idx.FieldName,
            FieldType:      idx.FieldType,
        })
        if err != nil {
            log.Printf("Failed to create index for %s: %v", idx.FieldName, err)
            return err
        }
    }
    
    return nil
}
```

### 9.4 辅助函数模板

```go
// 通用指针函数（Go 1.18+）
func Ptr[T any](v T) *T {
    return &v
}

// 类型特化辅助函数
func Uint64Ptr(v uint64) *uint64 { return &v }
func Float64Ptr(v float64) *float64 { return &v }
func StringPtr(v string) *string { return &v }
func BoolPtr(v bool) *bool { return &v }

// 安全提取 Payload 字段
func GetStringPayload(payload map[string]*qdrant.Value, key string) string {
    if val, ok := payload[key]; ok && val != nil {
        return val.GetStringValue()
    }
    return ""
}

func GetIntPayload(payload map[string]*qdrant.Value, key string) int64 {
    if val, ok := payload[key]; ok && val != nil {
        return val.GetIntegerValue()
    }
    return 0
}

func GetFloatPayload(payload map[string]*qdrant.Value, key string) float64 {
    if val, ok := payload[key]; ok && val != nil {
        return val.GetDoubleValue()
    }
    return 0.0
}

func GetBoolPayload(payload map[string]*qdrant.Value, key string) bool {
    if val, ok := payload[key]; ok && val != nil {
        return val.GetBoolValue()
    }
    return false
}
```

---

## 附录：核心概念速记卡

### 向量搜索 vs 标量过滤
- **向量搜索**：找"像不像"（语义）
- **标量过滤**：找"等于/大于/小于"（业务规则）
- **混合检索**：两者结合（向量数据库的核心优势）

### 稠密 vs 稀疏向量
- **稠密**：低维、非零值多、懂语义
- **稀疏**：高维、大部分为 0、懂精确匹配
- **最佳实践**：混合使用（Hybrid Search）

### 过滤逻辑
- **Must** = AND（必须满足）
- **Should** = OR（满足其一或加分）
- **MustNot** = NOT（必须不满足）

### 底层原理
- **HNSW**：负责快速找最近邻
- **Bitmap**：负责快速过滤
- **边搜边滤**：HNSW 遍历时实时检查 Bitmap
- **智能回退**：过滤太严时切换为暴力计算

### 性能优化
- **WithPayload: false**：只返回 ID + Score
- **ID 桥接模式**：去 MySQL/Redis 补全详情
- **建立索引**：务必为过滤字段 `CreateFieldIndex`
- **客户端嵌入**：永远不要在 Qdrant 端做嵌入

---

**文档结束**  
祝你复习顺利！🎉