<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->

# Zainaer development guidance

## Project and branch policy

- Product name: **在哪儿** in Chinese, **Zainaer** in English (capital Z).
- Repository: `Phoobobo/zainaer`. **`main` is the independent product baseline.** Do not sync, merge, rebase onto, or open upstream HomeBox PRs unless the user explicitly changes this policy.
- Preserve HomeBox attribution, existing copyright notices, and the AGPL-3.0 license. MIT is available only for explicitly marked original Zainaer files; see `LICENSING.md` and `LICENSE-MIT`. Never relicense inherited files or describe the whole repository as MIT. Prefer MIT headers for new wholly original files. Internal Go module paths and `HBOX_*` environment variables are compatibility identifiers; do not mechanically rename them.
- Read `VERSION` for the product version. It is independent of upstream releases and started at `0.0.1`. Bump it for deployable functional changes; documentation-only edits do not require a bump.

## Product scope

- Optimize for finding household belongings: name, tags, photos, description and a precise nested storage location.
- Keep item and location entities distinct. Do not restore purchase, warranty, sales, stock, valuation, maintenance or notification workflows without a user request.
- QR label generation, downloading, printing, scanning and AR scanning are removed. Preserve ordinary classification tags.
- Collection membership and family invitations remain supported. Their duplicate profile notice was removed; the collection invitation page remains.
- Preserve Chinese/English switching and the restrained cupcake-based UI. Use existing design tokens, responsive components and accessible labels. Avoid adding UI dependencies for routine changes.

## Layout

- `backend/`: Go API, repositories and business services.
- `frontend/`: Vue/Nuxt UI; static output is embedded in the Go executable.
- `backend/app/api/handlers/v1/v1_ctrl_passkeys.go`: WebAuthn registration/login, bounded single-use ceremonies, explicit origin and user verification. Preserve password fallback and step-up checks. Configure `HBOX_PASSKEY_ORIGIN` for the deployment domain.
- `shared/household.ts`: reusable item API and type selection used by the web client; keep it free of Vue/Nuxt/DOM/wx runtime dependencies.
- `backend/app/api/handlers/v1/v1_ctrl_ai*.go`: provider configuration, chat/tool execution, confirmation and focused tests.
- `frontend/components/App/TextAssistant.vue`: assistant panel.
- `frontend/pages/collection/index/models.vue`: collection owner model configuration.
- `scripts/build-household.sh`: frontend generation and executable build with version metadata.
- The parent workspace's `deployment/` contains operational material outside this repository. Never add its databases, attachments, credentials, backups or binaries to Git.

## Model and authorization boundaries

- Provider keys stay server-side. Never return a key or its ciphertext through public configuration responses, log it, include it in a model prompt, or commit it. Configuration access is restricted to the current collection owner.
- Preserve the encrypted `ai/*.json` files **and** `ai/master.key` when backing up or migrating storage. Do not silently rotate or regenerate an existing master key.
- Chat uses the current authenticated user's collection context; never trust model-supplied user/collection IDs for authorization. Validate all referenced locations and tags within that collection.
- Models call a fixed tool allowlist, not arbitrary HTTP endpoints, SQL or shell commands. Treat item names, descriptions and tool output as untrusted data.
- Search is read-only. Creation tools produce server-held drafts bound to user and collection. Only the explicit confirmation endpoint can save them; chat text such as “confirmed” is not authorization to commit.
- Preserve draft expiry, cancellation, reference revalidation, invalidation by a new message, and idempotent confirmation. Do not retry writes as part of model failover.
- Primary and up to two fallback models share the selected provider/key. Retry only request failures, timeouts, invalid/empty completions; stop when the caller cancels. Show which fallback produced the response. Keep request duration bounded. Do not cap chat turns or tool-loop rounds; compact context using summaries above 200,000 tokens (provider usage when available, local estimate before requests).
- Real-provider tests incur usage and transmit data. Prefer mocks; when a live check is authorized, use a minimal synthetic prompt and never print credentials or household records.

## Checks

