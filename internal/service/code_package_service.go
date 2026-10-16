// Package service code_package_service.go - 代码包管理
//
// 业务:
//   - List:按 (project_id, endpoint_id) 查代码包列表(build_time desc)
//   - Upload:上传文件到对象存储(S3 兼容),后端自动生成 version + 拼接 full_name
//   - Pull:触发部署(真实逻辑,经 JMS ad-hoc 任务跑 cd+curl+unzip 串)
//
// 命名规范:
//   - 前端上传时填 name(如 "前端包",不含版本号/扩展名)
//   - 后端生成 version(时间戳 v<unix>)
//   - full_name = name + "-" + version + "." + ext
package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"go_server/config"
	"go_server/internal/model"
	"go_server/pkg/jumpserver"
	"go_server/pkg/logger"
)

// 业务错误
var (
	ErrPackageNotFound      = errors.New("代码包不存在")
	ErrPackageExtInvalid    = errors.New("文件扩展名与端不匹配")
	ErrPackageTooLarge      = errors.New("代码包超过大小限制(200MB)")
	ErrPackageUpload        = errors.New("代码包上传失败")
	ErrPackageCreate        = errors.New("代码包入库失败")
	ErrPackageDelete        = errors.New("代码包删除失败")
	ErrProjectNoEndpoint    = errors.New("该项目未配置该端")
	ErrPackageNoDeployTarget = errors.New("代码包未配置部署目标")
	ErrPackagePullFail      = errors.New("代码包部署失败")
)

// JMS 拉取总超时:5min(100MB 包下载 + 解压足够,JMS 任务调度本身有 ~5s 延迟)
const codePackagePullTimeout = 5 * time.Minute

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

