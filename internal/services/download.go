package services

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jyjiangkai/stat/config"
	"github.com/jyjiangkai/stat/log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var (
	bucket = "ai-knowledgebase-files"
	// key    = "github|10882129/使用多维表格分组和筛选.pdf"
	// filename = "使用多维表格分组和筛选.pdf"
)

type DownloadService struct {
	cfg    config.S3
	closeC chan struct{}
}

func NewDownloadService(cfg config.S3) *DownloadService {
	os.Setenv("AWS_ACCESS_KEY_ID", cfg.AWSAccessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", cfg.AWSSecretAccessKey)
	return &DownloadService{
		cfg:    cfg,
		closeC: make(chan struct{}),
	}
}

func (ds *DownloadService) Start() error {
	return nil
}

func (ds *DownloadService) Stop() error {
	return nil
}

func (ds *DownloadService) Download(ctx *gin.Context, url string) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String("us-west-2"), // 根据实际情况调整区域
		Credentials: credentials.NewEnvCredentials(),
	}))

	// Create a downloader with the session and default options
	downloader := s3manager.NewDownloader(sess)

	// 创建临时缓存文件
	file, err := ioutil.TempFile("", "temp.file")
	if err != nil {
		log.Error(ctx).Err(err).Msg("failed to create temp file")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to create temp file: %s", err.Error()),
		})
		return
	}
	defer os.Remove(file.Name()) // 使用完毕后删除临时文件

	// Write the contents of S3 Object to the file
	n, err := downloader.Download(file, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(url),
	})
	if err != nil {
		log.Error(ctx).Err(err).Str("url", url).Msg("failed to download file")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to download file: %s", err.Error()),
		})
		return
	}
	log.Info(ctx).Msgf("success to download %s, %d bytes", url, n)

	urls := strings.Split(url, "/")
	if len(urls) != 2 {
		log.Error(ctx).Err(err).Str("url", url).Msg("url error")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("url error: %s", err.Error()),
		})
		return
	}

	// 设置响应头
	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", strings.Split(url, "/")[1]))
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Content-Transfer-Encoding", "binary")
	ctx.Header("Expires", "0")
	ctx.Header("Cache-Control", "must-revalidate")
	ctx.Header("Pragma", "public")

	// 将文件内容写入响应体
	_, err = io.Copy(ctx.Writer, file)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to write response: %s", err.Error()),
		})
		return
	}
}
