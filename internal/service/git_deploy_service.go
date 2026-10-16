// Package service git_deploy_service.go - 后端 API 部署(Git 拉取)
//
// 区别于 code_package_service.go 的 Pull(走 OSS 包 + 解压):
// 本流程是"git 部署"——目标机器已经 git clone 过项目,部署时直接
//
//	cd $DEPLOY_DIR
//	zip 备份整个目录(留最近 5 个)
//	git reset --hard {branch}  切到指定分支
//	composer install            拉依赖(PHP 项目)
//	chown -R www.www            改属主
//
// 用于"前端包管理"和"代码部署"菜单里的"后端 API 部署"流程
package service

import (
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"go_server/config"
	"go_server/internal/model"
	"go_server/pkg/jumpserver"
	"go_server/pkg/logger"
)

// buildGitDeployCmd 拼 JMS 远端执行的 shell 命令串
//
// 参数:
//
//	deployPath: 部署目录(例 /data/www/web/ceshi);空字符串就用默认 cwd
//
// 流程(2026-10-16 v2:**branch 不再由前端传,服务端自己拿当前分支**):
//  1. 备份 deployPath 到 PARENT_DIR/ceshi_<ts>.zip
//  2. 清理老备份(留最近 5 个,跟 Pull 同套逻辑,portable sh)
//  3. cd deployPath
//  4. 自动拿当前分支:git symbolic-ref --short HEAD(优先) || git rev-parse --abbrev-ref HEAD
//  5. git fetch origin
//  6. git reset --hard origin/$BRANCH
//  7. composer install --no-interaction --prefer-dist
//  8. chown -R www.www {deployPath}
//  9. echo "deploy ok: branch=xxx"  终止符
func buildGitDeployCmd(deployPath string) string {
	quotedPath := "."
	if deployPath != "" {
		quotedPath = strconv.Quote(deployPath)
	}

	var sb strings.Builder
	sb.WriteString("set +e\n")
	sb.WriteString("TS=$(date +%Y-%m-%d-%H:%M:%S)\n")
	if deployPath != "" {
		// 备份跟 Pull 共用 PARENT_DIR = dirname(deployPath)
		sb.WriteString(fmt.Sprintf("DEPLOY_DIR=%s\n", quotedPath))
	} else {
		sb.WriteString("DEPLOY_DIR=$(pwd)\n")
	}
	sb.WriteString("PARENT_DIR=$(dirname \"$DEPLOY_DIR\")\n")
	sb.WriteString("DIR_NAME=$(basename \"$DEPLOY_DIR\")\n")
	sb.WriteString("BACKUP_ZIP=\"${PARENT_DIR}/${DIR_NAME}_${TS}.zip\"\n")
	sb.WriteString("echo \"git deploy start: target=$DEPLOY_DIR\"\n")

	// 1. 备份整个 deployDir
	sb.WriteString("mkdir -p \"$DEPLOY_DIR\" 2>/dev/null\n")
	sb.WriteString("(cd \"$PARENT_DIR\" && zip -qr \"${DIR_NAME}_${TS}.zip\" \"$DIR_NAME\") 1>&2\n")
	sb.WriteString("ZIP_RC=$?\n")
	sb.WriteString("if [ $ZIP_RC -ne 0 ]; then\n")
	sb.WriteString("  echo \"backup failed: zip rc=$ZIP_RC, abort git deploy\" 1>&2\n")
	sb.WriteString("  exit $ZIP_RC\n")
	sb.WriteString("fi\n")
	sb.WriteString("echo \"backup ok: ${BACKUP_ZIP}\" 1>&2\n")

	// 1.5 清理老备份(留最近 5 个)— 跟 Pull 同套 portable sh
	sb.WriteString("KEEP_COUNT=5\n")
	sb.WriteString("find \"$PARENT_DIR\" -maxdepth 1 -type f -name \"${DIR_NAME}_*.zip\" -printf '%T@\\t%p\\n' 2>/dev/null \\\n")
	sb.WriteString("  | sort -rn \\\n")
	sb.WriteString("  | cut -f2- \\\n")
	sb.WriteString("  | tail -n +$((KEEP_COUNT + 1)) \\\n")
	sb.WriteString("  | xargs -r rm -f --\n")
	sb.WriteString("echo \"backup retention: kept recent ${KEEP_COUNT} of ${DIR_NAME}_*.zip\" 1>&2\n")

	// 2. cd + 自动拿当前分支
	sb.WriteString("cd \"$DEPLOY_DIR\" || { echo \"cd failed: $DEPLOY_DIR\" 1>&2; exit 1; }\n")
	// 验证是 git 仓库
	sb.WriteString("if [ ! -d .git ] && [ ! -f .git ]; then\n")
	sb.WriteString("  echo \"error: $DEPLOY_DIR is not a git repository\" 1>&2\n")
	sb.WriteString("  exit 2\n")
	sb.WriteString("fi\n")
	// 取当前分支:git symbolic-ref --short HEAD(更准) || git rev-parse --abbrev-ref HEAD(兼容 detached)
	sb.WriteString("BRANCH=$(git symbolic-ref --short HEAD 2>/dev/null || git rev-parse --abbrev-ref HEAD 2>/dev/null)\n")
	sb.WriteString("if [ -z \"$BRANCH\" ] || [ \"$BRANCH\" = \"HEAD\" ]; then\n")
	sb.WriteString("  echo \"error: cannot detect current branch (detached HEAD?)\" 1>&2\n")
	sb.WriteString("  exit 3\n")
	sb.WriteString("fi\n")
	sb.WriteString("echo \"current branch detected: $BRANCH\" 1>&2\n")

	// 3. git fetch
	sb.WriteString("echo \"git fetch origin\" 1>&2\n")
	sb.WriteString("git fetch origin 1>&2\n")
	sb.WriteString("FETCH_RC=$?\n")
	sb.WriteString("if [ $FETCH_RC -ne 0 ]; then\n")
	sb.WriteString("  echo \"git fetch failed: rc=$FETCH_RC, try local branch only\" 1>&2\n")
	sb.WriteString("fi\n")

	// 4. git reset
	sb.WriteString("echo \"git reset --hard origin/$BRANCH\" 1>&2\n")
	sb.WriteString("git reset --hard \"origin/$BRANCH\" 1>&2\n")
	sb.WriteString("RESET_RC=$?\n")
	sb.WriteString("if [ $RESET_RC -ne 0 ]; then\n")
	sb.WriteString("  echo \"git reset failed: rc=$RESET_RC\" 1>&2\n")
	sb.WriteString("  exit $RESET_RC\n")
	sb.WriteString("fi\n")
	sb.WriteString("echo \"git reset ok: HEAD=$(git rev-parse --short HEAD 2>/dev/null)\" 1>&2\n")

	// 5. composer install(2>&1 把 stdout/stderr 合并,install 输出主要在 stdout)
	//    双重保护:
	//      a) PHP/composer 不存在 → warn 后继续(只部署代码,不装 vendor)— 适合代码部署机
	//      b) 没有 composer.lock → skip,不进入 update 模式(避免网络全量拉最新极慢)
	//    有 lock → composer install 走 lock 模式,秒级完成
	sb.WriteString("echo \"composer install start\" 1>&2\n")
	sb.WriteString("if ! command -v composer >/dev/null 2>&1 || ! command -v php >/dev/null 2>&1; then\n")
	sb.WriteString("  echo \"skip composer install: php/composer not installed on this host (only git reset applied)\" 1>&2\n")
	sb.WriteString("elif [ ! -f \"$DEPLOY_DIR/composer.lock\" ]; then\n")
	sb.WriteString("  echo \"skip composer install: composer.lock not found in $DEPLOY_DIR\" 1>&2\n")
	sb.WriteString("  echo \"  hint: commit composer.lock to repo, or run 'composer install' manually on a build machine and commit vendor/ instead\" 1>&2\n")
	sb.WriteString("else\n")
	sb.WriteString("  composer install --no-interaction --prefer-dist --no-progress 1>&2\n")
	sb.WriteString("  COMPOSER_RC=$?\n")
	sb.WriteString("  if [ $COMPOSER_RC -ne 0 ]; then\n")
	sb.WriteString("    echo \"composer install failed: rc=$COMPOSER_RC, restore from backup\" 1>&2\n")
	sb.WriteString("    unzip -o \"$BACKUP_ZIP\" -d \"$PARENT_DIR\" 1>&2\n")
	sb.WriteString("    exit $COMPOSER_RC\n")
	sb.WriteString("  fi\n")
	sb.WriteString("  echo \"composer install ok\" 1>&2\n")
	sb.WriteString("fi\n")

	// 6. chown
	sb.WriteString("echo \"chown -R www.www $DEPLOY_DIR\" 1>&2\n")
	sb.WriteString(fmt.Sprintf("chown -R www.www \"%s\" 1>&2\n", deployPath))
	sb.WriteString("CHOWN_RC=$?\n")
	sb.WriteString("if [ $CHOWN_RC -ne 0 ]; then\n")
	sb.WriteString("  echo \"chown failed: rc=$CHOWN_RC, restore from backup\" 1>&2\n")
	sb.WriteString("  unzip -o \"$BACKUP_ZIP\" -d \"$PARENT_DIR\" 1>&2\n")
	sb.WriteString("  exit $CHOWN_RC\n")
	sb.WriteString("fi\n")
	sb.WriteString("echo \"chown ok\" 1>&2\n")

	sb.WriteString("echo \"deploy ok: branch=$BRANCH backup=$BACKUP_ZIP\"\n")
	sb.WriteString("exit 0\n")
	return sb.String()
}

