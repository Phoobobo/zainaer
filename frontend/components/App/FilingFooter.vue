<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->
<script setup lang="ts">
  const api = usePublicApi();
  const { data: records } = useAsyncData("site-filing-records", async () => {
    const { data } = await api.status();
    return { icp: data?.icpRecord || "", police: data?.policeRecord || "" };
  });
  const policeCode = computed(() => records.value?.police.match(/\d{14}/)?.[0] || "");
  const linkClass =
    "inline-flex items-center gap-1 rounded-sm underline-offset-4 hover:text-foreground hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary";
</script>

<template>
  <div
    v-if="records?.icp || policeCode"
    class="flex shrink-0 flex-wrap items-center justify-center gap-x-4 gap-y-2 p-4 text-center text-xs text-muted-foreground"
  >
    <a
      v-if="records?.icp"
      href="https://beian.miit.gov.cn/"
      target="_blank"
      rel="noopener noreferrer"
      :class="linkClass"
      >{{ records.icp }}</a
    >
    <a
      v-if="policeCode"
      :href="`https://beian.mps.gov.cn/#/query/webSearch?code=${policeCode}`"
      target="_blank"
      rel="noopener noreferrer"
      :class="linkClass"
    >
      <img src="/police-filing.png" alt="" width="20" height="20" class="size-5 shrink-0 object-contain" />
      <span>{{ records?.police }}</span>
    </a>
  </div>
</template>
