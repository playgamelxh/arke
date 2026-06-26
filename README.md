# Arke 知识库管理系统

Arke 是一个面向知识库管理、文档解析、向量索引、问答生成和知识问答的应用。系统以“知识库管理”为核心，支持将文档上传到本地目录或 RustFS 对象存储，解析并切分为知识片段，写入 Milvus 向量库，再结合 DashScope 大模型完成知识库问答和问答数据生成。

## 功能概览

- **工作台**：展示文档、解析、问答等关键统计数据。
- **知识库管理**：创建、编辑、删除知识库，管理知识库默认切分策略和索引类型。
- **文档管理**：上传文档、查看解析状态、重新解析、索引、编辑分段、删除文档。
- **文档解析**：支持原生解析和 MinerU 解析，支持 PDF 回退策略、图片分析、表格解析、公式解析等设置。
- **文档存储**：支持本地 `uploads` 目录和 RustFS/S3 兼容对象存储，系统设置中可检测 RustFS 配置。
- **向量索引**：将文档分段生成 embedding 后写入 Milvus，用于知识库检索。
- **知识库问答**：基于知识库召回片段生成答案，并展示引用来源。
- **问答生成**：基于知识库内容生成可管理的问答数据，支持预览、保存、编辑、启用/禁用和删除。
- **系统设置**：使用选项卡管理基础设置、上传位置、RustFS 配置和文档解析配置。

## 技术栈

| 层级 | 技术 |
|------|------|
| 前端 | Vue 3 + TypeScript + Vite + Tailwind CSS |
| 路由 | Vue Router |
| 图标 | lucide-vue-next |
| 后端 | Go + Gin + GORM |
| 数据库 | MySQL 8.4 |
| 对象存储 | RustFS / S3 兼容存储、本地 uploads |
| 文档解析 | 原生解析 + MinerU |
| 向量库 | Milvus |
| 向量桥接 | milvus-bridge Python 服务 |
| 大模型 | 阿里百炼 DashScope 兼容接口 |
| 部署 | Docker Compose |

## 快速启动

复制环境变量模板：

```bash
cp .env.example .env
```

编辑 `.env`，至少填写：

```bash
DASHSCOPE_API_KEY=你的密钥
```

启动完整服务：

```bash
docker compose up -d --build
```

默认访问地址：

```text
前端：http://localhost:18083
后端健康检查：http://localhost:8082/health
RustFS 控制台：http://localhost:19001
Milvus Bridge：http://localhost:18088
```

如果使用局域网访问，将 `localhost` 替换为宿主机 IP。

## Docker Compose 服务

| 服务 | 容器名 | 说明 |
|------|--------|------|
| `web` | `arke-web` | 前端静态资源与 Nginx 反向代理 |
| `server` | `arke-server` | Go 后端 API 服务 |
| `mysql` | `arke-mysql` | 业务数据库 |
| `rustfs` | `arke-rustfs` | RustFS/S3 兼容对象存储 |
| `milvus` | `arke-milvus` | 向量数据库 |
| `milvus-bridge` | `arke-milvus-bridge` | Milvus HTTP 桥接服务 |
| `etcd` | `milvus-etcd` | Milvus 依赖 |
| `minio` | `milvus-minio` | Milvus 内部对象存储依赖 |

## 目录结构

```text
arke/
├── backend/                    # Go 后端服务
│   ├── config/                 # YAML 配置与配置加载
│   ├── db/                     # 数据库初始化和迁移执行
│   ├── handlers/               # HTTP handler 层
│   ├── middleware/             # 中间件
│   ├── migrations/             # 数据库迁移 SQL
│   ├── models/                 # GORM 模型、请求响应结构
│   ├── routes/                 # Gin 路由注册
│   ├── services/               # 核心业务服务
│   ├── Dockerfile
│   ├── go.mod
│   └── main.go
├── frontend/                   # Vue 前端应用
│   ├── src/
│   │   ├── api/                # API 客户端
│   │   ├── components/         # 通用组件
│   │   ├── composables/        # 组合式逻辑
│   │   ├── layouts/            # 页面布局
│   │   ├── router/             # 前端路由
│   │   ├── types/              # TypeScript 领域类型
│   │   ├── utils/              # 工具函数
│   │   └── views/              # 页面视图
│   ├── Dockerfile
│   ├── nginx.conf
│   ├── package.json
│   └── vite.config.ts
├── milvus-bridge/              # Milvus HTTP 桥接服务
├── uploads/                    # 本地上传文件目录
├── .trae/documents/            # 产品和技术文档
├── docker-compose.yml
├── .env.example
└── README.md
```

## 前端实现说明

前端入口：

```text
frontend/src/main.ts
frontend/src/App.vue
```

主布局：

```text
frontend/src/layouts/AppLayout.vue
```

左侧菜单当前定位为：

```text
Knowledge Base
知识库管理
```

### 前端路由

路由文件：`frontend/src/router/index.ts`

