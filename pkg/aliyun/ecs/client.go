// Package ecs 提供精简版阿里云 ECS OpenAPI 客户端(RPC 风格 + 签名 v1)
//
// 设计原则(跟 jumpserver/client 一致):
//   - 只覆盖本项目实际用到的端点,不全量实现
//   - 凭据(access_key / secret_key)由调用方从 config.AppConfig.AliyunECS 传入
//     本 package 不读全局配置
//   - 返回值用领域类型(Image / Region)而不是裸 map
//
// 当前实现的 API(全部是 GET,form-encoded query):
//   - DescribeImages                  列自定义镜像(可按 region 筛)
//   - DescribeRegions                 列可用 region(辅助决策 B 还是 C)
//   - CopyImage                       跨地域复制镜像
//   - ModifyImageSharePermission      共享镜像给其他账号
//   - DescribeImageSharePermission    查询已共享给谁
//
// 不做什么:
//   - 不实现 ROA 风格 API(本项目只用 RPC)
//   - 不做连接池(每次 NewClient 一个 http.Client,30s timeout)
//   - 不做重试(阿里云服务端限流时,业务层重试)
//
// 签名算法(v1):
//  1. 公共参数 + 业务参数 merge,按 key 升序
//  2. URL-encode 每个 key=value,拼成 canonicalizedQuery
//  3. StringToSign = HTTPMethod + "&" + URLEncode("/") + "&" + URLEncode(canonicalizedQuery)
//  4. Signature = base64(HMAC-SHA1(AccessKeySecret + "&", StringToSign))
//  5. 把 Signature 加到 query 里一起发
//
// 参考:https://help.aliyun.com/zh/sdk/developer-reference/api-error-codes
package ecs

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// API 版本(EBS OpenAPI 2014-05-26,通用 ECS API 也是这个)
const apiVersion = "2014-05-26"

// Client ECS HTTP 客户端
type Client struct {
	accessKey     string
	secretKey     string
	defaultRegion string
	endpointHost  string // e.g. "ecs.cn-hangzhou.aliyuncs.com"
	http          *http.Client
}

// NewClient 构造一个 ECS Client
//   - accessKey: 阿里云 RAM AK(与 OSS / JMS 同一对即可,本项目复用 storage.access_key)
//   - secretKey: 同上
//   - defaultRegion: 默认 region(查询时如不指定,用这个);e.g. "cn-hangzhou"
//   - endpointSuffix: API 域名后缀,默认 "aliyuncs.com"(海外用 "aliyuncs.com" 不变)
func NewClient(accessKey, secretKey, defaultRegion, endpointSuffix string) *Client {
	if endpointSuffix == "" {
		endpointSuffix = "aliyuncs.com"
	}
	if defaultRegion == "" {
		defaultRegion = "cn-hangzhou"
	}
	// endpoint host 用 default region 拼;实际请求会在 query 里覆盖 RegionId
	return &Client{
		accessKey:     accessKey,
		secretKey:     secretKey,
		defaultRegion: defaultRegion,
		endpointHost:  fmt.Sprintf("ecs.%s.%s", defaultRegion, endpointSuffix),
		http:          &http.Client{Timeout: 30 * time.Second},
	}
}

// Image ECS 自定义镜像(DescribeImages.Image 的最小子集)
// 实际返回 30+ 字段,只取列表展示需要的几个
//
// 重要(2026-10-17):之前我加了 IsSelfShared bool,但阿里云 API 实际返回 string 类型
//
//	"true"/"false"(首字母可能大写),Go json.Unmarshaler 对 bool 字段收到字符串就报错;
//	即使不报错,本字段其实不参与分类 — 分类只需 ImageOwnerAlias 一个字段,
//	"self"=我的镜像,"others"=别人共享/社区镜像。所以这里**删掉**这个字段。
type Image struct {
	ImageID         string `json:"ImageId"`
	ImageName       string `json:"ImageName"`
	ImageType       string `json:"ImageType"`       // 阿里云 API 实际不会返回这个字段,保留占位以防万一
	ImageOwnerAlias string `json:"ImageOwnerAlias"` // system / self / others / marketplace / image_encryption
	RegionID        string `json:"RegionId"`
	OSName          string `json:"OSName"`       // e.g. "CentOS  8.5 64位"
	Architecture    string `json:"Architecture"` // x86_64 / i386 / arm64
	ImageVersion    string `json:"ImageVersion"`
	Size            int    `json:"Size"`         // 系统盘大小(GB)
	Status          string `json:"Status"`       // Unavailable / Available / Creating / Waiting / InUse
	CreationTime    string `json:"CreationTime"` // ISO8601
	Description     string `json:"Description"`
	IsPublic        bool   `json:"IsPublic"`
	Usage           string `json:"Usage"`
}

