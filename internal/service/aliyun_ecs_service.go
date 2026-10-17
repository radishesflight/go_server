// Package service aliyun_ecs_service.go - 阿里云 ECS 镜像管理业务层
//
// 封装 pkg/aliyun/ecs 的 5 个 API,提供面向 handler 的业务方法
// 凭据从 config.AppConfig.AliyunECS 拿,启动时初始化一次 client(单例)
package service

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"go_server/config"
	ecspkg "go_server/pkg/aliyun/ecs"
)

// 业务错误(给 handler 翻译用)
var (
	ErrAliyunECSNotConfig = errors.New("阿里云 ECS 未配置")
	ErrAliyunECSRegion    = errors.New("Region 非法")
)

// 全局 client 单例(读多写少,启动后不变)
var (
	ecsClientOnce sync.Once
	ecsClient     *ecspkg.Client
	ecsClientErr  error
)

// getECSClient 拿单例(懒加载)
func getECSClient() (*ecspkg.Client, error) {
	ecsClientOnce.Do(func() {
		cfg := config.AppConfig.AliyunECS
		if cfg.AccessKey == "" || cfg.SecretKey == "" {
			ecsClientErr = ErrAliyunECSNotConfig
			return
		}
		ecsClient = ecspkg.NewClient(cfg.AccessKey, cfg.SecretKey, cfg.DefaultRegion, cfg.EndpointSuffix)
	})
	return ecsClient, ecsClientErr
}

// AliyunImageService ECS 镜像服务
type AliyunImageService struct{}

// NewAliyunImageService 构造
func NewAliyunImageService() *AliyunImageService { return &AliyunImageService{} }

// ListImages 列镜像
//   - region: 必填(前端从 DescribeRegions 拿)
//   - pageSize / pageNumber: 分页
//
// 返回:(images, totalCount, error)
func (s *AliyunImageService) ListImages(region string, pageSize, pageNumber int) ([]ecspkg.Image, int, error) {
	if region == "" {
		return nil, 0, ErrAliyunECSRegion
	}
	c, err := getECSClient()
	if err != nil {
		return nil, 0, err
	}
	return c.DescribeImages(region, pageSize, pageNumber)
}

// ListImagesAllRegions 跨所有 region 聚合(2026-10-17 新增)
//
// 返回:(selfImages, sharedImages, error)
// 业务上 self 走 ImageType=Self,shared 走 ImageType=Others
// 前端列表合并展示,加 ImageType 列区分
//
// 缓存:跨 region 聚合要遍历 33 region × 2 组(self+shared),首次 25s+
// 加 5 分钟进程内缓存,项目单实例够用
type allImagesBundle struct {
	self   []ecspkg.Image
	shared []ecspkg.Image
}

var (
	allImagesCache   atomic.Pointer[allImagesBundle]
	allImagesCacheAt atomic.Int64 // unix seconds
	cacheTTLSeconds  = int64(300) // 5 分钟
)

func (s *AliyunImageService) ListImagesAllRegions() (selfImages, sharedImages []ecspkg.Image, err error) {
	// 缓存命中且未过期 → 直接返回
	if cached := allImagesCache.Load(); cached != nil {
		if time.Now().Unix()-allImagesCacheAt.Load() < cacheTTLSeconds {
			return cached.self, cached.shared, nil
		}
	}
	c, err := getECSClient()
	if err != nil {
		return nil, nil, err
	}
	self, shared, err := c.ListAllImagesAllRegions()
	if err == nil {
		selfCp := make([]ecspkg.Image, len(self))
		copy(selfCp, self)
		sharedCp := make([]ecspkg.Image, len(shared))
		copy(sharedCp, shared)
		allImagesCache.Store(&allImagesBundle{self: selfCp, shared: sharedCp})
		allImagesCacheAt.Store(time.Now().Unix())
	}
	return self, shared, err
}

// RefreshImagesAllRegions 清缓存 + 重拉,返回新数据
//
//	强制忽略 TTL 立即刷新(对应前端"重新同步"按钮)
func (s *AliyunImageService) RefreshImagesAllRegions() (selfImages, sharedImages []ecspkg.Image, err error) {
	allImagesCache.Store(nil)
	allImagesCacheAt.Store(0)

	c, err := getECSClient()
	if err != nil {
		return nil, nil, err
	}
	self, shared, err := c.ListAllImagesAllRegions()
	if err == nil {
		selfCp := make([]ecspkg.Image, len(self))
		copy(selfCp, self)
		sharedCp := make([]ecspkg.Image, len(shared))
		copy(sharedCp, shared)
		allImagesCache.Store(&allImagesBundle{self: selfCp, shared: sharedCp})
		allImagesCacheAt.Store(time.Now().Unix())
	}
	return self, shared, err
}

// ListSharedImagesAllRegions 列**共享镜像**跨 region 聚合(2026-10-17 新增)
//
// 业务:别人共享给当前账号的镜像(ImageType=Others,ImageOwnerAlias=others)
// 现已并入 ListImagesAllRegions,这个方法保留做兼容(返回 shared 部分)
func (s *AliyunImageService) ListSharedImagesAllRegions() ([]ecspkg.Image, error) {
	_, shared, err := s.ListImagesAllRegions()
	return shared, err
}

// RefreshSharedImagesAllRegions 清缓存 + 重拉(走合并接口)
func (s *AliyunImageService) RefreshSharedImagesAllRegions() ([]ecspkg.Image, error) {
	_, shared, err := s.RefreshImagesAllRegions()
	return shared, err
}

// ListRegions 列可用 region
func (s *AliyunImageService) ListRegions() ([]ecspkg.Region, error) {
	c, err := getECSClient()
	if err != nil {
		return nil, err
	}
	return c.DescribeAllRegions()
}

// CopyImage 跨地域复制(场景 C)
func (s *AliyunImageService) CopyImage(sourceRegion, imageID, destRegion string) (string, error) {
	if sourceRegion == "" || destRegion == "" || imageID == "" {
		return "", ErrAliyunECSRegion
	}
	c, err := getECSClient()
	if err != nil {
		return "", err
	}
	return c.CopyImage(sourceRegion, imageID, destRegion, "")
}

// ShareImage 共享镜像(场景 B)
func (s *AliyunImageService) ShareImage(region, imageID string, targetUIDs []string) error {
	if region == "" || imageID == "" {
		return ErrAliyunECSRegion
	}
	c, err := getECSClient()
	if err != nil {
		return err
	}
	return c.ModifyImageSharePermission(region, imageID, targetUIDs, nil)
}

// UnshareImage 撤销共享
func (s *AliyunImageService) UnshareImage(region, imageID string, targetUIDs []string) error {
	if region == "" || imageID == "" {
		return ErrAliyunECSRegion
	}
	c, err := getECSClient()
	if err != nil {
		return err
	}
	return c.ModifyImageSharePermission(region, imageID, nil, targetUIDs)
}

// ListSharedAccounts 查已共享给谁
func (s *AliyunImageService) ListSharedAccounts(region, imageID string) ([]string, error) {
	if region == "" || imageID == "" {
		return nil, ErrAliyunECSRegion
	}
	c, err := getECSClient()
	if err != nil {
		return nil, err
	}
	return c.DescribeImageSharePermission(region, imageID)
}
