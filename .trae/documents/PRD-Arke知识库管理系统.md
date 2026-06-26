# PRD - Arke 知识库管理系统

## 1. 产品定位

Arke 是一个面向企业和个人知识资产管理的知识库管理系统。系统围绕“知识库 - 文档 - 分段 - 向量索引 - 问答生成 - 知识问答”形成完整闭环，支持用户上传多类型文档，完成文档解析、内容切分、向量化索引、问答生成和基于知识库的智能问答。

当前项目不再定位为单纯的“文档解析问答中枢”，而是以“知识库管理”为核心，文档解析和问答生成是知识库管理流程中的能力模块。

## 2. 核心目标

1. 提供清晰的知识库管理入口，支持创建、维护和删除知识库。
2. 支持向知识库上传文档，并对文档进行解析、分段、编辑和索引。
3. 支持本地 uploads 与 RustFS/S3 兼容对象存储两种文档保存方式。
4. 支持原生解析与 MinerU 解析，并能在系统设置中检测解析配置是否正确。
5. 支持将知识库内容写入 Milvus，形成可检索向量库。
6. 支持基于知识库召回内容生成智能问答结果。
7. 支持基于文档或知识库内容生成可管理的问答数据。
8. 提供系统级设置能力，降低部署后的配置修改成本。

## 3. 用户角色

### 3.1 管理员

- 管理知识库、文档和系统设置。
- 配置 RustFS、MinerU、模型接口等系统参数。
- 检查上传、解析、索引、问答生成流程是否可用。

### 3.2 内容运营/知识维护人员

- 创建知识库。
- 上传和维护文档。
- 查看解析状态、编辑文档分段。
- 生成、编辑、启用或禁用问答。

### 3.3 普通使用者

- 在知识库问答页面选择知识库并提问。
- 查看答案、引用片段和召回来源。

## 4. 功能模块

## 4.1 工作台

对应前端页面：`DashboardView.vue`

能力：

- 展示文档、解析成功、解析失败、问答数量等统计信息。
- 展示最近上传文档。
- 为用户提供系统运行状态概览。

后端接口：

- `GET /api/stats`

## 4.2 知识库管理

对应前端页面：

- `KnowledgeBaseListView.vue`
- `KnowledgeBaseDetailView.vue`

能力：

- 创建知识库。
- 编辑知识库名称和描述。
- 删除知识库。
- 查看知识库下文档列表。
- 配置知识库默认切分策略。
- 配置向量索引类型。
- 查看知识库文档数和向量数。

后端接口：

- `GET /api/knowledge-bases`
- `POST /api/knowledge-bases`
- `GET /api/knowledge-bases/:id`
- `PUT /api/knowledge-bases/:id`
- `DELETE /api/knowledge-bases/:id`
- `PUT /api/knowledge-bases/:id/index`
- `GET /api/knowledge-bases/:id/documents`
- `POST /api/knowledge-bases/search`
- `GET /api/knowledge-bases/embedding-models`

## 4.3 文档管理

对应前端页面：

- `DocumentsView.vue`
- `DocumentDetailView.vue`

能力：

- 上传文档到指定知识库。
- 查看全局文档列表。
- 查看文档详情和解析状态。
- 手动重新解析文档。
- 手动将文档分段写入向量索引。
- 编辑文档名称和切分配置。
- 查看、编辑、删除文档分段。
- 删除文档。

支持文件类型：

- `pdf`
- `ppt` / `pptx`
- `xls` / `xlsx`
- `png` / `jpg` / `jpeg`
- `doc` / `docx`
- `md`
- `txt`

后端接口：

- `POST /api/documents/upload`
- `GET /api/documents`
- `GET /api/documents/:id`
- `PUT /api/documents/:id`
- `DELETE /api/documents/:id`
- `POST /api/documents/:id/parse`
- `POST /api/documents/:id/index`
- `GET /api/documents/:id/segments`
- `PUT /api/documents/:id/segments/:segmentId`
- `DELETE /api/documents/:id/segments/:segmentId`

## 4.4 文档解析

当前支持两类解析能力：

### 原生解析

适合：

- PDF 文本提取。
- Excel 表格解析。
- Markdown / TXT 文本解析。

### MinerU 解析

适合：

- PDF、Office、图片等复杂文档。
- 图片分析、表格解析、公式解析。

系统设置支持以下解析配置：

