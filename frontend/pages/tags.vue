<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->
<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import BaseContainer from "~/components/Base/Container.vue";
  import { Button } from "~/components/ui/button";
  import ItemCard from "~/components/Item/Card.vue";
  import type { EntitySummary } from "~/lib/api/types/data-contracts";
  import { useDialog } from "~/components/ui/dialog-provider";
  import { DialogID } from "~/components/ui/dialog-provider/utils";
  const { t } = useI18n();
  definePageMeta({ middleware: ["auth"] });
  useHead(() => ({ title: `${t("app.name")} | ${t("ui_upgrade.tags")}` }));
  const store = useTagStore();
  const api = useUserApi();
  const { openDialog } = useDialog();
  const selected = ref("");
  const items = ref<EntitySummary[]>([]);
  const total = ref(0);
  const page = ref(1);
  const busy = ref(false);
  const failed = ref(false);
  const locationFlatTree = useFlatLocations();
  let sequence = 0;
  async function load() {
    const current = ++sequence;
    busy.value = true;
    failed.value = false;
    items.value = [];
    try {
      const result = await api.items.getAll({
        tags: selected.value ? [selected.value] : [],
        page: page.value,
        pageSize: 20,
      });
      if (current !== sequence) return;
      if (result.error) throw new Error("Could not load tagged items");
      items.value = result.data.items || [];
      total.value = result.data.total;
    } catch {
      if (current === sequence) failed.value = true;
    } finally {
      if (current === sequence) busy.value = false;
    }
  }
  watch(selected, () => {
    page.value = 1;
    load();
  });
  watch(page, load);
  watch(
    () => store.tags,
    tags => {
      if (selected.value && !tags.some(t => t.id === selected.value)) selected.value = "";
    }
  );
  onMounted(load);
  onBeforeUnmount(() => {
    sequence++;
  });
  onServerEvent(ServerEvent.EntityMutation, load);
</script>
<template>
  <BaseContainer>
    <div class="mb-6 flex items-center justify-between gap-3">
      <h1 class="text-2xl font-semibold">{{ $t("ui_upgrade.tags") }}</h1>
      <Button variant="outline" @click="openDialog(DialogID.CreateTag)">{{ $t("ui_upgrade.create_tag") }}</Button>
    </div>
    <div class="mb-6 flex flex-wrap gap-2">
      <button
        v-for="tag in store.tags"
        :key="tag.id"
        type="button"
        class="tag-filter"
        :aria-pressed="selected === tag.id"
        @click="selected = selected === tag.id ? '' : tag.id"
      >
        {{ tag.name }}
      </button>
    </div>
    <div class="mb-4 flex items-center justify-between">
      <p class="text-sm text-muted-foreground">{{ $t("ui_upgrade.items_count", { count: total }) }}</p>
      <NuxtLink v-if="selected" :to="`/tag/${selected}`" class="text-sm text-primary hover:underline">{{
        $t("ui_upgrade.manage_tags")
      }}</NuxtLink>
    </div>
    <p v-if="busy" role="status" class="p-8">{{ $t("global.loading") }}</p>
    <div v-else-if="failed" role="alert">
      {{ $t("ui_upgrade.load_failed") }}<Button variant="outline" @click="load">{{ $t("ui_upgrade.retry") }}</Button>
    </div>
    <div v-else-if="items.length" class="inventory-grid">
      <ItemCard v-for="item in items" :key="item.id" :item="item" :location-flat-tree="locationFlatTree" />
    </div>
    <p v-else class="rounded-xl border border-dashed p-8 text-sm text-muted-foreground">{{ $t("items.no_results") }}</p>
    <div v-if="total > 20" class="mt-5 flex items-center justify-center gap-4">
      <Button variant="outline" :disabled="page === 1 || busy" @click="page--">{{ $t("ui_upgrade.previous") }}</Button
      ><span>{{ page }} / {{ Math.ceil(total / 20) }}</span
      ><Button variant="outline" :disabled="page * 20 >= total || busy" @click="page++">{{
        $t("ui_upgrade.next")
      }}</Button>
    </div>
  </BaseContainer>
</template>
