# dsh-work 知识库换票

**状态：** 部分实现（换票与绑定 API 已落地，待联调验收）  
**确认日期：** 2026-10-29  
**分类：** 给已登录的 dsh-work 员工换发 WeKnora 用户会话（服务端对服务端）  
**范围：** 两个内网接口。日常查阅的页面在 dsh-work，不在本仓库做新页面。

实施步骤见 [26-10-29 dsh-work 知识库换票](../实施计划/26-10-29%20dsh-work%20知识库换票.md)。

dsh-work 侧的页面、导航与员工端 API 见该仓库 `docs/SDD/规格文档/26-10-29 知识库页面.md`。

## 1. 要达成的事

dsh-work 的服务进程在确认员工身份之后，向 WeKnora 换一张**该员工本人**的访问令牌，再代为调用现有知识库 API。令牌的可见库、文档与预览权限与该员工在 WeKnora 网页里登录后相同。

成功时：

- 目录里已有的 ai-hub 身份可以换票，无需密码。
- 目录里没有的人不会因此被自动开户。
- 仅有 dsh-work 本地账号的员工，可以在本人登录 WeKnora 之后，把一次性绑定码交回 dsh-work。dsh-work 再用绑定码换到用户标识。
- 集成密钥泄露不能绕过「是哪个用户」：没有用户标识或绑定码时，密钥本身读不到任何知识库。

## 2. 现状

目录同步把 ai-hub 的 `user_id` 与 `subject` 记在 `user_directory_identities`。企业登录按 `subject`、主邮箱和别名查找**已经存在**的账号。找不到则失败，不新建用户。

现有 API Key 的 `signed_token` 会变成 `api_external_user`，不是网页用户，不能复用为本期换票。

知识库列表、文档、检索、文件预览已存在，本期不改它们的权限规则。

## 3. 不做什么

- 不新增员工可打开的页面，不改现有登录页的注册开关。
- 不接受 dsh-work 传来的密码。
- 不根据姓名合并账号。
- 不在换票时创建用户、个人空间或公司空间成员。
- 不改 `/mcp`、`/mcp/:endpoint_id` 与 `mcp_` 令牌。
- 不把集成密钥当作某个空间的 API Key 去列出该空间的全部库。

## 4. 谁可以调用

两个接口都要求请求头：

`Authorization: Bearer <集成密钥>`

密钥来自环境变量 `DSH_WORK_INTEGRATION_SECRET`。未设置时两个接口都返回 404，表现与未部署该能力相同。密钥不匹配为 401。

调用方应是 dsh-work 服务，不把这两个地址交给浏览器。绑定登录页例外：那一页由员工的浏览器打开，但签发绑定码的接口仍要集成密钥，由 dsh-work 服务在跳转前调用。

## 5. 换票

`POST /api/v1/integrations/dsh-work/sessions`

正文 JSON（snake_case），`subject` 与 `ai_hub_user_id` 至少有一个非空：

```json
{ "subject": "<oidc subject>", "ai_hub_user_id": "<ai-hub user_id>" }
```

查找顺序：

1. `ai_hub_user_id` 对上目录身份时用该用户。
2. 否则 `subject` 对上目录身份时用该用户。
3. 都没有：404，正文 `{"error":"account_not_provisioned"}`。不创建用户。

找到的用户已停用：403，`{"error":"account_disabled"}`。

成功 200：

```json
{
  "user_id": "<weknora user id>",
  "tenant_id": 1,
  "access_token": "<与网页登录相同种类的访问令牌>",
  "expires_in": 3600
}
```

`tenant_id` 是该用户的个人空间。`access_token` 能调用现有 `/api/v1` 知识库接口，权限与该用户在这个空间内的网页会话相同。令牌有效期不超过 1 小时。

不返回刷新令牌。dsh-work 过期后重新调用本接口。

## 6. 绑定码

给 dsh-work 本地账号或目录尚未对上的员工使用。本人必须先在 WeKnora 完成现有登录。

### 6.1 开始

`POST /api/v1/integrations/dsh-work/bind-codes`

请求头为集成密钥。正文：

```json
{
  "redirect_uri": "https://<dsh-work 主机>/api/workbench/v1/integrations/weknora/bind/callback",
  "state": "<dsh-work 生成的不透明字符串>"
}
```

`redirect_uri` 必须落在环境变量 `DSH_WORK_BIND_REDIRECT_ALLOWLIST`（逗号分隔的完整 URL）中。不在名单内为 400，`{"error":"redirect_not_allowed"}`。

成功 200：`{"url":"<本站登录地址，带一次性 bind 参数>"}`。该参数 10 分钟有效，与 `state`、`redirect_uri` 绑在一起。

### 6.2 员工登录之后

员工打开 `url` 并完成现有登录（密码或该部署已启用的企业登录）。登录成功后 WeKnora 生成一次性 `bind_code`（最多 10 分钟，用后即废），302 到：

`<redirect_uri>?state=<原 state>&bind_code=<code>`

登录失败或员工取消时，302 到 `<redirect_uri>?state=<原 state>&bind_error=access_denied`，不签发 `bind_code`。

未登录的人不能靠集成密钥单独换到 `bind_code`。

### 6.3 兑现

`POST /api/v1/integrations/dsh-work/bind-codes/exchange`

请求头为集成密钥。正文：`{"bind_code":"<code>"}`。

成功 200：`{"user_id":"<刚登录的 weknora user id>"}`。  
未知、过期或已使用：400，`{"error":"bind_code_invalid"}`。

兑现只返回 `user_id`，不返回访问令牌。dsh-work 只持久化 `user_id`。之后一律用第 5 节换票。没有目录身份的用户在第 5 节改用下面这种正文：

```json
{ "user_id": "<weknora user id>" }
```

仅当该用户存在且未停用时签发与第 5 节相同的会话。未知用户 404，`{"error":"account_not_provisioned"}`。此正文仍然要集成密钥，不能由浏览器直接调用。

这样本地绑定用户在令牌过期后仍能换票，而 dsh-work 不必保存 WeKnora 令牌。

## 7. 验收

1. 密钥缺失时，换票与开始绑定都是 404。
2. 密钥错误为 401。
3. 目录中的 `subject` 或 `ai_hub_user_id` 能换到该用户的令牌；用该令牌调用现有知识库列表，结果与该用户网页登录后一致。
4. 目录中不存在的 `subject` 得到 `account_not_provisioned`，用户表行数不增加。
5. 停用用户为 `account_disabled`。
6. 允许名单外的 `redirect_uri` 被拒绝。名单内的地址在本人登录后带回 `bind_code` 与原来的 `state`。兑现一次成功，第二次失败。
7. 只持有集成密钥、不持有 `subject`、`ai_hub_user_id`、`user_id` 或有效 `bind_code` 时，不能列出任何知识库。