// GitDeploy 在 JMS 资产上跑 git 部署(cd + 备份 + git reset + composer install + chown)
//
// 与 Pull 的区别:
//   - 不下载 OSS 包,不 unzip
//   - **branch 不由前端传,服务端自己拿当前分支**(2026-10-16 v2)
//   - 走 git fetch + git reset --hard origin/{current_branch} 拉代码
//   - 跑 composer install 拉 PHP 依赖
//   - 跑 chown 改属主
//
// 入参(2026-10-16 v3):
//
//	assetID    业务资产 id(business_assets.id)
//	deployPath 部署目录(**前端必传**,从 business_asset_paths.path 选 type='api' 的)
//	pathID     部署路径 id(2026-10-16 v3 新增,前端传 path.id,后端校验 type='api' 且 asset_id 匹配)
//
// 返回:成功时 map 含 success/branch(服务端拿到的)/status/jms_id/jms_output/...;
//
//	失败时返回 map + ErrPackagePullFail
func (s *BusinessAssetService) GitDeploy(assetID uint, pathID uint) (map[string]interface{}, error) {
	if assetID == 0 {
		return nil, ErrAssetNotFound
	}
	// 拿资产
	var asset model.BusinessAssets
	if err := model.DB.First(&asset, assetID).Error; err != nil {
		return nil, ErrAssetNotFound
	}
	if asset.Status != 1 {
		return nil, ErrAssetNotFound
	}
	if asset.JmsID == "" {
		return nil, ErrAssetNotFound
	}
	// 校验 pathID:必须是该 asset 的 type='api' 路径
	if pathID == 0 {
		return nil, ErrPackageNotFound // 复用"代码包未配置部署目标"业务码
	}
	var p model.BusinessAssetPaths
	if err := model.DB.First(&p, pathID).Error; err != nil {
		return nil, ErrPackageNotFound
	}
	if p.AssetID != assetID {
		return nil, ErrPackageNotFound
	}
	if p.Type != "api" {
		// git 部署只针对后端 API 路径(type=api);type=web 给前端包管理用
		return nil, ErrPackageNotFound
	}
	deployPath := p.Path
	if deployPath == "" {
		return nil, ErrPackageNotFound
	}

	// 拼命令 + 调 JMS(shell 内部 git symbolic-ref 拿当前 branch,失败则任务失败)
	cmd := buildGitDeployCmd(deployPath)
	logger.L.Info("git_deploy start",
		zap.Uint("asset_id", asset.ID),
		zap.String("asset_name", asset.Name),
		zap.Uint("path_id", p.ID),
		zap.String("deploy_path", deployPath),
	)
	jmsClient := jumpserver.NewClient(
		config.AppConfig.Jumpserver.BaseURL,
		config.AppConfig.Jumpserver.Token,
		config.AppConfig.Jumpserver.OrgID,
	)
	jmsRes, jmsErr := jmsClient.RunCommand(asset.JmsID, cmd, codePackagePullTimeout)

	out := map[string]interface{}{
		"success":     jmsErr == nil,
		"asset_id":    asset.ID,
		"asset_name":  asset.Name,
		"path_id":     p.ID,
		"path_name":   p.Name,
		"deploy_path": deployPath,
		"command":     cmd,
	}
	if jmsRes != nil {
		out["is_finished"] = jmsRes["is_finished"]
		out["is_success"] = jmsRes["is_success"]
		out["status"] = jmsRes["status"]
		out["time_cost"] = jmsRes["time_cost"]
		if m, ok := jmsRes["material"].(string); ok {
			out["jms_output"] = m
		}
		if sm, ok := jmsRes["summary"].(map[string]interface{}); ok {
			out["summary"] = sm
		}
		// 从远端 material 解析 "branch=xxx"(成功时的 echo "deploy ok: branch=xxx" 1>&2 写 stderr,
		// JMS 成功时不抓 stderr;但失败时 summary.failures 会含完整 stdout 里的 echo 输出)
		if m, ok := jmsRes["material"].(string); ok {
			if idx := strings.Index(m, "branch="); idx >= 0 {
				end := idx + len("branch=")
				for end < len(m) && m[end] != ' ' && m[end] != '\n' && m[end] != '\r' && m[end] != '\\' {
					end++
				}
				if end > idx+len("branch=") {
					out["branch"] = m[idx+len("branch=") : end]
				}
			}
		}
	}
	if jmsErr != nil {
		out["err"] = jmsErr.Error()
		logger.L.Warn("git_deploy failed",
			zap.Uint("asset_id", asset.ID),
			zap.Uint("path_id", p.ID),
			zap.String("err", jmsErr.Error()),
		)
		return out, ErrPackagePullFail
	}
	logger.L.Info("git_deploy success",
		zap.Uint("asset_id", asset.ID),
		zap.Uint("path_id", p.ID),
	)
	return out, nil
}
