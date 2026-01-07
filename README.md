# tilego

一个高性能的地图瓦片批量下载工具，支持多种配置选项和断点续传功能。

## 功能特性

- 支持多种瓦片 URL 模板格式
- 并发下载，可配置线程数
- 速率限制功能
- 断点续传，支持暂停和恢复
- HTTP/2 支持
- 代理支持
- 文件格式验证
- 实时进度监控
- 详细的错误统计

## 安装

```bash
# 克隆项目
git clone https://github.com/geoyee/tilego.git
cd tilego

# 安装依赖
go mod tidy

# 编译
go build -o tile-downloader ./cmd/tile-downloader
```

## 使用示例

```bash
# 下载指定范围的瓦片
./tile-downloader -url "http://tile.example.com/{z}/{x}/{y}.png" \
    -min-lon 116.0 -min-lat 39.0 -max-lon 117.0 -max-lat 40.0 \
    -min-zoom 10 -max-zoom 15 -dir ./tiles -threads 20

# 使用代理
./tile-downloader -url "http://tile.example.com/{z}/{x}/{y}.png" \
    -min-lon 116.0 -min-lat 39.0 -max-lon 117.0 -max-lat 40.0 \
    -min-zoom 10 -max-zoom 12 -proxy "http://127.0.0.1:7890"

# 限制下载速率
./tile-downloader -url "http://tile.example.com/{z}/{x}/{y}.png" \
    -min-lon 116.0 -min-lat 39.0 -max-lon 117.0 -max-lat 40.0 \
    -min-zoom 10 -max-zoom 12 -rate 5
```

## 参数说明

- `-url`: 瓦片 URL 模板 (必填，支持 {z}, {x}, {y}, {-y} 占位符)
- `-min-lon`: 最小经度
- `-min-lat`: 最小纬度
- `-max-lon`: 最大经度
- `-max-lat`: 最大纬度
- `-min-zoom`: 最小缩放级别 (0-18)
- `-max-zoom`: 最大缩放级别 (0-18)
- `-dir`: 保存目录 (默认: ./tiles)
- `-format`: 保存格式 (zxy/xyz/z/x/y，默认: zxy)
- `-threads`: 并发线程数 (默认: 20)
- `-timeout`: 超时时间(秒) (默认: 60)
- `-retries`: 重试次数 (默认: 5)
- `-proxy`: 代理 URL (例如: http://127.0.0.1:7890)
- `-user-agent`: User-Agent
- `-skip`: 跳过已存在的文件 (默认: true)
- `-md5`: 计算文件 MD5 (默认: false)
- `-min-size`: 最小文件大小(字节) (默认: 100)
- `-max-size`: 最大文件大小(字节) (默认: 2097152)
- `-rate`: 速率限制(请求/秒) (默认: 10)
- `-http2`: 启用 HTTP/2 (默认: true)
- `-keep-alive`: 启用 Keep-Alive (默认: true)

## 项目结构

```text
tiledownloader/
├── cmd/
│   └── tiledownloader/
│       └── main.go           # 程序入口
├── internal/
│   ├── client/               # HTTP客户端相关功能
│   │   └── http_client.go
│   ├── calculator/           # 瓦片坐标计算
│   │   ├── errors.go
│   │   └── tile_calculator.go
│   ├── download/             # 下载核心逻辑
│   │   ├── downloader.go
│   │   └── worker_pool.go
│   ├── model/                # 数据模型定义
│   │   └── types.go
│   ├── resume/               # 断点续传功能
│   │   └── resume_manager.go
│   ├── stats/                # 统计监控功能
│   │   └── monitor.go
│   └── util/                 # 工具函数
│       ├── error_utils.go
│       └── file_utils.go
├── go.mod
├── go.sum
└── README.md
```

## 注意事项

1. 请确保您有权限下载和使用目标地图服务的瓦片数据
2. 遵守目标服务器的使用条款和限制
3. 合理设置并发数和速率限制，避免对服务器造成过大压力
4. 对于大量瓦片的下载，建议分批进行