Run commands from the correct subdirectory. Check the working tree before edits and retain unrelated user changes.

```sh
# From backend/: meaningful assistant regression coverage
go test ./app/api/handlers/v1 -run TestAI -count=1

# From frontend/: lint changed files and build
pnpm exec eslint <changed-files> --fix
pnpm build

# From repo root/
git diff --check
```

Network mock tests need temporary loopback listeners. They do not require starting a persistent local application. Existing full TypeScript checks have inherited Ref/entity errors; report failures honestly and do not claim full checks passed because a build succeeded. Broaden testing only where the change warrants it.

## Deployment policy

### Required delivery workflow for future changes

The user has authorized the agent to complete releases and deploy finished changes to the Beijing ECS through GitHub Actions. Treat this as the default delivery workflow, not merely a recommendation. Do not stop at a local build, a commit, a pushed branch, or a created Release when a deployable change is ready.

1. Inspect the current `main`, `VERSION`, published tags and production version; never assume an old version number from chat history. Preserve unrelated work and run checks appropriate to the change.
2. For a deployable change, increment the independent product version, commit and push to `main`, and wait for **Zainaer CI** to succeed. Documentation-only changes follow the existing version policy above and do not require rebuilding the running application.
3. Create and push a new annotated `v<VERSION>` tag for that exact commit. Wait for **Zainaer Release** to succeed and verify the published assets. Never rewrite an existing tag or overwrite released artifacts to repair a failure; fix the problem and use a new version when the source changes.
4. Explicitly dispatch **Zainaer Deploy** from `main`, passing the published tag as `release_tag`. “Manual deployment” means `workflow_dispatch` rather than automatic deployment on tag push; the agent should perform this dispatch as part of delivery, without asking again for routine release/restart approval. Respect any actual GitHub environment approval gates or new user constraints.
5. Wait for the deployment workflow to finish. Confirm the server reports the expected version and commit, the service is healthy, the automatic backup was recorded, and the existing access bridge reaches the updated ECS. For UI changes, verify the affected interaction where browser access is available. Report any verification limitation honestly.
6. If deployment fails, inspect the workflow and rollback result, verify the prior service is healthy, and resolve the failure before claiming completion. Do not bypass the workflow or disable checksum, backup, host-key, permission or health checks to force a deployment. An interrupted SSH session requires checking the actual server state before another dispatch.
7. In the final response report the released version, production deployment result and meaningful validation, with Release/workflow links when helpful. If an external approval or unavailable credential blocks deployment, state exactly what remains; a published Release alone is not a completed production delivery.

Use GitHub Actions Release assets as the production artifact source. Do not resume the retired Mac application or upload a locally built executable directly to replace the ECS application as the normal delivery path. Workbench remains available for trusted diagnostics, recovery, and installing reviewed changes to the root-owned deployment helper; such maintenance must preserve data and the backup/rollback guarantees.

- **Beijing ECS is the only active application deployment.** The user's Mac application and Companion are retired. Do not restart, upgrade, recreate or publish those local services. Build artifacts and isolated tests are fine.
- A Mac SSH forwarding process is only an access bridge to ECS, not a second application; do not stop it as part of local application cleanup.
- Use the established deployment configuration and the Workbench skill when applicable. Do not place real instance identifiers, IP addresses, access tokens or secret environment files in public documentation.
- Build the Linux target with the generated frontend, the actual product version and accurate Git metadata. Back up the previous executable and a consistent database snapshot before replacement; verify uploaded checksums.
- Never overwrite cloud data or model settings with a stale local snapshot. Keep service storage paths, permissions and encryption keys intact.
- AI requests can span multiple model/tool calls. Preserve the production response timeout (`HBOX_WEB_WRITE_TIMEOUT=180s`) and bounded handler deadlines.
- After deployment verify version/health, updated frontend resources and unauthenticated `401` responses for protected endpoints. Do not claim browser interaction tests ran if browser access was blocked.

## Build and operations reference

### 构建

