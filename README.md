<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->

# 在哪儿 · Zainaer

别问我在哪儿，问“在哪儿”在哪儿。

Zainaer 是面向家庭的自托管物品定位工具：记录物品是什么、放在哪里，以及如何快速找到。它基于 [HomeBox](https://github.com/sysadminsmedia/homebox) 改造，现以本仓库的 **`main`** 为独立产品基线维护，不再与上游合并或同步。

[仓库](https://github.com/Phoobobo/zainaer) · [问题反馈](https://github.com/Phoobobo/zainaer/issues) · [许可证](LICENSE)

## 功能

- **家庭物品记录**：名称、标签、照片、描述和存放位置，收起购买、保修、销售和维护等库存管理信息。
- **多级位置**：按“客厅 → 餐边柜 → 左下第三个抽屉”建立位置层级，无需使用收纳箱。
- **AI助手**：多轮对话查找物品，显示真实记录和完整位置路径；准备新增物品、位置和标签。
- **确认后入库**：模型只生成草稿。用户点击确认后，服务端重新校验并保存；草稿绑定用户与集合，10 分钟过期，防止重复提交。
- **服务端 BYOK**：集合所有者管理提供商、主模型和两个备选模型。支持 OpenRouter、阿里云百炼（北京）和 DeepSeek。
- **故障切换**：主模型请求失败、超时或返回空响应时，依次尝试备选模型；共用所选提供商与 API Key，不因回答质量触发重试。
- **中英双语、移动端布局**，以及创建物品、位置、标签的悬浮入口。
- **家庭集合与邀请**：数据按集合隔离；模型查询使用当前登录用户的集合权限。

当前版本以 [VERSION](VERSION) 为准，独立版本号从 `0.0.1` 开始。网页文字助手新增实验性设备本地语音输入：需 HTTPS、浏览器支持强制本地识别，且已安装中文或英文语言包。可开始、停止转写或取消；结果先进入可编辑输入框，需手动发送，操作仍须卡片确认。不会回退云端转写或自动安装语言包，不支持时继续打字。切页、隐藏页面、关闭面板会取消录音，单次最多 60 秒。AI 照片识别和助手删除尚未实现。

## 模型配置

登录后进入 **集合 → 模型配置**，选择提供商，填写完整模型 ID 与 API Key，保存后测试连接。OpenRouter 的模型 ID 需要带厂商前缀；备选模型可留空。模型必须支持工具调用，费用及免费额度以提供商规则为准。

密钥使用 AES-GCM 加密，保存在服务端存储目录的 `ai/` 子目录，文件权限为 `0600`；页面只返回是否已配置，不回显密钥。**备份必须同时保留 `ai/*.json` 和 `ai/master.key`**。主密钥与密文保存在同一服务端存储目录，加密不能替代服务器和备份的访问控制。

助手会将对话和本次工具查询返回的记录发送给配置的模型提供商。模型不持有用户登录凭据，不执行 shell；新增记录必须经过独立确认接口。对话历史目前仅保留在当前页面内存中，刷新后清空；待确认草稿在服务重启后失效。

## 开发与运维

构建、验证、部署和备份说明见 [AGENTS.md](AGENTS.md)。

## 来源与许可证

感谢 [HomeBox / sysadminsmedia](https://github.com/sysadminsmedia/homebox) 及其贡献者。本项目保留相关历史与归属，整体及继承代码遵循 [GNU AGPL-3.0](LICENSE)。明确标注的 Zainaer 原创文件另按宽松的 [MIT](LICENSE-MIT) 许可提供，授权边界见 [LICENSING.md](LICENSING.md)；这不是将整个项目改为 MIT。既有上游文档和自动化可能仍包含 HomeBox 的部署方式或链接，不应视为 Zainaer 的官方发布配置。

---

**English:** Zainaer is a self-hosted household item locator, independently maintained from HomeBox on `main`. It offers nested locations, photos and tags, Chinese/English UI, server-side BYOK with two fallback models, and an AI assistant whose creation drafts require explicit user confirmation. The existing Go module paths and `HBOX_*` settings remain for compatibility. The combined application remains AGPL-3.0; explicitly marked original files are available under MIT. See [license scope](LICENSING.md).

## 通行密钥登录

保留邮箱和密码登录。用户先用密码登录，在 **个人设置 → 通行密钥** 输入当前账号密码，为现有账号添加密钥；之后可在登录页使用指纹、面容、设备 PIN 或安全密钥完成验证。新账号仍通过原注册流程创建。支持查看、删除已绑定密钥，不改变集合或物品数据。服务端只保存公钥凭据，不接收指纹或面容数据。

配置 `HBOX_PASSKEY_ORIGIN=https://your-domain.example` 后启用，留空则关闭。必须是固定的 HTTPS origin（不带路径），RP ID 使用该域名；不会信任请求 Host 或代理头来决定验证来源。本地测试允许 `http://localhost:<port>`。更换域名会影响已有密钥，迁移前确保密码可用。

浏览器/系统需支持 WebAuthn。微信内置浏览器等环境不保证支持，失败时使用保留的密码入口，或到系统浏览器打开网站。密钥能否跨设备同步取决于用户的系统和密码管理器。移除网站中的密钥不会自动删除密码管理器的副本。请保留密码及可用的账号恢复方式。

验证挑战保存在服务端内存，5 分钟过期、一次性使用，注册绑定当前登录会话并要求密码验证。应用重启后未完成的请求需重新发起；当前部署为单实例。通行密钥数据随数据库备份。原小程序源码和微信登录接口已移除，已发布的数据库迁移保留以支持升级。

开发验证：`scripts/check-passkeys.sh` 检查协议验证、账号权限、迁移和浏览器适配器类型。对独立的临时实例设置 `HBOX_PASSKEY_ORIGIN=http://localhost:17745`，可在 frontend 目录运行 `PASSKEY_TEST_ORIGIN=http://localhost:17745 node test/passkeys.e2e.cjs`，使用 Playwright Chromium 虚拟验证器验证绑定、登录、删除及密码回退。脚本会创建测试账号，仅允许 localhost，不能用于生产。
