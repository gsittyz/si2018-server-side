# Eureka Summer Internship 2018 API

# 技術スタック

- go 1.10.3
- dep
- go-swagger
- goose
- direnv
- xorm

# swagger

dockerで用意しています.
localhost:8081 で swagger-editor (エディタ), localhost:8082 で swagger-ui (APIドキュメント) が開きます。

```
docker-compose up -d
```

# how to run the app

```
# Goがインストールされている前提です。

# 必要なライブラリの取得

go get -u bitbucket.org/liamstask/goose/cmd/goose
go get -u github.com/golang/dep/cmd/dep
go get -u github.com/go-swagger/go-swagger/cmd/swagger
go get -u github.com/direnv/direnv

# 依存関係のインストール (dep ensureとか)
make init

# 環境変数を.envrc (direnv) で管理している
cp .envrc.sample .envrc
direnv allow

# ビルド
make build (ymlからgoファイルを生成

make init (生成されたgoファイルの依存関係取り込み

make build

# DBの初期化 & マイグレ
make setup-db

# サーバーを立ち上げる
make run
```

# dummy data

misc/dummy/ 下にダミーデータ生成のスクリプトを置いてます。以下makeコマンドでDBリセット & ダミー生成を行います.

```
make setup-db
```

# migration with goose

マイグレーションツールのgooseを使用しています。

```
# ./db/migrations/20180809183923_createUser.sql が作成される
goose create createHoge sql

# up
goose up

# down
goose down

# redo
goose redo
```