# 技术架构 - Arke 知识库管理系统

## 1. 总体架构

Arke 采用前后端分离架构，使用 Docker Compose 编排完整运行环境。

```text
Browser
  |
  | HTTP
  v
Nginx / Vue 静态资源容器 arke-web
  |
  | /api 反向代理
  v
Go Gin 后端 arke-server
  |--------- MySQL：业务数据、系统设置、文档和问答数据
  |--------- RustFS：文档对象存储
  |--------- MinerU：复杂文档解析服务
  |--------- DashScope：问答生成、答案生成、Embedding
  |--------- Milvus Bridge：HTTP 桥接 Milvus
                |
                v
              Milvus
```

## 2. 运行时服务

`docker-compose.yml` 当前包含以下服务：

| 服务 | 容器名 | 说明 |
|------|--------|------|
| `web` | `arke-web` | 前端静态资源与 Nginx 反向代理 |
| `server` | `arke-server` | Go 后端 API 服务 |
| `mysql` | `arke-mysql` | MySQL 业务数据库 |
| `rustfs` | `arke-rustfs` | RustFS/S3 兼容对象存储 |
| `milvus` | `arke-milvus` | 向量数据库 |
| `milvus-bridge` | `arke-milvus-bridge` | Milvus HTTP 桥接服务 |
| `etcd` | `milvus-etcd` | Milvus 依赖 |
| `minio` | `milvus-minio` | Milvus 内部对象存储依赖 |

默认端口：

| 服务 | 端口 |
|------|------|
| 前端 | `18083` |
| 后端 | `8082` |
| MySQL | `13406` |
| RustFS API | `19000` |
| RustFS Console | `19001` |
| Milvus | `19530` |
| Milvus Bridge | `18088` |

## 3. 前端架构

前端位于 `frontend/`，技术栈：

- Vue 3
- TypeScript
- Vite
- Tailwind CSS
- Vue Router
- lucide-vue-next

### 3.1 前端目录结构

```text
frontend/
├── src/
│   ├── api/
│   │   └── client.ts              # API 请求封装
│   ├── components/                # 通用 UI 组件
│   │   ├── ConfirmDialog.vue
│   │   ├── Empty.vue
│   │   ├── PaginationBar.vue
│   │   └── Toast.vue
│   ├── composables/               # 组合式逻辑
│   │   ├── useConfirm.ts
│   │   ├── useTheme.ts
│   │   └── useToast.ts
│   ├── layouts/
│   │   └── AppLayout.vue          # 左侧菜单与主布局
│   ├── router/
│   │   └── index.ts               # 路由定义
│   ├── types/
│   │   └── domain.ts              # 领域类型定义
│   ├── utils/                     # 工具函数
│   │   ├── csv.ts
│   │   ├── format.ts
│   │   └── text.ts
│   ├── views/                     # 页面级视图
│   │   ├── DashboardView.vue
│   │   ├── DocumentsView.vue
│   │   ├── DocumentDetailView.vue
│   │   ├── KnowledgeBaseListView.vue
│   │   ├── KnowledgeBaseDetailView.vue
│   │   ├── KnowledgeAskView.vue
│   │   ├── QAGenerateView.vue
│   │   ├── QAManageView.vue
│   │   ├── QASettingsView.vue
│   │   └── SettingsView.vue
│   ├── App.vue
│   ├── main.ts
│   └── style.css
├── Dockerfile
├── nginx.conf
├── package.json
└── vite.config.ts
```

### 3.2 前端路由

路由定义在 `frontend/src/router/index.ts`。

| 路径 | 页面 | 说明 |
|------|------|------|
| `/dashboard` | `DashboardView` | 工作台 |
| `/knowledge-bases` | `KnowledgeBaseListView` | 知识库列表 |
| `/knowledge-bases/:id` | `KnowledgeBaseDetailView` | 知识库详情 |
| `/knowledge-ask` | `KnowledgeAskView` | 知识库问答 |
| `/documents` | `DocumentsView` | 全局文档列表 |
| `/documents/:id` | `DocumentDetailView` | 文档详情 |
| `/qa-generate` | `QAGenerateView` | 问答生成 |
| `/qa` | `QAManageView` | 问答管理 |
| `/qa/settings` | `QASettingsView` | 问答设置 |
| `/settings` | `SettingsView` | 系统设置 |

