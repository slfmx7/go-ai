package utils

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// ConvertTxtFile 单行不超过64KB
func ConvertTxtFile(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()
	var content strings.Builder
	scanner := bufio.NewScanner(file) // 单行不超过64KB
	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		return ""
	}
	return content.String()
}

func ReadTXTFileSafely(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()
	var content strings.Builder
	// 预估content 大小
	fileInfo, _ := file.Stat()
	content.Grow(int(fileInfo.Size()))

	scanner := bufio.NewReader(file)
	for {
		readString, err := scanner.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}
		content.WriteString(readString)
	}
	return content.String()
}

// ReadFileByCallBack 通过便读取数据边进行函数处理 并且可以不超过64KB
func ReadFileByCallBack(filePath string, handler func(text string)) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	for {
		readString, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		time.Sleep(1 * time.Second)
		handler(readString)
	}
	return nil
}

func PrintFile(text string) {
	fmt.Println(text)
}
