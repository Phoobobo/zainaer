<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->
<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { MessageCircle, Send, Search, LoaderCircle } from "lucide-vue-next";
  import { Button } from "~/components/ui/button";
  import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription, SheetTrigger } from "~/components/ui/sheet";

  type Result = { id: string; name: string; description: string; kind: string; path: string; tags: string[] };
  type Proposal = {
    id: string;
    proposal: { kind: string; name: string; description: string };
    parentPath: string;
    tags: string[];
    state?: "saved" | "cancelled" | "replaced";
    result?: Result;
  };
  type Message = {
    role: "user" | "assistant";
    content: string;
    usedModel?: string;
    items?: Result[];
    drafts?: Proposal[];
  };
  const { t, locale } = useI18n();
  const zh = computed(() => locale.value.startsWith("zh"));
  const prefs = useViewPreferences();
  const open = ref(false);
  const draft = ref("");
  const messages = ref<Message[]>([]);
  const configured = ref(false);
  const checking = ref(false);
  const model = ref("");
  const busy = ref(false);
  const saving = ref("");
  const error = ref("");
  const scrollArea = ref<HTMLElement>();
  let abort: AbortController | undefined;
  let generation = 0;
  const suggestions = ["assistant.example_find", "assistant.example_add"];

  async function request(path: string, body?: object, signal?: AbortSignal) {
    const headers: Record<string, string> = { "Content-Type": "application/json" };
    if (prefs.value.collectionId) headers["X-Tenant"] = prefs.value.collectionId;
    const res = await fetch(`/api/v1/ai/${path}`, {
      method: body ? "POST" : "GET",
      credentials: "same-origin",
      headers,
      body: body ? JSON.stringify(body) : undefined,
      signal,
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || data.message || `HTTP ${res.status}`);
    return data;
  }
  async function status() {
    const current = generation;
    checking.value = true;
    try {
      const data = await request("status");
      if (current !== generation) return;
      configured.value = data.configured;
      model.value = data.model;
    } catch {
      if (current === generation) configured.value = false;
    } finally {
      if (current === generation) checking.value = false;
    }
  }
  async function scrollBottom() {
    await nextTick();
    scrollArea.value?.scrollTo({ top: scrollArea.value.scrollHeight, behavior: "smooth" });
  }
  function history() {
    return messages.value.slice(-16).map(m => ({
      role: m.role,
      content: (
        m.content +
        (m.drafts?.length
          ? "\n" +
            m.drafts
              .map(
                d =>
                  `${d.proposal.kind}: ${d.proposal.name}; ${d.parentPath}; tags: ${d.tags.join(", ")}; ${d.proposal.description}; ${d.state || "awaiting confirmation"}`
              )
              .join("\n")
          : "")
      ).slice(0, 4000),
    }));
  }
  async function send() {
    if (!draft.value.trim() || busy.value || saving.value || !configured.value) return;
    error.value = "";
    busy.value = true;
    for (const m of messages.value) for (const d of m.drafts || []) if (!d.state) d.state = "replaced";
    messages.value.push({ role: "user", content: draft.value.trim() });
    draft.value = "";
    const current = generation;
    abort = new AbortController();
    scrollBottom();
    try {
      const data = await request("chat", { messages: history() }, abort.signal);
      if (current !== generation) return;
      messages.value.push({
        role: "assistant",
        content: data.reply,
        usedModel: data.usedModel,
        items: data.items,
        drafts: data.drafts,
      });
    } catch (e) {
      if (current !== generation) return;
      // Remove failed user turn from the next model history, but keep it available to retry.
      const last = messages.value.pop();
      if (last?.role === "user") draft.value = last.content;
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      if (current === generation) {
        busy.value = false;
        scrollBottom();
      }
    }
  }
  async function confirm(d: Proposal, cancel = false) {
    if (saving.value || busy.value || d.state) return;
    const current = generation;
    saving.value = d.id;
    error.value = "";
    try {
      const result = await request("confirm", { draftId: d.id, cancel });
      if (current !== generation) return;
      d.state = cancel ? "cancelled" : "saved";
      if (!cancel) {
        d.result = result;
        messages.value.push({
          role: "assistant",
          content: `${zh.value ? "已保存" : "Saved"}: ${result.name}`,
          items: [result],
        });
        scrollBottom();
      }
    } catch (e) {
      if (current === generation) error.value = e instanceof Error ? e.message : String(e);
    } finally {
      if (current === generation) saving.value = "";
    }
  }
  function search() {
    if (!draft.value.trim()) return;
    navigateTo(`/items?q=${encodeURIComponent(draft.value.trim())}`);
    open.value = false;
  }
  function resultURL(item: Result) {
    return `/${item.kind}/${item.id}`;
  }
  function kindLabel(kind: string) {
    return kind === "tag" ? t("global.tags") : kind === "location" ? t("menu.locations") : zh.value ? "物品" : "Item";
  }
  watch(open, value => {
    if (value) {
      status();
      scrollBottom();
    }
  });
  watch(
    () => prefs.value.collectionId,
    () => {
      generation++;
      abort?.abort();
      messages.value = [];
      draft.value = "";
      configured.value = false;
      busy.value = false;
      saving.value = "";
      error.value = "";
      if (open.value) status();
    }
  );
  onBeforeUnmount(() => {
    generation++;
    abort?.abort();
  });
</script>

<template>
  <Sheet v-model:open="open">
    <SheetTrigger as-child>
      <Button variant="outline" class="h-9 justify-center gap-2 rounded-lg px-3" data-testid="text-assistant-button"
        ><MessageCircle class="size-5" />{{ t("ui_upgrade.assistant") }}</Button
      >
    </SheetTrigger>
    <SheetContent side="right" class="flex h-dvh w-full flex-col gap-0 p-0 sm:max-w-lg">
      <SheetHeader class="shrink-0 border-b p-5 text-left">
        <SheetTitle class="flex items-center gap-2"
          ><MessageCircle class="size-5 text-primary" />{{ t("assistant.title") }}</SheetTitle
        >
        <SheetDescription>{{
          zh ? "查找物品，或准备一条待确认的入库记录。" : "Find items or prepare a record to review and save."
        }}</SheetDescription>
        <div class="flex items-center justify-between gap-2 pt-2 text-xs text-muted-foreground">
          <span class="truncate">{{
            checking ? (zh ? "检查配置…" : "Checking…") : configured ? model : t("assistant.disconnected")
          }}</span>
          <NuxtLink to="/collection/models" class="shrink-0 text-primary hover:underline" @click="open = false">{{
            t("model_settings")
          }}</NuxtLink>
        </div>
      </SheetHeader>
      <div
        ref="scrollArea"
        class="min-h-0 flex-1 space-y-5 overflow-y-auto p-5"
        aria-live="polite"
        aria-relevant="additions"
      >
        <template v-if="!messages.length">
          <h2 class="mt-4 text-2xl font-semibold">{{ t("assistant.greeting") }}</h2>
          <p class="text-sm leading-relaxed text-muted-foreground">
            {{
              zh
                ? "试着问“车钥匙在哪”，或说“把充电器放在书房抽屉”。新记录会先给你确认，不会直接写入。"
                : "Ask where your keys are, or describe an item and its storage location. New records require your confirmation."
            }}
          </p>
          <div class="flex flex-col items-start gap-3">
            <button
              v-for="example in suggestions"
              :key="example"
              class="rounded-lg border bg-card px-4 py-3 text-left text-sm hover:border-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              @click="draft = t(example)"
            >
              {{ t(example) }}
            </button>
          </div>
        </template>
        <article v-for="(m, i) in messages" :key="i" class="space-y-3">
          <p class="text-xs text-muted-foreground">{{ m.role === "user" ? (zh ? "你" : "You") : t("app.name") }}</p>
          <p
            class="whitespace-pre-wrap break-words text-sm leading-relaxed"
            :class="m.role === 'user' ? 'rounded-lg bg-secondary p-3' : ''"
          >
            {{ m.content }}
          </p>
          <p v-if="m.usedModel && m.usedModel !== model" class="text-xs text-muted-foreground">
            {{ zh ? "本次使用备选模型：" : "Fallback model used: " }}{{ m.usedModel }}
          </p>
          <NuxtLink
            v-for="item in m.items"
            :key="item.id"
            :to="resultURL(item)"
            class="block rounded-lg border bg-card p-3 hover:border-primary"
            @click="open = false"
          >
            <p class="text-sm font-semibold">{{ item.name }}</p>
            <p class="mt-1 text-xs leading-relaxed text-primary">{{ item.path }}</p>
            <p v-if="item.description" class="mt-2 line-clamp-2 text-xs text-muted-foreground">
              {{ item.description }}
            </p>
          </NuxtLink>
          <div v-for="d in m.drafts" :key="d.id" class="space-y-3 rounded-lg border border-primary/30 bg-card p-4">
            <p class="text-xs font-medium text-primary">
              {{ zh ? "待确认草稿" : "Review draft" }} · {{ kindLabel(d.proposal.kind) }}
            </p>
            <p class="font-semibold">{{ d.proposal.name }}</p>
            <dl class="space-y-2 text-sm">
              <div>
                <dt class="text-xs text-muted-foreground">{{ zh ? "位置 / 上级" : "Location / parent" }}</dt>
                <dd>{{ d.parentPath || (zh ? "根级" : "Root") }}</dd>
              </div>
              <div v-if="d.tags.length">
                <dt class="text-xs text-muted-foreground">{{ t("global.tags") }}</dt>
                <dd>{{ d.tags.join("、") }}</dd>
              </div>
              <div v-if="d.proposal.description">
                <dt class="text-xs text-muted-foreground">{{ zh ? "描述" : "Description" }}</dt>
                <dd class="whitespace-pre-wrap break-words">{{ d.proposal.description }}</dd>
              </div>
            </dl>
            <p v-if="d.state" class="text-xs text-muted-foreground">
              {{
                d.state === "saved"
                  ? zh
                    ? "已保存"
                    : "Saved"
                  : d.state === "cancelled"
                    ? zh
                      ? "已取消"
                      : "Cancelled"
                    : zh
                      ? "已被新消息替代"
                      : "Replaced by a new message"
              }}
            </p>
            <template v-else>
              <div class="flex gap-2">
                <Button size="sm" :disabled="!!saving || busy" @click="confirm(d)">{{
                  saving === d.id ? (zh ? "处理中…" : "Working…") : zh ? "确认保存" : "Confirm save"
                }}</Button
                ><Button size="sm" variant="ghost" :disabled="!!saving || busy" @click="confirm(d, true)">{{
                  zh ? "取消" : "Cancel"
                }}</Button>
              </div>
              <p class="text-xs text-muted-foreground">
                {{
                  zh
                    ? "需要修改？在下方告诉助手。草稿 10 分钟后失效。"
                    : "To make changes, tell the assistant below. Drafts expire after 10 minutes."
                }}
              </p>
            </template>
          </div>
        </article>
        <p v-if="busy" class="flex items-center gap-2 text-sm text-muted-foreground" role="status">
          <LoaderCircle class="size-4 animate-spin" />{{ zh ? "正在查询和整理…" : "Looking things up…" }}
        </p>
        <p
          v-if="error"
          role="alert"
          class="break-words rounded-md border border-destructive/30 p-3 text-sm text-destructive"
        >
          {{ error }}
        </p>
      </div>
      <div class="shrink-0 border-t bg-card p-4" style="padding-bottom: max(1rem, env(safe-area-inset-bottom))">
        <label for="assistant-draft" class="sr-only">{{ t("assistant.input") }}</label>
        <textarea
          id="assistant-draft"
          v-model="draft"
          rows="2"
          maxlength="2000"
          :placeholder="t('assistant.placeholder')"
          class="w-full resize-none rounded-lg border bg-background p-3 text-sm leading-relaxed focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          :disabled="busy || !!saving"
          @keydown.ctrl.enter.prevent="send"
          @keydown.meta.enter.prevent="send"
        />
        <div class="mt-3 flex items-center justify-between gap-3">
          <Button variant="ghost" size="sm" :disabled="!draft.trim() || busy || !!saving" @click="search"
            ><Search class="size-4" />{{ t("assistant.keyword_search") }}</Button
          ><Button :disabled="!configured || checking || busy || !!saving || !draft.trim()" @click="send"
            ><Send class="size-4" />{{ t("assistant.send") }}</Button
          >
        </div>
        <p class="mt-3 text-xs leading-relaxed text-muted-foreground">
          {{
            configured
              ? zh
                ? "对话及本次查询到的记录会发送给已配置的模型提供商。"
                : "Messages and retrieved records are sent to your configured model provider."
              : zh
                ? "请让集合所有者在“模型配置”中保存并测试模型。"
                : "Ask the collection owner to configure and test a model."
          }}
        </p>
      </div>
    </SheetContent>
  </Sheet>
</template>
