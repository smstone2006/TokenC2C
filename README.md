<div align="center">
  <img src="assets/logo.svg" alt="TokenC2C Logo" width="88" />
  <h1>TokenC2C</h1>
  <p>共享账户，连接需求。</p>
</div>

TokenC2C 是用户共建的 AI 账户共享平台。用户自行添加支持的官方账户，经验证后进入共享池；使用者通过自己的 API Key 调用池中资源。买卖双方分别查看请求数、Token 用量、消费、收益及账户状态。

## 已实现功能

- 供应账户自行添加、验证、暂停、恢复与删除。
- 共享池调度及每日贡献上限。
- 买方计费与卖方收益在同一数据库事务中记账，防止重复扣款。
- 供应者获得计费金额的 90% 作为站内额度，平台服务费为 10%；收益待结算 24 小时。
- 双边统计、可用/待结算钱包和流水账本。
- TokenC2C 品牌、矢量 Logo、中英文响应式首页。

当前版本不包含现金提现、退款争议处理或固定收益承诺。

## 开发与部署

技术栈：Go、Vue 3、TypeScript、PostgreSQL、Redis、Docker Compose。

完整说明与测试命令见 [开发及部署文档](docs/C2C_MVP.md)。

```sh
cd frontend
pnpm install --frozen-lockfile
pnpm test:run
pnpm build
```

Mac 使用 `deploy/docker-compose.c2c.yml`；NAS 使用 `deploy/docker-compose.c2c.nas.yml` 和 `deploy/deploy-c2c-nas.sh`。为保护现有数据挂载及网络，已有容器、数据库和目录的内部 `subc2c` 标识保持兼容。环境配置从 `deploy/.env.c2c.example` 创建，实际密钥文件不进入 Git。

## 版本库与上游

本项目保留 Sub2API 的历史，TokenC2C 开发在本地分支中提交。`upstream` 用于追踪原项目；目前未创建或发布远程 TokenC2C 仓库。

TokenC2C 基于 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) 扩展。原文档保留于 [README_UPSTREAM.md](README_UPSTREAM.md)，原许可证与作者声明保留于 [LICENSE](LICENSE)。Go 模块路径保留上游路径以兼容现有源码引用。
