<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->
# 在哪儿 · 微信小程序

原生微信小程序入口，与网页连接同一个 Go 服务、数据库、物品、照片、集合和AI助手。无需部署另一套后端。当前尚未注册小程序账号，仓库使用 `touristappid`，不会伪造微信身份或自动开启生产微信登录。

## 本地构建

先按仓库说明安装 `frontend` 依赖（Node.js 22、pnpm 10.28）：

```sh
pnpm --dir frontend install --frozen-lockfile
node miniprogram/scripts/build.cjs
node --test miniprogram/tests/*.test.cjs
```

微信开发者工具中导入 **miniprogram/**，项目配置会使用生成的 `dist/`。源码是 `src/`，不要直接编辑 `dist/`。未注册时可以查看源码和构建产物；游客 AppID 不能完成真实微信身份校验。本机当前没有微信开发者工具，尚未执行 WXML 编译、模拟器或真机测试。

## 注册后的配置

1. 将微信平台分配的 AppID 填入 `project.config.json` 的 `appid`（AppID 是公开标识）。
2. 创建不入库的 `miniprogram/src/config.local.js`：

   ```js
   module.exports = { apiOrigin: 'https://your-domain.example' };
   ```

   地址只填写 HTTPS origin，不带末尾 `/` 或 `/api/v1`。构建器仅将 `apiOrigin` 写入产物；本地配置文件不会打包。
3. 在小程序后台配置业务服务对应的 `request`、`uploadFile` 和 `downloadFile` 合法域名。保持开发者工具的域名校验开启。
4. **仅在服务端**配置并重启服务：

   ```sh
   HBOX_WECHAT_ENABLED=true
   HBOX_WECHAT_APP_ID=<同一个小程序的AppID>
   HBOX_WECHAT_APP_SECRET=<小程序AppSecret>
   ```

   不要把 AppSecret 写入小程序、Git、聊天或日志。服务器须能够连接 `https://api.weixin.qq.com`。现有 `HBOX_OPTIONS_ALLOW_REGISTRATION` 决定是否允许创建微信账号；关闭后已绑定用户仍可登录。
5. 重新构建、开发者工具预览，完成下述真机验证后再按微信平台流程提交审核。服务端的常规发布不会上传或发布微信小程序。

## 登录与已有数据

- 点击“微信快捷登录”：`wx.login` → 短期 code → Go 服务 `code2Session` → 现有应用 Bearer 会话。AppSecret 与 session_key 都不会返回客户端。
- 未绑定时，用户明确选择“创建账号”或“绑定已有账号”；不会静默创建空集合。
- 绑定已有账号时仅需输入一次已有邮箱密码，先完成原有登录，再使用新微信 code 绑定当前会话的用户。绑定后原账号、集合与物品不变，原密码登录和 OIDC 字段保留。
- 同一用户只能绑定一个微信身份，同一微信身份只能对应一个用户；不能通过再次绑定覆盖。暂不提供解绑/换绑界面。已有账号若只支持 OIDC，请先通过现有账号管理设置可用的本地登录方式，或后续增加网页授权绑定流程；不能在小程序直接执行 OIDC 页面跳转。
- 身份键按 AppID + OpenID 生成摘要，数据库有唯一约束。匿名请求不能指定用户、集合或 OpenID 建立登录态。微信身份不按昵称、邮箱或手机号自动合并。
- 退出登录调用原会话撤销接口；401 清除本地凭据并回到登录页。不会自动重试创建、修改或助手确认请求。

## 功能和复用范围

- 小程序：中英文、搜索/分页、物品与位置列表、详情/路径、创建/编辑/删除、标签选择、拍照/选图上传、集合切换/命名、接受家庭邀请、AI助手查询与草稿确认/取消。
- `shared/household.ts`：网页和小程序共同使用的物品 CRUD、查询类型、请求协议、类型选择规则；数据类型直接引用现有前端 API contracts。网页 `ItemsApi` 已委托到此模块，不是复制另一份实现。
- `src/lib/api.js`：只处理微信网络请求、存储、登录及上传适配。原后端继续处理权限、集合隔离、附件、模型设置、模型调用、草稿有效期和幂等确认。
- 新外部账号的集合和默认数据创建与 OIDC 共享 `registerExternalUser`，微信身份存独立字段，不占用 OIDC 身份槽。
- Nuxt 页面与 DOM 组件无法直接运行在原生小程序中，因此 WXML/WXSS 页面独立实现。没有用 web-view 包装网页，也没有引入第二套业务后端或跨端框架。
- 高级集合管理、模型配置和批量导入/导出仍使用已有网页。小程序头像/手机号授权、扫码功能不在此版本内。

## 验证

自动测试覆盖身份校验失败、敏感错误隐藏、伪造身份拒绝、注册开关、唯一绑定、不覆盖原账号、会话签发、客户端集合请求头、401 清理和不自动注册。CI 同时运行共享客户端测试、后端回归、迁移与网页构建。

注册后必须验证：真实微信首次创建/绑定、再次登录、退出/过期、两个用户隔离、切换集合、照片上传/显示、物品和位置编辑、助手草稿确认/取消、网络错误以及 iOS/Android 展示。现有自动测试不代表真机验证通过。

官方协议参考：[小程序登录](https://developers.weixin.qq.com/miniprogram/dev/framework/open-ability/login.html)、[wx.login](https://developers.weixin.qq.com/miniprogram/dev/api/open-api/login/wx.login.html)、[code2Session](https://developers.weixin.qq.com/miniprogram/dev/server/API/user-login/api_code2session.html)。
