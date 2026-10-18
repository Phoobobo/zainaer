<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->
<script setup lang="ts">
  import { Home, MapPin, Plus, Minus } from "lucide-vue-next";
  import type { MapNode } from "./model";
  import MapBranch from "./Node.vue";
  defineProps<{ node: MapNode; expanded: Set<string>; selected: string; depth?: number }>();
  defineEmits<{ select: [id: string]; toggle: [id: string] }>();
</script>
<template>
  <li>
    <div class="storage-node" :class="{ selected: selected === node.id, root: node.id === 'root', room: depth === 1 }">
      <button
        type="button"
        class="storage-node-label"
        :aria-pressed="selected === node.id"
        @click="$emit('select', node.id)"
      >
        <Home v-if="node.id === 'root'" class="size-4 shrink-0" /><MapPin v-else class="size-4 shrink-0" />
        <span
          ><span class="line-clamp-2">{{ node.name }}</span
          ><small>{{ $t("ui_upgrade.items_count", { count: node.count }) }}</small></span
        >
      </button>
      <button
        v-if="node.children.length"
        type="button"
        class="storage-node-toggle"
        :aria-expanded="expanded.has(node.id)"
        :aria-label="
          $t(expanded.has(node.id) ? 'ui_upgrade.collapse_node' : 'ui_upgrade.expand_node', { name: node.name })
        "
        @click="$emit('toggle', node.id)"
      >
        <Minus v-if="expanded.has(node.id)" class="size-3" /><Plus v-else class="size-3" />
      </button>
    </div>
    <ul v-if="node.children.length && expanded.has(node.id)">
      <MapBranch
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :expanded="expanded"
        :selected="selected"
        :depth="(depth || 0) + 1"
        @select="$emit('select', $event)"
        @toggle="$emit('toggle', $event)"
      />
    </ul>
  </li>
</template>
