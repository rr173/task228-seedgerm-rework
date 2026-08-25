基于 Go 实现的农业种子萌发时序证据 Web 项目，一款后端服务，从图像时序检测胚根与叶鞘阶段事件、关联温湿度干预与停滞并发布不可变试验结果版本。

# 评测说明 (BENZHI) — task228-seedgerm

## 构建与运行

评测使用 `benzhi.Dockerfile` 与 `build_benzhi_docker.sh`：

```bash
bash build_benzhi_docker.sh seedgerm linux/amd64
bash build_benzhi_docker.sh seedgerm linux/arm64
docker run --rm seedgerm --smoke-test
```

## --smoke-test 契约

容器启动命令固定为 `--smoke-test`，不传任何路径参数。服务在 smoke 模式下：

1. 在临时 SQLite 数据库创建试验、种子、图像摘要、环境采样、阶段事件、人工观察、结果版本；
2. 关闭并重新打开同一数据库，验证上述数据持久化与重启恢复；
3. 以退出码 0 结束（任意断言失败则非 0）。

## 双架构证明

由 `scripts/docker_baseline_validation.py` 在 canonical `main` 健康基线提交后一次性生成
`.private/docker_baseline_validation.json`（status=passed），覆盖 `linux/amd64` 与 `linux/arm64`。

## API 形态

所有 API 以 `/api` 前缀暴露，详见 `README.md`。
