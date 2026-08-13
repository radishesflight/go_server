#!/bin/bash
# ===========================================================
# codeDeploy 上传测试脚本
#
# 跑法(在 192.168.1.196 终端):
#   bash test_upload.sh
#
# 前置:
#   - 后端 go_server 已在 192.168.1.196:8080 跑着
#   - 已有 admin 账号(admin/123456)
#   - 已有 b2b 业务项目(从前面 curl 创建的,project_id=1)
#   - 已有"前台web"端(endpoint_id=3,看 tree 确认)
#
# 测什么:
#   1. 登录拿 token
#   2. 准备一个小 zip(4KB 左右)
#   3. 调 /api/codeDeploy/packages 上传(走 OSS)
#   4. 查列表(看新包有没有)
#   5. 触发拉取(mock)
# ===========================================================

set -e

API="http://192.168.1.196:8080"
USER="admin"
PASS="123456"

echo "=========================================="
echo "  codeDeploy 上传端到端测试"
echo "=========================================="

# ---------- 1. 登录 ----------
echo -e "\n[1/5] 登录拿 token ..."
TOKEN=$(curl -s -X POST "$API/api/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USER\",\"password\":\"$PASS\"}" \
  | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('token',''))")

if [ -z "$TOKEN" ]; then
  echo "❌ 登录失败,token 为空"
  exit 1
fi
echo "✓ token: ${TOKEN:0:20}..."

AUTH=(-H "Authorization: $TOKEN")

# ---------- 2. 准备测试 zip ----------
echo -e "\n[2/5] 准备测试 zip ..."
WORK=/tmp/code-deploy-test
rm -rf "$WORK" && mkdir -p "$WORK/pkg"
echo "hello from test package $(date)" > "$WORK/pkg/index.html"
echo "build at $(date)" > "$WORK/pkg/build.log"
cd "$WORK" && zip -qr test-frontend.zip pkg/
ZIP_FILE="$WORK/test-frontend.zip"
ls -la "$ZIP_FILE"
echo "✓ 测试包: $ZIP_FILE ($(du -h "$ZIP_FILE" | cut -f1))"

# ---------- 3. 查树拿 ID ----------
echo -e "\n[3/5] 查项目树,确认 project_id / endpoint_id ..."
TREE=$(curl -s "${AUTH[@]}" "$API/api/codeDeploy/projects/tree")
echo "$TREE" | python3 -m json.tool

# 自动找 b2b 项目的"前台web"端(你也可以改成手填)
PROJECT_ID=$(echo "$TREE" | python3 -c "
import sys, json
d = json.load(sys.stdin)
for p in d.get('data', {}).get('list', []):
    if p.get('code') == 'b2b':
        print(p['id'])
        break
")
ENDPOINT_ID=$(echo "$TREE" | python3 -c "
import sys, json
d = json.load(sys.stdin)
for p in d.get('data', {}).get('list', []):
    if p.get('code') == 'b2b':
        for e in p.get('endpoints', []):
            if e.get('code') == 'web':
                print(e['id'])
                break
        break
")

if [ -z "$PROJECT_ID" ] || [ -z "$ENDPOINT_ID" ]; then
  echo "❌ 没找到 b2b 项目的'前台web'端,检查树或改手动"
  exit 1
fi
echo "✓ project_id=$PROJECT_ID  endpoint_id=$ENDPOINT_ID"

# ---------- 4. 上传 ----------
echo -e "\n[4/5] 上传代码包 ..."
UPLOAD_RES=$(curl -s -X POST "$API/api/codeDeploy/packages" \
  "${AUTH[@]}" \
  -F "file=@$ZIP_FILE" \
  -F "project_id=$PROJECT_ID" \
  -F "endpoint_id=$ENDPOINT_ID" \
  -F "name=test-frontend" \
  -F "note=自动化测试上传" \
  -F "build_time=$(date '+%Y-%m-%d %H:%M:%S')")
echo "$UPLOAD_RES" | python3 -m json.tool

PKG_ID=$(echo "$UPLOAD_RES" | python3 -c "
import sys, json
d = json.load(sys.stdin)
if d.get('code') == 0:
    print(d['data']['id'])
else:
    print('')
")
if [ -z "$PKG_ID" ]; then
  echo "❌ 上传失败,看上面响应"
  exit 1
fi
FULL_NAME=$(echo "$UPLOAD_RES" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['full_name'])")
FILE_URL=$(echo "$UPLOAD_RES" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['file_url'])")
echo "✓ 新包 ID=$PKG_ID"
echo "✓ full_name=$FULL_NAME"
echo "✓ file_url=$FILE_URL"

# ---------- 5. 查列表 + 拉取 ----------
echo -e "\n[5/5] 查列表(看新包) + 触发拉取 ..."
LIST=$(curl -s "${AUTH[@]}" \
  "$API/api/codeDeploy/packages?project_id=$PROJECT_ID&endpoint_id=$ENDPOINT_ID")
echo "--- 列表(最近 3 条) ---"
echo "$LIST" | python3 -c "
import sys, json
d = json.load(sys.stdin)
for p in d.get('data', {}).get('list', [])[:3]:
    print(f\"  {p['full_name']}  ({p['build_time']})  {p['size']} bytes  url={p['file_url']}\")
"

echo -e "\n--- 触发拉取(mock) ---"
PULL_RES=$(curl -s -X POST "${AUTH[@]}" "$API/api/codeDeploy/packages/$PKG_ID/pull")
echo "$PULL_RES" | python3 -m json.tool

echo -e "\n=========================================="
echo "  ✓ 全部测完"
echo "=========================================="
echo "验证清单(手工检查):"
echo "  1. 浏览器打开 $API 访问 /codeDeploy/apkPackage,选 b2b电商项目 → 前台web,看到新包"
echo "  2. 登录 OSS 控制台,看 code_packages/ 目录下有没有新文件"
echo "  3. 查 DB: SELECT * FROM code_packages WHERE id=$PKG_ID"
