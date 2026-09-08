# sing-box-ctl (sbctl)

基于 Web UI 的 sing-box 配置管理工具：管理节点配置来源（订阅/上传）与模板，实时生成可用的 sing-box 配置，**URL 即配置**——把配置参数拼进 URL，任何设备导入即用。

## 特性

- 内置 Web UI：Go 服务托管前端构建产物（Vue3 + Vite），无外部运行时依赖
- **Provider 管理**：订阅 URL 或本地文件上传作为节点来源；内容经解析后**只保留节点**并转为 sing-box outbounds 入库；支持一键"更新/上传"拉取最新内容
- **模板管理**：JSON 编辑器（Monaco，带 sing-box schema 补全/校验），多模板并存、可设默认
- **默认 Provider / 模板**：Provider 与模板各自支持默认项，URL 页打开自动选中
- **历史版本**：内容滚动保留 3 份（current/last/old），可还原任意历史版本
- **实时配置生成**：`/config/<provider-uuid>?<参数>` 将模板骨架 + 订阅节点合成 sing-box 配置
- 模式开关：`tun` / `mixed`（含系统代理）、sing-box api service（gRPC + Dashboard）
- **按关键词自动分组**：模板 outbound 支持 `组名@表达式` 语法，自动填充订阅节点
- 平台适配：windows / linux / android（Android 强制 Tun）
- 存储集中化：`working_dir` 下两个 `data.json` 管理全部数据（+ `.bak` 备份），变更延迟节流统一落盘
- 暗色/亮色主题（跟随系统可切换）+ 响应式布局

## 快速开始

依赖：Go 1.24+、Node.js + pnpm。

```bash
# 首次安装前端依赖（有缓存，之后一般无需再跑）
make install

# 一键构建：构建前端产物（webui/dist，供 go:embed）并编译根目录 sbctl
make build

# 启动（-d 指定工作目录，必填；首次启动自动初始化数据目录）
./sbctl serve -d ./working_dir

# 浏览器打开 https://127.0.0.1:9112
```

> 前端产物目录 `webui/dist` 不入库，每次改动前端后需重新 `make build`。
> 生成的配置面向 **sing-box ≥ 1.14.0**（services.api / Dashboard / http_clients 等新结构）。

### 命令行

```bash
sbctl serve -d <working_dir> [--listen <host>] [-p <port>] [--cert-file <crt> --key-file <key>]
# -d        工作目录（必填）
# --listen  监听地址，默认 127.0.0.1（仅本机）；局域网访问需 --listen 0.0.0.0
# -p        端口，默认 9112
# 证书（https）：--cert-file / --key-file 成对提供
```

### working_dir/config.json（可选）

监听与证书也可写入工作目录下的 `config.json`，生效优先级：**默认值 < config.json < 命令行参数**：

```json
{
  "listen": "0.0.0.0",
  "port": 9112,
  "certificate_file": "certs/server.crt",
  "certificate_key_file": "certs/server.key"
}
```

证书相对路径以 working_dir 为基准解析。未配证书时以 http 运行；配置后自动启用 https。

## 数据目录结构

```
working_dir/
├── config.json          # 监听/端口/证书（可选）
├── providers/
│   ├── data.json        # provider 全部数据
│   └── data.json.bak    # 保存前自动备份的上一个版本
└── templates/
    ├── data.json        # 模板全部数据
    └── data.json.bak
```

- `data.json` 由程序统一读写（变更延迟 10s 节流合并落盘，保存前备份 `.bak`，损坏自动回退）
- 老版本的散目录结构（`providers/<uuid>/`、`templates/<uuid>/`）在升级后**首次启动会自动迁移**进对应 `data.json`，确认无误后旧目录可删除

### providers/data.json

```json
{
  "default_provider": "<uuid>",
  "providers": [
    {
      "uuid": "...",
      "name": "...",
      "url": "...",
      "source": "url | upload",
      "message": "备注（可选）",
      "file_name": "上传文件名（upload 来源）",
      "nodes": [ { "data": [ "sing-box outbounds..." ] } ]
    }
  ]
}
```

- `nodes` 最多 3 槽（下标 0=当前 / 1=上次 / 2=最旧），**更新或上传一次存一个新版本**
- `data` 是解析后的 **sing-box 节点 outbounds**（JSON 数组）；订阅原文不入库
- 不认识的协议节点会被跳过；若订阅解析不出任何有效节点，创建/更新会报错

### templates/data.json

```json
{
  "default_template": "<uuid>",
  "templates": [
    { "uuid": "...", "name": "...", "created_at": "...", "nodes": [ { "data": "模板 JSON" } ] }
  ]
}
```

- 模板内容为 sing-box 配置 JSON 原文（读取时统一 2 空格缩进返回编辑器）
- `nodes` 最多 3 槽，**每次保存算一个新版本**
- **内置种子模板不落盘**：仅作为"新建模板"的来源（from=builtin），用户模板才写入该文件