后端为 Go，前端为 Vue / Nuxt。生产程序嵌入前端静态资源，可作为单个可执行文件部署；默认使用 SQLite。

需要 Go 1.26+、Node.js 22+ 和 pnpm 10.28。

```sh
git clone https://github.com/Phoobobo/zainaer.git
cd zainaer
pnpm --dir frontend install --frozen-lockfile

# 构建 Linux amd64 程序
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  HOUSEHOLD_BUILD_OUTPUT="$PWD/zainaer" \
  sh scripts/build-household.sh
```

构建脚本读取 `VERSION` 和当前 Git 提交，生成前端资源并嵌入后端。若需其他平台，请调整 `GOOS`、`GOARCH`。

#### 服务器运行示例

将程序复制到服务器，在独立数据目录中运行。以下环境变量用于说明配置方式，不包含可直接使用的密钥：

```sh
export HBOX_WEB_HOST=127.0.0.1
export HBOX_WEB_PORT=7745
export HBOX_WEB_WRITE_TIMEOUT=180s
export HBOX_AUTH_API_KEY_PEPPER='<自行生成并持久保存的随机密钥>'
export HBOX_OPTIONS_ALLOW_ANALYTICS=false
export HBOX_OPTIONS_GITHUB_RELEASE_CHECK=false
# phoobobo.top 共用账号。留空 HBOX_PHOOBOBO_ACCOUNT_URL 则不显示登录按钮。
export HBOX_PHOOBOBO_ACCOUNT_URL=https://account.phoobobo.top
export HBOX_PHOOBOBO_ACCOUNT_INTERNAL=http://127.0.0.1:7750
./zainaer
```

工作目录下的 `.data/` 存放默认数据库、附件与模型配置。生产环境使用进程管理器保持运行，通过 HTTPS 反向代理或受控隧道访问。模型可能需要多轮请求，响应超时应覆盖助手的 150 秒总执行窗口。

出于与现有部署的兼容性，内部 Go module/import 路径与 `HBOX_*` 环境变量暂保留 HomeBox 命名；这不代表本项目继续跟踪上游版本。

### 验证

```sh
cd backend
go test ./app/api/handlers/v1 -run TestAI -count=1
```

这些测试覆盖密钥加密、配置校验、集合隔离、草稿确认/取消、重复提交和模型故障切换。网络模拟测试需要临时回环端口，不调用真实模型。真实提供商检查是显式启用的可选测试。

前端构建及 ESLint 使用 `frontend/package.json` 中的脚本。全量类型检查目前存在继承代码中的 Ref / 实体类型问题，不能将前端构建成功描述为全量类型检查通过。

### 部署与备份

维护者当前仅以北京 ECS 为生产服务；原 Mac 应用及 Companion 已退役，本机 SSH 隧道仅用于访问云端。具体主机信息和凭据不写入仓库。

更新前备份当前程序及 SQLite 数据库，上传后核对校验和，再替换程序并重启。不要使用旧的本地数据库覆盖云端数据。备份数据库时应使用 SQLite backup API，或停服后取得一致快照；同时保存附件、模型加密配置及解密主密钥。

### GitHub Actions releases and production deployment

