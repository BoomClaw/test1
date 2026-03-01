# test1 - Komari-like Server Probe (Go + React)

功能：
- 定时探测 HTTP/TCP
- 实时状态面板（5s 刷新）
- Docker Compose 部署

## 启动
```bash
cd test1
docker compose up -d --build
```
打开：http://localhost:8090

## 配置目标
在 `docker-compose.yml` 的 `TARGETS_JSON` 修改目标列表。
