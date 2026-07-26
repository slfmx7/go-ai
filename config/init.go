package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

var ConfigInfo *Config

type Config struct {
	AIConfig     AIConfig     `yaml:"ai"`
	QdrantConfig QdrantConfig `yaml:"qdrant"`
}

type AIConfig struct {
	ApiKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
	BaseUrl string `yaml:"base_url"`
}

type QdrantConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	Dimension      int    `yaml:"dimension"` // 向量维度
	EmbeddingModel string `yaml:"embedding_model"`
}

// 读取配置文件
func loadConfig() *Config {
	// 1. 指定配置文件路径 (可以根据需要改为从环境变量读取路径)
	configPath := "config/config.yaml"

	// 2. 读取文件内容
	data, err := os.ReadFile(configPath)
	if err != nil {
		// 如果配置文件不存在或无法读取，直接终止程序（对于核心配置，这是合理的）
		log.Fatalf("无法读取配置文件 %s: %v", configPath, err)
	}

	// 3. 解析 YAML 数据到结构体
	var cfg *Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatalf("无法解析配置文件 %s: %v", configPath, err)
	}
	return cfg
}

// init 函数在 main 函数执行前自动运行
func init() {
	ConfigInfo = loadConfig()
}
