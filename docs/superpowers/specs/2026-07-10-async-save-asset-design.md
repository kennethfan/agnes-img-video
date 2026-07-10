# Async SaveAsset — 异步存储后台处理

**Date:** 2026-07-10
**Status:** Draft

## 问题

SaveAsset 保存远程 URL 到作品库时，`storeFile()` 同步做文件下载（可能几十 MB）+ GitHub 上传，导致 API 响应耗时 = 下载时间 + 上传时间（数秒到数十秒）。用户感知"点击保存后一直在转圈"。

## 方案

简单 goroutine 异步处理。SaveAsset 立即返回 asset ID，后台完成下载/上传并更新 DB。

### 新流程

```
请求到来
  → 解析参数、判断类型
  → 本地路径：与当前相同（同步，无等待）
  → 远程 URL：
      → 立即插入 asset（local_path="" github_url=""）
      → 返回 {"id": ID}
      → go processAssetStorage(id, url, type)   ← goroutine 后台跑
          → storeFile(url, type)                 ← 下载+上传
          → repo.UpdateStoragePaths(id, lp, gh)  ← 更新 DB 字段
          → 失败打日志（不做重试）
```

### 新增方法

```go
// processAssetStorage 异步处理资产存储（下载+上传）
func (h *AssetHandler) processAssetStorage(id int64, imageURL string, assetType string) {
    localPath, githubURL, err := h.storeFile(imageURL, assetType)
    if err != nil {
        log.Printf("[Asset] 异步处理存储失败 id=%d: %v", id, err)
        return
    }
    if err := h.repo.UpdateStoragePaths(id, localPath, githubURL); err != nil {
        log.Printf("[Asset] 异步更新存储路径失败 id=%d: %v", id, err)
    }
}
```

### 边界情况

| 场景 | 行为 |
|------|------|
| 本地路径 | 同步处理，不触发 goroutine |
| 远程 URL | 创建记录后立即返回，后台下载 |
| 下载失败 | 打日志，asset 保持空路径，用户可手动"转存"重试 |
| GitHub 上传失败 | 打日志，local_path 已有值但 github_url 为空 |
| 服务器重启 | goroutine 丢失，asset 空路径，手动转存恢复 |
| 并发 SaveAsset | 每个请求独立 goroutine，互不干扰 |

## 改动范围

仅 **1 个文件**：`internal/handler/asset.go`
- SaveAsset 方法：远程 URL 分支改为先插入后异步
- 新增 processAssetStorage 方法

无需新增路由、接口、DB 迁移。

## 回退方案

保留现有 `storeFile` 同步方法不变；TransferAsset handler 不变（仍然是同步调用 `storeFile`）。如需回退，只需 SaveAsset 恢复为同步调用 `storeFile`。
