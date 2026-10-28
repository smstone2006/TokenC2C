# TokenC2C Security Scan 修复与验证（2026-10-29）

日期：2026-10-29（Asia/Shanghai；验证跨 10 月 28 日与 29 日）。
基线：`06a6731e9774a09da7eaf9183336273691de6568`，对应 Security Scan 运行 `40290714588`。

后续更新：用户已批准替换 xlsx，本轮已完成官方 SheetJS 0.20.3 替换及回归。详细依据、命令和结果见文末 xlsx 章节；上一轮验证表保留替换前的结果，供追溯。

## 结论与范围

- 原有 15 条 Go 可达漏洞、13 条前端生产依赖未豁免漏洞均已消除。
- Go 构建、后端单元/集成测试、前端构建及全量测试通过；golangci-lint 为 0 issues。
- **本轮前端安全门禁已通过**：xlsx 改用官方修复版，两条过期豁免已删除；本地生产 audit 为 high=0、critical=0，例外校验通过。上一轮的 xlsx 阻塞已解除，但未提交或触发远程 GitHub Actions，不能据此声称远程运行已变绿。
- 未 commit、未 push、未部署；全部改动保留在工作区，暂存区为空。
- 本次没有改变业务规则。少量生产文件只增加兼容说明，或显式忽略原本忽略的只读查询清理返回值；详见下文。

## 版本与文件

下表为最终解析版本，不只是 package.json 中的范围。

| 项目 | 原版本 | 修复后 |
| --- | --- | --- |
| Go | 1.27.0 | 1.27.3 |
| golang.org/x/net | 0.56.0 | 0.62.0 |
| google.golang.org/grpc | 1.82.1 | 1.83.2 |
| golang.org/x/tools | 0.47.0 | 0.52.0 |
| golangci-lint（CI） | v2.13 | v2.15.0 |
| axios | 1.18.1 | 1.21.0 |
| dompurify | 3.4.14 | 3.4.17 |
| follow-redirects | 1.16.0 | 1.16.2 |
| source-map-js | 1.2.1 | 1.2.2 |
| Vue / @vue/server-renderer | 3.5.26 | 3.5.44 |
| xlsx | 0.18.5（npm） | 0.20.3（官方 CDN 固定 tarball，详见本轮补充） |

未停在原定的 x/net 0.60.0、grpc 1.83.1：实际扫描这两个版本仍报可达的 `GO-2026-6860`、`GO-2026-6443`，另有导入包级别的 `GO-2026-6866`；扫描给出的修复版本分别为 x/net 0.62.0、grpc 1.83.2。

依赖及工具配置改动：

- `backend/go.mod`、`backend/go.sum`：上述版本及必要依赖联动。x/crypto 0.53.0 → 0.58.0，x/mod 0.37.0 → 0.42.0，x/sync 0.21.0 → 0.24.0，x/term 0.44.0 → 0.47.0，x/sys 0.46.0 → 0.49.0，x/text 0.39.0 → 0.43.0；OpenTelemetry 1.43.0 → 1.44.0，otelhttp 0.49.0 → 0.61.0；genproto/rpc 联动，tidy 移除未使用的 google/subcommands。
- `.github/workflows/backend-ci.yml`、`release.yml`、`security-scan.yml`：四处硬编码版本检查统一为 `go1.27.3`；backend-ci 的 lint 固定到 v2.15.0。
- `Dockerfile`、`backend/Dockerfile`、`deploy/Dockerfile`：Go 构建镜像统一为 `golang:1.27.3-alpine`。
- `DEV_GUIDE.md`：同步 Go、lint 版本及安装说明。
- `frontend/package.json`、`frontend/pnpm-lock.yaml`：更新目标依赖，增加 follow-redirects/source-map-js 最低安全版本 override，更新 DOMPurify override。锁文件由 pnpm 9.15.9 生成；没有做整个依赖树的无差别更新。
- Vue 运行时、SFC/SSR 编译器、server-renderer 已对齐 3.5.44；vue-tsc 的开发依赖链仍保留其要求的旧 compiler-core/compiler-dom 副本，不是旧版运行时或 server-renderer。

### 兼容性处理：生产文件