| 路径 | 页面 | 说明 |
|------|------|------|
| `/dashboard` | `DashboardView.vue` | 工作台 |
| `/knowledge-bases` | `KnowledgeBaseListView.vue` | 知识库列表 |
| `/knowledge-bases/:id` | `KnowledgeBaseDetailView.vue` | 知识库详情 |
| `/knowledge-ask` | `KnowledgeAskView.vue` | 知识库问答 |
| `/documents` | `DocumentsView.vue` | 全局文档列表 |
| `/documents/:id` | `DocumentDetailView.vue` | 文档详情 |
| `/qa-generate` | `QAGenerateView.vue` | 问答生成 |
| `/qa` | `QAManageView.vue` | 问答管理 |
| `/qa/settings` | `QASettingsView.vue` | 问答设置 |
| `/settings` | `SettingsView.vue` | 系统设置 |

### 前端 API 层

文件：`frontend/src/api/client.ts`

职责：

- 统一封装 `fetch` 请求。
- 统一处理接口响应格式。
- 统一处理 413、502、504、空响应、非 JSON 响应等异常。
- 统一导出 `api` 对象供页面使用。

接口响应格式：

```ts
interface ApiResponse<T> {
  code: number
  message: string
  data: T
}
```

### 前端类型层

文件：`frontend/src/types/domain.ts`

核心类型：

- `DocumentItem`
- `DocumentSegment`
- `KnowledgeBase`
- `SearchHit`
- `KnowledgeAskResult`
- `QAItem`
- `QAGenerateTask`
- `SettingsMap`

### 系统设置页面

文件：`frontend/src/views/SettingsView.vue`

当前使用选项卡布局：

1. 基础设置
2. 上传位置
3. RustFS 配置
4. 文档解析

RustFS 检测只提交 RustFS 相关字段：

```text
rustfsEndpoint
rustfsAccessKey
rustfsSecretKey
rustfsBucket
rustfsRegion
rustfsUseSSL
```

文档解析检测只提交解析相关字段：

```text
parseEngine
parsePDFNativeFallback
mineruBaseURL
mineruTimeoutSeconds
mineruParseMethod
mineruEffort
mineruLanguage
mineruImageAnalysis
mineruTableEnable
mineruFormulaEnable
```

## 后端组织架构

后端入口：

```text
backend/main.go
```

启动流程：

1. 加载配置。
2. 初始化 Gin。
3. 初始化数据库和迁移。
4. 初始化系统设置服务。
5. 初始化运行时存储路由。
6. 初始化 MinerU、DashScope、Milvus Bridge、Embedding 客户端。
7. 初始化业务服务。
8. 初始化 handlers。
9. 注册路由。
10. 启动 HTTP 服务。

### 后端目录职责

| 目录 | 说明 |
|------|------|
| `config/` | YAML 配置加载和环境变量覆盖 |
| `db/` | 数据库连接和迁移 |
| `handlers/` | HTTP 入参、出参和错误处理 |
| `middleware/` | CORS 等中间件 |
| `migrations/` | 数据库迁移 SQL |
| `models/` | 数据模型、请求响应结构 |
| `routes/` | Gin 路由分组注册 |
| `services/` | 核心业务逻辑 |

### Service 层说明

| 文件 | 说明 |
|------|------|
| `settings.go` | 系统设置默认值、保存、校验、解析配置、存储配置 |
| `runtime_storage.go` | 根据系统设置动态选择本地存储或 RustFS |
| `storage.go` | RustFS/S3 和本地存储实现 |
| `kb_document.go` | 知识库文档上传、解析、分段、索引 |
| `knowledge_base.go` | 知识库 CRUD、索引配置、搜索 |
| `knowledge_ask.go` | 知识库问答、召回重排、生成答案 |
| `qa.go` | 问答生成任务、问答 CRUD、批量删除 |
| `bailian.go` | DashScope 问答生成和答案生成 |
| `embedding.go` | DashScope Embedding 调用 |
| `mineru.go` | MinerU 解析和连接检测 |
| `milvus_bridge.go` | 调用 Milvus Bridge 写入和搜索向量 |
| `chunker.go` | 文档分段切分策略 |

## 后端 API 概览

### 系统设置

```text
GET    /api/settings
PUT    /api/settings
POST   /api/settings/test-rustfs
POST   /api/settings/test-parse
```

### 工作台

```text
GET    /api/stats
```

### 知识库

```text
GET    /api/knowledge-bases
POST   /api/knowledge-bases
GET    /api/knowledge-bases/:id
PUT    /api/knowledge-bases/:id
DELETE /api/knowledge-bases/:id
PUT    /api/knowledge-bases/:id/index
GET    /api/knowledge-bases/:id/documents
POST   /api/knowledge-bases/search
GET    /api/knowledge-bases/embedding-models
```

### 文档

