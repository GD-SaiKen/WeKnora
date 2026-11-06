#!/usr/bin/env bash
# 校验迁移目录没有重复版本号。
#
# 背景：golang-migrate 在构造 migrator 时就会解析整个迁移目录并按版本号建索引。
# 同一版本出现两个 up（或两个 down）文件时，它以
#   duplicate migration file: 000135_xxx.down.sql
# 拒绝加载，进程启动即失败 —— 所以这类问题平时毫无征兆，只在部署那一刻爆出来。
#
# 跟踪上游的 fork 很容易踩到：本地迁移先占用了某个号，上游随后也用了同一个号，
# 合并时纯文本层面没有任何冲突。上游已有 TestMigrationDirectoriesLoad 覆盖此场景
# （internal/database/migration_versions_test.go），本脚本只是让它不需要本地 Go
# 工具链也能跑，供「同步上游之后、发版之前」快速自检。
#
# 用法: scripts/check-migrations.sh
#
# 环境变量:
#   GO_IMAGE          Docker 里用的 Go 镜像（默认与 docker/Dockerfile.app 的 builder 一致）
#   GOPROXY           模块代理（默认国内源；本机直连 proxy.golang.org 时改成 direct）
#   MODCACHE_VOLUME   复用模块缓存的 Docker 卷，避免每次重新下载依赖
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

GO_IMAGE="${GO_IMAGE:-golang:1.26-bookworm}"
GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
MODCACHE_VOLUME="${MODCACHE_VOLUME:-weknora-gomod-cache}"
TEST_PATTERN="${TEST_PATTERN:-TestMigrationDirectoriesLoad}"

run_check() {
    local go_args=(test ./internal/database -run "$TEST_PATTERN" -count=1)

    if command -v go >/dev/null 2>&1; then
        echo "==> 本地有 Go，直接跑"
        GOFLAGS=-mod=mod GOPROXY="$GOPROXY" go "${go_args[@]}"
        return
    fi

    if ! command -v docker >/dev/null 2>&1; then
        cat >&2 <<'EOF'
❌ 本机既没有 go 也没有 docker，无法执行校验。
   请安装 Go 后运行: go test ./internal/database -run TestMigrationDirectoriesLoad
EOF
        return 1
    fi

    echo "==> 本地没有 Go，改用 Docker ($GO_IMAGE)"
    docker volume create "$MODCACHE_VOLUME" >/dev/null
    docker run --rm \
        -e GOPROXY="$GOPROXY" \
        -e GOFLAGS=-mod=mod \
        -v "$ROOT":/src \
        -v "$MODCACHE_VOLUME":/go/pkg/mod \
        -w /src \
        "$GO_IMAGE" \
        go "${go_args[@]}"
}

if run_check; then
    echo
    echo "✅ 迁移目录可正常加载（versioned 与 sqlite 都没有重复版本号）"
    exit 0
fi

cat >&2 <<'EOF'

❌ 迁移目录无法加载，几乎可以确定是版本号重复。

   修复方式：把 fork 自己的迁移顺延到上游最高版本之后，并保持本地
   迁移彼此之间的相对顺序。例如上游占用了 000134：

       migrations/versioned/000134_xxx  ->  000135_xxx
       migrations/versioned/000135_yyy  ->  000136_yyy   # 原本排在后面的，一起顺延

   sqlite 是独立的 numbering，需要单独检查。
   改完重新运行本脚本确认。
EOF
exit 1
