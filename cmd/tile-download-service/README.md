# tile-download-service

瓦片下载 HTTP 服务提供了 RESTful API 接口，可以作为瓦片下载器的后端服务使用。

## 启动服务

```bash
# 编译服务
go build -o tile-download-service ./cmd/tile-download-service

# 运行服务（默认端口 8080）
./tile-download-service
```

服务启动后将在 `http://localhost:8080` 提供服务。

## API 接口文档

### 1. 健康检查

**GET** `/api/health`

检查服务是否正常运行。

**响应示例：**

```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "time": "2024-01-15T10:30:00Z"
  }
}
```

### 2. 创建下载任务

**POST** `/api/download`

创建并启动一个新的瓦片下载任务。

**请求体：**

```json
{
  "id": "task_001",
  "url_template": "https://server.example.com/{z}/{x}/{y}.png",
  "min_lon": 116.0,
  "min_lat": 39.0,
  "max_lon": 117.0,
  "max_lat": 40.0,
  "min_zoom": 10,
  "max_zoom": 15,
  "save_dir": "./tiles",
  "format": "zxy",
  "threads": 10,
  "timeout": 60,
  "retries": 5,
  "proxy_url": "",
  "user_agent": "Mozilla/5.0...",
  "referer": "",
  "skip_existing": true,
  "check_md5": false,
  "min_file_size": 100,
  "max_file_size": 2097152,
  "rate_limit": 10,
  "use_http2": true,
  "keep_alive": true,
  "batch_size": 1000,
  "buffer_size": 8192
}
```

**参数说明：**

| 参数名        | 类型   | 必填 | 默认值   | 描述              |
| ------------- | ------ | ---- | -------- | ----------------- |
| id            | string | 否   | 自动生成 | 任务 ID           |
| url_template  | string | 是   | -        | 瓦片 URL 模板     |
| min_lon       | float  | 是   | -        | 最小经度          |
| min_lat       | float  | 是   | -        | 最小纬度          |
| max_lon       | float  | 是   | -        | 最大经度          |
| max_lat       | float  | 是   | -        | 最大纬度          |
| min_zoom      | int    | 否   | 0        | 最小缩放级别      |
| max_zoom      | int    | 否   | 18       | 最大缩放级别      |
| save_dir      | string | 否   | ./tiles  | 保存目录          |
| format        | string | 否   | zxy      | 保存格式          |
| threads       | int    | 否   | 10       | 并发线程数        |
| timeout       | int    | 否   | 60       | 超时时间(秒)      |
| retries       | int    | 否   | 5        | 重试次数          |
| proxy_url     | string | 否   | -        | 代理 URL          |
| user_agent    | string | 否   | 默认UA   | User-Agent        |
| referer       | string | 否   | -        | Referer           |
| skip_existing | bool   | 否   | true     | 跳过已存在文件    |
| check_md5     | bool   | 否   | false    | MD5 校验          |
| min_file_size | int    | 否   | 100      | 最小文件大小      |
| max_file_size | int    | 否   | 2097152  | 最大文件大小      |
| rate_limit    | int    | 否   | 10       | 速率限制(请求/秒) |
| use_http2     | bool   | 否   | true     | 启用 HTTP/2       |
| keep_alive    | bool   | 否   | true     | 启用 Keep-Alive   |
| batch_size    | int    | 否   | 1000     | 批处理大小        |
| buffer_size   | int    | 否   | 8192     | 缓冲区大小        |

**响应示例：**

```json
{
  "success": true,
  "message": "Download task created",
  "data": {
    "task_id": "task_001"
  }
}
```

### 3. 查询任务状态

**GET** `/api/status/{task_id}`

获取指定任务的详细状态信息。

**响应示例：**

```json
{
  "success": true,
  "data": {
    "id": "task_001",
    "status": "running",
    "progress": 45.5,
    "total": 10000,
    "success": 4500,
    "failed": 50,
    "skipped": 0,
    "bytes_total": 52428800,
    "speed": 512.5,
    "start_time": "2024-01-15T10:30:00Z"
  }
}
```

**任务状态说明：**

| 状态     | 描述                 |
| -------- | -------------------- |
| pending  | 任务已创建，等待执行 |
| running  | 任务正在执行         |
| stopped  | 任务已停止           |
| complete | 任务已完成           |
| failed   | 任务失败             |

### 4. 停止任务

**POST** `/api/stop/{task_id}`

停止正在运行的下载任务。

**响应示例：**

```json
{
  "success": true,
  "message": "Task stopped"
}
```

### 5. 获取任务列表

**GET** `/api/tasks`

获取所有任务的列表。

**响应示例：**

```json
{
  "success": true,
  "data": [
    {
      "id": "task_001",
      "status": "complete",
      "progress": 100,
      "total": 10000,
      "success": 9950,
      "failed": 50,
      "skipped": 0,
      "bytes_total": 104857600,
      "speed": 1024.0,
      "start_time": "2024-01-15T10:30:00Z",
      "end_time": "2024-01-15T10:45:00Z"
    },
    {
      "id": "task_002",
      "status": "running",
      "progress": 25.0,
      "total": 5000,
      "success": 1250,
      "failed": 0,
      "skipped": 0,
      "bytes_total": 13107200,
      "speed": 256.0,
      "start_time": "2024-01-15T11:00:00Z"
    }
  ]
}
```

### 6. 删除任务

**DELETE** `/api/delete/{task_id}`

删除指定的任务记录。如果任务正在运行，会先停止任务。

**响应示例：**

```json
{
  "success": true,
  "message": "Task deleted"
}
```

## 使用示例

### cURL 示例

```bash
# 创建下载任务
curl -X POST http://localhost:8080/api/download \
  -H "Content-Type: application/json" \
  -d '{
    "url_template": "https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}",
    "min_lon": 116.0,
    "min_lat": 39.0,
    "max_lon": 117.0,
    "max_lat": 40.0,
    "min_zoom": 10,
    "max_zoom": 12
  }'

# 查询任务状态
curl http://localhost:8080/api/status/task_001

# 停止任务
curl -X POST http://localhost:8080/api/stop/task_001

# 获取任务列表
curl http://localhost:8080/api/tasks

# 删除任务
curl -X DELETE http://localhost:8080/api/delete/task_001
```

### JavaScript 示例

```javascript
// 创建下载任务
async function createTask() {
  const response = await fetch("http://localhost:8080/api/download", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      url_template: "https://server.example.com/{z}/{x}/{y}.png",
      min_lon: 116.0,
      min_lat: 39.0,
      max_lon: 117.0,
      max_lat: 40.0,
      min_zoom: 10,
      max_zoom: 12,
    }),
  });
  return await response.json();
}

// 查询任务状态
async function getTaskStatus(taskId) {
  const response = await fetch(`http://localhost:8080/api/status/${taskId}`);
  return await response.json();
}
```