### 3.3 前端 API 层

`frontend/src/api/client.ts` 封装所有 API 调用：

- `request<T>()`：统一处理 JSON 响应、错误码和空响应。
- `query()`：统一构造分页和筛选参数。
- `api` 对象：集中导出业务接口。

响应格式约定：

```ts
interface ApiResponse<T> {
  code: number
  message: string
  data: T
}
```

前端会将以下异常转为用户可读错误：

- HTTP 413：文件过大。
- HTTP 502 / 504：请求超时。
- 空响应：服务器返回空响应。
- 非 JSON 响应：服务器响应格式异常。

### 3.4 前端类型模型

`frontend/src/types/domain.ts` 定义核心类型：

- `DocumentItem`
- `DocumentSegment`
- `KnowledgeBase`
- `SearchHit`
- `KnowledgeAskResult`
- `QAItem`
- `QAGenerateTask`
- `SettingsMap`

其中 `SettingsMap` 覆盖：

- 上传限制。
- 问答生成设置。
- RustFS 配置。
- 文档解析配置。

### 3.5 系统设置页面实现

`SettingsView.vue` 当前采用选项卡布局：

1. 基础设置
2. 上传位置
3. RustFS 配置
4. 文档解析

RustFS 检测只提交 RustFS 相关字段：

```ts
rustfsEndpoint
rustfsAccessKey
rustfsSecretKey
rustfsBucket
rustfsRegion
rustfsUseSSL
```

文档解析检测只提交解析相关字段：

