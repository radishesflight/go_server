// Package service code_package_service.go - 代码包管理
//
// 业务:
//   - List:按 (project_id, endpoint_id) 查代码包列表(build_time desc)
//   - Upload:上传文件到 OSS,后端自动生成 version + 拼接 full_name
//   - Pull:触发部署(当前 mock,后续接堡垒机 ad-hoc)
//
// 命名规范:
//   - 前端上传时填 name(如 "前端包",不含版本号/扩展名)
//   - 后端生成 version(自增 v2.5.x)
//   - full_name = name + "-" + version + "." + ext
package service

import (
	"errors"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"go.uber.org/zap"

	"go_server/config"
	"go_server/internal/model"
	"go_server/pkg/logger"
)

// 业务错误
var (
	ErrPackageNotFound   = errors.New("代码包不存在")
	ErrPackageExtInvalid = errors.New("文件扩展名与端不匹配")
	ErrPackageTooLarge   = errors.New("代码包超过大小限制(200MB)")
	ErrPackageUpload     = errors.New("代码包上传失败")
	ErrPackageCreate     = errors.New("代码包入库失败")
	ErrPackageDelete     = errors.New("代码包删除失败")
	ErrProjectNoEndpoint = errors.New("该项目未配置该端")
)

// 最大 200MB(代码包比图片大很多)
const maxCodePackageSize = 200 * 1024 * 1024

// CodePackageService 代码包业务
type CodePackageService struct{}

// NewCodePackageService 构造
func NewCodePackageService() *CodePackageService { return &CodePackageService{} }

// formatPackage 把 model 序列化成给前端的 map
func formatPackage(p model.CodePackages) map[string]interface{} {
	return map[string]interface{}{
		"id":          p.ID,
		"project_id":  p.ProjectID,
		"endpoint_id": p.EndpointID,
		"name":        p.Name,
		"version":     p.Version,
		"full_name":   p.FullName,
		"ext":         p.Ext,
		"size":        p.Size,
		"file_url":    p.FileURL,
		"uploader_id": p.UploaderID,
		"build_time":  p.BuildTime,
		"note":        p.Note,
		"status":      p.Status,
		"created_at":  model.FormatTime(p.CreatedAt),
	}
}

// List 按 (project_id, endpoint_id) 查代码包列表
// 都为 0 表示不过滤
func (s *CodePackageService) List(projectID, endpointID uint) ([]map[string]interface{}, error) {
	db := model.DB.Model(&model.CodePackages{}).Where("status = ?", 1)
	if projectID > 0 {
		db = db.Where("project_id = ?", projectID)
	}
	if endpointID > 0 {
		db = db.Where("endpoint_id = ?", endpointID)
	}
	var pkgs []model.CodePackages
	if err := db.Order("build_time desc, id desc").Find(&pkgs).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, len(pkgs))
	for i, p := range pkgs {
		out[i] = formatPackage(p)
	}
	return out, nil
}

// Get 单条
func (s *CodePackageService) Get(id uint) (map[string]interface{}, error) {
	if id == 0 {
		return nil, ErrPackageNotFound
	}
	var p model.CodePackages
	if err := model.DB.First(&p, id).Error; err != nil {
		return nil, ErrPackageNotFound
	}
	return formatPackage(p), nil
}

// Upload 上传代码包
//
//	projectID, endpointID:必填
//	name:前端填的原始名(不含版本号/扩展名)
//	fileName, size, open:跟 upload_service 一致的回调
//	note:可空,uploaderID 从 token 取
//	build_time / uploader 由服务端自动生成(当前时间 + token.user_id)
//
// 返回:序列化后的 map(含 id, full_name, file_url 等)
func (s *CodePackageService) Upload(
	projectID, endpointID uint,
	name, fileName string, size int64,
	open func() (io.ReadCloser, error),
	note string, uploaderID uint,
) (map[string]interface{}, error) {
	if projectID == 0 || endpointID == 0 {
		return nil, ErrPackageNotFound
	}
	if name == "" {
		return nil, ErrPackageExtInvalid
	}
	// 校验端存在
	var ep model.CodeEndpoints
	if err := model.DB.First(&ep, endpointID).Error; err != nil {
		return nil, ErrEndpointNotFound
	}
	// 校验项目存在
	var proj model.BusinessProjects
	if err := model.DB.First(&proj, projectID).Error; err != nil {
		return nil, ErrProjectNotFound
	}
	// 校验端-项目已关联
	var rel model.BusinessProjectEndpoints
	if err := model.DB.Where("project_id = ? AND endpoint_id = ? AND status = ?", projectID, endpointID, 1).
		First(&rel).Error; err != nil {
		return nil, ErrProjectNoEndpoint
	}
	// 校验扩展名(不限制,任何后缀都允许,后端只存不校验)
	// 端 ext 只用于前端展示,不上传时强校验
	ext := strings.ToLower(path.Ext(fileName))
	if ext == "" {
		ext = "." + strings.ToLower(ep.Ext)
	}
	// 校验大小
	if size > maxCodePackageSize {
		return nil, ErrPackageTooLarge
	}

	// 推 OSS(单独函数)
	objectKey, fileURL, err := uploadCodePackageToOSS(fileName, size, open)
	if err != nil {
		return nil, err
	}

	// 生成 version + full_name(full_name 用上传文件的真实 ext,不用端 ext)
	version := nextVersion(projectID, endpointID)
	fullName := fmt.Sprintf("%s-%s%s", name, version, ext)

	// 入库
	pkg := model.CodePackages{
		ProjectID:  projectID,
		EndpointID: endpointID,
		Name:       name,
		Version:    version,
		FullName:   fullName,
		Ext:        ext,
		Size:       size,
		FileURL:    fileURL,
		FilePath:   objectKey,
		UploaderID: uploaderID,
		BuildTime:  time.Now().Format("2006-01-02 15:04:05"),
		Note:       note,
		Status:     1,
	}
	if err := model.DB.Create(&pkg).Error; err != nil {
		return nil, ErrPackageCreate
	}
	return formatPackage(pkg), nil
}

