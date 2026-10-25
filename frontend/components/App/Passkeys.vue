<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->
<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { toast } from "~/components/ui/sonner";
  import { Button } from "~/components/ui/button";
  import BaseCard from "~/components/Base/Card.vue";
  import BaseSectionHeader from "~/components/Base/SectionHeader.vue";
  import FormPassword from "~/components/Form/Password.vue";
  import FormTextField from "~/components/Form/TextField.vue";
  import { createPasskey, passkeysSupported, passkeyErrorKey, type PasskeySummary } from "~/lib/passkeys";

  const { t } = useI18n();
  const api = useUserApi();
  const publicApi = usePublicApi();
  const confirm = useConfirm();
  const enabled = ref(false);
  const supported = ref(false);
  const busy = ref(false);
  const keys = ref<PasskeySummary[]>([]);
  const name = ref("");
  const password = ref("");
  const loadFailed = ref(false);
  async function refresh() {
    const result = await api.user.listPasskeys();
    loadFailed.value = !!result.error;
    if (!result.error) keys.value = result.data;
  }
  onMounted(async () => {
    supported.value = passkeysSupported();
    const result = await publicApi.status();
    enabled.value = !result.error && result.data.passkeys;
    if (enabled.value) await refresh();
  });
  async function add() {
    busy.value = true;
    try {
      const start = await api.user.beginPasskeyRegistration(name.value.trim(), password.value);
      password.value = "";
      if (start.error) {
        toast.error(t("passkeys.verify_failed"));
        return;
      }
      const credential = await createPasskey(start.data.publicKey);
      const finish = await api.user.finishPasskeyRegistration(credential);
      if (finish.error) throw new Error("finish");
      name.value = "";
      toast.success(t("passkeys.added"));
      await refresh();
    } catch (error) {
      toast.error(t(passkeyErrorKey(error)));
    } finally {
      password.value = "";
      busy.value = false;
    }
  }
  async function remove(id: string) {
    const confirmation = await confirm.open(t("passkeys.delete_confirm"));
    if (confirmation.isCanceled) return;
    busy.value = true;
    try {
      const result = await api.user.deletePasskey(id);
      if (result.error) throw new Error("delete");
      toast.success(t("passkeys.deleted"));
      await refresh();
    } catch {
      toast.error(t("passkeys.failed"));
    } finally {
      busy.value = false;
    }
  }
</script>

<template>
  <BaseCard v-if="enabled">
    <template #title>
      <BaseSectionHeader>
        {{ t("passkeys.title") }}
        <template #description>{{ t("passkeys.description") }}</template>
      </BaseSectionHeader>
    </template>
    <div class="flex flex-col gap-4 p-4">
      <p v-if="loadFailed" role="alert">{{ t("passkeys.load_failed") }}</p>
      <p v-else-if="!keys.length" class="text-sm text-muted-foreground">{{ t("passkeys.empty") }}</p>
      <ul v-else class="flex flex-col gap-3">
        <li v-for="key in keys" :key="key.id" class="flex items-center justify-between gap-3">
          <div class="min-w-0">
            <p class="break-words font-medium">{{ key.name }}</p>
            <p class="text-sm text-muted-foreground">
              {{ t("passkeys.created") }}: {{ new Date(key.createdAt).toLocaleDateString() }}
            </p>
            <p class="text-sm text-muted-foreground">
              {{ t("passkeys.last_used") }}:
              {{ key.lastUsedAt ? new Date(key.lastUsedAt).toLocaleDateString() : t("passkeys.never") }}
            </p>
          </div>
          <Button
            type="button"
            variant="destructive"
            size="sm"
            :disabled="busy"
            :aria-label="`${t('global.delete')} ${key.name}`"
            @click="remove(key.id)"
            >{{ t("global.delete") }}</Button
          >
        </li>
      </ul>
      <form v-if="supported" class="flex flex-col gap-3" @submit.prevent="add">
        <FormTextField
          v-model="name"
          :label="t('passkeys.name')"
          name="passkey-name"
          :required="true"
          :max-length="80"
        />
        <FormPassword
          v-model="password"
          :label="t('passkeys.password')"
          name="passkey-password"
          autocomplete="current-password"
          :required="true"
        />
        <Button type="submit" :disabled="busy || !name.trim() || !password || keys.length >= 20">{{
          t("passkeys.add")
        }}</Button>
      </form>
      <p v-else class="text-sm text-muted-foreground">{{ t("passkeys.unsupported") }}</p>
    </div>
  </BaseCard>
</template>