## 使用流程

1. **Provider 管理页**：添加来源——URL 模式填订阅地址（保存时自动下载并解析入库）；upload 模式选择配置文件直接上传（一步创建，解析失败不会留下空条目）。卡片支持设为默认 / 编辑 / 更新或上传 / 版本还原 / 删除。
2. **模板页**：模板是 sing-box 配置骨架（dns/route/rule/inbounds 等）。从内置默认或已有模板"新建"，Monaco 内直接编辑保存（Ctrl-S）；工具栏支持设为默认 / 格式化 / 版本还原 / 删除（默认模板不可删）。
3. **URL 页**：选择 Provider 与模板（打开时自动选中默认项）与功能开关，生成/复制配置 URL：
   - 复制后在 sing-box 客户端导入，或浏览器直接打开查看生成的 JSON；
   - 也可粘贴已有 URL "导入"，自动回填页面状态。

## 配置生成参数（URL 页按勾选项生成）

`/config/<provider-uuid>?template=<模板uuid>&<参数>`

| 参数 | 说明 |
|---|---|
| `template` | 使用的模板 uuid，缺省为默认模板 |
| `platform` | `windows` / `linux` / `android`；android 强制启用 tun |
| `tun` | 启用 tun 模式 |
| `mixed` | 启用 mixed inbound（本机代理入口） |
| `mixed-listen` / `mixed-port` | 覆盖 mixed 的监听地址 / 端口 |
| `sys-proxy` | 开启 mixed 的系统代理 |
| `api` | 启用 sing-box api service（gRPC） |
| `api-listen` / `api-port` | 覆盖 api service 监听地址 / 端口 |
| `api-secret` | api service 访问密钥 |
| `api-dashboard=false` | 关闭内置 Dashboard（默认随模板开启） |

生成配置至少需要一个 inbound（tun / mixed 之一；模板自带常驻 inbound 除外）。

## 模板约定

模板即 sing-box 配置 JSON，约定会被 URL 生成页使用：

- `inbounds` 的**第一个**作为默认模式（URL 页默认勾选），建议同时配置 `mixed` 与 `tun` 便于切换；
- `outbounds` 的 tag 支持按关键词自动填充订阅节点：

  | tag 写法 | 效果 |
  |---|---|
  | `组名@all` | 组包含全部节点 |
  | `组名@keywords=关键词A,关键词B` | 仅包含名称命中任一关键词的节点 |
  | `组名@exclude=关键词C` | 排除名称命中任一关键词的节点 |
  | `组名`（无 @） | 原样保留，需自行维护 outbounds 列表 |

订阅节点解析目前支持 clash 订阅常用协议：shadowsocks / vmess / vless / trojan / anytls。

## 开发

```bash
# 前端单独开发服务器（后端需先起一个实例）
cd frontend
pnpm install
pnpm dev

# 类型检查
pnpm typecheck

# 后端测试
go test ./...
```

前端为 Vue3 + Vite + open-props + Monaco：
- 页面：`frontend/src/views/`（ProviderView / TemplateView / UrlView）
- 组件库：`frontend/src/components/ui/`（Btn / Card / Field / Select / Toast），样式 token 集中在 `styles/tokens.css`
- 状态与 API：`composables/useApi.ts`、`useTheme.ts`、`useToast.ts`
- 构建：`make build`（前端产物输出 `webui/dist` 供 Go embed）；仅前端 `make frontend`，仅后端 `make go`

后端结构：`webui/`（HTTP 服务）、`provider/`（ProviderManager / TemplateManager：内存态数据 + data.json 存取）、`converter/`（clash 解析与 sing-box 合成）、`throttle/`（延迟执行器）、`settings/`（URL 参数应用）。

## 待办与已知问题

> 细节 TODO 已按“就近记录”原则写在各实现处的注释里，这里只列概览。

**已完成主线**：Provider/模板两级模型与默认项、HTTPS/config.json、历史版本还原、视觉还原与整体美化（组件库 + tokens 设计规范 + Toast 通知 + 暗色/响应式）、后端存储重构（两个 data.json + 节流保存 + 旧数据迁移）。

**遗留事项（见代码注释定位）**：
1. 退出时的数据 flush（避免最后 10s 变更丢失）—— `provider/provider_manager.go`、`provider/template_manager.go`（`SaveNow` 处）
2. 延迟保存失败留痕 —— 同上两文件 `scheduleSave` 回调
3. 订阅下载可控性（超时 / User-Agent）—— `provider/provider.go` `DataFromSource`
4. 未知协议跳过数量的前端提示 —— `converter/converter.go` `convertNodes`
5. Monaco 控制台偶发 `ICodeLensCache` 内部告警 —— `frontend/src/composables/useMonaco.ts`