// List 按 (project_id, asset_id, asset_path_id) 查代码包列表
// 任何参数 = 0 表示不过滤
//
// 2026-10-16 改造:
//   - 不再支持 endpoint_id(已废弃,字典表不在截图页面用)
//   - 加 asset_id / asset_path_id 过滤,给"apk/前端包管理"页用
func (s *CodePackageService) List(projectID, assetID, assetPathID uint) ([]map[string]interface{}, error) {
	db := model.DB.Model(&model.CodePackages{}).Where("status = ?", 1)
	if projectID > 0 {
		db = db.Where("project_id = ?", projectID)
	}
	if assetID > 0 {
		db = db.Where("asset_id = ?", assetID)
	}
	if assetPathID > 0 {
		db = db.Where("asset_path_id = ?", assetPathID)
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
//	name:前端填的原始名(不含版本号/扩展名)
//	fileName, size, open:跟 upload_service 一致的回调
//	note:可空,uploaderID 从 token 取
//	build_time / uploader 由服务端自动生成(当前时间 + token.user_id)
//	assetID / assetPathID:必填,project_id 由 service 内部从 asset 反查
//
// 2026-10-16 改造:
//   - 3 个外键全部入库:project_id / asset_id / asset_path_id
//   - 不再写 code_package_deploy_targets(已废弃,功能并入本表)
//
// 返回:序列化后的 map(含 id, full_name, file_url, project_id, asset_id, asset_path_id)
func (s *CodePackageService) Upload(
	name, fileName string, size int64,
	open func() (io.ReadCloser, error),
	note string, uploaderID uint,
	assetID, assetPathID uint,
) (map[string]interface{}, error) {
	if name == "" {
		return nil, ErrPackageExtInvalid
	}
	if assetID == 0 {
		return nil, ErrPackageNotFound
	}
	// 校验资产存在且启用,同时拿到它的 project_id
	var asset model.BusinessAssets
	if err := model.DB.First(&asset, assetID).Error; err != nil {
		return nil, ErrPackageNotFound
	}
	if asset.Status != 1 {
		return nil, ErrPackageNotFound
	}
	// 校验路径(如果传了)存在并归属该资产
	if assetPathID > 0 {
		var p model.BusinessAssetPaths
		if err := model.DB.First(&p, assetPathID).Error; err != nil {
			return nil, ErrPackageNotFound
		}
		if p.AssetID != assetID {
			return nil, ErrPackageNotFound
		}
	}
	// 校验大小
	if size > maxCodePackageSize {
		return nil, ErrPackageTooLarge
	}

	// 推对象存储(S3 兼容,启动期由 main.go 注入)
	objectKey, fileURL, err := uploadCodePackageToStorage(fileName, size, open)
	if err != nil {
		return nil, err
	}

	// 扩展名(从文件名拿)
	ext := strings.ToLower(path.Ext(fileName))
	if ext == "" {
		ext = ".zip"
	}

	// version 用时间戳避免重复
	version := fmt.Sprintf("v%d", time.Now().Unix())
	fullName := fmt.Sprintf("%s-%s%s", name, version, ext)

	// 入库(单条 Insert,3 个外键全填)
	pkg := model.CodePackages{
		ProjectID:   asset.ProjectID,
		AssetID:     assetID,
		AssetPathID: assetPathID,
		Name:        name,
		Version:     version,
		FullName:    fullName,
		Ext:         ext,
		Size:        size,
		FileURL:     fileURL,
		FilePath:    objectKey,
		UploaderID:  uploaderID,
		BuildTime:   time.Now().Format("2006-01-02 15:04:05"),
		Note:        note,
		Status:      1,
	}
	if err := model.DB.Create(&pkg).Error; err != nil {
		// 事务回滚了,但对象存储已推成功,异步清理兜底
		logger.L.Warn("code_package create failed, object left in storage",
			zap.String("object_key", objectKey),
			zap.Error(err),
		)
		return nil, ErrPackageCreate
	}
	out := formatPackage(pkg)
	out["project_id"] = asset.ProjectID
	out["asset_id"] = assetID
	out["asset_path_id"] = assetPathID
	return out, nil
}

// Delete 硬删除代码包 + 同步删 OSS
// 策略:先硬删 DB,再删 OSS。OSS 删失败 log warning 但不返错(DB 已删,OSS 文件由定时任务兜底)
func (s *CodePackageService) Delete(id uint) error {
	if id == 0 {
		return ErrPackageNotFound
	}
	var pkg model.CodePackages
	if err := model.DB.First(&pkg, id).Error; err != nil {
		return ErrPackageNotFound
	}

	// 硬删 DB(项目要求不使用软删,Unscoped 跳过 gorm.DeletedAt)
	if err := model.DB.Unscoped().Delete(&pkg).Error; err != nil {
		return ErrPackageDelete
	}

	// 再删对象存储(失败不返错,只 log)
	if pkg.FilePath != "" {
		if err := Uploader().Delete(context.Background(), pkg.FilePath); err != nil {
			logger.L.Warn("storage DeleteObject failed after DB hard-delete",
				zap.String("object_key", pkg.FilePath),
				zap.Uint("package_id", id),
				zap.Error(err),
			)
		}
	}
	return nil
}

// GetDownloadURL 生成对象存储签名下载 URL(1小时有效,带 Content-Disposition: attachment)
// 前端拿这个 URL 后用 <a download> 触发浏览器下载
func (s *CodePackageService) GetDownloadURL(id uint) (string, error) {
	if id == 0 {
		return "", ErrPackageNotFound
	}
	var pkg model.CodePackages
	if err := model.DB.First(&pkg, id).Error; err != nil {
		return "", ErrPackageNotFound
	}

	uploader := Uploader()
	if uploader == nil {
		return "", ErrUploadOSSNotConfig
	}

	// 签名 URL + 强制 attachment,浏览器收到会下载而不是预览
	signedURL, err := uploader.PresignGet(
		context.Background(),
		pkg.FilePath,
		time.Hour,
		true, // forceDownload
	)
	if err != nil {
		return "", ErrUploadPut
	}
	// PresignGet 拿到的 URL 没带 content-disposition query(我们用 reqParams.Set
	// 设的 response-content-disposition 在 minio 客户端被忽略),这里手动拼接
	// query 让浏览器下载时用 full_name 作文件名
	if pkg.FullName != "" {
		sep := "?"
		if strings.Contains(signedURL, "?") {
			sep = "&"
		}
		signedURL = signedURL + sep + "response-content-disposition=attachment%3B%20filename%3D%22" +
			url.QueryEscape(pkg.FullName) + "%22"
	}
	return signedURL, nil
}

// Pull 触发部署(真实逻辑)
//
// 2026-10-16 改造:直接从 code_packages 拿 asset_id / asset_path_id,不再查 deploy_targets
//
// 流程:
//  1. 拿 package(asset_id / asset_path_id 都在主表)
//  2. 拿 business_assets(jms_id、address) + business_asset_paths(部署目录)
//  3. 生成对象存储签名下载 URL(1h 有效,curl 可直接下)
//  4. 拼 shell 命令:备份现有内容 → curl 下载 → unzip 解压 → 清理 zip
//  5. 调 JMS ad-hoc 任务,后端内部轮询直到终态,拿 stdout
//  6. 一次性返回完整结果给前端
//
// 失败语义:
//   - 任何一步配置缺失(没 asset_id / 资产禁用 / jms_id 空 / 路径不属于资产)→ 直接返回业务错
//   - JMS 任务执行失败(exit_code != 0 / status=failed)→ 返回 (result, ErrPackagePullFail)
//     此时 res 仍带 material(stdout),handler 用 ErrorWithData 透传给前端
func (s *CodePackageService) Pull(id uint) (map[string]interface{}, error) {
	if id == 0 {
		return nil, ErrPackageNotFound
	}

	// 1. 拿 package
	var pkg model.CodePackages
	if err := model.DB.First(&pkg, id).Error; err != nil {
		return nil, ErrPackageNotFound
	}
	if pkg.AssetID == 0 {
		return nil, ErrPackageNoDeployTarget
	}

	// 2. 拿资产
	var asset model.BusinessAssets
	if err := model.DB.First(&asset, pkg.AssetID).Error; err != nil {
		return nil, ErrPackageNotFound
	}
	if asset.Status != 1 {
		return nil, ErrPackageNotFound
	}
	if asset.JmsID == "" {
		return nil, ErrPackageNoDeployTarget
	}

	// 3. 拿路径(可空:asset_path_id=0 时不切目录,在默认 cwd 跑)
	var deployPath string
	if pkg.AssetPathID > 0 {
		var p model.BusinessAssetPaths
		if err := model.DB.First(&p, pkg.AssetPathID).Error; err != nil {
			return nil, ErrPackageNotFound
		}
		if p.AssetID != asset.ID {
			return nil, ErrPackageNotFound
		}
		deployPath = p.Path
	}

	// 5. 生成 OSS 签名下载 URL(1h 有效)
	uploader := Uploader()
	if uploader == nil {
		return nil, ErrUploadOSSNotConfig
	}
	signedURL, err := uploader.PresignGet(
		context.Background(),
		pkg.FilePath,
		time.Hour,
		false, // 服务器上 curl 下载,不要 attachment
	)
	if err != nil {
		logger.L.Error("storage PresignGet failed in Pull",
			zap.Uint("package_id", id),
			zap.String("file_path", pkg.FilePath),
			zap.Error(err),
		)
		return nil, ErrUploadPut
	}

	// 6. 拼 shell 命令:备份原内容 → curl → unzip → 删 zip
	//    备份用 ./_backup_<ts> 子目录,带隐藏文件(ls -A);解压后清理 zip
	//    注意:命令里涉及远端路径,做一层防注入:
	//      - path 用 strconv.Quote 防空格/$/反引号
	//      - signedURL 是 PresignGet 输出,天然安全
	//      - fullName 来自 pkg.FullName,后端生成,安全
	//      - zipFilename 用本机拼接的简单名(避免全路径),安全
	cmd := buildCodePackagePullCmd(deployPath, signedURL, pkg.FullName)

	logger.L.Info("code_package Pull start",
		zap.Uint("package_id", id),
		zap.Uint("asset_id", asset.ID),
		zap.String("asset_name", asset.Name),
		zap.String("jms_id", asset.JmsID),
		zap.String("deploy_path", deployPath),
		zap.String("full_name", pkg.FullName),
	)

	// 7. 调 JMS ad-hoc 任务(后端内部轮询到终态,5min 总超时)
	jmsClient := jumpserver.NewClient(
		config.AppConfig.Jumpserver.BaseURL,
		config.AppConfig.Jumpserver.Token,
		config.AppConfig.Jumpserver.OrgID,
	)
	jmsRes, jmsErr := jmsClient.RunCommand(asset.JmsID, cmd, codePackagePullTimeout)

	// 8. 拼最终响应
	out := map[string]interface{}{
		"success":     jmsErr == nil,
		"package_id":  id,
		"full_name":   pkg.FullName,
		"asset_id":    asset.ID,
		"asset_name":  asset.Name,
		"deploy_path": deployPath,
		"jms_id":      asset.JmsID,
		"command":     cmd,
	}
	if jmsRes != nil {
		out["is_finished"] = jmsRes["is_finished"]
		out["is_success"] = jmsRes["is_success"]
		out["status"] = jmsRes["status"]
		out["time_cost"] = jmsRes["time_cost"]
		if m, ok := jmsRes["material"].(string); ok {
			out["material"] = m
		}
		if sm, ok := jmsRes["summary"].(map[string]interface{}); ok {
			out["summary"] = sm
		}
	}
	if jmsErr != nil {
		out["err"] = jmsErr.Error()
		logger.L.Warn("code_package Pull failed",
			zap.Uint("package_id", id),
			zap.String("jms_id", asset.JmsID),
			zap.String("err", jmsErr.Error()),
		)
		return out, ErrPackagePullFail
	}
	logger.L.Info("code_package Pull success",
		zap.Uint("package_id", id),
		zap.String("jms_id", asset.JmsID),
	)
	return out, nil
}

// buildCodePackagePullCmd 拼 JMS 远端执行的 shell 命令串
//
//	deployPath: 部署目录(例 /data/www/web/ceshi);为空则不切目录,在 JMS 任务默认 cwd 跑
//	signedURL : 对象存储签名下载 URL(1h 有效,curl 可直接下)
//	fullName  : 代码包文件名(包含版本号 + .zip,后端生成)
//
// 流程:
//  1. mkdir -p deployPath(确保目标存在)
//  2. zip -qr 把 deployPath 整个目录打包成 <dir_name>_<ts>.zip 放到**父目录**里
//     (不 mv deployPath,保留原目录名;备份 zip 与原目录平级,带时间戳避免撞名)
//  3. curl 下载代码包 zip 到 deployPath 内
//  4. unzip -o 解压到 deployPath/__deploy_tmp_<pid>/
//  5. 智能平铺:
//     - 若临时目录第一层只有 1 个 entry 且是目录(如 ceshi/index.html 这种 "single-root"),
//       cd 进该目录后再平铺 — 这是 web 前端包最常见的结构
//     - 若临时目录第一层是多个 entry(扁平:index.html, dist/, package.json),
//       就在临时目录原地平铺
//     - 临时目录空 → 报"空 zip"错误
//  6. 把当前目录所有内容 mv 到 deployPath(同名先 rm -rf 再 mv,实现覆盖)
//  7. 清理临时目录和原 zip
//
// 失败时**不**回滚 backup(原目录已经被覆盖,backup 是救命稻草,留着)
func buildCodePackagePullCmd(deployPath, signedURL, fullName string) string { return buildCodePackagePullCmdImpl(deployPath, signedURL, fullName) }

func buildCodePackagePullCmdImpl(deployPath, signedURL, fullName string) string {
	// shell 引用,防特殊字符(/, $, 空格 等)
	quotedPath := "."
	if deployPath != "" {
		quotedPath = strconv.Quote(deployPath)
	}
	quotedFile := strconv.Quote(fullName)
	quotedURL := strconv.Quote(signedURL)

	var sb strings.Builder
	sb.WriteString("set +e\n")
	// 时间戳格式: 2026-10-16-11:47:11(用户指定,带 -)
	sb.WriteString("TS=$(date +%Y-%m-%d-%H:%M:%S)\n")
	// PARENT_DIR / DIR_NAME 后面 zip 包命名要用
	if deployPath != "" {
		sb.WriteString(fmt.Sprintf("DEPLOY_DIR=%s\n", quotedPath))
	} else {
		// 没传 deployPath 时,parent 是 . dir_name 是 deploy(占位)
		sb.WriteString("DEPLOY_DIR=$(pwd)\n")
	}
	sb.WriteString("PARENT_DIR=$(dirname \"$DEPLOY_DIR\")\n")
	sb.WriteString("DIR_NAME=$(basename \"$DEPLOY_DIR\")\n")
	sb.WriteString("BACKUP_ZIP=\"${PARENT_DIR}/${DIR_NAME}_${TS}.zip\"\n")
	// 临时解压目录:放在目标目录内,带 PID 避免重名
	sb.WriteString("TMP_DIR=\"${DEPLOY_DIR}/__deploy_tmp_$$\"\n")
	// 代码包最终位置(目标目录 + fullName)
	sb.WriteString("CODE_ZIP=\"${DEPLOY_DIR}/" + strings.ReplaceAll(fullName, "'", "'\\''") + "\"\n")

	// 0. 部署目录必须存在
	sb.WriteString("mkdir -p \"$DEPLOY_DIR\"\n")

	// 1. 备份原目录(整个目录打包成 zip,放在父目录下,与原目录平级)
	sb.WriteString("(cd \"$PARENT_DIR\" && zip -qr \"${DIR_NAME}_${TS}.zip\" \"$DIR_NAME\")\n")
	sb.WriteString("ZIP_RC=$?\n")
	sb.WriteString("if [ $ZIP_RC -ne 0 ]; then\n")
	sb.WriteString("  echo \"backup failed: zip rc=$ZIP_RC\"\n")
	sb.WriteString("  exit $ZIP_RC\n")
	sb.WriteString("fi\n")
	sb.WriteString("echo \"backup ok: ${BACKUP_ZIP}\"\n")

	// 2. curl 下载代码包到目标目录
	sb.WriteString("cd \"$DEPLOY_DIR\"\n")
	sb.WriteString(fmt.Sprintf("curl -sSL --fail %s -o %s\n", quotedURL, quotedFile))
	sb.WriteString("CURL_RC=$?\n")
	sb.WriteString("if [ $CURL_RC -ne 0 ]; then\n")
	sb.WriteString("  echo \"curl failed: rc=$CURL_RC\"\n")
	sb.WriteString("  exit $CURL_RC\n")
	sb.WriteString("fi\n")

	// 3. 解压到临时目录
	sb.WriteString("mkdir -p \"$TMP_DIR\"\n")
	sb.WriteString(fmt.Sprintf("unzip -o %s -d \"$TMP_DIR\"\n", quotedFile))
	sb.WriteString("UNZIP_RC=$?\n")
	sb.WriteString("if [ $UNZIP_RC -ne 0 ]; then\n")
	sb.WriteString("  echo \"unzip failed: rc=$UNZIP_RC, zip left at ${CODE_ZIP} for debug\"\n")
	sb.WriteString("  rm -rf \"$TMP_DIR\"\n")
	sb.WriteString("  exit $UNZIP_RC\n")
	sb.WriteString("fi\n")
	// 解压完删掉 zip(节省空间)
	sb.WriteString(fmt.Sprintf("rm -f %s\n", quotedFile))

	// 4. 智能平铺:探测第一层结构
	sb.WriteString("cd \"$TMP_DIR\"\n")
	sb.WriteString("top_count=$(ls -1A | wc -l)\n")
	sb.WriteString("if [ \"$top_count\" -eq 0 ]; then\n")
	sb.WriteString("  echo \"deploy failed: zip is empty (no entries after unzip)\"\n")
	sb.WriteString("  cd \"$DEPLOY_DIR\"\n")
	sb.WriteString("  rm -rf \"$TMP_DIR\"\n")
	sb.WriteString("  exit 1\n")
	sb.WriteString("fi\n")
	sb.WriteString("if [ \"$top_count\" -eq 1 ]; then\n")
	sb.WriteString("  only_entry=$(ls -1A | head -1)\n")
	sb.WriteString("  if [ -d \"$only_entry\" ]; then\n")
	sb.WriteString("    echo \"single-root detected: ${only_entry}/, flattening...\"\n")
	sb.WriteString("    cd \"$only_entry\"\n")
	sb.WriteString("  fi\n")
	sb.WriteString("fi\n")

	// 5. 平铺到 deployPath
	//    find -printf '%f\0' + while read -d '' 处理含空格/换行的文件名
	sb.WriteString("find . -mindepth 1 -maxdepth 1 -printf '%f\\0' | while IFS= read -r -d '' f; do\n")
	sb.WriteString("  target=\"${DEPLOY_DIR}/${f}\"\n")
	sb.WriteString("  if [ -e \"$target\" ] || [ -L \"$target\" ]; then\n")
	sb.WriteString("    rm -rf \"$target\"\n")
	sb.WriteString("  fi\n")
	sb.WriteString("  mv \"$f\" \"$target\" 2>/dev/null || true\n")
	sb.WriteString("done\n")

	// 6. 清理
	sb.WriteString("cd \"$DEPLOY_DIR\"\n")
	sb.WriteString("rm -rf \"$TMP_DIR\"\n")

	sb.WriteString("echo \"deploy ok: backup=${BACKUP_ZIP}\"\n")
	sb.WriteString("exit 0\n")
	return sb.String()
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

// uploadCodePackageToStorage 推文件到对象存储(S3 兼容)
// objectKey:code_packages/<unix_nano><ext>(prefix 由 storage 块统一加)
func uploadCodePackageToStorage(
	filename string, size int64,
	open func() (io.ReadCloser, error),
) (string, string, error) {
	uploader := Uploader()
	if uploader == nil {
		return "", "", ErrUploadOSSNotConfig
	}

	f, err := open()
	if err != nil {
		return "", "", ErrUploadRead
	}
	defer f.Close()

	ext := path.Ext(filename)
	objectKey := uploader.ObjectKey(fmt.Sprintf("code_packages/%d%s", time.Now().UnixNano(), ext))
	contentType := "application/octet-stream"
	switch strings.ToLower(ext) {
	case ".zip":
		contentType = "application/zip"
	case ".apk":
		contentType = "application/vnd.android.package-archive"
	}

	fileURL, err := uploader.Upload(context.Background(), objectKey, f, size, contentType)
	if err != nil {
		logger.L.Error("storage PutObject failed",
			zap.String("object_key", objectKey),
			zap.String("endpoint", config.AppConfig.Storage.Endpoint),
			zap.String("bucket", config.AppConfig.Storage.Bucket),
			zap.Int64("file_size", size),
			zap.Error(err),
		)
		return "", "", ErrUploadPut
	}
	return objectKey, fileURL, nil
}
