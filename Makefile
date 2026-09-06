.PHONY: install build frontend go clean

# 首次使用或依赖变更时安装前端依赖（pnpm 有缓存与 lockfile，之后一般不用再跑）
install:
	cd frontend && pnpm install

# 一键构建：前端产物（webui/dist，供 go:embed）→ Go 二进制
# 前提：已执行过 make install
build: frontend go

# 仅构建前端产物
frontend:
	cd frontend && pnpm build

# 仅编译 Go 二进制（依赖 webui/dist 已存在）
go:
	go build -o sbctl .

clean:
	rm -f sbctl
	rm -rf webui/dist
