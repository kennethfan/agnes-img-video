# 安全审计报告 — Agnes Creator Studio

> 扫描时间：2026-06-26
> 扫描范围：完整仓库（14 次提交历史）
> 扫描方式：静态分析 + 依赖检查 + Git 历史审计

---

## 裁决
**PASS WITH FINDINGS** — 3 个 Medium、3 个 Low，无 Critical 或 High。

---

## 发现总览

| Severity | Title | CWE | 文件 |
|----------|-------|-----|------|
| 🔶 Medium | API Key 明文存储到磁盘 | CWE-312 | `config.py`, `utils.py`, `app.py` |
| 🔶 Medium | Web UI 无身份认证 | CWE-306 | `app.py` |
| 🔶 Medium | 缺少 `.dockerignore` 文件 | CWE-200 | 根目录 |
| 🟡 Low | `show_error=True` 信息泄露 | CWE-209 | `app.py:927` |
| 🟡 Low | Debug Print 泄露请求细节 | CWE-532 | `api_client.py` |
| 🟡 Low | Supervisor Web 接口无认证 | CWE-306 | `supervisord.conf:5` |

---

## 详细发现

### 🔶 MEDIUM-1: API Key 明文存储到磁盘

**文件**: `config.py:26`, `utils.py:54-57`, `app.py:492-494`

`DEFAULT_API_KEY = os.getenv("AGNES_API_KEY", "") or _cached.get("api_key", "")`

当用户在 UI 中点击"保存配置"，API key 以明文写入 `.config.json`。该文件虽被 `.gitignore` 排除，但仍以明文落在磁盘上。

**攻击路径**:
1. 攻击者获取文件系统访问权限（容器卷被挂载到宿主机、共享主机、或横向移动）
2. 读取 `.config.json` 获取 API Key
3. 滥用 API Key 调用 Agnes AI 服务

**修复建议**:
- 仅将加密或部分掩码的标识符写入磁盘
- 添加文件权限 `chmod 600 .config.json`
- 在文档中警告存储风险，默认不持久化 key

---

### 🔶 MEDIUM-2: Web UI 无身份认证

**文件**: `app.py` — `demo.launch()` 无 `auth=` 参数

任何人只要能访问 Gradio URL（包括 Gradio Share 生成的公网链接），就能使用应用并消耗已配置 API Key 的额度。

**攻击路径**:
1. 部署时 `GRADIO_SHARE=true` 或暴露在公网
2. 攻击者访问 URL，自由调用生成服务
3. API 额度被盗用

**修复建议**:
```python
demo.launch(auth=("admin", "strong-password"), ...)
```
或通过反向代理（Nginx）添加 HTTP Basic Auth。

---

### 🔶 MEDIUM-3: 缺少 `.dockerignore` 文件

**文件**: 根目录 — 文件不存在

`docker build` 默认将上下文目录全部发送给构建守护进程。若 `.config.json`、`.env`、`.git/` 在构建时存在于工作目录，它们会被嵌入 Docker 镜像层。

**攻击路径**:
1. 开发环境有 `.config.json` 或 `.env`
2. 运行 `docker build` 构建镜像
3. 镜像被推送到仓库
4. 任意能拉取镜像的人提取敏感文件

**修复建议**:
创建 `.dockerignore`：
```
outputs/
history.json
.config.json
.env*
assets/
*.md
.git/
.gitignore
__pycache__/
*.pyc
.DS_Store
```

---

### 🟡 LOW-1: `show_error=True` 造成信息泄露

**文件**: `app.py:927`

`demo.launch(show_error=True)` 让 Gradio 在浏览器中显示 Python 回溯信息，可能暴露文件路径、环境变量和内部结构。

**修复建议**: 生产环境设为 `show_error=False`，或自定义错误处理。

---

### 🟡 LOW-2: Debug Print 泄露请求细节

**文件**: `api_client.py`（~20 处 `print()`）

大量 `print()` 记录了完整的请求 URL、payload 结构、响应摘要等。虽然 base64 图片数据已被过滤，但仍包含 URL 和 payload 结构信息。这些日志通过 supervisor 永久保留在 Docker 容器中。

**修复建议**: 使用 `logging` 模块替代 `print`，设置合理日志级别，配置日志轮转。

---

### 🟡 LOW-3: Supervisor Web 接口无认证

**文件**: `supervisord.conf:5`

`[inet_http_server] port=127.0.0.1:9001` 未配置 `username` 和 `password`。虽绑定在 localhost，但若攻击者进入容器，可完全控制进程管理。

**修复建议**: 添加 `username=` 和 `password=` 配置，或移除 `[inet_http_server]` 块（非必需）。

---

## 已降级或拒绝的候选发现

| 候选 | 理由 |
|------|------|
| SSRF via user-provided image URLs | 用户提供的 URL 发送给 **Agnes API 服务器**，非本应用直接 fetch。无 SSRF 攻击面。 |
| 路径遍历 (Image Upload) | `open(image_path)` 使用 Gradio 框架提供的临时路径，攻击者无法控制该参数。 |
| Git 历史密钥泄露 | 所有提交中搜索 `AGNES_API_KEY`，仅有代码引用，无实际密钥被提交。 |
| SSL 验证 | 所有 `requests` 调用使用默认 SSL 验证（`verify=True`），未禁用。 |

---

## 残余风险

- **依赖 CVE 未扫描**：未运行 `pip-audit`，建议补充：`pip install pip-audit && pip-audit`
- **Gradio Share**：`GRADIO_SHARE=true` 会通过 Gradio 公共隧道暴露应用，无认证
- **无速率限制**：无任何配额或限速，可被滥用
- **测试文件密钥**：`test_api.py` 和 `test_video.py` 通过 `os.getenv()` 读取密钥，CI 中可能泄露

---

## 建议修复顺序

| 优先级 | 修复项 | 工作量 | 影响 |
|--------|--------|--------|------|
| P0 | 创建 `.dockerignore` | 1 文件 | 防止密钥嵌入镜像 |
| P1 | Gradio UI 添加认证 | 1 行 | 防止未授权访问 |
| P1 | `.config.json` 权限控制 | 1 行 | 减少本地泄露风险 |
| P2 | 生产环境关闭 `show_error` | 1 行 | 减少信息泄露 |
| P3 | 使用 `logging` 替代 `print` | 中等 | 日志安全 |
| P3 | Supervisor 添加认证 | 2 行 | 容器内纵深防御 |