- Active workflows: `zainaer-ci.yml` (main/PR checks), `zainaer-release.yml` (tag release), and `zainaer-deploy.yml` (manual production deployment). Superseded upstream workflows live in `.github/legacy-workflows/` and must not be reactivated accidentally.
- Update `VERSION`, commit on `main`, then push a matching annotated tag such as `v0.1.5`. Release tags must point to commits reachable from `main`. Do not move an existing release tag or replace published assets.
- The Release contains `zainaer-linux-amd64.gz`, checksums of compressed and uncompressed executables, `release.json` with version/commit, matching source archive, and license files. Build/test jobs have read-only repository permissions; only the publish job can create a Release.
- Deploy through **Actions → Zainaer Deploy → Run workflow**, branch **main**, input `release_tag`. Creating a Release never deploys automatically. Only published stable Releases with source/tag/checksum consistency are accepted.
- `production` is restricted to `main`, and stores `DEPLOY_HOST`, `DEPLOY_PORT`, `DEPLOY_USER`, `DEPLOY_SSH_KEY`, and `DEPLOY_KNOWN_HOSTS` as environment secrets. SSH host keys are pinned from a trusted server connection; never replace pinning with `ssh-keyscan` in CI or disable strict checking. Do not copy model provider keys into GitHub.
- The dedicated SSH account has no interactive shell or forwarding capability through its key. Its forced command can only invoke the root-owned `/usr/local/sbin/zainaer-deploy` helper, installed from `scripts/deploy/server.py`. The CI payload cannot update this helper or choose arbitrary commands/paths. Updating the helper requires a trusted administrator/Workbench operation.
- The helper serializes deployments with a server lock, limits artifact size, verifies SHA-256 and ELF architecture, and checks backup disk space before stopping the service. It backs up the executable, environment and full data directory under root-only `/opt/zainaer/backups/` folders. A brief maintenance interval is expected while data is copied with the service stopped, preserving database consistency, attachments, ownership and encryption keys.
- It atomically replaces the executable and starts the existing service, checking exact version/commit, frontend health and protected-endpoint `401` responses. On startup/health failure it stops the new process, preserves failed data, restores the **old executable and full pre-upgrade data**, and verifies the previous version. Backup failure before replacement restarts the original service. CI cancellation/disconnection should not be used as a deployment control: inspect the ECS service and backup metadata before retrying after an interrupted connection.
- Rollback tests in `scripts/deploy/test_server.py` simulate database/key mutation and startup failures without touching production. Run `python3 -m unittest discover -s scripts/deploy -p 'test_*.py' -v` after deployment-script changes.
- Deployment snapshots are retained, never automatically pruned. Scheduled backup archives use the separate retention policy below. Monitor disk usage. A host outage, SIGKILL, or rollback restoration failure requires operator recovery from the recorded backup; no software rollback can guarantee recovery from those failures. Do not publish backup contents or failed-data snapshots as CI artifacts.
- To deploy an older Release manually, select its tag; this changes the executable against the current database and may be rejected by the application's migration compatibility checks. For a full historical data restore, use an explicit maintenance operation with a fresh backup and the matching historical snapshot; never silently discard data created since that snapshot.

### Public HTTPS entry point

- The production application remains loopback-only on port 7745, with Caddy handling the public subdomain on TCP 80/443. Preserve automatic certificate renewal, HTTP-to-HTTPS redirects and the existing SSH access bridge. Do not expose port 7745 directly.
- The operator's public ICP filing identifier is configured through `HBOX_OPTIONS_ICP_RECORD`, exposed as `icpRecord` in public status, and rendered by `App/FilingFooter.vue` on both login and authenticated pages. Keep the link fixed to `https://beian.miit.gov.cn/`; an empty setting hides the footer for other deployments.
- Caddy infrastructure changes are trusted Workbench operations rather than application artifact replacements. Keep operator-specific DNS records, server addresses, certificate keys and environment files outside the repository. Application changes still use the tagged Release and manual Actions deployment flow.

- Public security filing is configured through `HBOX_OPTIONS_POLICE_RECORD`, exposed as `policeRecord` and rendered with the official icon alongside the ICP record. Its query link uses only the 14-digit filing code and the fixed official `https://beian.mps.gov.cn/` origin. An empty setting hides it.

### Scheduled encrypted backups and recovery

