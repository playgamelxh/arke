# Arke 知识库管理系统

Arke 是一个面向知识库管理、文档解析、问答生成和知识问答的应用。系统支持将文档上传到本地目录或 RustFS 对象存储，解析后生成知识片段，并结合向量检索与大模型完成知识库问答。

## 功能概览

- **知识库管理**：创建知识库、上传文档、查看解析状态和文档分段。
- **文档存储**：支持本地 `uploads` 目录和 RustFS/S3 兼容对象存储。
- **文档解析**：支持原生解析与 MinerU 解析配置，可在系统设置中检测解析服务。
- **问答生成**：基于文档内容生成可管理的问答数据。
- **知识问答**：基于知识库检索相关片段并生成答案。
- **系统设置**：支持上传位置、RustFS、文档解析、模型参数等配置。

## 技术栈

| 层级 | 技术 |
|------|------|
| 前端 | Vue 3 + TypeScript + Vite + Tailwind CSS |
| 后端 | Go + Gin + GORM |
| 数据库 | MySQL 8.4 |
| 向量库 | Milvus |
| 对象存储 | RustFS / S3 兼容存储 |
| 大模型 | 阿里百炼 DashScope 兼容接口 |
| 部署 | Docker Compose |

## 快速启动

```bash
cp .env.example .env
```

编辑 `.env`，至少填写：

```bash
DASHSCOPE_API_KEY=你的密钥
```

启动服务：

```bash
docker compose up -d --build
```

默认访问地址：

```text
前端：http://localhost:18083
后端：http://localhost:18082/api/health
RustFS：http://localhost:19001
Milvus Bridge：http://localhost:18088
```

## 常用命令

前端检查：

```bash
cd frontend
npm run check
npm run lint
npm run build
```

后端检查：

```bash
cd backend
go test ./...
```

重新构建部署：

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

## 关键环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `WEB_PORT` | 前端端口 | `18083` |
| `SERVER_PORT` | 后端端口 | `8082` |
| `MYSQL_PORT` | MySQL 暴露端口 | `13406` |
| `RUSTFS_PORT` | RustFS API 端口 | `19000` |
| `RUSTFS_WEBUI_PORT` | RustFS 控制台端口 | `19001` |
| `DASHSCOPE_API_KEY` | 阿里百炼 API Key | 空，需自行配置 |
| `DASHSCOPE_BASE_URL` | DashScope 兼容接口地址 | 官方兼容模式地址 |
| `DASHSCOPE_MODEL` | 问答生成模型 | `qwen-plus` |
| `MINERU_BASE_URL` | MinerU 服务地址 | 按实际环境配置 |
| `S3_ENDPOINT` | RustFS/S3 Endpoint | `rustfs:9000` |
| `S3_ACCESS_KEY` | RustFS/S3 Access Key | `rustfsadmin` |
| `S3_SECRET_KEY` | RustFS/S3 Secret Key | `rustfsadmin` |
| `S3_BUCKET` | 文档存储 bucket | `documents` |

## 目录结构

```text
arke/
├── backend/              # Go 后端服务
│   ├── config/           # YAML 配置
│   ├── handlers/         # HTTP handlers
│   ├── migrations/       # 数据库迁移
│   ├── models/           # 数据模型
│   ├── routes/           # 路由注册
│   └── services/         # 业务服务
├── frontend/             # Vue 前端应用
│   ├── src/api/          # API 客户端
│   ├── src/layouts/      # 页面布局
│   ├── src/views/        # 页面视图
│   └── src/types/        # 类型定义
├── milvus-bridge/        # Milvus HTTP 桥接服务
├── uploads/              # 本地上传文件目录
├── docker-compose.yml
└── .env.example
```

## 安全说明

- 不要提交 `.env`。
- 不要把 DashScope API Key 写入 YAML 或源码。
- 生产环境建议修改 MySQL、RustFS 等默认密码。
