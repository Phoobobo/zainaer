<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->
<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { Button } from "~/components/ui/button";
  import { Input } from "~/components/ui/input";
  import BaseContainer from "~/components/Base/Container.vue";
  import BaseCard from "~/components/Base/Card.vue";

  definePageMeta({ middleware: ["auth"] });
  const { locale, t } = useI18n();
  useHead(() => ({ title: `${t("app.name")} | ${t("ui_upgrade.models")}` }));
  const zh = computed(() => locale.value.startsWith("zh"));
  const prefs = useViewPreferences();
  const provider = ref("openrouter");
  const model = ref("");
  const fallback1 = ref("");
  const fallback2 = ref("");
  const key = ref("");
  const hasKey = ref(false);
  const savedProvider = ref("openrouter");
  const busy = ref(false);
  const loading = ref(true);
  const allowed = ref(false);
  const error = ref("");
  const notice = ref("");
  const endpoints: Record<string, string> = {
    openrouter: "https://openrouter.ai/api/v1",
    bailian: "https://dashscope.aliyuncs.com/compatible-mode/v1",
    deepseek: "https://api.deepseek.com/v1",
  };

  async function request(method: string, suffix = "settings", body?: object) {
    const headers: Record<string, string> = { "Content-Type": "application/json" };
    if (prefs.value.collectionId) headers["X-Tenant"] = prefs.value.collectionId;
    const response = await fetch(`/api/v1/ai/${suffix}`, {
      method,
      headers,
      credentials: "same-origin",
      body: body ? JSON.stringify(body) : undefined,
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || data.message || `HTTP ${response.status}`);
    return data;
  }
  async function load() {
    loading.value = true;
    allowed.value = false;
    error.value = "";
    key.value = "";
    try {
      const data = await request("GET");
      provider.value = data.provider;
      savedProvider.value = data.provider;
      model.value = data.model;
      fallback1.value = data.fallbackModels?.[0] || "";
      fallback2.value = data.fallbackModels?.[1] || "";
      hasKey.value = data.hasKey;
      allowed.value = true;
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  }
  async function save(clearKey = false) {
    busy.value = true;
    error.value = "";
    notice.value = "";
    try {
      const data = await request("PUT", "settings", {
        provider: provider.value,
        model: model.value,
        fallbackModels: [fallback1.value, fallback2.value].map(v => v.trim()).filter(Boolean),
        apiKey: clearKey ? "" : key.value,
        clearKey,
      });
      hasKey.value = data.hasKey;
      savedProvider.value = data.provider;
      key.value = "";
      notice.value = zh.value
        ? "配置已保存。API Key 不会返回到页面。"
        : "Saved. The API key is never returned to the page.";
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      busy.value = false;
    }
  }
  async function test() {
    busy.value = true;
    error.value = "";
    notice.value = "";
    try {
      const data = await request("POST", "test");
      notice.value = (zh.value ? "连接成功，模型：" : "Connected: ") + data.model;
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      busy.value = false;
    }
  }
  watch(() => prefs.value.collectionId, load);
  onMounted(load);
</script>

<template>
  <BaseContainer
    ><BaseCard>
      <template #title>{{ zh ? "模型配置" : "Model settings" }}</template>
      <div class="max-w-2xl space-y-5 p-6">
        <p class="text-sm leading-relaxed text-muted-foreground">
          {{
            zh
              ? "为当前家庭集合配置模型。只有集合所有者可以管理，密钥加密保存在服务器。保存后，AI助手将使用此模型查询物品和准备入库草稿。"
              : "Configure a model for this collection. Only its owner can manage credentials, which are encrypted on the server. The AI assistant uses this model to search items and prepare records for confirmation."
          }}
        </p>
        <p v-if="loading" role="status">{{ zh ? "正在加载…" : "Loading…" }}</p>
        <p
          v-if="error"
          role="alert"
          class="break-words rounded-md border border-destructive/30 p-3 text-sm text-destructive"
        >
          {{ error }}
        </p>
        <p v-if="notice" role="status" class="rounded-md bg-secondary p-3 text-sm">{{ notice }}</p>
        <form v-if="allowed && !loading" class="space-y-5" @submit.prevent="save()">
          <div class="space-y-2">
            <label for="ai-provider" class="block text-sm font-medium">{{ zh ? "模型提供商" : "Provider" }}</label>
            <select
              id="ai-provider"
              v-model="provider"
              class="h-10 w-full rounded-md border bg-background px-3 text-sm"
              :disabled="busy"
              @change="
                key = '';
                notice = '';
              "
            >
              <option value="openrouter">OpenRouter</option>
              <option value="bailian">{{ zh ? "阿里云百炼（北京）" : "Alibaba Cloud (Beijing)" }}</option>
              <option value="deepseek">DeepSeek</option>
            </select>
            <p class="break-all text-xs text-muted-foreground">{{ endpoints[provider] }}</p>
          </div>
          <div class="space-y-2">
            <label for="ai-model" class="block text-sm font-medium">{{ zh ? "模型 ID" : "Model ID" }}</label>
            <Input
              id="ai-model"
              v-model="model"
              required
              maxlength="200"
              :disabled="busy"
              :placeholder="provider === 'openrouter' ? 'provider/model-name' : 'model-name'"
            />
            <p class="text-xs text-muted-foreground">
              {{
                zh
                  ? "填写提供商控制台中的完整模型 ID；OpenRouter 需要包含厂商前缀。"
                  : "Use the exact model ID from your provider. OpenRouter IDs include the organization prefix."
              }}
            </p>
            <a
              v-if="provider === 'openrouter'"
              href="https://openrouter.ai/models"
              target="_blank"
              rel="noopener noreferrer"
              class="inline-block text-sm text-primary hover:underline"
              >{{ zh ? "查看 OpenRouter 模型" : "Browse OpenRouter models" }}</a
            >
          </div>
          <div class="space-y-2">
            <label for="ai-fallback-1" class="block text-sm font-medium">{{
              zh ? "备选模型 1（可选）" : "Fallback model 1 (optional)"
            }}</label>
            <Input
              id="ai-fallback-1"
              v-model="fallback1"
              maxlength="200"
              :disabled="busy"
              :placeholder="zh ? '主模型不可用时尝试' : 'Try when the primary model is unavailable'"
            />
          </div>
          <div class="space-y-2">
            <label for="ai-fallback-2" class="block text-sm font-medium">{{
              zh ? "备选模型 2（可选）" : "Fallback model 2 (optional)"
            }}</label>
            <Input
              id="ai-fallback-2"
              v-model="fallback2"
              maxlength="200"
              :disabled="busy"
              :placeholder="zh ? '前两个模型均不可用时尝试' : 'Try when both earlier models fail'"
            />
            <p class="text-xs leading-relaxed text-muted-foreground">
              {{
                zh
                  ? "按主模型 → 备选 1 → 备选 2 顺序调用，共用当前提供商和密钥。单次请求最多等待 20 秒再切换；不会因为模型回答不理想而重试。备选模型也可能产生费用。"
                  : "Try primary → fallback 1 → fallback 2 using this provider and key. Each attempt times out after 20 seconds. Answer quality does not trigger retries. Fallback models may also incur charges."
              }}
            </p>
          </div>
          <div class="space-y-2">
            <label for="ai-key" class="block text-sm font-medium">API Key</label>
            <Input
              id="ai-key"
              v-model="key"
              type="password"
              autocomplete="new-password"
              spellcheck="false"
              maxlength="4096"
              :disabled="busy"
              :placeholder="
                hasKey && savedProvider === provider
                  ? zh
                    ? '已保存；留空保留，填写则替换'
                    : 'Saved; leave blank to keep it'
                  : zh
                    ? '在这里粘贴 API Key'
                    : 'Paste your API key here'
              "
            />
            <p class="text-xs text-muted-foreground">
              {{
                zh
                  ? "密钥只发送给本站服务端及你选择的提供商，不保存在浏览器存储中。切换提供商需要填写对应的新密钥。"
                  : "The key is sent only to this server and the selected provider, and is not stored in browser storage. Switching providers requires a new key."
              }}
            </p>
          </div>
          <div class="flex flex-wrap gap-3">
            <Button type="submit" :disabled="busy || !model.trim()">{{ zh ? "保存配置" : "Save settings" }}</Button>
            <Button
              type="button"
              variant="outline"
              :disabled="busy || !hasKey || !!key || provider !== savedProvider"
              @click="test"
              >{{ zh ? "测试已保存配置" : "Test saved settings" }}</Button
            >
            <Button
              v-if="hasKey"
              type="button"
              variant="ghost"
              :disabled="busy || provider !== savedProvider"
              @click="save(true)"
              >{{ zh ? "移除密钥" : "Remove key" }}</Button
            >
          </div>
          <p class="text-xs leading-relaxed text-muted-foreground">
            {{
              zh
                ? "测试会向已保存的模型发送“Reply only with OK.”，可能产生少量费用，不发送物品数据。修改后请先保存再测试。"
                : "Testing sends ‘Reply only with OK.’ to the saved model and may incur a small charge. No inventory data is sent. Save changes before testing."
            }}
          </p>
        </form>
      </div>
    </BaseCard></BaseContainer
  >
</template>
