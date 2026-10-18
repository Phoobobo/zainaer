<script setup lang="ts">
  definePageMeta({ middleware: ["auth"] });
  await navigateTo("/models", { replace: true });
</script>
<template><div /></template>
