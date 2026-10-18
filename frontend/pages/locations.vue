<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->
<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import BaseContainer from "~/components/Base/Container.vue";
  import { Button } from "~/components/ui/button";
  import MapBranch from "~/components/Location/Map/Node.vue";
  import { storageMap, flattenLocations } from "~/components/Location/Map/model";
  import ItemCard from "~/components/Item/Card.vue";
  import type { EntitySummary } from "~/lib/api/types/data-contracts";
  import { useDialog } from "~/components/ui/dialog-provider";
  import { DialogID } from "~/components/ui/dialog-provider/utils";
  const { t } = useI18n();
  definePageMeta({ middleware: ["auth"] });
  useHead(() => ({ title: `${t("app.name")} | ${t("ui_upgrade.storage")}` }));
  const api = useUserApi();
  const { selectedCollection } = useCollections();
  const { openDialog } = useDialog();
  const {
    data: tree,
    status,
    error,
    refresh,
  } = useAsyncData("storage-map", async () => {
    const result = await api.items.getTree({ withItems: true });
    if (result.error) throw new Error("Could not load storage map");
    return result.data;
  });
  const pending = computed(() => status.value === "pending");
  const root = computed(() => storageMap(tree.value || [], selectedCollection.value?.name || t("ui_upgrade.home")));
  const nodes = computed(() => flattenLocations(root.value));
  const expanded = ref(new Set(["root"]));
  const selectedId = ref("root");
  const selected = computed(() => nodes.value.find(n => n.id === selectedId.value) || root.value);
  function select(id: string) {
    selectedId.value = id;
    expanded.value = new Set([...expanded.value, id]);
  }
  function toggle(id: string) {
    const next = new Set(expanded.value);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    expanded.value = next;
  }
  const pan = reactive({ x: 0, y: 0 });
  const canvas = ref<HTMLElement>();
  let drag: { id: number; x: number; y: number; originX: number; originY: number } | null = null;
  const dragging = ref(false);
  function down(e: PointerEvent) {
    if (e.button !== 0 || (e.target as HTMLElement).closest("button, a")) return;
    drag = { id: e.pointerId, x: e.clientX, y: e.clientY, originX: pan.x, originY: pan.y };
    canvas.value?.setPointerCapture(e.pointerId);
    dragging.value = true;
  }
  function move(e: PointerEvent) {
    if (drag && drag.id === e.pointerId) {
      pan.x = drag.originX + e.clientX - drag.x;
      pan.y = drag.originY + e.clientY - drag.y;
    }
  }
  function up(e: PointerEvent) {
    if (drag?.id !== e.pointerId) return;
    drag = null;
    dragging.value = false;
    if (canvas.value?.hasPointerCapture(e.pointerId)) canvas.value.releasePointerCapture(e.pointerId);
  }
  function wheel(e: WheelEvent) {
    if (e.ctrlKey || e.metaKey) return;
    e.preventDefault();
    pan.x -= e.deltaX;
    pan.y -= e.deltaY;
  }
  function keyboard(e: KeyboardEvent) {
    if (e.target !== canvas.value) return;
    const delta: Record<string, [number, number]> = {
      ArrowLeft: [40, 0],
      ArrowRight: [-40, 0],
      ArrowUp: [0, 40],
      ArrowDown: [0, -40],
    };
    const d = delta[e.key];
    if (d) {
      e.preventDefault();
      pan.x += d[0];
      pan.y += d[1];
    }
  }
  function reset() {
    pan.x = 0;
    pan.y = 0;
  }
  const items = ref<EntitySummary[]>([]);
  const busy = ref(false);
  const itemError = ref(false);
  const page = ref(1);
  const total = ref(0);
  let request = 0;
  const locationFlatTree = useFlatLocations();
  async function loadItems() {
    const seq = ++request;
    busy.value = true;
    itemError.value = false;
    items.value = [];
    const ids = selected.value.id === "root" ? [] : flattenLocations(selected.value).map(n => n.id);
    try {
      const result = await api.items.getAll({ parentIds: ids, page: page.value, pageSize: 12, orderBy: "name" });
      if (seq !== request) return;
      if (result.error) throw new Error("Could not load items");
      items.value = result.data.items || [];
      total.value = result.data.total;
    } catch {
      if (seq === request) itemError.value = true;
    } finally {
      if (seq === request) busy.value = false;
    }
  }
  watch(selectedId, () => {
    page.value = 1;
    loadItems();
  });
  watch(page, loadItems);
  watch(tree, () => {
    if (!nodes.value.some(n => n.id === selectedId.value)) selectedId.value = "root";
    loadItems();
  });
  onMounted(loadItems);
  onBeforeUnmount(() => {
    request++;
  });
  onServerEvent(ServerEvent.EntityMutation, () => refresh());
