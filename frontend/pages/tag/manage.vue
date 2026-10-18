<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->
<script setup lang="ts">
  import { useDialog, DialogID } from "~/components/ui/dialog-provider/utils";
  import { useI18n } from "vue-i18n";
  import { Pencil, Trash2 } from "lucide-vue-next";
  import BaseContainer from "~/components/Base/Container.vue";
  import { Button } from "~/components/ui/button";
  import { Dialog, DialogContent, DialogHeader, DialogTitle } from "~/components/ui/dialog";
  import FormTextField from "~/components/Form/TextField.vue";
  import FormTextArea from "~/components/Form/TextArea.vue";
  import ColorSelector from "~/components/Form/ColorSelector.vue";
  import IconSelector from "~/components/Form/IconSelector.vue";
  import { getIconComponent } from "~/lib/icons";
  import { toast } from "~/components/ui/sonner";
  import type { TagOut } from "~/lib/api/types/data-contracts";
  definePageMeta({ middleware: ["auth"] });
  const { t } = useI18n();
  useHead(() => ({ title: `${t("app.name")} | ${t("ui_upgrade.manage_tags")}` }));
  const store = useTagStore();
  const api = useUserApi();
  const confirm = useConfirm();
  const { openDialog, closeDialog } = useDialog();
  const busy = ref(false);
  const form = reactive({ id: "", name: "", description: "", color: "", icon: "" });
  function edit(tag?: TagOut) {
    Object.assign(form, {
      id: tag?.id || "",
      name: tag?.name || "",
      description: tag?.description || "",
      color: tag?.color || "",
      icon: tag?.icon || "",
    });
    openDialog(DialogID.UpdateTag);
  }
  async function save() {
    if (busy.value || !form.name.trim()) return;
    busy.value = true;
    try {
      const body = { name: form.name.trim(), description: form.description, color: form.color, icon: form.icon };
      const result = form.id ? await api.tags.update(form.id, body) : await api.tags.create(body);
      if (result.error) throw new Error("save");
      await store.refresh();
      closeDialog(DialogID.UpdateTag);
    } catch {
      toast.error(t("tags.toast.failed_update_tag"));
    } finally {
      busy.value = false;
    }
  }
  async function remove(tag: TagOut) {
    if (busy.value) return;
    const { isCanceled } = await confirm.open(`${tag.name} — ${t("tags.tag_delete_confirm")}`);
    if (isCanceled) return;
    busy.value = true;
    try {
      const result = await api.tags.delete(tag.id);
      if (result.error) throw new Error("delete");
      await store.refresh();
    } catch {
      toast.error(t("ui_upgrade.load_failed"));
    } finally {
      busy.value = false;
    }
  }
</script>
<template>
  <BaseContainer>
    <div class="mb-6 flex items-center justify-between gap-3">
      <h1 class="text-2xl font-semibold">{{ $t("ui_upgrade.manage_tags") }}</h1>
      <Button @click="edit()">{{ $t("ui_upgrade.create_tag") }}</Button>
    </div>
    <div class="divide-y rounded-xl border bg-card">
      <div v-for="tag in store.tags" :key="tag.id" class="flex items-center gap-3 p-4">
        <component :is="getIconComponent(tag.icon)" class="size-6 shrink-0 text-primary" />
        <div class="min-w-0 flex-1">
          <p class="break-words font-medium">{{ tag.name }}</p>
          <p class="truncate text-sm text-muted-foreground">{{ tag.description }}</p>
        </div>
        <Button variant="ghost" size="icon" :aria-label="`${$t('global.edit')} ${tag.name}`" @click="edit(tag)"
          ><Pencil class="size-4"
        /></Button>
        <Button
          variant="ghost"
          size="icon"
          :disabled="busy"
          :aria-label="`${$t('global.delete')} ${tag.name}`"
          @click="remove(tag)"
          ><Trash2 class="size-4"
        /></Button>
      </div>
    </div>
    <Dialog :dialog-id="DialogID.UpdateTag">
      <DialogContent class="max-h-[90dvh] overflow-y-auto">
        <DialogHeader
          ><DialogTitle>{{ form.id ? $t("global.edit") : $t("ui_upgrade.create_tag") }}</DialogTitle></DialogHeader
        >
        <form class="space-y-4" @submit.prevent="save">
          <FormTextField
            v-model="form.name"
            :label="$t('components.tag.create_modal.tag_name')"
            :max-length="50"
            :min-length="1"
          />
          <FormTextArea
            v-model="form.description"
            :label="$t('components.tag.create_modal.tag_description')"
            :max-length="1000"
          />
          <ColorSelector v-model="form.color" :label="$t('components.tag.create_modal.tag_color')" />
          <IconSelector v-model="form.icon" :label="$t('components.tag.create_modal.tag_icon')" />
          <Button type="submit" :disabled="busy || !form.name.trim()">{{ $t("global.save") }}</Button>
        </form>
      </DialogContent>
    </Dialog>
  </BaseContainer>
</template>