// DescribeImagesResp DescribeImages 响应
type DescribeImagesResp struct {
	RequestID  string `json:"RequestId"`
	TotalCount int    `json:"TotalCount"` // 注意:阿里云这里返回的是 int 但分页要看 PageNumber/PageSize
	PageNumber int    `json:"PageNumber"`
	PageSize   int    `json:"PageSize"`
	Images     struct {
		Image []Image `json:"Image"`
	} `json:"Images"`
	Code    string `json:"Code"`    // 错误时存在
	Message string `json:"Message"` // 错误时存在
}

// Region DescribeRegions 响应里的单个 region
type Region struct {
	RegionID  string `json:"RegionId"`
	LocalName string `json:"LocalName"`
}

// DescribeRegionsResp DescribeRegions 响应
type DescribeRegionsResp struct {
	RequestID string `json:"RequestId"`
	Regions   struct {
		Region []Region `json:"Region"`
	} `json:"Regions"`
}

// CopyImageResp CopyImage 响应
type CopyImageResp struct {
	RequestID string `json:"RequestId"`
	ImageID   string `json:"ImageId"` // 目标 region 的新镜像 ID
	Code      string `json:"Code"`
	Message   string `json:"Message"`
}

// ModifyImageSharePermissionResp 共享响应
type ModifyImageSharePermissionResp struct {
	RequestID string `json:"RequestId"`
	Code      string `json:"Code"`
	Message   string `json:"Message"`
}

// SharedAccount 已共享给某个账号的记录
type SharedAccount struct {
	Account struct {
		AliyunID string `json:"AliyunId"` // 对方账号 UID
	} `json:"Account"`
}

