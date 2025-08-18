package server

import (
	"compress/gzip"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// StaticFileServer 静态文件服务器
type StaticFileServer struct {
	config          *Config
	rootDir         string
	urlPrefix       string
	enableDirectory bool
	enableGzip      bool
	cacheMaxAge     time.Duration
}

// NewStaticFileServer 创建静态文件服务器
func NewStaticFileServer(config *Config) *StaticFileServer {
	return &StaticFileServer{
		config:          config,
		rootDir:         config.StaticDir,
		urlPrefix:       config.StaticURLPrefix,
		enableDirectory: config.EnableDirectoryList,
		enableGzip:      config.EnableCompression,
		cacheMaxAge:     24 * time.Hour, // 默认缓存24小时
	}
}

// SetupRoutes 设置静态文件路由
func (sfs *StaticFileServer) SetupRoutes(engine *gin.Engine) {
	// 确保URL前缀以/开头但不以/结尾（除非是根路径）
	urlPattern := strings.TrimSuffix(sfs.urlPrefix, "/")
	if urlPattern == "" {
		urlPattern = "/"
	}
	urlPattern += "/*filepath"

	// 注册静态文件处理器
	engine.GET(urlPattern, sfs.ServeStaticFile)
	engine.HEAD(urlPattern, sfs.ServeStaticFile)
}

// ServeStaticFile 处理静态文件请求
func (sfs *StaticFileServer) ServeStaticFile(c *gin.Context) {
	// 获取文件路径
	filePath := c.Param("filepath")
	if filePath == "" {
		filePath = "/"
	}

	// 安全检查：防止目录遍历攻击
	if !sfs.isSecurePath(filePath) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
			"code":  http.StatusForbidden,
		})
		return
	}

	// 构建完整的文件系统路径
	fullPath := filepath.Join(sfs.rootDir, filepath.Clean(filePath))

	// 检查文件是否存在
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "File not found",
				"code":  http.StatusNotFound,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
				"code":  http.StatusInternalServerError,
			})
		}
		return
	}

	// 如果是目录
	if fileInfo.IsDir() {
		sfs.handleDirectory(c, fullPath, filePath)
		return
	}

	// 处理文件
	sfs.handleFile(c, fullPath, fileInfo)
}

// handleDirectory 处理目录请求
func (sfs *StaticFileServer) handleDirectory(c *gin.Context, fullPath, urlPath string) {
	// 尝试查找索引文件
	indexFiles := []string{"index.html", "index.htm", "default.html"}
	for _, indexFile := range indexFiles {
		indexPath := filepath.Join(fullPath, indexFile)
		if fileInfo, err := os.Stat(indexPath); err == nil && !fileInfo.IsDir() {
			sfs.handleFile(c, indexPath, fileInfo)
			return
		}
	}

	// 如果启用了目录列表，显示目录内容
	if sfs.enableDirectory {
		sfs.serveDirectoryListing(c, fullPath, urlPath)
		return
	}

	// 否则返回403
	c.JSON(http.StatusForbidden, gin.H{
		"error": "Directory listing disabled",
		"code":  http.StatusForbidden,
	})
}

// handleFile 处理文件请求
func (sfs *StaticFileServer) handleFile(c *gin.Context, fullPath string, fileInfo os.FileInfo) {
	// 设置缓存头
	sfs.setCacheHeaders(c, fileInfo)

	// 检查If-Modified-Since头
	if sfs.checkNotModified(c, fileInfo) {
		c.Status(http.StatusNotModified)
		return
	}

	// 设置Content-Type
	contentType := sfs.getContentType(fullPath)
	c.Header("Content-Type", contentType)

	// 设置Content-Length
	c.Header("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

	// 检查是否支持压缩
	if sfs.shouldCompress(c, contentType) {
		sfs.serveCompressedFile(c, fullPath)
		return
	}

	// 直接提供文件
	c.File(fullPath)
}

// serveDirectoryListing 提供目录列表
func (sfs *StaticFileServer) serveDirectoryListing(c *gin.Context, fullPath, urlPath string) {
	// 读取目录内容
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read directory",
			"code":  http.StatusInternalServerError,
		})
		return
	}

	// 排序目录项
	sort.Slice(entries, func(i, j int) bool {
		// 目录在前，文件在后
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})

	// 生成HTML目录列表
	html := sfs.generateDirectoryHTML(urlPath, entries)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// generateDirectoryHTML 生成目录列表HTML
