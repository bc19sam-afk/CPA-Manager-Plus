# CPA Panel 模式

CPA Panel 模式适合继续从 CPA 端口打开面板的环境。浏览器持有 CPA Management Key，面板由 CPA 托管；如果你需要完整的历史监控、模型价格、导入导出和服务端巡检，优先使用完整 Docker 或原生 Manager Server 模式。

## 和完整 Docker 模式的区别

| 模式 | 面板托管 | 登录凭证 | 适用场景 |
|---|---|---|---|
| 完整 Docker | Manager Server `:18317` | `cmp_admin_...` 管理员密钥 | 独立部署 CPAMP。 |
| CPA Panel | CPA `:8317` | CPA Management Key | 希望继续从 CPA 访问面板。 |
| 前端开发 | Vite dev 或静态 HTML | 浏览器本地 CPA URL + key | 本地开发和调试。 |

## 注意事项

- CPA Panel 模式使用 CPA Management Key 登录，不需要 CPAMP 管理员密钥。
- CPA Management Key 保存在浏览器侧，符合 CPA 托管面板的访问方式。
- 完整 Docker 模式会把 CPA Management Key 加密后保存到 SQLite。
- 面板入口相同，但可用数据取决于托管模式；完整历史监控、模型价格和服务端巡检来自 Manager Server 模式。

## 何时使用

选择 CPA Panel 模式：

- 已经习惯从 CPA 端口访问管理面板。
- 不想让用户直接访问 Manager Server 面板端口。
- 希望 CPA Management Key 继续作为面板访问凭证。

选择完整 Docker 模式：

- 希望 CPAMP 独立托管。
- 希望 Manager Server 统一保存配置。
- 需要管理员密钥和 server-side 加密保存 CPA Management Key。
