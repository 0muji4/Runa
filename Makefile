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

# adb の場所。PATH に無い環境でも動くよう ANDROID_HOME 配下を既定にする。
ADB ?= $(ANDROID_HOME)/platform-tools/adb

# iOS シミュレータ名。`xcrun simctl list devices available` の値で上書き可。
IOS_SIM ?= iPhone 17 Pro

# ---- 実機（physical device）向けの設定 --------------------------------------
# 実機からはホストの localhost / 10.0.2.2 に届かない。同じ LAN のホスト IP
# (`make lan-ip`) か、デプロイ済みの URL を RUNA_BASE_URL で渡すこと。
#   make android-run RUNA_BASE_URL=http://192.168.1.10:8080
#   make ios-run IOS_TEAM=XXXXXXXXXX IOS_DEVICE=00008110-... RUNA_BASE_URL=https://...
RUNA_BASE_URL ?=

# Apple Developer の Team ID。実機ビルドの署名に必須（Xcode の Signing & Capabilities
# か Apple Developer の Membership で確認できる 10 文字）。
IOS_TEAM ?=
# 対象実機の UDID。`make ios-devices` で確認する。
IOS_DEVICE ?=

# 実機の bundle id / applicationId（project.yml と androidApp/build.gradle.kts が正）。
IOS_BUNDLE_ID  := com.runa
ANDROID_APP_ID := com.runa
# 実機ビルドの成果物。DerivedData は apps/swift/build/ 配下（gitignore 済み）。
IOS_DEVICE_APP := $(SWIFT_DIR)/build/device/Build/Products/Debug-iphoneos/Runa.app

# KMP + SKIE が生成する iOS 向けバイナリ。SharedKit/Package.swift が参照する固定パス。
XCFRAMEWORK := $(KOTLIN_DIR)/shared/build/XCFrameworks/release/Shared.xcframework

.DEFAULT_GOAL := help

# ---- 集約 -------------------------------------------------------------------

.PHONY: verify
verify: check-theme check-strings server-verify shared-test android-build ios-build ## 全レイヤーを検証（PR 前の総合チェック / iOS ビルド含むため重い）

.PHONY: check-theme
check-theme: ## テーマトークンの整合を検証（README 正典 ⇔ Android/iOS/colors.xml の色定義。ビルド不要）
	./hack/check-theme-tokens.sh

.PHONY: check-strings
check-strings: ## UI 文言の整合を検証（strings.xml ⇔ iOS Strings.swift、iOS に生の文言が無いこと。ビルド不要）
	./hack/check-ui-strings.sh

.PHONY: ios-strings
ios-strings: ## iOS: strings.xml から apps/swift/Runa/Strings.swift を再生成（文言を直したら必ず実行）
	./hack/gen-ios-strings.sh

# ---- Server (Go) ------------------------------------------------------------

# テストは2層: -short は Postgres / MinIO のコンテナを要するものを飛ばす。CI は全部。

.PHONY: server-verify
server-verify: ## Go: vet + build + 全テスト（go-ci.yaml と同じ。要 Docker / -race 込みで遅い）
	cd $(GO_DIR) && go vet ./... && go build ./... && go test -race -shuffle=on -count=1 ./...

.PHONY: server-test
server-test: ## Go: 速いテストのみ（-short。Docker 不要。書きながら回す用）
	cd $(GO_DIR) && go test -short -count=1 ./...

.PHONY: server-test-all
server-test-all: ## Go: 実 Postgres / MinIO を含む全テスト（要 Docker）
	cd $(GO_DIR) && go test -count=1 ./...

.PHONY: server-cover
server-cover: ## Go: 本番コード基準のカバレッジを測る（-coverpkg=./...）。未実行の関数も出す。要 Docker
	cd $(GO_DIR) && go test -count=1 -coverpkg=./... -coverprofile=cover.out ./...
	cd $(GO_DIR) && go tool cover -func=cover.out | tail -1
	@echo '--- 0% の関数（未実行の本番コード） ---'
	cd $(GO_DIR) && go tool cover -func=cover.out | awk '$$NF == "0.0%"' || true

.PHONY: server-cover-html
server-cover-html: server-cover ## Go: カバレッジを HTML で開く
	cd $(GO_DIR) && go tool cover -html=cover.out

