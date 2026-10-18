<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->
<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import type { TagOut } from "~/lib/api/types/data-contracts";
  import { Button } from "~/components/ui/button";
  import { Input } from "~/components/ui/input";
  import { toast } from "~/components/ui/sonner";
  const props = defineProps<{ tags: TagOut[] }>();
  const model = defineModel<string[]>({ default: () => [] });
  const { t } = useI18n();
  const api = useUserApi();
  const store = useTagStore();
  const name = ref("");
  const busy = ref(false);
  const created = ref<TagOut[]>([]);
  const choices = computed(() => [...new Map([...props.tags, ...created.value].map(tag => [tag.id, tag])).values()]);
  async function add() {
    const value = name.value.trim();
    if (!value || busy.value) return;
    const existing = choices.value.find(tag => tag.name.toLocaleLowerCase() === value.toLocaleLowerCase());
    if (existing) {
      if (!model.value.includes(existing.id)) model.value = [...model.value, existing.id];
      name.value = "";
      return;
    }
    busy.value = true;
    try {
      const { data, error } = await api.tags.create({ name: value, color: "", description: "", icon: "" });
      if (error || !data) {
        toast.error(t("components.tag.create_modal.toast.create_failed"));
        return;
      }
      created.value.push(data);
      model.value = [...model.value, data.id];
      name.value = "";
      await store.refresh();
    } catch {
      toast.error(t("components.tag.create_modal.toast.create_failed"));
    } finally {
      busy.value = false;
    }
  }
</script>
<template>
  <fieldset class="min-w-0 space-y-3">
    <legend class="mb-2 text-sm">
      {{ $t("global.tags") }}
      <span class="ml-2 text-xs text-muted-foreground">{{ $t("ui_upgrade.multi_select") }}</span>
    </legend>
    <div class="flex max-h-44 flex-wrap gap-2 overflow-y-auto p-1">
      <label v-for="tag in choices" :key="tag.id" class="tag-choice"
        ><input v-model="model" type="checkbox" :value="tag.id" /><span>{{ tag.name }}</span></label
      >
    </div>
    <div class="flex gap-2">
      <Input
        v-model="name"
        :maxlength="50"
        :placeholder="$t('ui_upgrade.new_tag')"
        :aria-label="$t('ui_upgrade.new_tag')"
        @keydown.enter.prevent="add"
      /><Button type="button" variant="outline" :disabled="busy || !name.trim()" @click="add">{{
        $t("ui_upgrade.create_tag")
      }}</Button>
    </div>
  </fieldset>
</template>
