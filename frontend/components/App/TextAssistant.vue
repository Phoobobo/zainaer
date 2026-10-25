<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->
<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { MessageCircle, Send, Search, LoaderCircle, Mic, Square } from "lucide-vue-next";
  import { Button } from "~/components/ui/button";
  import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription, SheetTrigger } from "~/components/ui/sheet";

  import { createLocalVoice, type VoiceState, type LocalRecognition } from "~/lib/local-voice";

  type Result = { id: string; name: string; description: string; kind: string; path: string; tags: string[] };
  type Proposal = {
    id: string;
    proposal: { action?: string; kind: string; name: string; description: string };
    before?: Result;
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
  let contextSummary = "";
  let historyStart = 0;
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

  const voiceState = ref<VoiceState>("idle");
  const voiceError = ref("");
  const voiceActive = computed(() => voiceState.value !== "idle");
  const voice = createLocalVoice({
    recognition: () => {
      if (!window.isSecureContext) return undefined;
      return (window as unknown as { SpeechRecognition?: LocalRecognition }).SpeechRecognition;
    },
    state: state => {
      voiceState.value = state;
    },
    error: code => {
      voiceError.value = code;
    },
    text: text => {
      const combined = [draft.value.trimEnd(), text].filter(Boolean).join(" ");
      draft.value = combined.slice(0, 2000);
      if (combined.length > 2000) voiceError.value = "length";
    },
  });
  const voiceNotice = computed(() => {
    const notices: Record<string, [string, string]> = {
      unsupported: [
        "此浏览器不支持设备本地转写，请使用文字输入。",
        "On-device transcription is unavailable in this browser. Type instead.",
      ],
      language: [
        "当前语言的本地语音包不可用。请在浏览器中安装语言包后重试，或使用文字输入。",
        "The local language pack is unavailable. Install it in your browser and retry, or type instead.",
      ],
      "not-allowed": [
        "麦克风权限被拒绝。请在浏览器设置中允许后重试。",
        "Microphone permission denied. Allow it in browser settings and retry.",
      ],
      "service-not-allowed": [
        "浏览器禁止本地语音识别，请检查权限或使用文字输入。",
        "Local speech recognition is blocked. Check permissions or type instead.",
      ],
      "no-speech": ["未识别到语音，请重新录音。", "No speech recognized. Record again."],
      length: [
        "输入已达到 2000 字限制，请检查并编辑。",
        "Input reached the 2,000 character limit. Review and edit it.",
      ],
      timeout: [
        "录音或转写超时，请重新录音（每次不超过 60 秒）。",
        "Recording or transcription timed out. Record again (up to 60 seconds).",
      ],
      "audio-capture": ["麦克风不可用，请检查设备后重试。", "Microphone unavailable. Check your device and retry."],
      failed: ["转写失败或录音中断，请重新录音。", "Transcription failed or recording was interrupted. Record again."],
    };
    return (notices[voiceError.value] || notices.failed!)[zh.value ? 0 : 1];
  });
  function cancelVoice() {
    voice.cancel();
    voiceError.value = "";
  }
  onMounted(() => {
    document.addEventListener("visibilitychange", hideVoice);
    window.addEventListener("pagehide", cancelVoice);
  });
  function hideVoice() {
    if (document.hidden) cancelVoice();
  }
  onBeforeUnmount(() => {
    cancelVoice();
    document.removeEventListener("visibilitychange", hideVoice);
    window.removeEventListener("pagehide", cancelVoice);
  });
  const route = useRoute();
  watch(() => route.fullPath, cancelVoice);
  watch(locale, cancelVoice);

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
    const recent = messages.value.slice(historyStart).map(m => ({
      role: m.role,
      content:
        m.content +
        (m.drafts?.length
          ? "\n" +
            m.drafts
              .map(
                d =>
                  `${d.proposal.action || "create"} ${d.proposal.kind}: ${d.proposal.name}; ${d.parentPath}; tags: ${d.tags.join(", ")}; ${d.proposal.description}; ${d.state || "awaiting confirmation"}`
              )
              .join("\n")
          : ""),
    }));
    return contextSummary ? [{ role: "assistant", content: contextSummary }, ...recent] : recent;
  }
  async function send(followUp?: string) {
    const content = followUp ?? draft.value;
    if (!content.trim() || busy.value || saving.value || !configured.value || voiceActive.value) return;
    error.value = "";
    busy.value = true;
    for (const m of messages.value) for (const d of m.drafts || []) if (!d.state) d.state = "replaced";
    messages.value.push({ role: "user", content: content.trim() });
    if (!followUp) draft.value = "";
    const current = generation;
    abort = new AbortController();
    scrollBottom();
    try {
      const data = await request("chat", { messages: history() }, abort.signal);
      if (current !== generation) return;
      if (data.summary) {
        contextSummary = data.summary;
        historyStart = messages.value.length - 1;
      }
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
      if (last?.role === "user" && !followUp) draft.value = last.content;
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      if (current === generation) {
        busy.value = false;
        scrollBottom();
      }
    }
  }
  async function confirm(d: Proposal, cancel = false, grouped = false) {
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
          content: `${zh.value ? "已保存" : "Saved"} ${kindLabel(result.kind)}: ${result.name}${result.path ? ` (${result.path})` : ""}`,
          items: [result],
        });
        scrollBottom();
        if (!grouped && d.proposal.action !== "update" && (result.kind === "location" || result.kind === "tag")) {
          saving.value = "";
          await send(
            zh.value
              ? "前置的位置或标签已通过确认按钮保存。请继续完成我此前尚未完成的创建或编辑、批量移动请求：查询并使用刚保存的记录，生成完整的一组待确认操作卡片；若还缺少前置记录，继续准备对应卡片。如果此前没有待完成的请求，无需创建任何内容。"
              : "The prerequisite location or tag was saved using the confirmation button. Continue my outstanding creation, edit or batch move request: look up the saved record and prepare the complete action group for review, or prepare any remaining prerequisites. If no request remains outstanding, do not create anything."
          );
        }
      }
    } catch (e) {
      if (current === generation) error.value = e instanceof Error ? e.message : String(e);
    } finally {
      if (current === generation) saving.value = "";
    }
  }
  async function confirmGroup(m: Message, cancel = false) {
    for (const d of m.drafts || []) {
      if (d.state) continue;
      await confirm(d, cancel, true);
      // Keep successful cards saved and leave remaining cards untouched on failure.
      if (error.value) break;
    }
  }
  function search() {
    if (!draft.value.trim() || voiceActive.value) return;
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
    if (!value) cancelVoice();
    if (value) {
      status();
      scrollBottom();
    }
  });
  watch(
    () => prefs.value.collectionId,
    () => {
      cancelVoice();
      generation++;
      abort?.abort();
      messages.value = [];
      contextSummary = "";
      historyStart = 0;
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
        ><MessageCircle class="size-5" />{{ t("assistant.title") }}</Button
      >
    </SheetTrigger>
    <SheetContent side="right" class="flex h-dvh w-full flex-col gap-0 p-0 sm:max-w-lg">
      <SheetHeader class="shrink-0 border-b p-5 text-left">
        <SheetTitle class="flex items-center gap-2"
          ><MessageCircle class="size-5 text-primary" />{{ t("assistant.title") }}</SheetTitle
        >
        <SheetDescription>{{
          zh
            ? "查找物品，通过操作卡片确认创建、编辑或批量移动。"
            : "Find items and review actions to create, edit or move records."
        }}</SheetDescription>
        <div class="flex items-center justify-between gap-2 pt-2 text-xs text-muted-foreground">
          <span class="truncate">{{
            checking ? (zh ? "检查配置…" : "Checking…") : configured ? model : t("assistant.disconnected")
          }}</span>
          <NuxtLink to="/models" class="shrink-0 text-primary hover:underline" @click="open = false">{{
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
                ? "试着问“车钥匙在哪”，或说“把充电器放在书房抽屉”。创建和编辑都会先生成操作卡片，确认后才执行。"
                : "Ask where your keys are, or describe an item and its storage location. Creation and edits require action card confirmation."
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
          <div v-if="(m.drafts?.filter(d => !d.state).length || 0) > 1" class="flex flex-wrap items-center gap-2">
            <span class="text-sm"
              >{{ zh ? "整组操作" : "Action group" }} · {{ m.drafts?.filter(d => !d.state).length }}</span
            >
            <Button size="sm" :disabled="!!saving || busy" @click="confirmGroup(m)">{{
              zh ? "确认全部执行" : "Confirm all actions"
            }}</Button>
            <Button size="sm" variant="ghost" :disabled="!!saving || busy" @click="confirmGroup(m, true)">{{
              zh ? "取消全部" : "Cancel all"
            }}</Button>
          </div>
          <div v-for="d in m.drafts" :key="d.id" class="space-y-3 rounded-lg border border-primary/30 bg-card p-4">
            <p class="text-xs font-medium text-primary">
              {{ zh ? "Action · 操作卡片" : "Action card" }} ·
              {{ d.proposal.action === "update" ? (zh ? "编辑" : "Edit") : zh ? "创建" : "Create" }} ·
              {{ kindLabel(d.proposal.kind) }}
            </p>
            <p class="font-semibold">{{ d.proposal.name }}</p>
            <dl v-if="d.before" class="space-y-2 rounded-md bg-secondary/50 p-3 text-sm">
              <dt class="text-xs text-muted-foreground">{{ zh ? "修改前" : "Before" }}</dt>
              <dd>{{ d.before.name }}</dd>
              <dd>{{ d.before.path || (zh ? "根级" : "Root") }}</dd>
              <dd v-if="d.before.tags?.length">{{ d.before.tags.join("、") }}</dd>
              <dd v-if="d.before.description" class="whitespace-pre-wrap break-words">{{ d.before.description }}</dd>
            </dl>
            <p v-if="d.before" class="text-xs text-primary">{{ zh ? "修改为" : "After" }}</p>
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
                  saving === d.id ? (zh ? "处理中…" : "Working…") : zh ? "确认执行" : "Confirm action"
                }}</Button
                ><Button size="sm" variant="ghost" :disabled="!!saving || busy" @click="confirm(d, true)">{{
                  zh ? "取消" : "Cancel"
                }}</Button>
              </div>
              <p class="text-xs text-muted-foreground">
                {{
                  zh
                    ? "需要修改？在下方告诉助手。操作卡片 10 分钟后失效。"
                    : "To make changes, tell the assistant below. Action cards expire after 10 minutes."
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
          :disabled="busy || !!saving || voiceActive"
          @keydown.ctrl.enter.prevent="send()"
          @keydown.meta.enter.prevent="send()"
        />
        <div class="mt-2 flex flex-wrap items-center gap-2">
          <Button
            v-if="!voiceActive"
            variant="outline"
            size="sm"
            :disabled="busy || !!saving || draft.length >= 2000"
            @click="voice.start(zh ? 'zh-CN' : 'en-US')"
          >
            <Mic class="size-4" />{{
              voiceError ? (zh ? "重新录音" : "Record again") : zh ? "语音输入" : "Voice input"
            }}
          </Button>
          <template v-else>
            <span role="status" class="text-sm text-primary">{{
              voiceState === "recording"
                ? zh
                  ? "正在录音…"
                  : "Recording…"
                : voiceState === "transcribing"
                  ? zh
                    ? "正在转写…"
                    : "Transcribing…"
                  : zh
                    ? "准备麦克风…"
                    : "Preparing microphone…"
            }}</span>
            <Button v-if="voiceState === 'recording'" variant="outline" size="sm" @click="voice.stop()"
              ><Square class="size-4" />{{ zh ? "停止并转写" : "Stop and transcribe" }}</Button
            >
            <Button variant="ghost" size="sm" @click="cancelVoice">{{ zh ? "取消录音" : "Cancel recording" }}</Button>
          </template>
        </div>
        <p v-if="voiceError" role="alert" class="mt-2 text-xs text-destructive">{{ voiceNotice }}</p>
        <p class="mt-2 text-xs text-muted-foreground">
          {{
            zh
              ? "仅设备本地转写（需浏览器及语言包支持）。最多录音 60 秒；结果可编辑，点击发送才提交。"
              : "On-device transcription only; browser and language pack support required. Up to 60 seconds. Edit the result, then send."
          }}
        </p>
        <div class="mt-3 flex items-center justify-between gap-3">
          <Button variant="ghost" size="sm" :disabled="!draft.trim() || busy || !!saving || voiceActive" @click="search"
            ><Search class="size-4" />{{ t("assistant.keyword_search") }}</Button
          ><Button
            :disabled="!configured || checking || busy || !!saving || !draft.trim() || voiceActive"
            @click="send()"
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
