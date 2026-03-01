# test1 - Komari-like 服务器状态探针（Go + React）

你说“代码太少”，这版我加了更完整的能力：

## 功能
- 多目标探测（HTTP / TCP）
- 每个目标可配置 interval/timeout
- 5 秒刷新实时状态面板
- 24h 可用率统计
- SQLite 持久化探测历史
- 节点趋势火花图（Sparkline）
- 节点历史详情表
- Docker Compose 一键部署

## 项目结构
- `backend/`：Go API + 探测调度 + SQLite
- `frontend/`：React 实时看板
- `docker-compose.yml`：部署编排

## 启动
```bash
cd test1
docker compose up -d --build
```
访问：<http://localhost:8090>

## 后端 API
- `GET /health`
- `GET /api/status`：当前状态列表
- `GET /api/history?name=xxx`：单节点历史记录
- `GET /api/targets`：目标配置

## 配置目标
编辑 `docker-compose.yml` 的 `TARGETS_JSON`，示例：
```json
[
  {"name":"Google","type":"http","addr":"https://www.google.com","intervalSec":15,"timeoutSec":6},
  {"name":"DNS","type":"tcp","addr":"1.1.1.1:53","intervalSec":15,"timeoutSec":4}
]
```

## 说明
- 持久化数据库：`./data/probe.db`
- 若后续你要，我可以继续加：告警（Telegram/Webhook）、登录管理、节点分组、地图视图、Prometheus 指标。
