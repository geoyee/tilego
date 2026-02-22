# tilego

地图瓦片批量下载工具集，支持多种配置选项和断点续传功能。

## 项目结构

```text
tilego/
├── cmd/
│   ├── tile-downloader/        # 命令行下载工具
│   └── tile-download-service/  # HTTP 下载服务
├── internal/
│   ├── client/                 # HTTP 客户端
│   ├── calculator/             # 瓦片坐标计算
│   ├── download/               # 下载核心逻辑
│   ├── model/                  # 数据模型
│   ├── resume/                 # 断点续传
│   ├── stats/                  # 统计监控
│   └── util/                   # 工具函数
├── go.mod
├── go.sum
└── README.md
```

## 组件说明

### [tile-downloader](./cmd/tile-downloader/README.md)

命令行瓦片下载工具，适合批量下载地图瓦片。

**主要特性：**

- 支持多种瓦片 URL 模板格式
- 高效并发下载
- 断点续传
- 代理服务器支持
- 实时进度监控

**快速开始：**

```bash
go build -o tile-downloader ./cmd/tile-downloader

./tile-downloader -url "http://tile.example.com/{z}/{x}/{y}.png" \
    -min-lon 116.0 -min-lat 39.0 -max-lon 117.0 -max-lat 40.0 \
    -min-zoom 10 -max-zoom 15
```

👉 [查看完整文档](./cmd/tile-downloader/README.md)

### [tile-download-service](./cmd/tile-download-service/README.md)

HTTP 瓦片下载服务，提供 RESTful API 接口，可作为后端服务使用。

**主要特性：**

- RESTful API 接口
- 异步任务管理
- 实时进度查询
- 多任务并发

**快速开始：**

```bash
go build -o tile-download-service ./cmd/tile-download-service
./tile-download-service
```

**API 接口：**

| 接口               | 方法   | 描述         |
| ------------------ | ------ | ------------ |
| `/api/health`      | GET    | 健康检查     |
| `/api/download`    | POST   | 创建下载任务 |
| `/api/status/{id}` | GET    | 查询任务状态 |
| `/api/stop/{id}`   | POST   | 停止任务     |
| `/api/tasks`       | GET    | 获取任务列表 |
| `/api/delete/{id}` | DELETE | 删除任务     |

👉 [查看完整文档](./cmd/tile-download-service/README.md)

## 安装

```bash
# 克隆项目
git clone https://github.com/geoyee/tilego.git
cd tilego

# 安装依赖
go mod tidy
```

## 共同特性

- 支持多种瓦片 URL 模板格式（{z}, {x}, {y}, {-y} 占位符）
- 高效并发下载，可配置线程数
- 智能速率限制功能
- 断点续传，支持暂停和恢复
- HTTP/2 协议支持
- 代理服务器支持
- 文件格式验证（PNG、JPG、WEBP）
- 实时进度监控和统计
- 详细的错误分类和统计
- 自动重试机制（指数退避）
- 文件大小验证
- MD5 校验支持

## 注意事项

1. 请确保您有权限下载和使用目标地图服务的瓦片数据
2. 遵守目标服务器的使用条款和限制
3. 合理设置并发数和速率限制，避免对服务器造成过大压力
4. 对于大量瓦片的下载，建议分批进行