- `backend/internal/service/c2c_service.go`：两处只读查询的 `defer rows.Close()` 改为显式忽略关闭返回值；仍返回原来的扫描错误和 `rows.Err()`，不改变业务行为。
- `backend/internal/repository/http_upstream.go`、`backend/internal/server/http.go`、`backend/internal/service/openai_codex_models_service.go`：仅在现有 x/net HTTP/2 API 使用处增加带原因的局部 `nolint:staticcheck` 注释。x/net 升级后这些 API 被标为弃用；本次保留已测试的 PING、H2C 流控与 typed GOAWAY 重试行为。迁移到纯标准库 HTTP/2 是后续独立工作，不在安全升级中混入行为改写。没有全局关闭 staticcheck 或安全检查。

Go 1.27.3 的导出数据版本为 V5，旧 x/tools 导致 Ent schema loader 失败（`version 5 greater than maximum supported version 4`）。升级 x/tools 0.52.0 后该测试通过；lint 同步使用内含新版工具链依赖的 v2.15.0。

### 测试文件修复

- `backend/internal/server/api_contract_test.go`：默认站名期望改为已有的 `TokenC2C`，不更改服务返回值。
- `backend/internal/repository/c2c_billing_integration_test.go`：为两个提交真实事务的 C2C 用例增加限定 account ID 的清理，按 ledger → trade → provider 的外键顺序执行，避免后续用例因残留引用清理失败。清理错误会使测试失败，不改生产删除逻辑。
- `frontend/src/components/layout/__tests__/docUrlSanitization.spec.ts`、`siteLogoSanitization.spec.ts`：源码断言容忍单双引号和换行，保留 URL 清理安全要求。
- `backend/internal/repository/http_upstream_http2_keepalive_test.go`、`backend/internal/service/openai_codex_models_service_test.go`：局部说明保留的旧 HTTP/2 API 回归覆盖。
- `backend/internal/service/openai_ws_client_read_test.go`、`openai_ws_passthrough_turn_pricing_test.go`、`openai_ws_v2_passthrough_lifecycle_test.go`：新版 errcheck 检查 error channel 接收；仅在已有“等待退出、不关心终止错误”的接收点增加解释性注释，不删除断言、不改变测试流程。

对于 errcheck 与 staticcheck S1005 的冲突，核对官方文档后比较了：断言终止错误、显式丢弃接收值、逐行说明忽略、回退工具版本四种方案。终止错误不属于这些同步检查的断言目标；直接 `_ = <-channel` 会触发 S1005；回退工具存在 Go 导出格式兼容问题。因此采用具体接收点的说明，不增加全局豁免。

## 上一轮验证命令与结果（xlsx 替换前）

后端命令在 `backend/`，前端命令在 `frontend/`；例外检查在仓库根目录。
本机是 macOS/arm64；前端执行环境为 Node 26.8.1、pnpm 10.30.3。另用 CI 同系列 pnpm 9.15.9 生成并验证 frozen lockfile、执行生产 audit，但没有宣称已在 GitHub 的 Node 20/Linux runner 完整执行 CI。