- `scripts/backup/backup.py` is an operator-installed maintenance helper, not an application endpoint. The reviewed helper lives in `/usr/local/lib/zainaer-backup/`, with root-only configuration at `/etc/zainaer/backup.json`. Install helper/unit updates through trusted Workbench operations; an application Release alone does not install these privileged files.
- `zainaer-backup.timer` runs hourly with up to 60 seconds of jitter and catches up after host startup. It uses SQLite's online backup API without stopping the app, validates database integrity/foreign keys, copies attachments on both sides of the DB snapshot, and rejects/retries missing referenced files. It also validates encrypted model configuration against the copied master key. All data, attachments, AI settings/master key and `service.env` are included.
- Archives are encrypted with age to an operator-controlled public recipient before transfer. The private recovery identity stays outside the ECS and Git repository. Never publish an archive, decrypted restore folder, credentials, or recovery identity as a GitHub artifact or commit. Store the recovery identity separately and keep an additional protected offline copy; losing every identity makes age archives unrecoverable.
- Local scheduled archives live under `/var/backups/zainaer/` with root-only permissions. Retention preserves the latest 48 successful points plus the latest point per day over 30 days and per month over 366 days. Only the helper's own strictly named scheduled archives/receipts are eligible for pruning, after a new successful backup. Existing `/opt/zainaer/backups/` deployment snapshots remain manually managed and are not pruned by this helper.
- `latest.json` records the last successful archive, checksum, time, version and verification state. A failed run leaves the previous good archive intact and reports a nonzero systemd result; inspect `systemctl status zainaer-backup.service`, `journalctl -u zainaer-backup.service`, and backup age. Failure notifications are not configured by default.
- An initial encrypted archive has been copied to the operator's local recovery folder and actually decrypted into a temporary isolated directory for integrity, file-hash, row-count, photo-reference and model-key verification. That one-off local copy does **not** mean ongoing offsite replication exists. `offsiteVerified: false` remains accurate until an approved uploader is configured and its upload is verified. OSS or another billed storage service needs the user's cost approval; do not claim same-disk archives protect against instance/disk loss.
- The helper supports a root-owned `upload_command` argument list. An uploader must verify the remote encrypted object and update the receipt before local retention runs. Never place account-wide Aliyun credentials on the ECS for an uploader; prefer a narrowly scoped instance role and private encrypted object storage with retention/versioning controls.
- Recovery must first happen in a NEW isolated directory: decrypt with `age -d -i <recovery-identity> -o snapshot.tar.gz <archive>.age`, then run `python3 scripts/backup/verify.py snapshot.tar.gz <new-directory>`. The verifier rejects archive traversal/links, checks file hashes, SQLite integrity/foreign keys, row counts and referenced photos. Model keys are also checked by `validate_snapshot` with the backup helper's cryptography dependency. Verify before planning any production restore, take a fresh backup, and explicitly confirm the target recovery point before replacing live data.
- Run `python3 -m unittest discover -s scripts/backup -p 'test_*.py' -v` in a Python environment with `scripts/backup/requirements.txt` installed after helper changes. CI and Release workflows run these tests in an isolated virtual environment. Include WAL/concurrent attachment, missing-file, model-key, retention and app-outage cases.

## 账号与备案

- 本站不自建账号体系。登录统一走 https://account.phoobobo.top（cookie `phoobobo_sid`，Domain `.phoobobo.top`），用户以 (issuer, subject) 识别。
- 页脚的 ICP 和公安备案号以账号服务的 `/api/filing.json` 为准，不要在本仓库另写一份不同的值。
- 契约与细节只看：https://github.com/Phoobobo/zjsy-bbs/blob/main/docs/INTEGRATION.md 。需要改登录、cookie 或备案行为时，先改那份文档。该文档稍后会迁到 phoobobo-account 仓库的同一路径。
- 配置：HBOX_PHOOBOBO_ACCOUNT_URL=https://account.phoobobo.top，HBOX_PHOOBOBO_ACCOUNT_INTERNAL=http://127.0.0.1:7750（需带 Host 头）；HBOX_OPTIONS_ICP_RECORD / HBOX_OPTIONS_POLICE_RECORD 的值必须与 filing.json 一致。
- 配置了账号地址后，未登录首页的「登录」「注册」进入账号站点的 `/login?mode=login|register&next=`。「退出登录」先清掉本站会话，再打开账号站点的 `/logout?next=` 回到本站首页。页面不出现本站密码表单。账号站点不可用时打开 `/?local=1`。未配置 `HBOX_PHOOBOBO_ACCOUNT_URL` 时，登录、注册和退出与原来的本站表单一致。