- 解析引擎：`auto` / `mineru` / `native`
- PDF 原生回退：开启 / 关闭
- MinerU 服务地址
- MinerU 超时时间
- MinerU 解析方法：`auto` / `txt` / `ocr`
- MinerU 解析精度：`low` / `medium` / `high`
- MinerU 语言列表
- 图片分析
- 表格解析
- 公式解析

检测能力：

- 文档解析设置页提供“检测解析配置”。
- 原生解析模式只校验配置合法性。
- 自动或强制 MinerU 模式会检测 MinerU 服务可访问性。

后端接口：

- `POST /api/settings/test-parse`

## 4.5 文档存储

系统支持两种上传存储方式：

### 本地 uploads

- 默认路径：`/app/uploads`
- Docker 环境映射到宿主机项目目录：`./uploads`

### RustFS / S3 兼容存储

配置项：

- Endpoint
- Access Key
- Secret Key
- Bucket
- Region
- 是否使用 SSL

检测能力：

- 系统设置页提供“检测 RustFS”。
- 检测包含连接、bucket 检查/创建、写入、读取、删除。

后端接口：

- `POST /api/settings/test-rustfs`

运行时策略：

- 上传时严格按照当前系统设置选择本地或 RustFS。
- 下载和删除历史文件时兼容旧存储位置。

## 4.6 知识库问答

对应前端页面：`KnowledgeAskView.vue`

能力：

- 选择知识库。
- 输入问题。
- 配置召回数量、使用数量、重排模式。
- 查看回答结果。
- 查看引用来源、召回分数和原始距离。

重排模式：

- `similarity`：按向量相似度排序。
- `keyword`：按问题关键词命中排序。
- `length`：按内容长度排序。

后端流程：

1. 根据问题调用知识库搜索。
2. 从 Milvus 召回片段。
3. 根据重排模式排序。
4. 截取使用片段。
5. 调用 DashScope 生成答案。
6. 返回答案、置信度、引用来源。

后端接口：

- `POST /api/knowledge-ask`

## 4.7 问答生成与管理

对应前端页面：

- `QAGenerateView.vue`
- `QAManageView.vue`
- `QASettingsView.vue`

能力：

- 按知识库生成问答预览。
- 查看生成任务进度。
- 保存生成结果。
- 手动新增问答。
- 编辑问答。
- 启用 / 禁用问答。
- 删除问答。
- 批量删除问答。
- 根据指定文档和问题生成答案。

后端接口：

- `POST /api/qa/generate-preview`
- `GET /api/qa/generate-tasks/:id`
- `POST /api/qa/save-generated`
- `POST /api/qa/generate-answer`
- `GET /api/qa`
- `POST /api/qa`
- `GET /api/qa/:id`
- `PUT /api/qa/:id`
- `PATCH /api/qa/:id/status`
- `DELETE /api/qa/:id`
- `DELETE /api/qa/batch`

## 4.8 系统设置

对应前端页面：`SettingsView.vue`

当前页面采用选项卡布局：

1. 基础设置
2. 上传位置
3. RustFS 配置
4. 文档解析

基础设置包含：

- 最大文件大小
- 允许上传类型
- 默认问答数量
- 问答生成批次大小
- 模型接口地址

后端接口：

- `GET /api/settings`
- `PUT /api/settings`
- `POST /api/settings/test-rustfs`
- `POST /api/settings/test-parse`

## 5. 非功能需求

### 5.1 可部署性

- 使用 Docker Compose 一键启动完整依赖。
- 包含 MySQL、RustFS、Milvus、Milvus Bridge、后端和前端服务。

### 5.2 可配置性

- 配置来自 YAML、环境变量和数据库系统设置。
- 环境变量优先覆盖 YAML。
- 系统设置用于运行时调整上传位置、RustFS、解析等参数。

### 5.3 安全性

- `.env` 不提交。
- DashScope API Key 不写入 YAML 和源码。
- 检测接口只接收当前模块相关字段，后端使用白名单过滤。

### 5.4 可维护性

- 前端按 API、类型、布局、视图、组件拆分。
- 后端按 config、db、handlers、middleware、migrations、models、routes、services 拆分。
- 业务逻辑集中在 services 层。

## 6. 当前已知边界

1. 文档解析是同步接口，复杂大文件解析可能耗时较长。
2. Office 文档原生解析能力有限，推荐使用 MinerU。
3. 系统设置修改不会自动迁移已上传文件。
4. 删除或切换存储位置时，历史文件通过兼容逻辑尽量读取和删除。
5. Milvus 不可用时，知识库检索和问答能力不可用，但其他基础功能仍可运行。
