# 雪球组合持仓分布查看

从 cookies.txt 与组合列表拉取雪球组合当前持仓分布，支持聚合查看和按组合筛选。

## 快速开始

```bash
# 构建
task build

# 启动 HTTP 服务（默认 :8080）
./xq server
```

## Task 任务

### build

构建项目为 Linux AMD64 二进制文件。

```bash
task build
```

### deploy

部署到远程服务器（默认主机 `beer`，路径 `/opt/xq`）。

```bash
# 使用默认配置部署
task deploy

# 自定义主机、二进制路径、远程路径
task deploy HOST=myserver BINARY=bin/xq REMOTE=/opt/xq
```

**流程**
1. `ssh <host> systemctl stop xq` - 停止服务
2. `scp <binary> <host>:<remote>` - 上传二进制
3. `ssh <host> systemctl restart xq.service` - 重启服务
4. `ssh <host> systemctl status xq.service` - 检查状态

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
- `GET /api/config` - 提醒配置（只读）
- `POST /api/notify/run` - 手动触发一次提醒

**页面**
- 根路径 `/` - 聚合持仓，支持按组合筛选

## 提醒配置

提醒规则通过配置文件设置：`$HOME/.xq_config.json`

| 配置项 | 说明 |
|--------|------|
| enabled | 启用定时提醒（true/false） |
| interval_minutes | 检查间隔（分钟），如 30 表示每 30 分钟检查一次 |
| weight_threshold | 持仓比例变化阈值（%），如 5 表示变化超过 5% 才发送提醒；设为 0 时无变化也发 |
| feishu_app_id | 飞书应用的 App ID |
| feishu_app_secret | 飞书应用的 App Secret |
| feishu_receive_id | 接收消息的 ID（用户 open_id/user_id/union_id 或群 chat_id），主动推送时使用 |
| feishu_receive_type | 接收者类型：open_id、user_id、union_id、chat_id，默认 open_id |

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
    "interval_minutes": 30,
    "weight_threshold": 5,
    "feishu_app_id": "cli_xxxxxxxxxxxxx",
    "feishu_app_secret": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "feishu_receive_id": "ou_xxxxxxxxxxxxx",
    "feishu_receive_type": "open_id"
  }
}
```

### 飞书应用配置

1. 在飞书开放平台创建应用（https://open.feishu.cn/app）
2. 获取 App ID 和 App Secret
3. 在「权限管理」中申请并开通权限：
   - `im:message` (发消息)
   - `im:message:send_as_bot` (以机器人身份发送)
4. 在「事件订阅」中配置订阅事件（如需长连接交互）
5. 配置接收者（用于主动推送）：
   - 个人消息：填写用户的 open_id / user_id / union_id
   - 群消息：填写群的 chat_id，receive_type 设为：chat_id

## 日志

- 输出到 `runtime.log`（当前工作目录）和 stderr
- 追加写入，重启不覆盖

## 依赖

### Get cookies.txt LOCALLY 插件

https://github.com/kairi003/Get-cookies.txt-LOCALLY

导出 cookies.txt 后放入项目目录或指定路径。