// DescribeImageSharePermissionResp 共享查询响应
type DescribeImageSharePermissionResp struct {
	RequestID   string   `json:"RequestId"`
	ShareGroups struct{} `json:"-"`
	Accounts    struct {
		Account []SharedAccount `json:"Account"`
	} `json:"Accounts"`
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

// describeImages 列镜像(底层 API 调用)
//   - region: 必填(阿里云要求);空时用 defaultRegion
//   - imageOwnerAlias: 关键过滤参数(system/self/others/marketplace,空=全部),
//     分类依据见官方文档 https://api.aliyun.com/document/Ecs/2014-05-26/DescribeImages
//   - status:   过滤状态,空=全部;"Available"=可用
//   - pageSize: 1-100,默认 50
//   - pageNumber: 默认 1
//
// 历史踩坑(2026-10-17):这里原来叫 "imageType" 并传 "Self" / "Others",
//
//	但阿里云 DescribeImages **没有 ImageType 这个请求参数**(文档只列 ImageOwnerAlias)。
//	把参数名改正,语义才准确。
func (c *Client) describeImages(region, imageOwnerAlias, status string, pageSize, pageNumber int) (*DescribeImagesResp, error) {
	if region == "" {
		region = c.defaultRegion
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}
	if pageNumber <= 0 {
		pageNumber = 1
	}
	params := url.Values{}
	params.Set("RegionId", region)
	if imageOwnerAlias != "" {
		params.Set("ImageOwnerAlias", imageOwnerAlias)
	}
	if status != "" {
		params.Set("Status", status)
	}
	params.Set("PageSize", strconv.Itoa(pageSize))
	params.Set("PageNumber", strconv.Itoa(pageNumber))

	var out DescribeImagesResp
	if err := c.call("DescribeImages", region, params, &out); err != nil {
		return nil, err
	}
	// 阿里云 DescribeImages 响应里 Image 不带 RegionId(只在请求里指定),
	// 这里手动补上,前端跨 region 展示才能区分
	for i := range out.Images.Image {
		if out.Images.Image[i].RegionID == "" {
			out.Images.Image[i].RegionID = region
		}
	}
	return &out, nil
}

// DescribeImages 列可用镜像(给前端用,固定 ImageType=Self Status=Available)
//
// 双层过滤:
//
//	Layer 1:按 ImageOwnerAlias 显式值过滤
//	  - "system" / "others" / "marketplace" / "image_encryption" → 跳过
//	  - "self" / ""(字段缺失) → 进入 Layer 2
//
//	Layer 2:按镜像名特征过滤(因为阿里云系统镜像的 OwnerAlias 也可能是空字符串,
//	  跟用户私有但字段缺失的镜像无法用 OwnerAlias 区分)
//	  - 镜像名含 "_alibase_" → 跳过(阿里云官方系统镜像命名约定,用户不会用)
//	  - 镜像名以 "win" / "anolisos" 开头 → 跳过(阿里云系统镜像常见前缀)
//	  - 其余 → 保留
//
// 这两层的目的是把"用户私有创建"的镜像过滤出来,排除阿里云预置的 11k+ 公共系统镜像。
// 用户自定义镜像名一般不含这些前缀。
func (c *Client) DescribeImages(region string, pageSize, pageNumber int) ([]Image, int, error) {
	// 用 ImageOwnerAlias=self 直接拉"用户私有+主动共享给他人的"那批,跳过 system/marketplace 等
	// 留下来的就是用户自己的镜像,无论是否分享出去(ImageOwnerAlias=self 都属于"自定义")
	r, err := c.describeImages(region, "self", "Available", pageSize, pageNumber)
	if err != nil {
		return nil, 0, err
	}
	filtered := make([]Image, 0, len(r.Images.Image))
	for _, img := range r.Images.Image {
		name := strings.ToLower(img.ImageName)
		// 阿里云系统镜像命名约定(_alibase_ 几乎所有官方镜像都有)
		// (DescribeImages 传了 ImageOwnerAlias=self 之后,正常不会出现 system 镜像,
		//  但历史数据 / 边缘 case 仍可能混进来,这里 Layer 2 兜底)
		if strings.Contains(name, "_alibase_") ||
			strings.HasPrefix(name, "win") ||
			strings.HasPrefix(name, "anolisos") {
			continue
		}
		filtered = append(filtered, img)
	}
	return filtered, len(filtered), nil
}

// DescribeImagesAllRegions 列**所有 region** 的自定义镜像(2026-10-17 新增)
//
// 业务背景:ECS DescribeImages 单 API 只支持单 region,前端想看"全部区域镜像"必须聚合
//
// 实现:
//  1. 先 DescribeAllRegions 拿到 region 列表
//  2. 并发(限流 5)调各 region 的 DescribeImages,PageSize=100 自动翻页直到拉完
//  3. 合并到统一 list
//
// 失败策略:单个 region 失败不影响其他 region,只 log warning
func (c *Client) DescribeImagesAllRegions() ([]Image, error) {
	regions, err := c.DescribeAllRegions()
	if err != nil {
		return nil, fmt.Errorf("列 region 失败: %w", err)
	}

	type regResult struct {
		region string
		images []Image
		err    error
	}
	results := make(chan regResult, len(regions))
	sem := make(chan struct{}, 5) // 限流:同时最多 5 个 region 在跑
	var wg sync.WaitGroup

	for _, r := range regions {
		wg.Add(1)
		sem <- struct{}{}
		go func(rg Region) {
			defer wg.Done()
			defer func() { <-sem }()
			imgs, err := c.fetchAllImagesInRegion(rg.RegionID)
			results <- regResult{region: rg.RegionID, images: imgs, err: err}
		}(r)
	}
	wg.Wait()
	close(results)

	all := make([]Image, 0, 256)
	var failedRegions []string
	for r := range results {
		if r.err != nil {
			failedRegions = append(failedRegions, r.region)
			continue
		}
		all = append(all, r.images...)
	}
	if len(failedRegions) > 0 {
		// 不算 fatal,只是告诉调用方有 region 没拉到
		return all, fmt.Errorf("部分 region 查询失败: %v", failedRegions)
	}
	return all, nil
}

// fetchAllImagesInRegion 拉单个 region 的所有**用户私有(self)**镜像(自动翻页)
//
// 走 ImageOwnerAlias=self,后台 API 直接过滤,本地不再过滤
// (DescribeImages 已经把 system 等跳了)
func (c *Client) fetchAllImagesInRegion(region string) ([]Image, error) {
	const pageSize = 100
	all := make([]Image, 0, 50)
	for page := 1; ; page++ {
		imgs, _, err := c.DescribeImages(region, pageSize, page)
		if err != nil {
			return all, err
		}
		all = append(all, imgs...)
		if len(imgs) < pageSize {
			break
		}
	}
	return all, nil
}

// ListSharedImages 列**共享镜像**(别人共享给当前账号的,2026-10-17 新增)
//
// 业务:阿里云控制台"共享镜像"tab — 区分 ImageType=Self(自定义)和 ImageType=Others(共享)
// 这里只跨 region 聚合 Others 类型(单 region 模式不实现,业务不需要)
//
// 注意:阿里云 API "Shared" 在不同 region 的镜像列表里,**镜像的 RegionId 是源 region**(拥有者),
// 不会自动同步到本账号 region。需要用 ShareImage 跨账号共享才能跨账号访问。
func (c *Client) ListSharedImages() ([]Image, error) {
	regions, err := c.DescribeAllRegions()
	if err != nil {
		return nil, fmt.Errorf("列 region 失败: %w", err)
	}
	type regResult struct {
		region string
		images []Image
		err    error
	}
	results := make(chan regResult, len(regions))
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	for _, r := range regions {
		wg.Add(1)
		sem <- struct{}{}
		go func(rg Region) {
			defer wg.Done()
			defer func() { <-sem }()
			imgs := c.fetchAllSharedImagesInRegion(rg.RegionID)
			results <- regResult{region: rg.RegionID, images: imgs}
		}(r)
	}
	wg.Wait()
	close(results)

	all := make([]Image, 0, 64)
	for r := range results {
		if r.err != nil {
			continue
		}
		all = append(all, r.images...)
	}
	return all, nil
}

// fetchAllSharedImagesInRegion 拉单个 region 的所有"别人共享/社区"镜像(ImageOwnerAlias=others,自动翻页)
//
// 走 ImageOwnerAlias=others 参数让阿里云 API 端直接返回"别人共享给我 + 公开社区镜像"。
// (API 层面已经按 OwnerAlias=others 过滤,不再混 system 等;本地只兜底命名约定)
func (c *Client) fetchAllSharedImagesInRegion(region string) []Image {
	const pageSize = 100
	all := make([]Image, 0, 32)
	for page := 1; ; page++ {
		r, err := c.describeImages(region, "others", "Available", pageSize, page)
		if err != nil {
			return all
		}
		for _, img := range r.Images.Image {
			// 兜底过滤系统镜像命名约定(理论上 ImageOwnerAlias=others 不会进来)
			name := strings.ToLower(img.ImageName)
			if strings.Contains(name, "_alibase_") ||
				strings.HasPrefix(name, "win") ||
				strings.HasPrefix(name, "anolisos") {
				continue
			}
			all = append(all, img)
		}
		if len(r.Images.Image) < pageSize {
			break
		}
	}
	return all
}

// ListAllImagesAllRegions 跨所有 region 列**自定义 + 共享**镜像合并(2026-10-17 新增)
//
// 业务:用户希望一个统一列表展示所有镜像,前端用 ImageType 列区分
//
// 业务:用户希望一个统一列表展示所有镜像,前端用 image_type 列区分
//
// 并发 5 跑两组:
//   - self 镜像(ImageOwnerAlias=self:用户私有/主动分享给他人的),由 fetchAllImagesInRegion 内部传 ImageOwnerAlias=self 调 API
//   - shared 镜像(ImageOwnerAlias=others:别人共享给当前账号的 + 公开社区镜像),由 fetchAllSharedImagesInRegion 内部传 ImageOwnerAlias=others
//
// 跨 self/shared 去重:同一 ImageId 在 self 与 shared 都返回时,只在 self 端保留
//
//	(ImageOwnerAlias=self 才是"我的镜像",ImageOwnerAlias=others 表示"别人分享给我")
func (c *Client) ListAllImagesAllRegions() (selfList, sharedList []Image, err error) {
	regions, err := c.DescribeAllRegions()
	if err != nil {
		return nil, nil, fmt.Errorf("列 region 失败: %w", err)
	}

	type regResult struct {
		region string
		self   []Image
		shared []Image
		err    error
	}
	results := make(chan regResult, len(regions))
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	for _, r := range regions {
		wg.Add(1)
		sem <- struct{}{}
		go func(rg Region) {
			defer wg.Done()
			defer func() { <-sem }()
			self, _ := c.fetchAllImagesInRegion(rg.RegionID)
			shared := c.fetchAllSharedImagesInRegion(rg.RegionID)
			results <- regResult{region: rg.RegionID, self: self, shared: shared}
		}(r)
	}
	wg.Wait()
	close(results)

	// 去重:同一 ImageId 在多个 region 出现时只保留一份(第一次出现为准)
	// 跨 self/shared 兜底:同一 ImageId 在两边都出现时,**只保留 self 端**
	selfMap := make(map[string]Image, 64)
	sharedMap := make(map[string]Image, 32)
	for r := range results {
		if r.err != nil {
			continue
		}
		for _, img := range r.self {
			if _, ok := selfMap[img.ImageID]; !ok {
				selfMap[img.ImageID] = img
			}
		}
		for _, img := range r.shared {
			if _, inSelf := selfMap[img.ImageID]; inSelf {
				continue
			}
			if _, ok := sharedMap[img.ImageID]; !ok {
				sharedMap[img.ImageID] = img
			}
		}
	}
	selfAll := make([]Image, 0, len(selfMap))
	for _, img := range selfMap {
		selfAll = append(selfAll, img)
	}
	sharedAll := make([]Image, 0, len(sharedMap))
	for _, img := range sharedMap {
		sharedAll = append(sharedAll, img)
	}
	return selfAll, sharedAll, nil
}

// DescribeAllRegions 列所有可用 region
func (c *Client) DescribeAllRegions() ([]Region, error) {
	// DescribeRegions 不需要 RegionId 参数,但 endpoint host 还是要 region,这里用默认
	var out DescribeRegionsResp
	if err := c.call("DescribeRegions", "", url.Values{}, &out); err != nil {
		return nil, err
	}
	return out.Regions.Region, nil
}

// CopyImage 跨地域复制
//   - sourceRegion: 源 region(必填)
//   - imageID: 镜像 ID(必填)
//   - destRegion: 目标 region(必填)
//   - destImageName: 可选,新镜像名(默认追加 "-<destRegion>-<timestamp>")
//
// 返回:目标 region 的新 ImageID(异步任务,这里只发起)
func (c *Client) CopyImage(sourceRegion, imageID, destRegion, destImageName string) (string, error) {
	if sourceRegion == "" {
		sourceRegion = c.defaultRegion
	}
	params := url.Values{}
	params.Set("RegionId", sourceRegion)
	params.Set("ImageId", imageID)
	params.Set("DestinationRegionId", destRegion)
	if destImageName == "" {
		destImageName = imageID + "-" + destRegion + "-" + time.Now().Format("20060102-150405")
	}
	params.Set("DestinationImageName", destImageName)

	var out CopyImageResp
	if err := c.call("CopyImage", sourceRegion, params, &out); err != nil {
		return "", err
	}
	return out.ImageID, nil
}

// ModifyImageSharePermission 共享/撤销镜像
//   - region: 镜像所在 region
//   - imageID: 镜像 ID
//   - addAccount: 要添加的账号 UID 列表
//   - removeAccount: 要移除的账号 UID 列表
//
// 阿里云规定 add 和 remove 不能同时传,这里强制一次只做一件事
func (c *Client) ModifyImageSharePermission(region, imageID string, addAccount, removeAccount []string) error {
	if region == "" {
		region = c.defaultRegion
	}
	if len(addAccount) > 0 && len(removeAccount) > 0 {
		return fmt.Errorf("add 和 remove 不能同时传")
	}
	if len(addAccount) == 0 && len(removeAccount) == 0 {
		return fmt.Errorf("addAccount / removeAccount 至少传一个")
	}
	params := url.Values{}
	params.Set("RegionId", region)
	params.Set("ImageId", imageID)
	for i, uid := range addAccount {
		params.Set("AddAccount."+strconv.Itoa(i+1), uid)
	}
	for i, uid := range removeAccount {
		params.Set("RemoveAccount."+strconv.Itoa(i+1), uid)
	}

	var out ModifyImageSharePermissionResp
	if err := c.call("ModifyImageSharePermission", region, params, &out); err != nil {
		return err
	}
	return nil
}

// DescribeImageSharePermission 查询已共享给哪些账号
func (c *Client) DescribeImageSharePermission(region, imageID string) ([]string, error) {
	if region == "" {
		region = c.defaultRegion
	}
	params := url.Values{}
	params.Set("RegionId", region)
	params.Set("ImageId", imageID)

	var out DescribeImageSharePermissionResp
	if err := c.call("DescribeImageSharePermission", region, params, &out); err != nil {
		return nil, err
	}
	uids := make([]string, 0, len(out.Accounts.Account))
	for _, a := range out.Accounts.Account {
		uids = append(uids, a.Account.AliyunID)
	}
	return uids, nil
}

// call 通用 RPC 调用:加公共参数、签名、发请求、解析响应
func (c *Client) call(action, region string, biz url.Values, out interface{}) error {
	// 1. 公共参数
	common := url.Values{}
	common.Set("Action", action)
	common.Set("Format", "JSON")
	common.Set("Version", apiVersion)
	common.Set("AccessKeyId", c.accessKey)
	common.Set("SignatureMethod", "HMAC-SHA1")
	common.Set("SignatureNonce", randomNonce())
	common.Set("SignatureVersion", "1.0")
	common.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	if region != "" {
		common.Set("RegionId", region)
	}

	// 2. 合并(业务参数覆盖公共参数,允许 RegionId 一致)
	all := url.Values{}
	for k, v := range common {
		for _, vv := range v {
			all.Add(k, vv)
		}
	}
	for k, v := range biz {
		for _, vv := range v {
			all.Add(k, vv)
		}
	}

	// 3. 签名
	signature := c.signature(http.MethodGet, all)
	all.Set("Signature", signature)

	// 4. endpoint host(如果有 region,用 region-specific host;否则 default)
	host := c.endpointHost
	if region != "" && region != c.defaultRegion {
		host = fmt.Sprintf("ecs.%s.aliyuncs.com", region)
	}

	// 5. 构造请求
	reqURL := fmt.Sprintf("https://%s/?%s", host, all.Encode())
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("ECS 请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("ECS %s -> HTTP %d: %s", action, resp.StatusCode, truncate(string(body), 300))
	}

	// 6. 先反序列化到 map 检查业务级 Code(阿里云错误时 HTTP 200 但 body 里有 Code 非空)
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("ECS 响应解析失败: %w (body 前 200: %.200s)", err, string(body))
	}
	if code, _ := raw["Code"].(string); code != "" && !strings.EqualFold(code, "OK") && !strings.EqualFold(code, "Success") {
		msg, _ := raw["Message"].(string)
		return fmt.Errorf("ECS %s 业务错误: %s - %s", action, code, msg)
	}

	// 7. 再 unmarshal 到结构体
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("ECS 响应解析失败: %w", err)
	}
	return nil
}