| 命令 | 结果 |
| --- | --- |
| `go version`；`go version \| grep -q 'go1.27.3'` | Go 1.27.3；版本硬校验通过，四处 workflow 字符串已核对 |
| `go mod verify` | all modules verified |
| `go build ./...` | 通过 |
| `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$SEC_DIR/tokenc2c-linux-amd64" ./cmd/server` | Linux/amd64 交叉构建通过 |
| `govulncheck ./...`（x/vuln v1.9.0） | 退出 0；可达漏洞 0、导入包漏洞 0 |
| `GOOS=linux GOARCH=amd64 govulncheck -show verbose ./...` | 退出 0；同样为 0/0；原 15 个 ID 均未出现 |
| `go test -json -tags=unit ./... -count=1` | 退出 0；17,987 个测试/子测试 pass，16 skip，无 fail |
| `CI=1 go test -json -tags=integration ./... -count=1` | 退出 0；11,038 个测试/子测试 pass，14 skip，无 fail；repository 的 PostgreSQL/Redis Testcontainers 实际运行 |
| `golangci-lint run --timeout=30m --max-same-issues=0 --max-issues-per-linter=0`（v2.15.0） | 退出 0，0 issues；未使用自动修复 |
| `pnpm install --frozen-lockfile` | 通过 |
| `npx --yes pnpm@9.15.9 install --lockfile-only --frozen-lockfile --ignore-scripts --modules-dir "$SEC_DIR/pnpm9-modules"` | 通过；pnpm 9 frozen lock 兼容性验证 |
| `pnpm build` | 类型检查与 Vite 构建通过；仍有现有大 chunk 提示 |
| `pnpm test:run` | 253 个文件、1,833 项测试全部通过 |
| `npx --yes pnpm@9.15.9 audit --prod --audit-level=high --json` | 退出 1；仅剩 xlsx 两条 high，critical 0；原 13 条目标告警逐 ID 比对均消失 |
| `python3 tools/check_pnpm_audit_exceptions.py --audit "$SEC_DIR/audit-after.json" --exceptions .github/audit-exceptions.yml` | 退出 1；仅报告 xlsx 两条豁免过期，没有用改日期来掩盖结果 |
| `pnpm audit --json`（额外检查，包含 devDependencies） | 退出 1；34 high、3 critical、36 moderate、4 low，包含上述 xlsx 告警；Vite/Vitest 等开发工具链另需专项处理 |
| `git diff --check` | 通过 |
| `git diff --exit-code -- .github/audit-exceptions.yml` | 通过；例外文件未改，xlsx 的声明、锁定版本与包记录亦未改 |

测试计数包含 Go 子测试，不能简单当作相互独立的场景数；unit 与 integration 两次运行也存在重叠。跳过项来自已有条件：外部 OpenAI/TLS capture/插件配置、单独的 prompt-audit Redis/PostgreSQL 环境变量未配置，以及原有 sentinel/TODO。没有新增 skip，也没有把这些项算作通过。

govulncheck 另报告 **7 条未调用的模块级告警**（x/image、compress、openpgp 等）。这不是 7 条可达漏洞，但也不应描述为“整个依赖树零漏洞”。生产前端 audit 另有 13 moderate、2 low，不在原 high/critical 门禁目标内；含开发依赖的全量 audit 也没有清零。本次未扩展到整个构建工具链的主版本升级。

### 逐项漏洞核对

后端以下 15 个 ID 在最终 Linux verbose 扫描中均不再出现：

```text
GO-2026-6617 GO-2026-6612 GO-2026-6611 GO-2026-6610 GO-2026-6603
GO-2026-6809 GO-2026-6613 GO-2026-6609 GO-2026-6605 GO-2026-6608
GO-2026-6607 GO-2026-6600 GO-2026-6599 GO-2026-6443 GO-2026-6348
```

前端原 13 个未豁免 ID 在最终生产 audit 中均不再出现：

| 包 | 已消除的 advisory |
| --- | --- |
| axios（7 high） | GHSA-c29m-xwm3-cm6r、GHSA-mghh-pgcx-3jjj、GHSA-x97p-jq2g-jp4f、GHSA-3pq3-5fj3-cg6v、GHSA-542g-h47m-68v8、GHSA-m8m8-qj5v-23w3、GHSA-r4gj-5m52-g5wh |
| dompurify（1 critical、1 high） | GHSA-cg2g-xh64-v4q7、GHSA-mrv2-3pqr-j5jr |
| follow-redirects（2 high） | GHSA-q3jh-wrc6-jgq8、GHSA-8r9p-f939-6h3c |
| source-map-js（1 high） | GHSA-68fv-2mgg-jv7q |
| @vue/server-renderer（1 high） | GHSA-g2v6-rqmx-r4w6 |

本机原始日志目录：`/tmp/tokenc2c-security-20261028.QqsVFV`（即以上命令的 `SEC_DIR`，临时目录不保证永久保存）。主要文件：`govulncheck-linux.log`、`go-unit-final.jsonl`、`go-integration-final.jsonl`、`golangci-lint-final.log`、`frontend-build.log`、`frontend-tests-final.log`、`audit-before.json`、`audit-after.json`、`audit-all.json`、`audit-exceptions.log`。

## xlsx：初始评估与后续修复