</script>
<template>
  <BaseContainer>
    <div class="mb-6 flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="text-2xl font-semibold">{{ $t("ui_upgrade.storage") }}</h1>
        <p class="mt-2 text-xs text-muted-foreground">{{ $t("ui_upgrade.map_hint") }}</p>
      </div>
      <div class="flex gap-2">
        <Button variant="outline" size="sm" @click="reset">{{ $t("ui_upgrade.reset") }}</Button
        ><Button variant="outline" size="sm" @click="expanded = new Set(nodes.map(n => n.id))">{{
          $t("ui_upgrade.expand")
        }}</Button
        ><Button
          variant="outline"
          size="sm"
          @click="
            expanded = new Set(['root']);
            reset();
          "
          >{{ $t("ui_upgrade.collapse") }}</Button
        >
      </div>
    </div>
    <div v-if="error" class="rounded-xl border bg-card p-8" role="alert">
      {{ $t("ui_upgrade.load_failed") }}
      <Button variant="outline" @click="refresh()">{{ $t("ui_upgrade.retry") }}</Button>
    </div>
    <p v-else-if="pending" class="p-8" role="status">{{ $t("global.loading") }}</p>
    <template v-else>
      <div
        ref="canvas"
        class="storage-canvas"
        :class="{ dragging }"
        tabindex="0"
        :aria-label="$t('ui_upgrade.map_label')"
        :style="{ backgroundPosition: `${pan.x}px ${pan.y}px` }"
        @pointerdown="down"
        @pointermove="move"
        @pointerup="up"
        @pointercancel="up"
        @wheel="wheel"
        @keydown="keyboard"
      >
        <ul class="storage-tree" :style="{ transform: `translate(${pan.x}px, ${pan.y}px)` }">
          <MapBranch
            :node="root"
            :expanded="expanded"
            :selected="selected.id"
            :depth="0"
            @select="select"
            @toggle="toggle"
          />
        </ul>
      </div>
      <div class="mb-5 mt-7 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="text-lg font-semibold">{{ selected.name }}</h2>
          <p class="mt-1 text-xs text-muted-foreground">{{ selected.path.join(" › ") }}</p>
        </div>
        <div class="flex gap-2">
          <Button v-if="selected.id !== 'root'" as-child size="sm" variant="outline"
            ><NuxtLink :to="`/location/${selected.id}`">{{ $t("ui_upgrade.open_location") }}</NuxtLink></Button
          ><Button
            size="sm"
            variant="outline"
            @click="
              openDialog(DialogID.CreateEntity, {
                params: {
                  baseType: selected.id === 'root' ? 'location' : 'item',
                  parentId: selected.id === 'root' ? undefined : selected.id,
                },
              })
            "
            >{{ $t("ui_upgrade.add_here") }}</Button
          >
        </div>
      </div>
      <p v-if="busy" role="status" class="py-8 text-muted-foreground">{{ $t("global.loading") }}</p>
      <div v-else-if="itemError" role="alert">
        {{ $t("ui_upgrade.load_failed")
        }}<Button variant="outline" @click="loadItems">{{ $t("ui_upgrade.retry") }}</Button>
      </div>
      <div v-else-if="items.length" class="inventory-grid">
        <ItemCard v-for="item in items" :key="item.id" :item="item" :location-flat-tree="locationFlatTree" />
      </div>
      <p v-else class="rounded-xl border border-dashed p-8 text-sm text-muted-foreground">
        {{ $t("ui_upgrade.empty_location") }}
      </p>
      <div v-if="total > 12" class="mt-5 flex items-center justify-center gap-4">
        <Button variant="outline" :disabled="page === 1 || busy" @click="page--">{{ $t("ui_upgrade.previous") }}</Button
        ><span>{{ page }} / {{ Math.ceil(total / 12) }}</span
        ><Button variant="outline" :disabled="page * 12 >= total || busy" @click="page++">{{
          $t("ui_upgrade.next")
        }}</Button>
      </div>
    </template>
  </BaseContainer>
</template>
