# shortener

一个基于 Go 和 go-zero 的短链接服务，提供长链缩短与短链跳转能力，适合用于内容分发、活动链接管理、系统内链统一等场景。

## 目录

- [项目介绍](#项目介绍)
- [核心技术栈](#核心技术栈)
- [核心亮点](#核心亮点)
- [快速启动](#快速启动)
- [项目结构](#项目结构)

## 项目介绍

`shortener` 目前提供两个核心接口：

- `POST /v1/shorturl/shorten`：将长链接转换为短链接（需要 JWT）
- `GET /v1/shorturl/:short_url`：通过短码查询并重定向到原始长链接（302）

核心处理流程：

1. 参数校验与 URL 连通性检查
2. 对长链接做 MD5，优先查询是否已有映射
3. 未命中时用自增序列 + Base62 生成短码
4. 过滤敏感词短码，写入映射表并加入布隆过滤器
5. 查询短链时先过布隆过滤器，再查库返回长链

对应代码：

- 服务入口：`shortener.go`
- 路由注册：`internal/handler/routes.go`
- 生成逻辑：`internal/logic/shortenLogic.go`
- 解析逻辑：`internal/logic/showLogic.go`

## 核心技术栈

- 语言与框架
  - Go `1.24.1`
  - `github.com/zeromicro/go-zero`（REST、配置、日志、缓存）
- 存储与缓存
  - MySQL（`short_url_map`、`sequence`）
  - Redis（序列缓存、布隆过滤器）
- 核心组件
  - `validator/v10`：参数校验
  - `godotenv`：环境变量加载（`.env` + `.env.<APP_ENV>`）
  - 自定义 `errorx`：统一错误码、HTTP 映射、错误包装
  - `pkg/base62`、`pkg/md5`、`pkg/sensitive`、`pkg/urlTool`

## 核心亮点

- 去重友好：同一长链通过 MD5 映射避免重复入库与重复生成。
- 性能优化：短链查询先走布隆过滤器，减少无效数据库访问。
- 合规控制：短码生成过程集成敏感词过滤，避免不合规短码输出。
- 分层清晰：`handler -> logic -> repository/model/pkg`，职责明确，便于维护与测试。
- 配置灵活：配置文件通过环境变量注入，支持不同环境快速切换。

## 快速启动

### 1) 准备依赖

- Go（建议与 `go.mod` 一致）
- MySQL
- Redis

### 2) 初始化数据库

在 MySQL 中执行以下脚本：

```sql
source ddl/sequence.sql;
source ddl/shortUrlMap.sql;
```

### 3) 配置环境变量

项目会先加载根目录 `.env`，再根据 `APP_ENV` 加载 `.env.<APP_ENV>`（例如 `.env.dev`）。

配置模板位于：`etc/shortener-api.yaml`

至少需要确认以下变量已设置：

- 应用：`APP_ENV`、`APP_PORT`、`OPERATOR`、`SHORT_URL_DOMAIN`、`SHORT_URL_PATH`
- ShortUrlMap MySQL：`SHORT_URL_MAP_DB_USER`、`SHORT_URL_MAP_DB_PASSWORD`、`SHORT_URL_MAP_DB_HOST`、`SHORT_URL_MAP_DB_PORT`、
  `SHORT_URL_MAP_DB_NAME`
- Sequence MySQL：`SEQUENCE_DB_USER`、`SEQUENCE_DB_PASSWORD`、`SEQUENCE_DB_HOST`、`SEQUENCE_DB_PORT`、`SEQUENCE_DB_NAME`
- Sequence Redis：`SEQUENCE_REDIS_HOST`、`SEQUENCE_REDIS_PORT`、`SEQUENCE_REDIS_PASSWORD`、`SEQUENCE_REDIS_TYPE`
- Filter Redis：`SHORT_URL_FILTER_REDIS_HOST`、`SHORT_URL_FILTER_REDIS_PORT`、`SHORT_URL_FILTER_REDIS_PASSWORD`、
  `SHORT_URL_FILTER_REDIS_TYPE`
- Cache Redis：`CACHE_REDIS_HOST`、`CACHE_REDIS_PORT`、`CACHE_REDIS_PASSWORD`
- 鉴权：`ACCESS_SECRET`

### 4) 启动服务

```bash
go run shortener.go -f etc/shortener-api.yaml
```

### 5) 调用示例

创建短链（需要 JWT）：

```bash
curl -X POST "http://127.0.0.1:${APP_PORT}/v1/shorturl/shorten" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-jwt-token>" \
  -d '{"long_url":"https://example.com/path/to/page"}'
```

访问短链（302 重定向）：

```bash
curl -i "http://127.0.0.1:${APP_PORT}/v1/shorturl/<short_code>"
```

### 6) 运行测试

```bash
go test ./...
```

## 项目结构

```text
shortener/
├── shortener.go                 # 程序入口
├── shortUrl.api                 # API 声明（goctl）
├── etc/
│   └── shortener-api.yaml       # 服务配置（通过环境变量注入）
├── ddl/
│   ├── sequence.sql             # 序列表 DDL
│   └── shortUrlMap.sql          # 长短链映射表 DDL
├── assets/                      # 敏感词及替换规则词典
├── internal/
│   ├── config/                  # 配置定义与环境变量加载
│   ├── handler/                 # HTTP 处理与统一响应
│   ├── logic/                   # 核心业务逻辑
│   ├── model/                   # 数据模型
│   ├── repository/              # 数据访问层（DB/缓存/序列）
│   ├── svc/                     # ServiceContext 依赖组装
│   └── types/                   # API 请求/响应结构
└── pkg/
    ├── base62/                  # Base62 编码
    ├── errorx/                  # 错误体系
    ├── sensitive/               # 敏感词过滤
    ├── urlTool/                 # URL 工具与连通性检查
    └── validate/                # 参数校验规则
```