以下调用面、风险与成本评估保留上一轮结论；用户随后批准使用官方修复版，执行结果记录于本节后半部分。

### 实际调用面

唯一生产入口是 `frontend/src/views/admin/UsageView.vue:575` 的 `exportToExcel`：管理员点击导出时动态加载 xlsx，把管理员 usage API 返回的数据按每页 100 行取回，输出 32 列的 `.xlsx` 文件，带进度及取消功能。

只使用 `aoa_to_sheet`、`sheet_add_aoa`、`book_new`、`book_append_sheet`、`write`，再由 Blob/file-saver 下载。全仓相关调用检索没有发现 `XLSX.read/readFile` 或上传工作簿解析入口。前端路由有 `requiresAdmin`，后端 `/api/v1/admin/usage` 也在管理员认证中间件内。

权限和动态加载减少了暴露面，但不能把所有导出内容视为可信：API key 名称、user-agent、模型等字段可来自用户或上游。全部分页结果最终聚合到浏览器工作簿，分页不等于限制总导出量；大文件内存压力也需单独评估。

现有 `UsageView.spec.ts` mock 了 xlsx，主要验证列及下载调用，不能证明真实生成文件可正确打开。因此无论升级还是换库，都应补真实文件生成/解析验证、中文与特殊字符、金额精度、字符串类型、大量行、取消和进度测试。

### 两条漏洞不能混为一谈

- `GHSA-4r6h-8v6p-xvw6`：官方 advisory 明确触发条件是读取特制工作簿，并说明纯导出工作流不受影响。结合现有调用面，该触发路径目前不可达；如果新增导入功能，必须重新评估。
- `GHSA-5pgg-2g8v-p4x9`：advisory 指出 0.20.2 之前存在 ReDoS，但没有给出同等明确的“全部导出路径不受影响”结论。本次没有完成底层正则的逐路径漏洞复现，不能保证零风险；管理员导出和不解析外部工作簿仅降低暴露面。
- npm registry 的 xlsx 仍是 0.18.5；直接 `pnpm update xlsx` 不能解决这两项。官方发布渠道另有已修复版本，当前安装文档展示 0.20.3 的 tarball。

### 方案与改动成本（基于当前代码的估计）

| 方案 | 预计改动量 | 风险与取舍 |
| --- | --- | --- |
| 短期续豁免 | 最小：例外文件中的两条记录，补真实责任人、理由及短期截止日；若增加导出上限或防导入回归，则另改代码/测试 | 只是接受风险，不是修复。原型污染有明确不可达依据；ReDoS 证据仍不足。不建议无期限或反复续期 |
| 换用官方已修复 SheetJS tarball/本地 vendor（至少 0.20.2，文档示例 0.20.3） | 低：package/lock 两个文件、可选 vendor 包、真实导出测试；现有 API 理论上可保留，需验证 | 最接近现有行为；需核验来源、完整性、许可证、离线可重复安装及 audit 对非 registry 依赖的识别，不能以扫描器“不识别”为通过依据 |
| 改用 ExcelJS 等 npm 管理的库 | 中：一个约 55 行导出函数、mock/真实导出测试、package/lock，约 4–6 个文件；一般不需后端改动 | 保留 XLSX 功能，但需重验 32 列顺序、数值/字符串类型、六位金额、取消/进度、中文、浏览器内存与产物体积；不能预设新库更小或自动无漏洞 |
| 删除 XLSX 导出，或改为 CSV | 代码改动小至中：导出 UI/函数、测试、依赖，通常 3–5 个文件 | 删除会损失功能；CSV 不是完全等价替换，需处理 BOM、引号/换行、单元格类型及公式注入。均需要用户明确接受功能变化 |

建议：若优先减少行为变化，先选择官方已修复 SheetJS 并补真实文件测试；若优先统一 npm 供应链，则评估 ExcelJS。短期豁免只适合作为有明确到期日的过渡。**上一轮仅评估，未执行任何方案，也未改豁免日期；本轮按用户决定执行以下替换。**

### 本轮执行：官方 SheetJS CE 0.20.3（2026-10-29）

#### 版本选择、来源核验及可复现安装

