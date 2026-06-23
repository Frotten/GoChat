# GoAI

基于 **Go + Gin** 的智能对话后端，配套 **Vue 3** 前端。支持多模型（OpenAI 兼容 / Ollama / 火山方舟）、会话管理、RAG 知识库检索，以及通过 RabbitMQ 异步落库消息。支持简单工具调用（如文件搜索、读取、写入等），可扩展自定义工具。

## 技术栈

| 层级 | 技术 |
|------|------|
| HTTP | [Gin](https://github.com/gin-gonic/gin) |
| ORM | GORM + MySQL |
| 缓存 | Redis（验证码等） |
| 消息队列 | RabbitMQ（聊天消息异步写入 MySQL） |
| AI 编排 | [CloudWeGo Eino](https://github.com/cloudwego/eino) |
| 向量检索 | Qdrant + Ollama Embedding |
| 鉴权 | JWT（`Authorization: Bearer <token>`） |
| 前端 | Vue 3 + Element Plus + Axios |
| API 文档 | [go-swagger](https://github.com/go-swagger/go-swagger) |

## 系统架构

```
┌─────────────┐     HTTP/SSE      ┌──────────────────────────────────────────┐
│ vue-frontend│ ────────────────► │ Gin Router (/api/v1)                      │
│  :8080      │                   │  ├─ /user/*        注册/登录/验证码       │
└─────────────┘                   │  └─ /AI/chat/*     JWT 鉴权后的对话接口     │
                                  └───────────┬──────────────────────────────┘
                                              │
                    ┌─────────────────────────┼─────────────────────────┐
                    ▼                         ▼                         ▼
             controller              service/session              middleware/jwt
                    │                         │
                    ▼                         ▼
              dao (user/session/message)   aihelper.Manager
                    │                         │
         ┌──────────┴──────────┐    ┌─────────┴─────────┐
         ▼                     ▼    ▼                   ▼
      MySQL              RabbitMQ  Eino Model      RAG (Qdrant)
   用户/会话/消息           异步持久化   + Tools        Info/ 知识库
```

## 核心链路

### 1. 用户注册 / 登录

1. `POST /api/v1/user/captcha`：向邮箱发送验证码（Redis 存储）。
2. `POST /api/v1/user/register`：校验验证码后创建用户，返回 JWT。
3. `POST /api/v1/user/login`：校验密码，返回 JWT。

### 2. AI 对话（同步）

1. 前端携带 `Authorization: Bearer <token>` 调用 AI 接口。
2. JWT 中间件解析出 `userName`，进入 `controller/session`。
3. `service/session` 通过 `aihelper.Manager` 按「用户 + sessionId」获取或创建 `AIHelper`。
4. `AIHelper.GenerateResponse`：
   - 从内存加载会话历史；
   - 可选：RAG 从 Qdrant 检索 `Info/` 下已索引文档片段；
   - 通过 Eino Workflow 先归一化消息，再调用 ReAct Agent；
   - Agent 继续支持 `search_files` / `read_file` 等工具。
5. 用户消息与 AI 回复经 RabbitMQ 异步写入 MySQL；启动时 `main.readDataFromDB` 会把历史消息灌回内存。

### 3. 流式对话（SSE）

- `POST /api/v1/AI/chat/send-stream-new-session`：先创建会话，首条 SSE 返回 `sessionId`，再流式输出模型内容。
- `POST /api/v1/AI/chat/send-stream`：在已有会话中流式回复。

### 4. RAG 知识库

启动时若 `Env.env` 中配置了 Ollama Embedding 模型与 Qdrant，`rag.Service` 会扫描 `PROJECT_ROOT`（默认 `./Info`）下的 `.txt` / `.md` / `.go`，经 Ollama `/api/embed` 向量化后写入 Qdrant。对话时按用户问题检索 Top-K 片段注入上下文。

## 目录结构

```
GoAI/
├── main.go                 # 入口：加载配置、初始化 DB/RAG/Redis/MQ、启动 HTTP
├── config/                 # config.toml + 配置读取
├── router/                 # 路由注册
├── controller/             # HTTP 入参/出参
├── service/                # 业务逻辑
├── dao/                    # 数据访问
├── model/                  # GORM 模型
├── middleware/jwt/         # JWT 鉴权
├── common/
│   ├── aihelper/           # Eino 模型封装、会话内存、工具调用
│   ├── rag/                # 向量化 + Qdrant
│   ├── mysql/ redis/ rabbitmq/ email/
├── vue-frontend/           # Vue 3 前端
├── Info/                   # RAG 知识库示例文档
├── Env.env                 # 环境变量（模型、RAG、密钥）
└── config/config.toml      # 服务、MySQL、Redis、RabbitMQ、JWT 等
```

## 快速开始

### 环境要求

- Go 1.24+
- MySQL、Redis、RabbitMQ（地址见 `config/config.toml`）
- 可选：Qdrant、Ollama（RAG 向量化；对话模型仍可通过 OpenAI 兼容 API / Ollama / 方舟配置）
- Node.js（前端）

### 后端

```bash
# 1. 编辑项目根目录 Env.env（模型密钥、RAG、Qdrant 等，勿提交真实密钥）

# 2. 修改 config/config.toml 中的 MySQL / Redis / RabbitMQ 地址

# 3. 启动
go run .
```

默认监听：`http://0.0.0.0:9090`（见 `config.toml` 的 `mainConfig`）。

### 前端

```bash
cd vue-frontend
npm install
npm run serve
```

开发地址：`http://localhost:8080`（代理/API 基址见 `vue-frontend/.env`）。


### 安装工具

```bash
go install github.com/go-swagger/go-swagger/cmd/swagger@latest
```

### 接口概览

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | `/api/v1/user/captcha` | 发送验证码 | 否 |
| POST | `/api/v1/user/register` | 注册 | 否 |
| POST | `/api/v1/user/login` | 登录 | 否 |
| GET | `/api/v1/AI/chat/sessions` | 会话列表 | JWT |
| POST | `/api/v1/AI/chat/send-new-session` | 新建会话并对话 | JWT |
| POST | `/api/v1/AI/chat/send` | 继续对话 | JWT |
| POST | `/api/v1/AI/chat/history` | 历史记录 | JWT |
| POST | `/api/v1/AI/chat/delete-session` | 删除会话 | JWT |
| POST | `/api/v1/AI/chat/send-stream-new-session` | 新建会话 + SSE | JWT |
| POST | `/api/v1/AI/chat/send-stream` | 流式对话 SSE | JWT |

统一响应体包含 `status_code`（`1000` 表示成功）与 `status_msg`，详见 [`common/code/code.go`](common/code/code.go)。

## 环境变量（Env.env）

| 变量 | 说明 |
|------|------|
| `OPENAI_API_KEY` / `OPENAI_BASE_URL` / `OPENAI_MODEL` | OpenAI 兼容对话模型 |
| `OPENAI_TYPE` | `openai` / `ollama` / `ark` 等 |
| `OLLAMA_BASE_URL` / `OLLAMA_MODEL` | 本地 Ollama |
| `OLLAMA_EMBEDDING_MODEL` | Ollama Embedding 模型名（如 `bge-m3`） |
| `EMBEDDING_BASE_URL` | Ollama 地址，默认同 `OLLAMA_BASE_URL` 或 `http://localhost:11434` |
| `EMBEDDING_DIM` | 向量维度（须与模型输出及 Qdrant collection 一致） |
| `PROJECT_ROOT` | RAG 文档根目录，默认 `./Info` |
| `QDRANT_HTTP_URL` / `QDRANT_COLLECTION` | Qdrant 地址与集合名 |
| `RAG_TOP_K` / `RAG_CHUNK_SIZE` / `RAG_CHUNK_OVERLAP` | 检索与分块参数 |
| `CAPTCHA_DEV_MODE` | 开发模式可跳过部分验证码校验 |

## 状态码说明

业务状态码定义于 [`common/code/code.go`](common/code/code.go)，例如：

- `1000`：成功
- `2001`：参数错误
- `2006`：Token 无效
- `4001`：服务繁忙
- `5001`–`5003`：AI 模型相关错误

HTTP 状态码多为 `200`，请以响应 JSON 中的 `status_code` 为准。
