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

当前版本以 [VERSION](VERSION) 为准，独立版本号从 `0.0.1` 开始。语音、AI 照片识别、助手修改/删除现有记录尚未实现；修改和删除仍使用常规界面。

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

## 微信小程序

原生小程序源码位于 [`miniprogram/`](miniprogram/README.md)。与网页共用 Go 服务、集合权限和 `shared/household.ts` 物品 API；支持微信快捷登录及绑定已有账号。微信登录默认关闭，注册小程序后按接入说明配置 AppID、服务端 AppSecret 和 HTTPS 合法域名。