核对官方安装文档、CDN 首页与 `xlsx.lst` 版本清单：截至本次核验，官方文档指向且版本清单列出的最新 CE 版本为 **0.20.3**，不使用 npm 上停留在 0.18.5 的包，也不使用第三方镜像或浮动 `latest` 地址。

官方来源：

```text
https://docs.sheetjs.com/docs/getting-started/installation/nodejs/
https://cdn.sheetjs.com/
https://cdn.sheetjs.com/xlsx.lst
https://cdn.sheetjs.com/xlsx-0.20.3/xlsx-0.20.3.tgz
https://cdn.sheetjs.com/advisories/CVE-2023-30533
https://cdn.sheetjs.com/advisories/CVE-2024-22363
```

两条公告分别要求至少 **0.19.3**（GHSA-4r6h-8v6p-xvw6 / CVE-2023-30533）和 **0.20.2**（GHSA-5pgg-2g8v-p4x9 / CVE-2024-22363）；0.20.3 均满足。该结论来自官方公告与实际包版本核对，不是仅凭 audit 对非 registry 依赖未报警推断。

通过 HTTPS 获取官方文档与压缩包，核验主机名分别为 `docs.sheetjs.com`、`cdn.sheetjs.com`，响应 HTTP 200、重定向次数 0、TLS 证书校验成功（curl `ssl_verify_result=0`）。压缩包为 **2,409,319 bytes**；包内元数据为 `name=xlsx`、`version=0.20.3`、`license=Apache-2.0`，homepage 为 `https://sheetjs.com/`，repository 为 `https://git.sheetjs.com/SheetJS/sheetjs`，无生产依赖及 install/postinstall 脚本。

对下载内容计算并锁定的哈希如下（不是厂商签名验证）：

```text
SHA-256: 8dc73fc3b00203e72d176e85b50938627c7b086e607c682e8d3c22c02bb99fe8
SHA-512 SRI: sha512-oLDq3jw7AcLqKWH2AhCpVTZl8mf6X2YReP+Neh0SJUzV/BdZYjth94tG5toiMB1PPrYtxOCfaoUCkvtuH+3AJA==
```

`frontend/package.json` 使用完整固定 URL，保留包名 `xlsx` 和原导入 API：

```json
"xlsx": "https://cdn.sheetjs.com/xlsx-0.20.3/xlsx-0.20.3.tgz"
```

用 pnpm 9.15.9 更新 lockfile。该版本对 URL 依赖生成的记录未自动包含 integrity，因此补入上述已验证 SHA-512；锁文件同时保留版本、tarball URL 与 integrity。新增回归测试防止后续重写锁文件时丢失哈希。独立目录、独立 pnpm store 的 frozen 安装验证：正确哈希可安装，故意错误哈希被 `ERR_PNPM_TARBALL_INTEGRITY` 拒绝；错误值只存在于临时 fixture，未写入项目锁文件。

本机 Node 访问 CDN 时曾在 IPv6 下载上停滞；只对相关下载命令设置 `NODE_OPTIONS=--dns-result-order=ipv4first` 后成功，未修改全局网络配置。固定 URL 与哈希可复现依赖内容，但首次下载仍需要官方 CDN 可用；本轮未把 tarball vendor 到仓库。

#### 本轮改动文件

| 文件 | 本轮改动 |
| --- | --- |
| `frontend/package.json` | xlsx 从 npm 0.18.5 改为官方固定 URL 的 0.20.3 |
| `frontend/pnpm-lock.yaml` | pnpm 更新解析结果，锁定官方 URL 与 SHA-512，移除旧 xlsx 及不再需要的 8 个传递依赖 |
| `.github/audit-exceptions.yml` | 仅删除 xlsx 两条已于 2026-10-06 过期的豁免；不延长日期、不新增豁免 |
| `frontend/src/views/admin/__tests__/UsageView.export.integration.spec.ts` | 新增 4 项测试：官方来源/hash 锁定、真实分页 XLSX、空数据 XLSX、进度/取消/重复导出保护 |
| `Makefile` | 将真实 XLSX 导出集成测试纳入 `FRONTEND_CRITICAL_VITEST` |
| `SECURITY_REMEDIATION_2026-10-29.md` | 追加本轮证据并更新当前结论，保留上轮验证历史 |

