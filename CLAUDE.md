# igt - Interactive gitignore CLI

## 概要

`igt`（**i**gnore + **T**UI）は、[gibo](https://github.com/simonwhitaker/gibo) のような
gitignoreテンプレート管理ツールを、対話型TUIで実現するGo製CLIツール。

GitHub公式の [github/gitignore](https://github.com/github/gitignore) リポジトリから
テンプレートを取得し、ファジー検索付きの複数選択UIで選んだテンプレートを
`.gitignore` にセクション管理付きで追記する。

検索性対応: READMEやGitHub Aboutには `gitignore cli`, `gitignore tui`,
`interactive gitignore generator` 等のキーワードを含めること
（"IGT" は International Game Technology 等との名前衝突があるため）。

## 技術スタック

- 言語: Go
- TUIフレームワーク: [Bubble Tea](https://github.com/charmbracelet/bubbletea)
  + [Bubbles](https://github.com/charmbracelet/bubbles)（`list` コンポーネント、フィルタ機能内蔵）
- テンプレート取得: GitHub API（`github/gitignore` リポジトリ）
- キャッシュ: ローカルファイルシステム（`os.UserCacheDir()` 経由、XDG準拠）

## アーキテクチャ

```
┌──────────────────────┐
│ GitHub取得層           │  GitHub APIで github/gitignore の一覧・内容を取得
├──────────────────────┤
│ ローカルキャッシュ層     │  ~/.cache/igt/templates/ にTTL付きで保存
├──────────────────────┤
│ 対話型選択UI層          │  Bubble Tea: ファジー検索 + カテゴリ別グループ + 複数選択
├──────────────────────┤
│ セクション解析・置換層   │  既存.gitignoreをパースし、マーカー単位で削除・再配置
├──────────────────────┤
│ 出力層                 │  .gitignore へ書き込み（末尾に追記）
└──────────────────────┘
```

## 機能仕様

### 1. テンプレート取得・キャッシュ

- 取得元: GitHub API経由で `github/gitignore` リポジトリの内容を取得
  - 対象: ルート直下の `*.gitignore`、`Global/`、`community/` 配下
- キャッシュ先: `~/.cache/igt/templates/`
- 更新ポリシー:
  - TTL方式（デフォルト7日）で自動更新チェック
  - `-r, --refresh` フラグで強制更新
  - オフライン時や取得失敗時はキャッシュをそのまま使用（エラーで落とさない）
- レート制限対策:
  - GitHub APIの未認証レート制限（60回/時）を考慮し、キャッシュヒット時はAPIを叩かない
  - 環境変数 `GITHUB_TOKEN` があれば認証付きリクエストに切り替え、レート制限を緩和（5000回/時）
  - レート制限到達時はキャッシュにフォールバックし、警告メッセージを表示

### 2. 対話型選択UI

- ライブラリ: `bubbletea` + `bubbles/list`
- 表示形式: カテゴリ別（言語 / Global / community）にグループ化
- 操作:
  - インクリメンタル入力でファジーフィルタ
  - `Space` または `Tab` で複数選択（チェックマーク表示）
  - `Enter` で選択確定
  - `Esc` / `Ctrl+C` でキャンセル（ファイルへの書き込みなしで終了）
- 既存 `.gitignore` に含まれるテンプレートのサイドパネル表示は将来タスク（初期実装では対象外）

### 3. CLIフラグ

```
igt [flags] [<template>...]

Flags:
  -o, --output string   出力先ファイル（デフォルト: ./.gitignore）
  -r, --refresh         キャッシュを強制更新
  -l, --list            テンプレート一覧を表示して終了（対話UIなし、grep用）
  -n, --dry-run         実際には書き込まず、変更内容をプレビュー表示
  -h, --help            ヘルプ表示
  -v, --version         バージョン表示
```

- `<template>...` を引数で直接指定した場合は対話UIをスキップして即座に処理（gibo互換の使い方も可能にする）
- 引数なしで実行した場合のみ対話UIを起動

### 4. .gitignoreへの出力・マージロジック

#### マーカーフォーマット

```
### Go ###
(Goテンプレートの内容)
### Go ###

### Node ###
(Nodeテンプレートの内容)
### Node ###
```

- 開始・終了を同じマーカー行 `### <TemplateName> ###` で挟む
- 既存内容とマーカーセクションの間には空行を1つ入れる
- テンプレート間にも空行を1つ入れる
- 各テンプレート本文の末尾の余分な空行はtrimする

#### 再選択時の置換ロジック

1. `.gitignore` 全体をパースし、`### <name> ###` 〜 次の同名 `### <name> ###` までを1セクションとして認識
2. 選択されたテンプレートに対応するセクションが既存にあれば、そのセクションを丸ごと削除
3. 削除後、最新のテンプレート内容を**常にファイル末尾**に追加（元の位置は保持しない）
4. 新規テンプレート（既存セクションなし）もそのまま末尾に追加

#### エッジケース

- `.gitignore` が存在しない場合: 新規作成
- `.gitignore` はあるがマーカー管理外の内容のみの場合: 既存内容はそのまま保持し、末尾に新規セクションを追加
- 同一実行内で同じテンプレートが複数回指定された場合: 重複排除して1回のみ処理

### 5. dry-runモード

- `-n, --dry-run` 指定時は実際のファイル書き込みを行わず、diff形式（追加行を`+`、削除行を`-`）で変更内容を標準出力にプレビュー表示する

## 将来検討事項（初期実装スコープ外）

- 現在 `.gitignore` に含まれるテンプレート一覧を確認する機能（gibo `dump` 相当）
  - 対話型UIのサイドパネルとして表示する方向で検討中
- tarball一括取得への切り替え（レート制限がボトルネックになった場合）
- カスタムテンプレート（ユーザー定義）のサポート

## ディレクトリ構成（想定）

```
igt/
├── cmd/
│   └── igt/
│       └── main.go
├── internal/
│   ├── fetcher/       # GitHub API取得・キャッシュ処理
│   ├── ui/            # Bubble Tea TUIコンポーネント
│   ├── merger/        # セクション解析・マージロジック
│   └── template/       # テンプレートのデータモデル
├── go.mod
├── go.sum
└── README.md
```

## 開発時の注意点

- テストはロジック層（`fetcher`, `merger`）を中心にユニットテストを書く。TUI部分（`ui`）は手動確認を優先し、必要に応じて `bubbletea` のテストヘルパーを使う
- GitHub API呼び出しはモック可能な形でインターフェース化し、レート制限やネットワーク断のテストをしやすくする
- クロスコンパイル（`GOOS`/`GOARCH`）を前提に、OS依存のパスは `os.UserCacheDir()` 等の標準ライブラリ経由で解決する
