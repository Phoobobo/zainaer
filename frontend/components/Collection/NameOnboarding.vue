<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->
<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { DialogRoot } from "reka-ui";
  import { DialogContent, DialogHeader, DialogTitle, DialogDescription } from "~/components/ui/dialog";
  import { Button } from "~/components/ui/button";
  import FormTextField from "~/components/Form/TextField.vue";
  import { useDialog } from "~/components/ui/dialog-provider";

  const { t } = useI18n();
  const api = useUserApi();
  const route = useRoute();
  const { activeDialog } = useDialog();
  const { collections, load } = useCollections();
  const pendingId = ref<string | null>(null);
  const name = ref("");
  const saving = ref(false);
  const error = ref("");
  const pendingCollection = computed(() => collections.value.find(c => c.id === pendingId.value));
  const open = computed(() => !!pendingCollection.value && !activeDialog.value && !route.query.token);

  watch(pendingCollection, collection => {
    if (collection) name.value = collection.name;
  });

  onMounted(async () => {
    const { data } = await api.user.self();
    pendingId.value = data?.item?.collectionNamingId ?? null;
  });

  async function save() {
    if (saving.value || !name.value.trim()) return;
    saving.value = true;
    error.value = "";
    try {
      const result = await api.user.completeCollectionNaming(name.value.trim());
      if (result.error) {
        error.value = t("collection_onboarding.save_error");
        return;
      }
      pendingId.value = null;
      const ctx = useAuthContext();
      if (ctx.user) ctx.user.collectionNamingId = null;
      await load();
    } catch {
      error.value = t("collection_onboarding.save_error");
    } finally {
      saving.value = false;
    }
  }
</script>

<template>
  <DialogRoot :open="open">
    <DialogContent disable-close @escape-key-down.prevent @interact-outside.prevent>
      <DialogHeader>
        <DialogTitle>{{ t("collection_onboarding.title") }}</DialogTitle>
        <DialogDescription>{{ t("collection_onboarding.description") }}</DialogDescription>
      </DialogHeader>
      <form class="space-y-4" data-testid="collection-naming-form" @submit.prevent="save">
        <FormTextField
          v-model="name"
          :label="t('collection_onboarding.name')"
          :required="true"
          :max-length="255"
          :autofocus="true"
        />
        <p v-if="error" role="alert" class="text-sm text-destructive">{{ error }}</p>
        <Button class="w-full" type="submit" :disabled="saving || !name.trim()">
          {{ saving ? t("global.loading") : t("collection_onboarding.continue") }}
        </Button>
      </form>
    </DialogContent>
  </DialogRoot>
</template>
