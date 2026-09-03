.PHONY: build

# 一键构建：先构建前端产物（输出到 webui/dist，该目录不入库），再编译 Go 二进制
# 前置：Node.js + pnpm（go:embed 依赖 webui/dist 存在）
build:
	cd frontend && pnpm install && pnpm build
