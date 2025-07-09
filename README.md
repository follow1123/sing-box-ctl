## Sing-box 辅助工具

### 特性

- 开启或关闭 web ui
- 模式切换：`tun` `mixed` `mixed(with system proxy)`
- 订阅转换（支持 clash 转 sing-box 的部分协议）
- 订阅更新
- 配置生产服务（用于内网设备获取转换后的配置）

### 安装

### 手动构建

#### 依赖

- [go 1.24.4](https://go.dev/)

```bash
go build -o sbctl
```

### 使用 `sytsemd` 管理服务

```toml
[Unit]
Description=Sing-box configuration generator and subscription converter

[Service]
ExecStart=/usr/local/bin/sbctl serve
```
