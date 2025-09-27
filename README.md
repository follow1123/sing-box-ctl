## Sing-box 辅助工具

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

# 获取配置并启动
sbctl provider fetch -r
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

#### 服务管理

```bash
# 启动、停止、重启服务
sbctl [start|stop|restart]
```

---

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

# 获取配置
sbctl provider fetch

# 获取配置并重启服务
sbctl provider fetch -r

# 恢复订阅配置（用于恢复自己修改后的配置）
sbctl provider restore
```

---

#### 配置更新

> 详细参考 `sbctl update -h`

```bash
# 开启 mixed 模式下的局域网共享并重启
sbctl update --allow-lan -r

# 切换为 tun 模式并重启
sbctl update --tun -r

# 禁用 external controller (webui) 并重启
sbctl update --disable-webui -r

# 格式化配置
sbctl update --format
```

---

#### 其他

```bash
# 查看状态信息
sbctl status

# 共享配置
sbctl share

# 使用系统默认浏览器打开 web ui
sbctl webui
```

---

### 相关文件

配置文件 Windows 在：`%LOCALAPPDATA%/singboxctl`，Linux 在：`/etc/singboxctl`

Linux 下使用 `systemd` 管理 sing-box 服务，名称为 `sing-box.service`，服务文件存放在 `/etc/systemd/system/sing-box.service`

---

### 可选配置

#### 集成 Windows 右键菜单

> Windows 10 下测试可用，Windows 11 未测试

```bash
# 初始化右键菜单
sbctl rcmenu --init
```
