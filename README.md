## Sing-box 配置辅助工具

### 特性

- external controller(Web UI) 相关配置
- 模式切换：`tun` `mixed`
- `mixed` 下的系统代理、局域网共享开关
- 订阅转换（支持 clash 转 sing-box 的部分协议）
- 配置共享，将转换后的配置使用 http 接口提供给内网的其他设备

---

### 快速开始

```bash
# 设置订阅信息
sbctl provider add <name> <your_provider_sub_url>

# 获取并转换配置
sbctl provider fetch
```

---

### 安装

### 手动构建

#### 依赖

- [go 1.24.4](https://go.dev/)

```bash
make

# 或（没有 make 命令）
go build -o sbctl
```

---

### 功能

#### `provider` 子命令

```bash
# 添加订阅链接信息
sbctl provider add <name> <your_provider_sub_ulr>

# 添加时设置为默认
sbctl provider add <name> <your_provider_sub_ulr> -d
# 或
sbctl provider update <name> -d

# 更新订阅
sbctl provider update <name> <new_url>

# 删除
sbctl provider delete <name>

# 获取并转换配置
sbctl provider fetch

# 恢复订阅配置（用于恢复自己修改后的配置）
sbctl provider restore
```

---

#### 配置更新

> 详细参考 `sbctl update -h`

```bash
# 开启 mixed 模式下的局域网共享
sbctl update --enable-proxy-sharing

# 切换为 tun 模式
sbctl update -t

# 禁用 external controller (webui)
sbctl update -W

# 格式化配置
sbctl update --format
```

---

#### 其他

```bash
# 共享配置
sbctl share

# 从上次下载的订阅配置恢复到配置文件（手动修改配置后可使用这个命令恢复）
sbctl restore
```

---

### 相关文件

配置文件 Windows 在：`%LOCALAPPDATA%/singboxctl`，Linux 在：`$HOME/.config/singboxctl`
