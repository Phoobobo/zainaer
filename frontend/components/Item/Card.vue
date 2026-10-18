<template>
  <Card class="inventory-card relative isolate overflow-hidden transition-colors hover:border-primary/40">
    <div class="inventory-cover">
      <img v-if="item.imageId" loading="lazy" :src="imageUrl" :alt="item.name" />
      <MdiPackageVariant v-else class="size-12 text-primary/25" />
    </div>
    <NuxtLink
      :to="`/item/${item.id}`"
      class="absolute inset-0 z-10 rounded-[inherit] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary"
      :aria-label="item.name"
    />
    <div v-if="tableRow" class="absolute left-2 top-2 z-30">
      <Checkbox
        class="size-5 bg-card"
        :model-value="tableRow.getIsSelected()"
        :aria-label="$t('components.item.view.selectable.select_card')"
        @update:model-value="tableRow.toggleSelected()"
      />
    </div>
    <div class="pointer-events-none relative z-20 space-y-2 p-3">
      <h2 class="line-clamp-2 text-sm font-semibold leading-snug">{{ item.name }}</h2>
      <NuxtLink
        v-if="item.parent"
        :to="`/location/${item.parent.id}`"
        class="pointer-events-auto flex items-start gap-1 text-xs leading-relaxed text-primary hover:underline"
      >
        <MdiMapMarkerOutline class="mt-0.5 size-3 shrink-0" /><span class="line-clamp-2">{{ locationString }}</span>
      </NuxtLink>
      <div v-if="itemTags.length" class="pointer-events-auto flex flex-wrap gap-1">
        <TagChip v-for="tag in itemTags.slice(0, 3)" :key="tag.id" :tag="tag" size="sm" :ancestors="tag.ancestors" />
        <span v-if="itemTags.length > 3" class="text-xs text-muted-foreground">+{{ itemTags.length - 3 }}</span>
      </div>
    </div>
  </Card>
</template>

<script setup lang="ts">
  import MdiMapMarkerOutline from "~icons/mdi/map-marker-outline";
  import MdiPackageVariant from "~icons/mdi/package-variant";
  import type { EntityOut, EntitySummary } from "~~/lib/api/types/data-contracts";
  import { Card } from "@/components/ui/card";
  import TagChip from "@/components/Tag/Chip.vue";
  import type { Row } from "@tanstack/vue-table";
  import { Checkbox } from "@/components/ui/checkbox";

  const api = useUserApi();

  const imageUrl = computed(() => {
    if (!props.item.imageId) {
      return "/no-image.jpg";
    }
    if (props.item.thumbnailId) {
      return api.authURL(`/entities/${props.item.id}/attachments/${props.item.thumbnailId}`);
    } else {
      return api.authURL(`/entities/${props.item.id}/attachments/${props.item.imageId}`);
    }
  });

  const itemTags = computed(() => {
    return useTagStore().withAncestors(props.item.tags);
  });

  const props = defineProps({
    item: {
      type: Object as () => EntityOut | EntitySummary,
      required: true,
    },
    locationFlatTree: {
      type: Array as () => FlatTreeItem[],
      required: false,
      default: () => [],
    },
    tableRow: {
      type: Object as () => Row<EntitySummary>,
      required: false,
      default: () => null,
    },
  });

  const locationString = computed(
    () => props.locationFlatTree.find(l => l.id === props.item.parent?.id)?.treeString || props.item.parent?.name
  );
</script>

<style lang="css"></style>
