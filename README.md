<<<<<<< HEAD
# arke
Arke is knowledge project
=======
# 文档解析问答管理系统

基于 PRD 与技术架构文档实现的文档解析与问答管理系统。支持上传 PDF、PPTX、XLSX 文档，自动解析文本内容，并通过阿里百炼大模型生成可管理的问答。

## 功能概览

- **文档上传**：原始文件直接保存到项目根目录 `uploads/`
- **内容解析**：优先调用 MinerU 服务（`/file_parse`）解析 PDF、Office、图片等文档
- **问答生成**：调用阿里百炼 DashScope API（默认 `qwen-plus`）生成问答预览
- **问答管理**：编辑、启用/停用、删除、批量操作
- **数据库迁移**：启动时自动执行 `server/migrations/` 下的 Go migrations

## 技术栈

| 层级 | 技术 |
|------|------|
| 前端 | Vue 3 + TypeScript + Vite + Tailwind CSS |
| 后端 | Go + Gin + GORM |
| 数据库 | MySQL 8.4.9 |
| 大模型 | 阿里百炼 DashScope 兼容接口 |
| 部署 | Docker Compose |

## 快速启动（Docker）

```bash
# 1. 复制环境变量并填写阿里百炼 API Key
cp .env.example .env
# 编辑 .env，设置 DASHSCOPE_API_KEY=你的密钥

# 2. 启动全部服务
docker compose up -d --build

# 3. 访问
# 前端：http://localhost:8083
# 后端 API：http://localhost:8082/api/health
# MySQL（宿主机）：localhost:3406
```

上传的文件会持久化到 `./uploads` 目录，MySQL 数据保存在 Docker volume 中。

## 本地开发

### 后端

```bash
# 启动 MySQL（可使用 docker compose 仅启动 mysql）
docker compose up -d mysql

# 配置环境变量（项目根目录 .env 会自动加载）
cp .env.example .env

# 运行后端
cd server
go run ./cmd/api
```

默认连接 `127.0.0.1:3406`（Docker 映射端口），上传目录自动解析为项目根目录 `uploads/`。

### 前端

```bash
cd web
npm install
npm run dev
```

开发服务器默认代理到 `http://localhost:8080/api`。

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `MYSQL_DSN` | MySQL 连接串 | `docqa:docqa_password@tcp(127.0.0.1:3406)/doc_parse_qa?...` |
| `UPLOAD_DIR` | 上传目录 | `uploads`（项目根目录） |
| `DASHSCOPE_API_KEY` | 阿里百炼 API Key | 必填（问答生成） |
| `DASHSCOPE_MODEL` | 模型名称 | `qwen-plus` |
| `DASHSCOPE_BASE_URL` | API 地址 | DashScope 兼容模式 |
| `DASHSCOPE_TIMEOUT_SECONDS` | 百炼 API 超时（秒，最大 300） | `300` |
| `MINERU_BASE_URL` | MinerU 服务地址 | `http://10.3.17.83:7861` |
| `MINERU_TIMEOUT_SECONDS` | MinerU 解析超时（秒，最大 300） | `300` |
| `PORT` | 后端端口 | `8082`（Docker 映射） |

> **安全提示**：API Key 仅通过 `.env` 或部署环境变量注入，切勿提交到 Git。

## 百炼 API 常见问题

### 1. DNS 解析失败（`no such host`）

Docker 容器内无法解析 `dashscope.aliyuncs.com` 时，`docker-compose.yml` 已为 server 服务配置公共 DNS（223.5.5.5 等）。修改后重建后端：

```bash
docker compose up -d --build server
```

### 2. 403 IP 白名单限制（`IP access denied by API-Key restriction`）

说明当前 API Key 在百炼控制台启用了 **IP 访问限制**，而服务器出口 IP 不在白名单中。

**解决步骤：**

1. 登录 [阿里云百炼控制台](https://bailian.console.aliyun.com/) → **API Key 管理**
2. 找到对应 Key，选择以下任一方式：
   - **关闭 IP 限制**（本地开发推荐）
   - **将本机公网 IP 加入白名单**（生产环境推荐）
3. 查看当前公网 IP：浏览器访问 https://ifconfig.me 或在终端执行 `curl ifconfig.me`
4. 修改后无需改代码，直接重试生成问答

## API 概览

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/documents/upload` | 上传文档到 uploads 并解析 |
| GET | `/api/documents` | 文档列表 |
| POST | `/api/qa/generate-preview` | 调用百炼生成问答预览 |
| POST | `/api/qa/save-generated` | 保存预览问答 |
| GET | `/api/qa` | 问答列表 |

完整接口定义见 `.trae/documents/技术架构-文档解析问答管理系统.md`。

## 目录结构

```
doc-parse-qa/
├── uploads/              # 上传文件存储（持久化）
├── server/
│   ├── cmd/api/main.go   # 后端入口
│   └── migrations/       # 数据库迁移 SQL
├── web/                  # Vue 前端
├── docker-compose.yml
└── .env.example
```
>>>>>>> cf24f4b (init)