// signature 阿里云 OpenAPI 签名 v1
func (c *Client) signature(method string, params url.Values) string {
	// 1. 按 key 升序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. 拼 canonicalizedQuery:k1=v1&k2=v2(每个 key/value URL-encoded)
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(encodeURI(k))
		sb.WriteByte('=')
		sb.WriteString(encodeURI(params.Get(k)))
	}
	canonicalizedQuery := sb.String()

	// 3. StringToSign
	stringToSign := method + "&" + encodeURI("/") + "&" + encodeURI(canonicalizedQuery)

	// 4. HMAC-SHA1(AccessKeySecret + "&", StringToSign)
	mac := hmac.New(sha1.New, []byte(c.secretKey+"&"))
	mac.Write([]byte(stringToSign))
	sigBytes := mac.Sum(nil)

	// 5. base64
	return base64.StdEncoding.EncodeToString(sigBytes)
}

// encodeURI 阿里云的 URL encoding:
//   - RFC3986 风格(空格变 %20,不是 +)
//   - ~ 不编码
//   - 其他特殊字符按 RFC3986 percent-encode
func encodeURI(s string) string {
	// url.QueryEscape 对 " "(空格)编码成 "+",阿里云要 "%20"
	// 用 url.PathEscape 然后手动把 + 转 %20 也行,但更稳的是重写
	out := make([]byte, 0, len(s)*3)
	for i := 0; i < len(s); i++ {
		c := s[i]
		// unreserved:字母数字 + - _ . ~
		if isUnreserved(c) {
			out = append(out, c)
			continue
		}
		// 其他全部 percent-encode
		out = append(out, '%', hexUpper(c>>4), hexUpper(c&0x0f))
	}
	return string(out)
}

func isUnreserved(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~'
}

func hexUpper(b byte) byte {
	if b < 10 {
		return '0' + b
	}
	return 'A' + (b - 10)
}

// randomNonce 16 字节随机(32 个 hex 字符)
func randomNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// reflectCode / reflectMessage 已被 call() 内的 map[string]interface{} 检查取代
// 保留注释:不同响应类型的 Code/Message 字段名一致,直接 map lookup 更直观