.PHONY: server-fuzz
server-fuzz: ## Go: 手書きパーサ（argon2 PHC / JWKS）を短時間ファジングする
	cd $(GO_DIR) && go test -run=XXX -fuzz=FuzzDecodeArgon2Hash -fuzztime=30s ./internal/auth/
	cd $(GO_DIR) && go test -run=XXX -fuzz=FuzzParseJWKS -fuzztime=30s ./internal/auth/

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
	cd $(KOTLIN_DIR) && $(GRADLEW) :androidApp:assembleDebug $(if $(RUNA_BASE_URL),-PRUNA_BASE_URL=$(RUNA_BASE_URL),)

.PHONY: android-devices
android-devices: ## Android: 接続中の実機/エミュレータを一覧（adb devices -l）
	$(ADB) devices -l

.PHONY: android-install
android-install: ## Android: 実機にデバッグ APK をビルドしてインストール（USB デバッグ ON。RUNA_BASE_URL 推奨）
	cd $(KOTLIN_DIR) && $(GRADLEW) :androidApp:installDebug $(if $(RUNA_BASE_URL),-PRUNA_BASE_URL=$(RUNA_BASE_URL),)

.PHONY: android-run
android-run: android-install ## Android: 実機にインストールしてアプリを起動する
	$(ADB) shell monkey -p $(ANDROID_APP_ID) -c android.intent.category.LAUNCHER 1

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
		-configuration Debug CODE_SIGNING_ALLOWED=NO \
		$(if $(RUNA_BASE_URL),RUNA_BASE_URL=$(RUNA_BASE_URL),) build

# 実機ターゲットの前提チェック。xcframework の重いビルドより先に落としたいので、
# レシピの中ではなく最初の前提条件として置く（## を付けない＝ help に出さない）。
.PHONY: require-ios-team
require-ios-team:
	@test -n "$(IOS_TEAM)" || { echo 'IOS_TEAM が未設定: make ios-device-build IOS_TEAM=XXXXXXXXXX （Apple Developer の Membership で確認できる 10 文字）'; exit 1; }

.PHONY: require-ios-device
require-ios-device:
	@test -n "$(IOS_DEVICE)" || { echo 'IOS_DEVICE が未設定: make ios-devices で UDID を確認して渡す'; exit 1; }

.PHONY: ios-devices
ios-devices: ## iOS: 接続中の実機を一覧（UDID を IOS_DEVICE に渡す）
	xcrun devicectl list devices

.PHONY: ios-device-build
ios-device-build: require-ios-team xcframework xcodeproj ## iOS: 実機向けに署名してビルド（要 IOS_TEAM）
	cd $(SWIFT_DIR) && xcodebuild -project Runa.xcodeproj -scheme Runa \
		-destination 'generic/platform=iOS' -configuration Debug \
		-derivedDataPath build/device -allowProvisioningUpdates \
		DEVELOPMENT_TEAM=$(IOS_TEAM) CODE_SIGN_STYLE=Automatic \
		$(if $(RUNA_BASE_URL),RUNA_BASE_URL=$(RUNA_BASE_URL),) build

.PHONY: ios-install
ios-install: require-ios-device ios-device-build ## iOS: ビルドした .app を実機へインストール（要 IOS_DEVICE。Xcode 15+/iOS 17+）
	xcrun devicectl device install app --device $(IOS_DEVICE) $(IOS_DEVICE_APP)

.PHONY: ios-run
ios-run: ios-install ## iOS: 実機にインストールしてアプリを起動する
	xcrun devicectl device process launch --device $(IOS_DEVICE) $(IOS_BUNDLE_ID)

# ---- 雑務 -------------------------------------------------------------------

.PHONY: lan-ip
lan-ip: ## 実機から届くホストの LAN IP を表示（RUNA_BASE_URL の組み立てに使う）
	@ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null \
		|| { echo 'LAN IP を取得できなかった。ifconfig で確認する。'; exit 1; }

.PHONY: clean
clean: ## 生成物を削除（Gradle build / 生成された xcodeproj / 実機ビルドの DerivedData）
	cd $(KOTLIN_DIR) && $(GRADLEW) clean
	rm -rf $(SWIFT_DIR)/Runa.xcodeproj $(SWIFT_DIR)/build

.PHONY: help
help: ## このヘルプを表示
	@echo 'Runa — make targets:'
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-18s\033[0m %s\n",$$1,$$2}'
