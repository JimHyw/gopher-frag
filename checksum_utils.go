package gopher_frag

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
)

// 摘要算法名称常量
const (
	// ChecksumMd5 MD5摘要算法
	ChecksumMd5 = "MD5"
	// ChecksumSha1 SHA1摘要算法
	ChecksumSha1 = "SHA1"
	// ChecksumSha256 SHA256摘要算法
	ChecksumSha256 = "SHA256"
)

// 计算文件摘要值
//
//   - filePath 要计算的文件路径
//   - algorithm 摘要算法名称，支持： gopher_frag.ChecksumMd5 gopher_frag.ChecksumSha1 gopher_frag.ChecksumSha256
//   - excepted 期望的摘要值，16进制字符串，不区分大小写
//
// 当文件的摘要值和excepted相同时，返回true
func computeFileChecksum(filePath, algorithm, excepted string) (bool, error) {
	// 打开文件
	file, e := os.Open(filePath)
	if e != nil {
		return false, e
	}
	defer func() {
		_ = file.Close()
	}()
	// 根据算法选择哈希函数
	var hashChecker hash.Hash
	switch algorithm {
	case ChecksumMd5:
		hashChecker = md5.New()
	case ChecksumSha1:
		hashChecker = sha1.New()
	case ChecksumSha256:
		hashChecker = sha256.New()
	default:
		return false, fmt.Errorf("不支持的摘要算法：%s", algorithm)
	}
	// 计算摘要
	_, e = io.Copy(hashChecker, file)
	if e != nil {
		logger.ErrorLine("计算文件摘要出错！")
		return false, e
	}
	// 对比
	fileHash := strings.ToLower(fmt.Sprintf("%x", hashChecker.Sum(nil)))
	exceptedLower := strings.ToLower(excepted)
	logger.InfoLine("计算摘要完成！")
	logger.Info("期望：%s\n", exceptedLower)
	logger.Info("实际：%s\n", fileHash)
	return fileHash == exceptedLower, nil
}