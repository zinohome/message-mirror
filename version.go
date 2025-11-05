package main

import (
	"bufio"
	"os"
	"strings"
)

var (
	// Version 版本号，通过VERSION文件读取
	Version = "0.1.1"
	// BuildTime 构建时间，编译时注入
	BuildTime = "unknown"
	// GitCommit Git提交哈希，编译时注入
	GitCommit = "unknown"
)

func init() {
	// 尝试从VERSION文件读取版本号
	if versionFile, err := os.Open("VERSION"); err == nil {
		defer versionFile.Close()
		scanner := bufio.NewScanner(versionFile)
		if scanner.Scan() {
			version := strings.TrimSpace(scanner.Text())
			if version != "" {
				Version = version
			}
		}
	}
}