```ts
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

这样避免检测接口传递无关配置，后端也使用白名单再次过滤。

## 4. 后端架构

后端位于 `backend/`，技术栈：

- Go
- Gin
- GORM
- MySQL
- MinIO SDK / S3 兼容协议
- DashScope 兼容 API

### 4.1 后端目录结构

```text
backend/
├── config/
│   ├── config.go              # 配置加载、YAML + 环境变量覆盖
│   ├── config.yaml            # 通用默认配置
│   ├── config.dev.yaml        # 开发配置
│   └── config.prod.yaml       # 生产配置
├── db/
│   └── db.go                  # 数据库初始化与迁移
├── handlers/                  # HTTP 入口层
│   ├── document.go
│   ├── kb_document.go
│   ├── knowledge_ask.go
│   ├── knowledge_base.go
│   ├── qa.go
│   └── settings.go
├── middleware/
│   └── cors.go                # CORS 中间件
├── migrations/                # SQL 迁移文件
├── models/                    # 数据模型、请求响应结构
├── routes/
│   └── routes.go              # 路由注册
├── services/                  # 业务服务层
│   ├── bailian.go             # DashScope 问答生成
│   ├── chunker.go             # 文档切分
│   ├── document.go            # 历史文档服务
│   ├── embedding.go           # Embedding 调用
│   ├── file_helper.go         # 文件辅助逻辑
│   ├── kb_document.go         # 知识库文档上传/解析/索引
│   ├── knowledge_ask.go       # 知识库问答
│   ├── knowledge_base.go      # 知识库管理
│   ├── milvus_bridge.go       # Milvus Bridge 客户端
│   ├── mineru.go              # MinerU 客户端
│   ├── qa.go                  # 问答生成与管理
│   ├── runtime_storage.go     # 运行时存储路由
│   ├── settings.go            # 系统设置
│   └── storage.go             # 本地 / RustFS 存储实现
├── Dockerfile
├── go.mod
└── main.go
```

### 4.2 后端启动流程

入口：`backend/main.go`

流程：

1. `config.Load()` 加载配置。
2. 初始化 Gin。
3. 初始化数据库和迁移。
4. 初始化 `SettingsService`。
5. 初始化运行时存储 `RuntimeStorage`。
6. 初始化 MinerU、DashScope、Milvus Bridge、Embedding 客户端。
7. 初始化业务 services。
8. 初始化 handlers。
9. 注册 routes。
10. 启动 HTTP 服务。

### 4.3 配置加载

配置由三层组成：

1. YAML：`config/config.{env}.yaml`
2. 环境变量覆盖
3. 数据库系统设置覆盖运行时行为

`APP_ENV` 默认是 `dev`。

环境变量覆盖项：

```text
DASHSCOPE_API_KEY
DASHSCOPE_BASE_URL
DASHSCOPE_MODEL
DASHSCOPE_TIMEOUT_SECONDS
MINERU_BASE_URL
MINERU_TIMEOUT_SECONDS
S3_ENDPOINT
S3_ACCESS_KEY
S3_SECRET_KEY
S3_BUCKET
S3_REGION
S3_USE_SSL
```

### 4.4 路由组织

路由注册在 `backend/routes/routes.go`。

主要路由组：

```text
/api/settings
/api/knowledge-bases
/api/documents
/api/knowledge-ask
/api/qa
```

健康检查：

```text
GET /health
```

### 4.5 Handler 层

Handler 层负责：

- 读取 URL 参数、Query 和 JSON/FormData。
- 调用 service。
- 返回统一响应格式。

典型文件：

- `handlers/settings.go`
- `handlers/kb_document.go`
- `handlers/knowledge_base.go`
- `handlers/knowledge_ask.go`
- `handlers/qa.go`

### 4.6 Service 层

Service 层承载核心业务逻辑。

#### SettingsService

文件：`services/settings.go`

职责：

- 提供系统设置默认值。
- 读取数据库中的设置。
- 校验和保存设置。
- 生成 RustFS 存储配置。
- 生成文档解析配置。
- 检测文档解析配置。

#### RuntimeStorage

文件：`services/runtime_storage.go`

职责：

- 每次上传、下载、删除时读取当前系统设置。
- 根据 `storageMode` 路由到本地存储或 RustFS。
- RustFS 检测时只使用 RustFS 白名单字段。
- 下载和删除历史文件时兼容旧存储位置。

#### StorageClient / LocalStorageClient

文件：`services/storage.go`

职责：

- `StorageClient`：RustFS/S3 存储实现。
- `LocalStorageClient`：本地文件系统存储实现。
- 共同实现统一的 `StorageInterface`。

#### KBDocumentService

文件：`services/kb_document.go`

职责：

- 上传文档并关联知识库。
- 解析文档。
- 切分分段。
- 管理文档分段。
- 将分段写入 Milvus。
- 更新知识库文档数和向量数。

#### MinerUClient

文件：`services/mineru.go`

职责：

- 调用 MinerU `/file_parse`。
- 按系统设置传递解析参数。
- 检测 MinerU 服务连接。
- 将 MinerU 返回的 Markdown 内容转换为文档分段。

#### KnowledgeBaseService

文件：`services/knowledge_base.go`

职责：

- 知识库 CRUD。
- Embedding 模型信息。
- Milvus collection 管理。
- 知识库检索。

#### KnowledgeAskService

文件：`services/knowledge_ask.go`

职责：

- 接收用户问题。
- 调用知识库检索。
- 支持 similarity、keyword、length 重排。
- 调用 DashScope 生成最终答案。

#### QAService

文件：`services/qa.go`

职责：

- 生成问答预览。
- 管理生成任务。
- 保存问答。
- CRUD 问答。
- 批量删除。

#### BailianClient

文件：`services/bailian.go`

职责：

- 调用 DashScope 兼容接口。
- 生成问答。
- 生成单题答案。
- 生成知识库问答答案。

#### EmbeddingClient

文件：`services/embedding.go`

职责：

- 调用 DashScope Embedding 模型。
- 为文档分段和问题生成向量。

#### MilvusBridge

文件：`services/milvus_bridge.go`

职责：

- 通过 HTTP 调用 `milvus-bridge`。
- 创建 collection。
- 写入向量。
- 搜索向量。
- 健康检查。

## 5. 数据模型

核心模型位于 `backend/models/`。

主要模型：

- `Document`
- `DocumentSegment`
- `KnowledgeBase`
- `QA`
- `QAGenerateTask`
- `SystemSetting`

迁移文件位于 `backend/migrations/`，当前包含：

```text
000001_init_schema
000002_qa_generate_tasks
000003_qa_generate_instruction
000004_knowledge_bases
000005_document_chunk_config
000006_qa_generate_by_kb
```

## 6. 文档上传、解析和索引流程

```text
用户选择知识库并上传文件
  |
  v