func (sfs *StaticFileServer) generateDirectoryHTML(urlPath string, entries []os.DirEntry) string {
	var html strings.Builder

	html.WriteString("<!DOCTYPE html>\n")
	html.WriteString("<html>\n<head>\n")
	html.WriteString("<meta charset=\"utf-8\">\n")
	html.WriteString(fmt.Sprintf("<title>Directory listing for %s</title>\n", urlPath))
	html.WriteString("<style>\n")
	html.WriteString(sfs.getDirectoryCSS())
	html.WriteString("</style>\n")
	html.WriteString("</head>\n<body>\n")

	html.WriteString(fmt.Sprintf("<h1>Directory listing for %s</h1>\n", urlPath))
	html.WriteString("<hr>\n")
	html.WriteString("<table>\n")
	html.WriteString("<thead>\n")
	html.WriteString("<tr><th>Name</th><th>Size</th><th>Modified</th></tr>\n")
	html.WriteString("</thead>\n")
	html.WriteString("<tbody>\n")

	// 添加返回上级目录的链接
	if urlPath != "/" {
		parentPath := path.Dir(urlPath)
		if parentPath == "." {
			parentPath = "/"
		}
		html.WriteString(fmt.Sprintf("<tr><td><a href=\"%s\">../</a></td><td>-</td><td>-</td></tr>\n", parentPath))
	}

	// 添加目录项
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}

		size := "-"
		if !entry.IsDir() {
			size = sfs.formatFileSize(info.Size())
		}

		modTime := info.ModTime().Format("2006-01-02 15:04:05")
		href := path.Join(urlPath, entry.Name())
		if entry.IsDir() {
			href += "/"
		}

		html.WriteString(fmt.Sprintf("<tr><td><a href=\"%s\">%s</a></td><td>%s</td><td>%s</td></tr>\n",
			href, name, size, modTime))
	}

	html.WriteString("</tbody>\n")
	html.WriteString("</table>\n")
	html.WriteString("<hr>\n")
	html.WriteString("<footer>Powered by Gin Static File Server</footer>\n")
	html.WriteString("</body>\n</html>\n")

	return html.String()
}

// getDirectoryCSS 获取目录列表的CSS样式
func (sfs *StaticFileServer) getDirectoryCSS() string {
	return `
body { font-family: Arial, sans-serif; margin: 40px; }
h1 { color: #333; }
table { border-collapse: collapse; width: 100%; }
th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
th { background-color: #f2f2f2; }
tr:nth-child(even) { background-color: #f9f9f9; }
a { text-decoration: none; color: #0066cc; }
a:hover { text-decoration: underline; }
footer { margin-top: 20px; color: #666; font-size: 12px; }
`
}

// formatFileSize 格式化文件大小
func (sfs *StaticFileServer) formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// isSecurePath 检查路径是否安全（防止目录遍历攻击）
func (sfs *StaticFileServer) isSecurePath(filePath string) bool {
	// 清理路径
	cleanPath := filepath.Clean(filePath)

	// 检查是否包含..或其他危险模式
	if strings.Contains(cleanPath, "..") {
		return false
	}

	// 确保路径以/开头
	if !strings.HasPrefix(cleanPath, "/") {
		return false
	}

	return true
}

// setCacheHeaders 设置缓存头
func (sfs *StaticFileServer) setCacheHeaders(c *gin.Context, fileInfo os.FileInfo) {
	// 设置Last-Modified
	c.Header("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))

	// 设置Cache-Control
	maxAge := int(sfs.cacheMaxAge.Seconds())
	c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))

	// 设置ETag
	etag := fmt.Sprintf("\"%x-%x\"", fileInfo.ModTime().Unix(), fileInfo.Size())
	c.Header("ETag", etag)
}

// checkNotModified 检查文件是否未修改
func (sfs *StaticFileServer) checkNotModified(c *gin.Context, fileInfo os.FileInfo) bool {
	// 检查If-Modified-Since
	ifModifiedSince := c.Request.Header.Get("If-Modified-Since")
	if ifModifiedSince != "" {
		if t, err := time.Parse(http.TimeFormat, ifModifiedSince); err == nil {
			if fileInfo.ModTime().Before(t.Add(1 * time.Second)) {
				return true
			}
		}
	}

	// 检查If-None-Match (ETag)
	ifNoneMatch := c.Request.Header.Get("If-None-Match")
	if ifNoneMatch != "" {
		etag := fmt.Sprintf("\"%x-%x\"", fileInfo.ModTime().Unix(), fileInfo.Size())
		if ifNoneMatch == etag {
			return true
		}
	}

	return false
}

// getContentType 获取文件的Content-Type
func (sfs *StaticFileServer) getContentType(filePath string) string {
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return contentType
}

// shouldCompress 检查是否应该压缩文件
func (sfs *StaticFileServer) shouldCompress(c *gin.Context, contentType string) bool {
	if !sfs.enableGzip {
		return false
	}

	// 检查客户端是否支持gzip
	if !strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") {
		return false
	}

	// 检查内容类型是否适合压缩
	compressibleTypes := []string{
		"text/",
		"application/javascript",
		"application/json",
		"application/xml",
		"application/css",
	}

	for _, compressibleType := range compressibleTypes {
		if strings.HasPrefix(contentType, compressibleType) {
			return true
		}
	}

	return false
}

// serveCompressedFile 提供压缩文件
func (sfs *StaticFileServer) serveCompressedFile(c *gin.Context, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to open file",
			"code":  http.StatusInternalServerError,
		})
		return
	}
	defer file.Close()

	// 设置压缩头
	c.Header("Content-Encoding", "gzip")
	c.Header("Vary", "Accept-Encoding")

	// 创建gzip写入器
	gzipWriter := gzip.NewWriter(c.Writer)
	defer gzipWriter.Close()

	// 复制文件内容到gzip写入器
	if _, err := io.Copy(gzipWriter, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to compress file",
			"code":  http.StatusInternalServerError,
		})
		return
	}
}