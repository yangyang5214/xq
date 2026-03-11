# 雪球组合持仓分布对比与提醒

从 cookies.txt 与组合列表拉取雪球组合当前持仓分布，与上次快照对比；比例变化超过阈值时通过飞书群机器人发送提醒。

## 快速开始

```bash
# 构建
go build -o xq ./cmd/xq

# 启动 HTTP 服务（默认 :8080）
./xq server
```

## 子命令

### server

启动 HTTP 服务，提供组合列表与持仓 API，以及内置的定时提醒。

```bash
./xq server [flags]
```

| 参数 | 说明 | 默认 |
|------|------|------|
| `-f, --cubes-file` | 组合列表文件路径，每行一个 symbol，支持 # 注释 | cubes.txt |
| `--cookies-file` | Get cookies.txt LOCALLY 导出的 cookies.txt 路径 | cookies.txt |
| `-a, --addr` | 监听地址 | :8080 |

**API**
- `GET /api/cubes` - 组合列表
- `GET /api/cubes/{symbol}` - 指定组合持仓
- `GET /api/config` - 提醒配置
- `PUT /api/config` - 保存提醒配置
- `POST /api/notify/run` - 手动触发一次提醒

**页面**
- 根路径 `/` - 聚合持仓、按组合查看
- `#config` - 提醒配置（启用、间隔、阈值、收件人）

## 提醒配置

在页面「提醒配置」中可设置：

| 配置项 | 说明 |
|--------|------|
| 启用定时提醒 | 勾选后按间隔定时检查 |
| 检查间隔（分钟） | 每隔多少分钟检查一次 |
| 持仓比例变化阈值 | 超过此阈值才发送提醒；设为 0 时无变化也发 |
| 飞书 Webhook URL | 飞书群机器人的 webhook 地址 |

**规则**
- 仅在**交易日 9:00-15:00**（北京时间）执行检查
- 持仓比例变化超过阈值时会触发提醒（包括新增、调整、调出）
- 快照保存在 `$HOME/.xq_snapshots/`

## 配置文件

### 提醒配置 `$HOME/.xq_config.json`

可通过环境变量 `XQ_CONFIG` 指定路径，否则默认 `$HOME/.xq_config.json`。

格式示例：
```json
{
  "notify": {
    "enabled": true,
    "feishu_webhook": "https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
    "weight_threshold": 5,
    "interval_minutes": 30
  }
}
```

### 飞书群机器人配置

1. 在飞书群中添加「自定义机器人」
2. 获取 webhook URL
3. 将 URL 填入配置的 `feishu_webhook` 字段

## 日志

- 输出到 `runtime.log`（当前工作目录）和 stderr
- 追加写入，重启不覆盖

## 依赖

### Get cookies.txt LOCALLY 插件

https://github.com/kairi003/Get-cookies.txt-LOCALLY

导出 cookies.txt 后放入项目目录或指定路径。