`UsageView.vue`、`UsageFilters.vue` 生产导出实现未修改，也不需要业务兼容性改动。测试中针对 jsdom 的 FileReader 保留真实 setImmediate，并将所用静态英文 locale 标签转换为 vue-i18n runtime-only 可执行的 message functions；这些调整仅在测试中生效。

#### 实际导出与替换前后比较

在升级运行库之前，使用 npm 0.18.5 执行真实导出基线；升级后对同一组数据重复执行。测试挂载真实 `UsageView` 与 `UsageFilters`，点击实际导出按钮，运行生产分页、格式化、SheetJS 写入和 Blob 下载逻辑。API 使用固定测试数据，file-saver 下载边界被接管以保存文件；无关展示组件被 stub，**xlsx 未 mock**。随后从磁盘读取实际 `.xlsx`，用真实 SheetJS 解析并断言工作表、数据与类型。

这是本地组件集成测试与真实文件回归，**不是登录生产 NAS 后的浏览器 E2E**；本轮没有访问生产数据库或部署。

覆盖与结果：

- 分页场景生成 **101 条数据、32 列**，工作表名 `Usage`，范围 `A1:AF102`；每页 100 条，两次请求均验证 `exact_total=true`。
- 比较全部表头、行数据、数值/字符串类型及单元格格式；金额维持六位小数文本，倍率维持 `toPrecision(4)` 的四位有效数字文本。
- 中文、日文、emoji、引号、换行、XML 特殊字符、0 值、前导零请求 ID 保持一致；`=SUM(1,2)` 保持字符串且没有公式字段。
- 空数据仍生成 32 列表头、范围 `A1:AF1` 的有效文件；分页进度、取消后不下载及重复点击保护通过。
- 新旧文件读回后，包含下载文件名、工作表名、完整 `Sheets` 的语义 JSON 经 `diff -u` 比较完全一致。比较的是解析语义，不要求 ZIP 二进制逐字节相同。
- 新版分页与空表文件均通过 `unzip -t`，未检测到 ZIP 错误。

原始文件和语义快照：

```text
/tmp/tokenc2c-sheetjs-20261029.dOe18K/exports/
  usage-paginated-0.18.5.xlsx / .json
  usage-paginated-0.20.3.xlsx / .json
  usage-empty-0.18.5.xlsx / .json
  usage-empty-0.20.3.xlsx / .json
```

两份分页语义 JSON 的 SHA-256 相同：

```text
1c5e27219eb837c14ade12cc934dab9cce8a7af444aaf18bf47143a4e00633d6
```

#### 本轮验证命令与结果

以下 `SHEETJS_DIR` 指 `/tmp/tokenc2c-sheetjs-20261029.dOe18K`。前端命令在 `frontend/` 执行，`make`、例外校验及 git 命令在仓库根目录执行。环境仍为 macOS/arm64、Node 26.8.1，本机 pnpm 10.30.3，另验证 CI 同系列 pnpm 9.15.9；没有声称在 Node 20/Linux runner 完整复跑。