前端 FormData 提交 /api/documents/upload
  |
  v
KBDocumentHandler.Upload
  |
  v
KBDocumentService.UploadDocument
  |
  v
RuntimeStorage.PutObject
  |
  |-- storageMode=local  -> LocalStorageClient
  |-- storageMode=rustfs -> StorageClient(RustFS)
  v
写入 documents 表，状态 uploaded
  |
  v
用户触发解析 /api/documents/:id/parse
  |
  v
读取系统解析设置
  |
  |-- native -> 原生解析
  |-- mineru -> MinerU 解析
  |-- auto   -> 优先 MinerU，PDF 可回退原生
  v
生成 document_segments
  |
  v
用户触发索引 /api/documents/:id/index
  |
  v
EmbeddingClient 生成向量
  |
  v
MilvusBridge 写入 Milvus
```

## 7. 知识库问答流程

```text
用户在 KnowledgeAskView 提问
  |
  v
POST /api/knowledge-ask
  |
  v
KnowledgeAskService.Ask
  |
  v
KnowledgeBaseService.Search
  |
  v
EmbeddingClient 生成问题向量
  |
  v
MilvusBridge 搜索相似分段
  |
  v
rerankSources 根据模式重排
  |
  v
BailianClient.GenerateKnowledgeAnswer
  |
  v
返回 answer、confidence、sources
```

## 8. 系统设置运行时行为

系统设置数据存储在 `system_settings` 表。

默认值来自 `SettingsService.Defaults()`，其中包含：

- 上传大小与类型。
- 问答生成数量与批次。
- DashScope 模型地址。
- 上传存储模式。
- RustFS 配置。
- MinerU 解析配置。

保存设置使用 upsert：

```text
setting_key 唯一
存在则更新 setting_value
不存在则创建
```

检测接口设计：

- RustFS 检测只接收 RustFS 相关字段。
- 文档解析检测只接收解析相关字段。
- 后端白名单过滤无关字段。
- 检测时会和数据库已保存设置合并，避免前端必须提交全部系统设置。

## 9. 安全与配置约定

- `.env` 不提交。
- DashScope API Key 不写入 YAML 和源码。
- `config.dev.yaml` 和 `config.prod.yaml` 中 `dashscope.api_key` 保持为空。
- 运行时通过 `DASHSCOPE_API_KEY` 注入。
- RustFS 和 MySQL 默认密码仅用于本地开发，生产环境必须修改。

## 10. 开发和质量检查

前端：

```bash
cd frontend
npm run check
npm run lint
npm run build
```

后端：

```bash
cd backend
go test ./...
```

部署：

```bash
docker compose up -d --build
```

只重建前端：

```bash
docker compose up -d --build web
```

只重建后端：

```bash
docker compose up -d --build server
```

## 11. 后续架构优化建议

1. 文档解析和索引可改为异步任务，避免长请求阻塞。
2. 增加任务队列，统一管理解析、索引、问答生成任务。
3. 增加用户认证和权限控制。
4. 对系统设置中的敏感字段增加脱敏显示。
5. 增加后端单元测试和接口测试。
6. 增加 Docker healthcheck 到 server 和 web。
7. 为 Milvus Bridge 增加更完整的错误码和请求日志。
