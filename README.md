# sing-box-ctl (sbctl)

基于 Web UI 的 sing-box 配置管理工具：管理节点配置来源与模板，并通过 URL 参数实时生成可用的 sing-box 配置。

## 特性

- 内置 Web UI（Go 服务托管前端构建产物，无外部依赖）
- 节点配置源管理：支持订阅 URL 或本地文件上传，更新内容滚动保留历史
- 模板管理：JSON 编辑器内可直接编辑模板，支持设为默认
- 实时配置生成：`/config/<id>?<参数>` 将模板与节点数据转换为 sing-box 配置，URL 即配置
- 模式开关：`tun` / `mixed`（含系统代理），以及 sing-box api service（gRPC + Dashboard）
- 节点按关键词自动分组：模板 outbound 支持 `组名@表达式` 语法
- 平台适配：windows / linux / android 的 tun 参数自动调整（Android 强制 Tun）

## 快速开始

依赖：Go 1.24+、Node.js（pnpm）。前端产物输出到 `webui/dist` 并随二进制内嵌（该目录不入库，需先构建一次）。

> 生成的配置面向 **sing-box ≥ 1.14.0**（使用 services.api / Dashboard / http_clients 等新版配置结构）。

```bash
# 一键构建（构建 webui/dist 并编译根目录 sbctl 二进制）
cd frontend
pnpm install
pnpm build
cd ..

# 启动：-d 指定工作目录（必填，首次启动自动初始化目录结构）
./sbctl serve -d ./working_dir

# 浏览器打开
# http://127.0.0.1:8080
```

完整命令：

```bash
sbctl serve -d <working_dir> [--listen <host>] [-p <port>]
# -d        工作目录（必填），存放 providers / templates 数据
# --listen  监听地址，默认 127.0.0.1（仅本机）；局域网访问请指定 --listen 0.0.0.0
# -p        端口，默认 8080
```

## 使用流程

1. **Provider 页**：添加节点配置来源——填订阅 URL，或直接上传配置文件；可手动"更新"拉取最新内容。
2. **模板页**：模板是 sing-box 配置的基础骨架（dns/route/规则等），可多份并存、切换默认。
3. **URL 页**：选择 Provider、模板与要开启的功能，生成配置 URL：
   - 复制 URL 后，本机或局域网内的其它设备可用 sing-box 直接导入；
   - 也可用浏览器直接打开 URL 查看生成的 JSON。

## 配置生成参数（URL 页会按勾选项生成）

`/config/<provider-uuid>?template=<模板uuid>&<参数>`

| 参数 | 说明 |
|---|---|
| `template` | 使用的模板 uuid，缺省为默认模板 |
| `platform` | `windows` / `linux` / `android`；android 强制启用 tun |
| `tun` | 启用 tun 模式 |
| `mixed` | 启用 mixed inbound（本机代理入口） |
| `mixed-listen` / `mixed-port` | 覆盖 mixed 的监听地址 / 端口 |
| `sys-proxy` | 开启 mixed 的系统代理（`set_system_proxy`） |
| `api` | 启用 sing-box api service（gRPC，远程查看/控制） |
| `api-listen` / `api-port` | 覆盖 api service 的监听地址 / 端口 |
| `api-secret` | api service 的访问密钥 |
| `api-dashboard=false` | 关闭内置 Dashboard（默认随模板开启） |

生成时至少需要一个 inbound（tun / mixed 之一，模板自带常驻 inbound 除外）。

## 模板约定

模板即 sing-box 配置 JSON，以下约定会被 Web UI / URL 生成页使用：

- `inbounds` 的**第一个**作为默认模式，URL 生成页默认勾选它；
- 建议同时配置 `mixed` 与 `tun` 两种 inbound，便于随时切换（Android 强制 Tun）；
- `outbounds` 的 tag 支持按关键词自动填充订阅节点：

  | tag 写法 | 效果 |
  |---|---|
  | `组名@all` | 组包含全部节点 |
  | `组名@keywords=关键词A,关键词B` | 仅包含名称命中任一关键词的节点 |
  | `组名@exclude=关键词C` | 排除名称命中任一关键词的节点 |
  | `组名`（无 @） | 原样保留，需自行维护 outbounds 列表 |

节点数据转换目前支持 clash 订阅中的常用协议（shadowsocks / vmess / vless / trojan / anytls）。

## 数据目录结构

```
working_dir/
├── providers/<uuid>/   # 节点配置来源
│   ├── name url / file_name message
│   └── current last old   # 订阅内容滚动保留 3 份
└── templates/<uuid>/   # 模板
    ├── name default(默认标记)
    └── current last old
```

## 开发

前端为 Vue3 + Vite，前后端分离开发：

```bash
# 后端（先启动，供前端代理）
go run . serve -d ./working_dir -p 9112

# 前端开发服务器（另开终端，/api、/config 代理到 127.0.0.1:9112）
cd frontend
pnpm install
pnpm dev            # http://localhost:5173
```

生产构建（前端产物输出到 `webui/dist`，随 Go 二进制 embed）：

```bash
cd frontend
pnpm build          # pnpm build:web && pnpm build:go（构建 webui/dist 并编译 ../sbctl）
```
