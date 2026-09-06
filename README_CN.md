<div align="center">
  <img src="assets/logo.svg" alt="TokenC2C 标志" width="100" />
  <h1>TokenC2C</h1>
  <p><strong>共享账户，连接需求，每笔都有账可查。</strong></p>
  <p>可自行部署的用户共建 AI 账户共享池，为买方和供应者分别计量、记账。</p>
  <p><a href="README.md">English</a> · <a href="README_CN.md"><strong>简体中文</strong></a> · <a href="README_JA.md">日本語</a></p>
  <img src="https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white" alt="Go 1.27" />
  <img src="https://img.shields.io/badge/Vue-3-4FC08D?logo=vue.js&logoColor=white" alt="Vue 3" />
  <img src="https://img.shields.io/badge/PostgreSQL-18-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL 18" />
  <img src="https://img.shields.io/badge/Redis-8-DC382D?logo=redis&logoColor=white" alt="Redis 8" />
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-LGPL--3.0-blue" alt="许可证：LGPL-3.0" /></a>
</div>

## 项目简介

TokenC2C 基于 [Sub2API](https://github.com/Wei-Shaw/sub2api)，增加用户自有账户的供应机制。供应者自行添加支持的上游账户，验证通过后参与共享调度；使用者通过自己的平台 API Key 调用 AI 服务。系统同时记录买方用量和供应者收益。

同一用户可以同时消费和供给。这里的 **Token** 指 AI 用量单位及平台使用额度，不是加密货币。

## 核心功能

- **自助供给：** 添加、验证、暂停、恢复和删除自己的供应账户。
- **验证后入池：** 验证通过前，账户不会参与调度。
- **共享调度：** 复用 Sub2API 的鉴权、路由、限流及用量计费能力。
- **供给控制：** 可设置每日贡献上限，查看每个账户的共享池状态。
- **双边计量：** C2C API 分别提供请求数、Token 总量、买方消费和供应者收益；控制台提供钱包、账户和流水视图。
- **原子结算：** 买方扣费、供应者收益、平台费用在同一数据库事务中记录。
- **防止重复记账：** 请求/API Key 标识和账本幂等键避免重复结算。
- **响应式界面：** 原创 TokenC2C 品牌和中英文首页；项目文档提供英文、中文、日文。

### 与上游的关系

| 领域 | 继承的 Sub2API 基础 | TokenC2C 扩展 |
| --- | --- | --- |
| 账户管理 | 管理员管理上游账户 | 用户自有账户、自助验证与供给控制 |
| 请求路由 | 共享调度器与 API 网关 | 通过验证的用户账户参与调度 |
| 计费结算 | 消费用量与余额扣费 | 供应者收益、平台费用及待结算额度 |
| 数据可见性 | 用量记录和管理面板 | 买卖双方独立汇总与结算流水 |

## 工作流程

```text
供应者账户 -> 上游验证 -> 共享账户池
使用者 API Key -> 网关与调度器 -> 共享池 -> 实际计费调用

实际计费调用
  |-- 买方扣费
  |-- 供应者待结算额度 -> 24 小时后转入可用额度
  `-- 平台费用
```

### 当前结算规则

| 项目 | 规则 |
| --- | --- |
| 供应者分成 | 买方计费金额的 90%，以站内额度记入 |
| 平台服务费 | 买方计费金额的 10% |
| 待结算周期 | 24 小时后转入供应者可用额度 |
| 使用自己的供给 | 支持，仍适用同一套记账和平台费用规则 |
| 现金提现 | 尚未实现 |

以上是当前 MVP 代码中的规则，不构成固定收益承诺。收益来自实际被调度并成功计费的调用。自动退款、争议处理与现金付款不属于当前版本的功能。

## 支持的账户来源

C2C 表单支持 OpenAI、Anthropic、Gemini、Antigravity、Grok、Kimi、智谱/GLM 和 DeepSeek。不同平台允许的凭证类型不同，以添加账户表单中的选项为准；实际模型访问能力取决于账户和运营配置。

贡献流程只接受支持的官方上游，拒绝用户自定义 Base URL 和通用中转账户类型。买方使用自己的平台 API Key；账户列表和统计接口不会返回供应者的上游凭证。

## Docker Compose 快速开始

准备 Git、Docker 和 Compose v2，并确保可以访问构建依赖及镜像仓库。随附配置使用 PostgreSQL 18 和 Redis 8。

```sh
git clone https://github.com/smstone2006/TokenC2C.git
cd TokenC2C
cp deploy/.env.c2c.example deploy/.env.c2c
```

编辑 `deploy/.env.c2c`：设置管理员邮箱和密码，使用独立随机值替换**所有**占位符。`JWT_SECRET` 和 `TOTP_ENCRYPTION_KEY` 分别使用一次 `openssl rand -hex 32` 生成。妥善保管实际环境配置文件。

```sh
docker compose --env-file deploy/.env.c2c -f deploy/docker-compose.c2c.yml up -d --build

# 等待一次性设置初始化完成，输出的退出码必须为 0。
docker wait subc2c-bootstrap

# 初始化完成后，重新加载品牌与注册设置。
docker compose --env-file deploy/.env.c2c -f deploy/docker-compose.c2c.yml restart app
docker compose --env-file deploy/.env.c2c -f deploy/docker-compose.c2c.yml up -d --wait --wait-timeout 180 app
curl -fsS http://127.0.0.1:18080/health
```

访问 [http://127.0.0.1:18080](http://127.0.0.1:18080)，使用配置的管理员账户登录。初始化默认启用邀请码注册，并为初始部署关闭邮箱验证。邀请成员前，请在管理员界面创建邀请码。

默认仅监听本机回环地址。需要远程访问时，请配置自己的 HTTPS 反向代理和访问策略。PostgreSQL、Redis 不发布宿主机端口。长期运行的服务采用 `unless-stopped`，一次性初始化容器不自动重启。

### NAS 部署

QNAP 模板位于 [`deploy/docker-compose.c2c.nas.yml`](deploy/docker-compose.c2c.nas.yml)，[`deploy/deploy-c2c-nas.sh`](deploy/deploy-c2c-nas.sh) 负责启动及挂接 qnet。该模板依赖具体环境：使用前需调整 Docker 路径、数据目录、qnet 名称，并选择未占用的局域网 IP。先按目标架构构建并加载模板使用的镜像 `subc2c:nas`，再部署。详见[部署说明](docs/C2C_MVP.md)。

容器、数据卷和数据库内部保留 `subc2c` 标识，以兼容已有安装；对外产品名称为 TokenC2C。

## 开发与测试

| 组件 | 技术 |
| --- | --- |
| 后端 | Go 1.27、Gin、Ent |
| 前端 | Vue 3、TypeScript、Vite、Tailwind CSS |
| 数据库 / 缓存 | PostgreSQL / Redis |
| 测试 | Go tests、Testcontainers、Vitest、Vue Test Utils |

```sh
cd frontend
pnpm install --frozen-lockfile  # 使用 pnpm 9
pnpm typecheck
pnpm test:run
pnpm build

cd ../backend
go test -tags=unit ./internal/service ./internal/repository ./migrations -run 'TestC2C|TestApplyC2CTrade' -count=1
CI=1 go test -v -tags=integration ./internal/repository -run '^TestC2CBilling' -count=1
```

集成测试需要 Docker，会启动隔离的 PostgreSQL/Redis 测试容器，覆盖买方扣费、卖方待结算收益、费用拆分、重复请求、到期结算及自用场景。前端测试覆盖不同角色入口、注册设置、语言切换和既有首页模式。设置 `CI=1` 后，Docker 不可用将明确报错，而不是跳过集成测试。

## C2C API

以下路径均以 `/api/v1/c2c` 为前缀，需要用户登录。供应账户的变更操作限定为账户所有者。

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET / POST | `/provider-accounts` | 列出 / 添加自己的账户 |
| POST | `/provider-accounts/:id/verify` | 验证账户 |
| PUT | `/provider-accounts/:id/pool-state` | 暂停或恢复供给 |
| DELETE | `/provider-accounts/:id` | 删除账户 |
| GET | `/wallet` | 可用和待结算额度 |
| GET | `/stats` | 买方与供应者汇总 |
| GET | `/ledger?limit=50` | 最近账本记录 |

## 项目结构

```text
backend/
  internal/handler/c2c_handler.go          # 面向用户的 C2C HTTP 接口
  internal/service/c2c_service.go          # 归属、验证与到期结算
  internal/repository/c2c_billing.go       # 事务结算
  migrations/234_c2c_marketplace.sql       # 账户归属与账本结构
frontend/src/
  components/home/                        # TokenC2C 首页及文案
  views/user/C2CView.vue                   # 供应、钱包与账本面板
  api/c2c.ts                              # 类型化 API 客户端
deploy/                                   # Docker 与 NAS 模板
docs/C2C_MVP.md                            # 实现与部署说明
```

## 部署边界

当前版本面向受信任的邀请码制 MVP 场景，复用上游的凭证存储；贡献账户凭证的额外应用层静态加密尚未实现。扩大运营范围前，应完善适合自身部署的凭证保护、滥用控制、争议和退款流程。仅贡献你有权使用、且服务商条款允许共享的账户。

不要提交真实环境配置、Token、私钥或数据库备份。向用户介绍服务时，应明确上述平台费用和额度规则。

## 参与贡献

欢迎通过 [Issues](https://github.com/smstone2006/TokenC2C/issues) 提交可复现问题和具体建议，或提交包含相关测试的 Pull Request。请勿在问题或补丁中附带真实上游凭证及用户个人数据。

## 上游与许可证

TokenC2C 基于 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) 开发，保留 Git 历史和许可证声明。Go 模块路径保留原值以兼容源码引用，原英文 README 存于 [README_UPSTREAM.md](README_UPSTREAM.md)。

仓库采用 [GNU LGPL v3](LICENSE)，第三方组件遵循各自许可证。感谢 Sub2API 作者和贡献者提供网关基础。
