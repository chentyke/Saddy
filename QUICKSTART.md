# 🚀 Saddy 快速启动指南

感谢使用 Saddy！本指南将帮助你快速上手。

## 📦 文件结构

解压后你会看到以下文件：

```
saddy-xxx/
├── saddy-xxx          # 可执行文件 (Linux/macOS) 或 saddy-xxx.exe (Windows)
├── README.md          # 详细文档
├── LICENSE            # 许可证
├── CHANGELOG.md       # 更新日志
├── configs/           # 配置文件目录
│   ├── config.yaml           # 当前配置 (可能不存在)
│   └── config.yaml.example   # 配置示例
└── web/               # Web 管理界面文件
    ├── static/        # 静态资源 (CSS, JS)
    └── templates/     # HTML 模板
```

## ⚡ 快速开始

### 1️⃣ 配置 Saddy

首先复制配置示例文件：

```bash
# Linux/macOS
cp configs/config.yaml.example configs/config.yaml

# Windows (PowerShell)
Copy-Item configs\config.yaml.example configs\config.yaml
```

### 2️⃣ 编辑配置文件

打开 `configs/config.yaml` 进行配置：

```yaml
server:
  host: "0.0.0.0"
  port: 80                # HTTP 端口
  https_port: 443         # HTTPS 端口
  admin_port: 8081        # 管理后台端口
  auto_https: false       # 是否启用自动 HTTPS

# Web 管理界面
web_ui:
  enabled: true
  username: "admin"       # 修改为你的用户名
  password: "admin123"    # 修改为你的密码

# 代理规则示例
proxy:
  rules:
    - domain: "example.com"
      target: "http://localhost:3000"
      cache:
        enabled: true
        ttl: 300
      ssl:
        enabled: false
```

### 3️⃣ 运行 Saddy

**Linux/macOS:**

```bash
# 添加执行权限
chmod +x saddy-*

# 运行（使用默认配置路径）
./saddy-*

# 或指定配置文件
./saddy-* -config configs/config.yaml
```

**Windows (PowerShell):**

```powershell
# 运行
.\saddy-windows-amd64.exe

# 或指定配置文件
.\saddy-windows-amd64.exe -config configs\config.yaml
```

### 4️⃣ 访问管理界面

服务启动后，访问 Web 管理界面：

```
http://localhost:8081
```

使用配置文件中设置的用户名和密码登录。

## 🔧 常见配置

### 配置反向代理

在配置文件中添加代理规则：

```yaml
proxy:
  rules:
    - domain: "app.example.com"
      target: "http://localhost:3000"
      cache:
        enabled: true
        ttl: 300
        max_size: "100MB"
      ssl:
        enabled: true
        force_https: true
```

### 启用自动 HTTPS

```yaml
server:
  auto_https: true
  https_port: 443

proxy:
  rules:
    - domain: "yourdomain.com"
      target: "http://localhost:3000"
      ssl:
        enabled: true
        force_https: true
```

**注意：** 
- 需要确保域名已解析到服务器
- 需要开放 80 和 443 端口
- Let's Encrypt 会自动获取和续期证书

### 配置缓存

```yaml
cache:
  enabled: true
  storage: "memory"      # memory 或 file
  max_size: "1GB"
  cleanup_interval: "10m"
  file_cache:
    directory: "./cache"
```

## 🐳 使用 Docker

如果你更喜欢 Docker，可以使用官方镜像：

```bash
docker pull chentyke/saddy:latest

docker run -d \
  --name saddy \
  -p 80:80 \
  -p 443:443 \
  -p 8081:8081 \
  -v $(pwd)/configs:/app/configs:ro \
  -v saddy-certs:/app/certs \
  chentyke/saddy:latest
```

## 🔐 安全建议

1. **修改默认密码：** 首次运行前务必修改 `web_ui.password`
2. **限制管理端口访问：** 建议使用防火墙限制 8081 端口只能内网访问
3. **使用 HTTPS：** 生产环境建议启用 `auto_https`

## 📖 更多帮助

- **完整文档：** 查看 README.md
- **API 文档：** http://localhost:8081/api/v1/
- **GitHub Issues：** https://github.com/chentyke/saddy/issues

## 🆘 故障排查

### 端口被占用

如果启动时提示端口被占用，修改配置文件中的端口号：

```yaml
server:
  port: 8080        # 改为其他未占用端口
  admin_port: 8082
```

### 无法访问管理界面

1. 检查服务是否正常启动
2. 确认防火墙是否开放相应端口
3. 检查配置文件中 `web_ui.enabled` 是否为 `true`

### Web 界面样式异常

确保 `web/` 目录完整，包含：
- `web/static/app.js`
- `web/static/style.css`
- `web/templates/index.html`
- `web/templates/login.html`

### 权限问题 (Linux/macOS)

```bash
# 确保二进制文件有执行权限
chmod +x saddy-*

# 如果需要绑定 80/443 端口，可能需要 sudo
sudo ./saddy-*
```

## 💡 提示

- 配置修改后需要重启 Saddy 才能生效
- 建议使用系统服务管理器（如 systemd）来管理 Saddy 进程
- 生产环境建议配置日志输出到文件

---

**享受使用 Saddy！** 如有问题，欢迎提交 Issue。


