# tilego

一个高性能的地图瓦片批量下载工具，支持多种配置选项和断点续传功能。

## 功能特性

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

# 使用代理服务器
./tile-downloader -url "http://tile.example.com/{z}/{x}/{y}.png" \
    -min-lon 116.0 -min-lat 39.0 -max-lon 117.0 -max-lat 40.0 \
    -min-zoom 10 -max-zoom 12 -proxy "http://127.0.0.1:7890"

# 限制下载速率
./tile-downloader -url "http://tile.example.com/{z}/{x}/{y}.png" \
    -min-lon 116.0 -min-lat 39.0 -max-lon 117.0 -max-lat 40.0 \
    -min-zoom 10 -max-zoom 12 -rate 5

# 启用 MD5 校验
./tile-downloader -url "http://tile.example.com/{z}/{x}/{y}.png" \
    -min-lon 116.0 -min-lat 39.0 -max-lon 117.0 -max-lat 40.0 \
    -min-zoom 10 -max-zoom 12 -md5
```

## 参数说明

| 参数名        | 描述                                            | 默认值                                                       | 必填 |
| ------------- | ----------------------------------------------- | ------------------------------------------------------------ | ---- |
| `-url`        | 瓦片 URL 模板 (支持 {z}, {x}, {y}, {-y} 占位符) | -                                                            | 是   |
| `-min-lon`    | 最小经度                                        | -                                                            | 是   |
| `-min-lat`    | 最小纬度                                        | -                                                            | 是   |
| `-max-lon`    | 最大经度                                        | -                                                            | 是   |
| `-max-lat`    | 最大纬度                                        | -                                                            | 是   |
| `-min-zoom`   | 最小缩放级别 (0-18)                             | 0                                                            | 否   |
| `-max-zoom`   | 最大缩放级别 (0-18)                             | 18                                                           | 否   |
| `-dir`        | 保存目录                                        | ./tiles                                                      | 否   |
| `-format`     | 保存格式 (zxy/xyz/z/x/y)                        | zxy                                                          | 否   |
| `-threads`    | 并发线程数                                      | 10                                                           | 否   |
| `-timeout`    | 超时时间(秒)                                    | 60                                                           | 否   |
| `-retries`    | 重试次数                                        | 5                                                            | 否   |
| `-proxy`      | 代理 URL (例如: <http://127.0.0.1:7890>)        | -                                                            | 否   |
| `-user-agent` | User-Agent                                      | Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 | 否   |
| `-skip`       | 跳过已存在的文件                                | true                                                         | 否   |
| `-md5`        | 计算文件 MD5                                    | false                                                        | 否   |
| `-min-size`   | 最小文件大小(字节)                              | 100                                                          | 否   |
| `-max-size`   | 最大文件大小(字节)                              | 2097152                                                      | 否   |
| `-rate`       | 速率限制(请求/秒)                               | 10                                                           | 否   |
| `-http2`      | 启用 HTTP/2                                     | true                                                         | 否   |
| `-keep-alive` | 启用 Keep-Alive                                 | true                                                         | 否   |
| `-batch`      | 批处理大小                                      | 1000                                                         | 否   |
| `-buffer`     | 下载缓冲区大小                                  | 8192                                                         | 否   |
| `-resume`     | 断点续传文件名                                  | .tilego-resume.json                                          | 否   |
| `-resume`     | Referer 请求头                                  | -                                                            | 否   |

## 项目结构

```text
tilego/
├── cmd/
│   └── tile-downloader/
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
5. 代理连接测试失败不会中断程序，但会记录警告信息
6. 程序会自动跳过已下载的文件（除非设置 `-skip=false`）
7. 下载过程中会显示实时进度和统计信息

## 输出说明

程序运行时会输出以下信息：

- **初始化阶段**：显示代理配置、速率限制等设置信息
- **下载阶段**：每 10 秒显示一次进度，包括：
  - 已处理瓦片数/总瓦片数（百分比）
  - 下载速度（KB/s 和 瓦片/秒）
  - 活跃线程数
  - 成功、失败、跳过的瓦片数量
- **完成阶段**：显示最终统计信息，包括：
  - 总耗时
  - 总瓦片数
  - 成功下载、跳过、失败的瓦片数
  - 重试次数
  - 下载总大小
  - 平均下载速度

## 错误处理

程序会对错误进行分类和统计，常见的错误类型包括：

- **网络错误**：超时、连接失败、DNS 解析失败等
- **客户端错误**：HTTP 4xx 错误（404、403、401、400）
- **服务器错误**：HTTP 5xx 错误
- **文件错误**：文件大小超出限制、格式验证失败等

所有错误都会在程序结束时显示统计信息，帮助用户了解下载过程中遇到的问题。