// Delete 软删除代码包(OSS 文件保留 — 后续做定时清理)
func (s *CodePackageService) Delete(id uint) error {
	if id == 0 {
		return ErrPackageNotFound
	}
	if err := model.DB.Delete(&model.CodePackages{}, id).Error; err != nil {
		return ErrPackageDelete
	}
	return nil
}

// Pull 触发部署
// 当前是 mock:睡 1 秒,80% 成功 20% 失败
// 后续接堡垒机 ad-hoc:从 project_endpoint_deploy_targets 查目标资产,跑 git pull
func (s *CodePackageService) Pull(id uint) (map[string]interface{}, error) {
	pkg, err := s.Get(id)
	if err != nil {
		return nil, ErrPackageNotFound
	}
	// mock 延迟
	time.Sleep(1 * time.Second)
	if time.Now().UnixNano()%5 == 0 {
		return map[string]interface{}{
			"success":    false,
			"err":        "mock: 部署失败,目标主机不可达",
			"package_id": id,
			"full_name":  pkg["full_name"],
		}, nil
	}
	return map[string]interface{}{
		"success":    true,
		"time_cost":  1.0,
		"material":   fmt.Sprintf("cd /data/www/%s && curl -O %s && unzip -o %s", pkg["name"], pkg["file_url"], pkg["full_name"]),
		"package_id": id,
		"full_name":  pkg["full_name"],
	}, nil
}

// nextVersion 生成下一个自增版本号(v2.5.x)
// 查当前 (project_id, endpoint_id) 下最大 version 的最后一段数字 +1
func nextVersion(projectID, endpointID uint) string {
	var latest model.CodePackages
	prefix := "v2.5."
	if err := model.DB.Where("project_id = ? AND endpoint_id = ? AND version LIKE ?", projectID, endpointID, prefix+"%").
		Order("id desc").First(&latest).Error; err != nil {
		// 没有历史版本,从 0 开始
		return prefix + "0"
	}
	parts := strings.Split(latest.Version, ".")
	if len(parts) < 4 {
		return prefix + "0"
	}
	n, err := strconv.Atoi(parts[3])
	if err != nil {
		return prefix + "0"
	}
	return fmt.Sprintf("%s%d", prefix, n+1)
}

// uploadCodePackageToOSS 推文件到 OSS
// objectKey:code_packages/<project_id>/<endpoint_id>/<timestamp><ext>
func uploadCodePackageToOSS(
	filename string, size int64,
	open func() (io.ReadCloser, error),
) (string, string, error) {
	cfg := config.AppConfig.Aliyun.OSS
	if cfg.Endpoint == "" || cfg.AccessKeyID == "" || cfg.BucketName == "" {
		return "", "", ErrUploadOSSNotConfig
	}
	client, err := oss.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret)
	if err != nil {
		return "", "", ErrUploadClient
	}
	bucket, err := client.Bucket(cfg.BucketName)
	if err != nil {
		return "", "", ErrUploadBucket
	}
	f, err := open()
	if err != nil {
		return "", "", ErrUploadRead
	}
	defer f.Close()

	ext := path.Ext(filename)
	objectKey := fmt.Sprintf("code_packages/%d%s", time.Now().UnixNano(), ext)
	if err := bucket.PutObject(objectKey, f); err != nil {
		logger.L.Error("OSS PutObject failed",
			zap.String("object_key", objectKey),
			zap.String("endpoint", cfg.Endpoint),
			zap.String("bucket", cfg.BucketName),
			zap.Int64("file_size", size),
			zap.Error(err),
		)
		return "", "", ErrUploadPut
	}

	url := cfg.BucketDomain
	if url == "" {
		url = fmt.Sprintf("https://%s.%s/%s", cfg.BucketName, cfg.Endpoint, objectKey)
	} else {
		url = strings.TrimRight(url, "/") + "/" + objectKey
	}
	return objectKey, url, nil
}
