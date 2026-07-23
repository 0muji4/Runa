# Runa monorepo — 各レイヤー（Go バックエンド / KMP shared / Android / iOS）の
# ビルド・テスト・ローカル起動をまとめる単一の入口。CONTRIBUTING.md と各 README の
# 手順をコマンド化しただけで、新しいツールは足していない。
#
# 使い方: `make`（= help）でターゲット一覧。PR 前の検証は `make verify`。
#
# macOS 標準の GNU Make 3.81 前提のため .ONESHELL は使わず、各レシピ行は独立シェル。
# ディレクトリ移動は行内 `cd ... &&` で完結させている。

SHELL := /usr/bin/env bash

GO_DIR     := apps/go
KOTLIN_DIR := apps/kotlin
SWIFT_DIR  := apps/swift
GRADLEW    := ./gradlew

# Gradle（Android/KMP）は Android SDK の場所を要求する。local.properties は
# gitignore で未コミットのため、既定を与えて `make` 単体で通るようにする。
# 環境変数が既にあればそちら優先（`?=`）。CI や非 macOS では上書きする。
ANDROID_HOME ?= $(HOME)/Library/Android/sdk
export ANDROID_HOME

# iOS シミュレータ名。`xcrun simctl list devices available` の値で上書き可。
IOS_SIM ?= iPhone 17 Pro

# KMP + SKIE が生成する iOS 向けバイナリ。SharedKit/Package.swift が参照する固定パス。
XCFRAMEWORK := $(KOTLIN_DIR)/shared/build/XCFrameworks/release/Shared.xcframework

.DEFAULT_GOAL := help

# ---- 集約 -------------------------------------------------------------------

.PHONY: verify
verify: check-theme server-verify shared-test android-build ios-build ## 全レイヤーを検証（PR 前の総合チェック / iOS ビルド含むため重い）

.PHONY: check-theme
check-theme: ## テーマトークンの整合を検証（README 正典 ⇔ Android/iOS/colors.xml の色定義。ビルド不要）
	./hack/check-theme-tokens.sh

# ---- Server (Go) ------------------------------------------------------------

.PHONY: server-verify
server-verify: ## Go: vet + build + test（go-ci.yaml と同じ内容）
	cd $(GO_DIR) && go vet ./... && go build ./... && go test ./...

.PHONY: server-test
server-test: ## Go: テストのみ
	cd $(GO_DIR) && go test ./...

.PHONY: server-fmt
server-fmt: ## Go: gofmt で整形（-w で上書き）
	cd $(GO_DIR) && gofmt -l -w .

.PHONY: server-run
server-run: ## Go: Docker なしで API を起動（go run ./cmd/api）
	cd $(GO_DIR) && go run ./cmd/api

.PHONY: server-up
server-up: ## ローカル環境を起動（Postgres + MinIO + API を docker compose で）
	cd $(GO_DIR) && docker compose up --build

.PHONY: server-down
server-down: ## ローカル環境を停止・撤去
	cd $(GO_DIR) && docker compose down

.PHONY: seed
seed: ## 今日のコンテンツ（quote/song）を投入。要 ADMIN_API_TOKEN（サーバ起動時と一致）
	./hack/seed-today.sh

# ---- Shared (KMP) -----------------------------------------------------------

.PHONY: shared-test
shared-test: ## shared: 共通ロジックの単体テスト（:shared:testDebugUnitTest）
	cd $(KOTLIN_DIR) && $(GRADLEW) :shared:testDebugUnitTest

.PHONY: xcframework
xcframework: ## shared: iOS 向け Shared.xcframework をビルド（Gradle が差分ビルド）
	cd $(KOTLIN_DIR) && $(GRADLEW) :shared:assembleSharedReleaseXCFramework

# ---- Android ----------------------------------------------------------------

.PHONY: android-build
android-build: ## Android: デバッグ APK をビルド（shared を含めてコンパイル検証）
	cd $(KOTLIN_DIR) && $(GRADLEW) :androidApp:assembleDebug

# ---- iOS --------------------------------------------------------------------
# iOS は shared の XCFramework が先に要る（SharedKit が binaryTarget で参照）。
# xcframework → xcodeproj 生成 → xcodebuild の順を依存関係で固定する。

.PHONY: xcodeproj
xcodeproj: ## iOS: project.yml から Runa.xcodeproj を生成（xcodegen、成果物は gitignore）
	cd $(SWIFT_DIR) && xcodegen generate

.PHONY: ios-build
ios-build: xcframework xcodeproj ## iOS: シミュレータ向けにビルド（XCFramework とプロジェクト生成を含む）
	cd $(SWIFT_DIR) && xcodebuild -project Runa.xcodeproj -scheme Runa \
		-destination 'platform=iOS Simulator,name=$(IOS_SIM)' \
		-configuration Debug CODE_SIGNING_ALLOWED=NO build

# ---- 雑務 -------------------------------------------------------------------

.PHONY: clean
clean: ## 生成物を削除（Gradle build / 生成された xcodeproj）
	cd $(KOTLIN_DIR) && $(GRADLEW) clean
	rm -rf $(SWIFT_DIR)/Runa.xcodeproj

.PHONY: help
help: ## このヘルプを表示
	@echo 'Runa — make targets:'
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-14s\033[0m %s\n",$$1,$$2}'