| 命令/检查 | 结果 |
| --- | --- |
| `NODE_OPTIONS=--dns-result-order=ipv4first npx --yes pnpm@9.15.9 install --lockfile-only --ignore-scripts --fetch-timeout 30000 --fetch-retries 1 --modules-dir "$SHEETJS_DIR/pnpm9-modules"` | 成功更新锁文件，随后补入已核验的 integrity |
| `npx --yes pnpm@9.15.9 install --lockfile-only --frozen-lockfile --ignore-scripts --modules-dir "$SHEETJS_DIR/pnpm9-modules"` | 退出 0，最终锁文件兼容 pnpm 9 |
| `pnpm install --frozen-lockfile` | 退出 0；最终实际安装 xlsx 0.20.3，URL/SHA-512 锁定不变 |
| 独立 store 中 pnpm 9 frozen 安装的正确/错误 SRI 对照 | 正确值安装成功；错误值按预期被完整性检查拒绝 |
| `pnpm audit --prod --audit-level=high` | 退出 0；high=0、critical=0，仍有 13 moderate、2 low |
| `npx --yes pnpm@9.15.9 audit --prod --audit-level=high` | 退出 0，同样 high=0、critical=0 |
| `npx --yes pnpm@9.15.9 audit --prod --audit-level=high --json > "$SHEETJS_DIR/audit-prod.json"` | JSON 中 high=0、critical=0；此 JSON 模式命令仍退出 1，见下方说明 |
| `python3 tools/check_pnpm_audit_exceptions.py --audit "$SHEETJS_DIR/audit-prod.json" --exceptions .github/audit-exceptions.yml` | 退出 0，`Audit exceptions validated.` |
| `jq -e '.metadata.vulnerabilities.high == 0 and .metadata.vulnerabilities.critical == 0' "$SHEETJS_DIR/audit-prod.json"` | true，退出 0 |
| `pnpm build` | 最终工作区再次构建通过，包含类型检查；仅有现有大 chunk 提示 |
| `TOKENC2C_XLSX_ARTIFACT_DIR="$SHEETJS_DIR/exports" pnpm test:run` | **254 个文件、1,837 项测试全部通过** |
| `make test-frontend-critical` | **14 个文件、172 项测试通过**，包括新增真实导出测试 |
| `pnpm exec eslint src/views/admin/__tests__/UsageView.export.integration.spec.ts` | 退出 0 |
| `diff -u "$SHEETJS_DIR/exports/usage-paginated-0.18.5.json" "$SHEETJS_DIR/exports/usage-paginated-0.20.3.json"` | 退出 0，无差异 |
| `diff -u "$SHEETJS_DIR/exports/usage-empty-0.18.5.json" "$SHEETJS_DIR/exports/usage-empty-0.20.3.json"` | 退出 0，无差异 |
| `unzip -t`（两个新版真实 XLSX 文件） | 均无错误 |
| `git diff --exit-code -- frontend/src/views/admin/UsageView.vue frontend/src/components/admin/usage/UsageFilters.vue` | 退出 0，生产导出代码未改 |
| `git diff --check`；`git diff --cached --exit-code` | 通过；暂存区为空 |

pnpm 9 的 JSON audit 此次在只剩 moderate/low 的情况下仍退出 1，不能记录为该命令退出 0。仓库现有 workflow 使用 JSON audit `|| true` 收集报告，再由例外校验脚本执行 high/critical 门禁；此完整链路现在通过，非 JSON 的 high 门禁也明确退出 0。没有修改 checker 或放宽门禁；生产 high/critical 清零不代表全部严重性清零，也不代表开发依赖的全量 audit 清零。

证据日志在 `SHEETJS_DIR`：`frontend-build-final.log`、`frontend-tests.log`、`frontend-critical-tests.log`、`export-baseline.log`、`audit-prod.json`、`audit-prod.log`、`audit-prod-pnpm9.log`、`audit-exceptions.log`、`pnpm9-frozen-lock.log`、`pnpm-frozen-install.log`、`integrity-valid.log`、`integrity-invalid.log`，以及官方文档、版本清单和压缩包。临时目录不保证长期保留；新增测试可重新生成当前版本产物，默认临时文件在测试结束后清理，指定 `TOKENC2C_XLSX_ARTIFACT_DIR` 时保留。

本轮未 commit、未 push、未部署。HEAD 仍为上述 `06a6731e9774a09da7eaf9183336273691de6568`；保留上一轮全部未提交修复，改动可通过 `git status` / `git diff` 检查。

### 核对依据

官方资料（核对日期 2026-10-29）：

```text
https://pkg.go.dev/vuln/GO-2026-6443
https://pkg.go.dev/vuln/GO-2026-6860
https://pkg.go.dev/vuln/GO-2026-6866
https://github.com/advisories/GHSA-4r6h-8v6p-xvw6
https://github.com/advisories/GHSA-5pgg-2g8v-p4x9
https://cdn.sheetjs.com/advisories/CVE-2024-22363
https://docs.sheetjs.com/docs/getting-started/installation/nodejs/
https://github.com/exceljs/exceljs
https://golangci-lint.run/docs/product/changelog/
https://golangci-lint.run/docs/linters/false-positives/
https://staticcheck.dev/docs/checks/#S1005
https://pkg.go.dev/github.com/kisielk/errcheck
```
