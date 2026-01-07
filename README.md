# tilego

高性能地图瓦片下载器

> 注意：实际运行时，根据网络情况和服务器限制，适当调整线程数、速率限制等参数，以避免被封 IP。另外，程序还支持断点续传，默认会生成一个.resume.json 文件记录下载进度。如果下载中断，再次运行相同的命令（不强制重新下载）会跳过已下载的瓦片。如果希望重新下载，可以使用--force=true 参数。

## 编译

```shell
go mod tidy
go build -o tile-downloader cmd/tile-downloader/main.go
```

## 使用

### 参数说明

```text
# 基本参数
--url="..."                   # 图源URL模板，使用{x} {y} {z}占位符
--min-lon=...                 # 最小经度
--max-lon=...                 # 最大经度
--min-lat=...                 # 最小纬度
--max-lat=...                 # 最大纬度
--min-zoom=...                # 最小层级
--max-zoom=...                # 最大层级
--save-dir="..."              # 保存目录

# 性能参数
--threads=50                  # 并发线程数（默认50）
--batch-size=1000             # 批量大小（默认1000）
--buffer-size=65536           # 缓冲区大小（默认64KB）
--timeout=30                  # 超时时间（秒）
--retries=3                   # 重试次数

# 网络参数
--proxy="..."                 # 代理URL
--http2=true                  # 启用HTTP/2（默认true）
--keep-alive=true             # 保持长连接（默认true）
--rate-limit=0                # 速率限制（0表示无限制）

# 文件处理
--format="zxy"                # 保存格式（zxy, xyz, z/x/y）
--type="auto"                 # 文件类型（自动检测）
--skip-existing=true          # 跳过已存在的文件
--force=false                 # 强制重新下载

# 校验和断点
--check-md5=false             # 启用MD5校验
--min-size=100                # 最小文件大小（字节）
--max-size=2097152            # 最大文件大小（默认2MB）
--resume-file=".resume.json"  # 断点续传文件
```

### 常用图源

- OSM 图源：http://tile.openstreetmap.org/{z}/{x}/{y}.png
- Google 无偏卫星图：http://mt0.google.com/vt/lyrs=s&hl=en&x={x}&y={y}&z={z}

### 示例

```shell
./tile-downloader \
  --url="http://tile.openstreetmap.org/{z}/{x}/{y}.png" \
  --min-lon=116.0 \
  --min-lat=39.0 \
  --max-lon=117.0 \
  --max-lat=40.0 \
  --min-zoom=10 \
  --max-zoom=15 \
  --save-dir="./tiles_with_proxy" \
  --proxy="http://127.0.0.1:7890" \
  --threads=50 \         # 设置线程数量
  --rate-limit=50 \      # 限制请求速率
  --batch-size=500 \     # 减小批量大小
  --timeout=60 \         # 增加超时时间
  --retries=5 \          # 增加重试次数
  --rate-limit=50
```