```text
POST   /api/documents/upload
GET    /api/documents
GET    /api/documents/:id
PUT    /api/documents/:id
DELETE /api/documents/:id
POST   /api/documents/:id/parse
POST   /api/documents/:id/index
GET    /api/documents/:id/segments
PUT    /api/documents/:id/segments/:segmentId
DELETE /api/documents/:id/segments/:segmentId
```

### 知识库问答

```text
POST   /api/knowledge-ask
```

### 问答

```text
POST   /api/qa/generate-preview
GET    /api/qa/generate-tasks/:id
POST   /api/qa/save-generated
POST   /api/qa/generate-answer
GET    /api/qa
POST   /api/qa
GET    /api/qa/:id
PUT    /api/qa/:id
PATCH  /api/qa/:id/status
DELETE /api/qa/:id
DELETE /api/qa/batch
```

### 健康检查

```text
GET    /health
```

## 核心流程

### 文档上传、解析、索引

```text
上传文档
  -> RuntimeStorage 按系统设置保存到 local 或 RustFS
  -> 写入 documents 表
  -> 手动触发解析
  -> 根据解析设置选择 native / MinerU / auto
  -> 生成 document_segments
  -> 手动触发索引
  -> DashScope Embedding 生成向量
  -> Milvus Bridge 写入 Milvus
```

### 知识库问答

```text
用户提问
  -> 生成问题向量
  -> Milvus 召回知识片段
  -> 按 similarity / keyword / length 重排
  -> 选择指定数量片段
  -> DashScope 生成最终答案
  -> 返回答案、置信度、引用来源
```

## 配置说明

配置来源优先级：

```text
环境变量 > YAML 配置 > 系统默认值
```

系统设置保存在数据库 `system_settings` 表中，用于运行时覆盖上传位置、RustFS、文档解析等配置。

### 关键环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `WEB_PORT` | 前端端口 | `18083` |
| `SERVER_PORT` | 后端端口 | `8082` |
| `MYSQL_PORT` | MySQL 暴露端口 | `13406` |
| `RUSTFS_PORT` | RustFS API 端口 | `19000` |
| `RUSTFS_WEBUI_PORT` | RustFS 控制台端口 | `19001` |
| `MILVUS_PORT` | Milvus 端口 | `19530` |
| `MILVUS_BRIDGE_PORT` | Milvus Bridge 端口 | `18088` |
| `DASHSCOPE_API_KEY` | DashScope API Key | 空，需自行配置 |
| `DASHSCOPE_BASE_URL` | DashScope 兼容接口地址 | 官方兼容模式地址 |
| `DASHSCOPE_MODEL` | 问答生成模型 | `qwen-plus` |
| `DASHSCOPE_TIMEOUT_SECONDS` | DashScope 超时秒数 | `300` |
| `MINERU_BASE_URL` | MinerU 服务地址 | 按实际环境配置 |
| `MINERU_TIMEOUT_SECONDS` | MinerU 超时秒数 | `300` |
| `S3_ENDPOINT` | RustFS/S3 Endpoint | `rustfs:9000` |
| `S3_ACCESS_KEY` | RustFS/S3 Access Key | `rustfsadmin` |
| `S3_SECRET_KEY` | RustFS/S3 Secret Key | `rustfsadmin` |
| `S3_BUCKET` | 文档存储 bucket | `documents` |
| `S3_REGION` | S3 Region | `us-east-1` |
| `S3_USE_SSL` | 是否使用 SSL | `false` |

## 常用开发命令

前端：

```bash
cd frontend
npm install
npm run dev
npm run check
npm run lint
npm run build
```

后端：

```bash
cd backend
go mod download
go run .
go test ./...
```

Docker：

```bash
docker compose up -d --build
docker compose ps
docker compose logs -f server
docker compose logs -f web
```

只重建前端：

```bash
docker compose up -d --build web
```

只重建后端：

```bash
docker compose up -d --build server
```

## 文档资料

项目文档位于：

```text
.trae/documents/PRD-Arke知识库管理系统.md
.trae/documents/技术架构-Arke知识库管理系统.md
```

## 安全说明

- 不要提交 `.env`。
- 不要把 DashScope API Key 写入 YAML、README 或源码。
- `backend/config/config.dev.yaml` 和 `backend/config/config.prod.yaml` 中的 `dashscope.api_key` 应保持为空。
- 生产环境必须修改 MySQL、RustFS 等默认账号密码。
- 系统设置中的敏感字段建议后续增加脱敏显示。

## 质量检查

提交或部署前建议执行：

```bash
cd frontend
npm run check
npm run lint
npm run build
```

```bash
cd backend
go test ./...
```

## 后续优化建议

- 将文档解析、索引和问答生成改为统一异步任务队列。
- 为后端 services 增加单元测试。
- 增加登录认证和权限控制。
- 为系统设置中的 Access Key / Secret Key 增加脱敏和二次确认。
- 增加 server / web 的 Docker healthcheck。
- 增加更详细的解析任务进度和失败重试能力。
