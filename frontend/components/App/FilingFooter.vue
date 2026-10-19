<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->
<script setup lang="ts">
  const api = usePublicApi();
  const { data: record } = useAsyncData("site-icp-record", async () => {
    const { data } = await api.status();
    return data?.icpRecord || "";
  });
</script>

<template>
  <p v-if="record" class="shrink-0 p-4 text-center text-xs text-muted-foreground">
    <a
      href="https://beian.miit.gov.cn/"
      target="_blank"
      rel="noopener noreferrer"
      class="rounded-sm underline-offset-4 hover:text-foreground hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
      >{{ record }}</a
    >
  </p>
</template>
