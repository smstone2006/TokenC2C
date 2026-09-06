<div align="center">
  <img src="assets/logo.svg" alt="TokenC2C ロゴ" width="100" />
  <h1>TokenC2C</h1>
  <p><strong>アカウントを共有し、需要につなぎ、利用を明確に記録する。</strong></p>
  <p>利用者と提供者の利用量・収支を個別に管理できる、セルフホスト型 AI アカウント共有プール。</p>
  <p><a href="README.md">English</a> · <a href="README_CN.md">简体中文</a> · <a href="README_JA.md"><strong>日本語</strong></a></p>
  <img src="https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white" alt="Go 1.27" />
  <img src="https://img.shields.io/badge/Vue-3-4FC08D?logo=vue.js&logoColor=white" alt="Vue 3" />
  <img src="https://img.shields.io/badge/PostgreSQL-18-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL 18" />
  <img src="https://img.shields.io/badge/Redis-8-DC382D?logo=redis&logoColor=white" alt="Redis 8" />
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-LGPL--3.0-blue" alt="ライセンス：LGPL-3.0" /></a>
</div>

## 概要

TokenC2C は [Sub2API](https://github.com/Wei-Shaw/sub2api) を拡張し、ユーザー自身が所有するアカウントを供給できる仕組みを追加したプロジェクトです。提供者が対応する上流アカウントを登録し、検証に成功すると共有スケジューラーの対象になります。利用者は自分のプラットフォーム API キーで AI サービスを呼び出し、システムが利用量と提供者の収益を記録します。

同じユーザーが利用者と提供者を兼ねることもできます。ここでいう **Token** は AI の使用量単位およびプラットフォーム内クレジットを指し、暗号資産ではありません。

## 主な機能

- **セルフサービスでの供給：** 自分のアカウントを追加・検証・一時停止・再開・削除できます。
- **検証後の参加：** 検証に成功するまで、アカウントはリクエストの割り当て対象になりません。
- **共有スケジューリング：** Sub2API の認証、ルーティング、レート制限、使用量課金を継承します。
- **供給量の制御：** 日次提供上限を設定し、アカウントごとのプール状態を確認できます。
- **双方の集計：** C2C API はリクエスト数、Token 総量、利用者の支出、提供者の収益を返します。管理画面にはウォレット、アカウント、台帳を表示します。
- **原子的な精算：** 利用者への課金、提供者のクレジット、プラットフォーム手数料を単一のデータベーストランザクションで記録します。
- **二重記帳の防止：** リクエスト/API キーの識別子と台帳の冪等キーで重複精算を防ぎます。
- **レスポンシブ UI：** 独自の TokenC2C ブランドと英語・中国語のホームページ。ドキュメントは英語・中国語・日本語に対応します。

### Sub2API からの拡張点

| 分野 | Sub2API から継承する基盤 | TokenC2C の拡張 |
| --- | --- | --- |
| アカウント管理 | 管理者による上流アカウント管理 | ユーザー所有のアカウント、自主検証、供給制御 |
| ルーティング | 共有スケジューラーと API ゲートウェイ | 検証済みユーザーアカウントによる供給 |
| 課金・精算 | 使用量の記録と残高からの課金 | 提供者収益、手数料、確定待ちクレジット |
| 可視化 | 使用履歴と管理ダッシュボード | 利用者・提供者別の集計と精算台帳 |

## 仕組み

```text
提供者のアカウント -> 上流サービスで検証 -> 共有アカウントプール
利用者の API キー -> ゲートウェイとスケジューラー -> 共有プール -> 課金対象の呼び出し

課金対象の呼び出し
  |-- 利用者への課金
  |-- 提供者の確定待ちクレジット -> 24 時間後に利用可能なクレジットへ
  `-- プラットフォーム手数料
```

### 現在の精算ルール

| 項目 | ルール |
| --- | --- |
| 提供者の取り分 | 利用者への課金額の 90% をプラットフォーム内クレジットとして付与 |
| プラットフォーム手数料 | 利用者への課金額の 10% |
| 確定待ち期間 | 24 時間後に提供者の利用可能クレジットへ移行 |
| 自分の供給の利用 | 対応。同じ精算と手数料のルールを適用 |
| 現金の出金 | 未実装 |

これは現在の MVP の実装ルールであり、固定収益を保証するものではありません。収益は実際に割り当てられ、課金された利用によって発生します。自動返金、紛争処理、現金での支払いは現行版に含まれません。

## 対応するアカウント

C2C の登録フォームは OpenAI、Anthropic、Gemini、Antigravity、Grok、Kimi、Zhipu/GLM、DeepSeek に対応しています。利用できる認証情報の種類はプロバイダーごとに異なるため、フォームの選択肢を確認してください。実際に利用できるモデルはアカウントと運営設定に依存します。

供給フローで受け付けるのは対応する公式上流サービスです。ユーザー指定の Base URL と汎用中継アカウントは拒否されます。利用者は自分のプラットフォーム API キーを使用し、アカウント一覧・統計 API に提供者の上流認証情報は含まれません。

## Docker Compose で起動

Git、Docker、Compose v2 と、ビルド依存関係・コンテナレジストリへのネットワーク接続が必要です。付属構成では PostgreSQL 18 と Redis 8 を使用します。

```sh
git clone https://github.com/smstone2006/TokenC2C.git
cd TokenC2C
cp deploy/.env.c2c.example deploy/.env.c2c
```

`deploy/.env.c2c` を編集し、管理者のメールアドレス・パスワードを設定して、**すべて**のプレースホルダーを個別のランダム値に置き換えてください。`JWT_SECRET` と `TOTP_ENCRYPTION_KEY` は、それぞれ `openssl rand -hex 32` で生成します。実際の環境設定ファイルは非公開で管理してください。

```sh
docker compose --env-file deploy/.env.c2c -f deploy/docker-compose.c2c.yml up -d --build

# 初期設定用の単発コンテナの終了を待ちます。終了コードは 0 が必要です。
docker wait subc2c-bootstrap

# 初期設定後にブランド名と登録設定を再読み込みします。
docker compose --env-file deploy/.env.c2c -f deploy/docker-compose.c2c.yml restart app
docker compose --env-file deploy/.env.c2c -f deploy/docker-compose.c2c.yml up -d --wait --wait-timeout 180 app
curl -fsS http://127.0.0.1:18080/health
```

[http://127.0.0.1:18080](http://127.0.0.1:18080) を開き、設定した管理者アカウントでログインします。初期設定では招待コード制の登録が有効になり、メール認証は無効になります。メンバーを招待する前に、管理画面で招待コードを作成してください。

初期状態ではループバックアドレスだけにバインドします。外部からアクセスする場合は、HTTPS リバースプロキシとアクセス方針を設定してください。PostgreSQL と Redis のホストポートは公開しません。常駐サービスは `unless-stopped` を使用し、単発の初期設定コンテナは自動再起動しません。

### NAS へのデプロイ

QNAP 用テンプレートは [`deploy/docker-compose.c2c.nas.yml`](deploy/docker-compose.c2c.nas.yml)、起動と qnet 接続用スクリプトは [`deploy/deploy-c2c-nas.sh`](deploy/deploy-c2c-nas.sh) です。環境に依存するため、Docker のパス、データディレクトリ、qnet 名、未使用の LAN IP を調整してください。先に対象アーキテクチャ向けに、テンプレートが使用する `subc2c:nas` イメージをビルド・ロードします。詳細は[デプロイ手順](docs/C2C_MVP.md)を参照してください。

既存環境との互換性を保つため、コンテナ・ボリューム・データベース内部の `subc2c` 識別子は維持しています。公開製品名は TokenC2C です。

## 開発とテスト

| コンポーネント | 技術 |
| --- | --- |
| バックエンド | Go 1.27、Gin、Ent |
| フロントエンド | Vue 3、TypeScript、Vite、Tailwind CSS |
| データベース / キャッシュ | PostgreSQL / Redis |
| テスト | Go tests、Testcontainers、Vitest、Vue Test Utils |

```sh
cd frontend
pnpm install --frozen-lockfile  # pnpm 9 を使用
pnpm typecheck
pnpm test:run
pnpm build

cd ../backend
go test -tags=unit ./internal/service ./internal/repository ./migrations -run 'TestC2C|TestApplyC2CTrade' -count=1
CI=1 go test -v -tags=integration ./internal/repository -run '^TestC2CBilling' -count=1
```

統合テストには Docker が必要で、隔離された PostgreSQL/Redis コンテナを起動します。利用者への課金、提供者の確定待ちクレジット、手数料配分、重複リクエスト、確定処理、自己利用を検証します。フロントエンドのテストは役割別の入口、登録設定、言語切り替え、既存ホームページモードを対象とします。`CI=1` を指定すると、Docker が利用できない場合にテストをスキップせず、明示的に失敗させます。

## C2C API

以下のパスの共通プレフィックスは `/api/v1/c2c` です。ログインが必要で、供給アカウントの変更操作は所有者に限定されます。

| メソッド | パス | 用途 |
| --- | --- | --- |
| GET / POST | `/provider-accounts` | 自分のアカウントの一覧 / 追加 |
| POST | `/provider-accounts/:id/verify` | アカウントの検証 |
| PUT | `/provider-accounts/:id/pool-state` | 供給の一時停止・再開 |
| DELETE | `/provider-accounts/:id` | アカウントの削除 |
| GET | `/wallet` | 利用可能・確定待ちクレジット |
| GET | `/stats` | 利用者・提供者の集計 |
| GET | `/ledger?limit=50` | 最近の台帳エントリ |

## ディレクトリ構成

```text
backend/
  internal/handler/c2c_handler.go          # ユーザー向け C2C HTTP ハンドラー
  internal/service/c2c_service.go          # 所有権、検証、確定処理
  internal/repository/c2c_billing.go       # トランザクション内の精算
  migrations/234_c2c_marketplace.sql       # アカウント所有権と台帳スキーマ
frontend/src/
  components/home/                        # TokenC2C ホームページと文言
  views/user/C2CView.vue                   # 供給・ウォレット・台帳画面
  api/c2c.ts                              # 型付き API クライアント
deploy/                                   # Docker / NAS テンプレート
docs/C2C_MVP.md                            # 実装とデプロイの説明
```

## 運用上の範囲

現行版は信頼できるメンバーを招待して運用する MVP です。上流の認証情報ストレージを再利用しており、提供アカウントの保存時の追加アプリケーション層暗号化は未実装です。運用範囲を広げる前に、環境に適した認証情報保護、不正利用対策、紛争・返金処理を整備してください。使用権限があり、プロバイダーの規約上共有が認められるアカウントのみを提供してください。

実際の環境設定、Token、秘密鍵、データベースバックアップをコミットしないでください。サービスを案内する際は、上記の手数料とクレジットのルールを明確にしてください。

## コントリビューション

再現可能な不具合や具体的な提案は [Issues](https://github.com/smstone2006/TokenC2C/issues) へ、コードの変更は関連テストを添えた Pull Request でお寄せください。実際の上流認証情報やユーザーの個人データを含めないでください。

## 上流プロジェクトとライセンス

TokenC2C は [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) を基に開発し、Git 履歴とライセンス表記を保持しています。ソース互換性のため Go モジュールパスは変更していません。元の英語 README は [README_UPSTREAM.md](README_UPSTREAM.md) に保存しています。

本リポジトリのライセンスは [GNU LGPL v3](LICENSE) です。第三者コンポーネントにはそれぞれのライセンスが適用されます。ゲートウェイ基盤を提供する Sub2API の作者とコントリビューターに感謝します。
