# tile-downloader

地图瓦片批量下载命令行工具，支持多种配置选项和断点续传功能。

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

| 参数名        | 描述                                                           | 默认值                                                       | 必填 |
| ------------- | -------------------------------------------------------------- | ------------------------------------------------------------ | ---- |
| `-url`        | 瓦片 URL 模板 (支持 {z}, {x}, {y}, {-y} 占位符)                | -                                                            | 是   |
| `-min-lon`    | 最小经度                                                       | -                                                            | 是   |
| `-min-lat`    | 最小纬度                                                       | -                                                            | 是   |
| `-max-lon`    | 最大经度                                                       | -                                                            | 是   |
| `-max-lat`    | 最大纬度                                                       | -                                                            | 是   |
| `-min-zoom`   | 最小缩放级别 (0-18)                                            | 0                                                            | 否   |
| `-max-zoom`   | 最大缩放级别 (0-18)                                            | 18                                                           | 否   |
| `-dir`        | 保存目录                                                       | ./tiles                                                      | 否   |
| `-format`     | 保存格式 (zxy/xyz/z/x/y)                                       | zxy                                                          | 否   |
| `-threads`    | 并发线程数                                                     | 10                                                           | 否   |
| `-timeout`    | 超时时间(秒)                                                   | 60                                                           | 否   |
| `-retries`    | 重试次数                                                       | 5                                                            | 否   |
| `-proxy`      | 代理 URL (例如:http://127.0.0.1:7890)                          | -                                                            | 否   |
| `-user-agent` | User-Agent                                                     | Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 | 否   |
| `-referer`    | Referer                                                        | -                                                            | 否   |
| `-skip`       | 跳过已存在的文件                                               | true                                                         | 否   |
| `-md5`        | 计算文件 MD5                                                   | false                                                        | 否   |
| `-min-size`   | 最小文件大小(字节)                                             | 100                                                          | 否   |
| `-max-size`   | 最大文件大小(字节)                                             | 2097152                                                      | 否   |
| `-rate`       | 速率限制(请求/秒)                                              | 10                                                           | 否   |
| `-http2`      | 启用 HTTP/2                                                    | true                                                         | 否   |
| `-keep-alive` | 启用 Keep-Alive                                                | true                                                         | 否   |
| `-batch`      | 批处理大小                                                     | 1000                                                         | 否   |
| `-buffer`     | 下载缓冲区大小                                                 | 8192                                                         | 否   |
| `-resume`     | 断点续传文件名                                                 | .tilego-resume.json                                          | 否   |
