<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->
<script setup lang="ts">
  import { UserRound, UsersRound, Cpu, ChevronRight, LogOut } from "lucide-vue-next";
  import { useI18n } from "vue-i18n";
  import BaseContainer from "~/components/Base/Container.vue";
  import CollectionSelector from "~/components/Collection/Selector.vue";
  import LanguageQuickSwitch from "~/components/App/LanguageQuickSwitch.vue";
  import { Button } from "~/components/ui/button";
  const { t } = useI18n();
  definePageMeta({ middleware: ["auth"] });
  useHead(() => ({ title: `${t("app.name")} | ${t("ui_upgrade.settings")}` }));
  const auth = useAuthContext();
  const api = useUserApi();
  async function logout() {
    await auth.logout(api);
    await navigateTo("/");
  }
  const links = [
    { to: "/profile", label: "ui_upgrade.personal", icon: UserRound },
    { to: "/collection/members", label: "ui_upgrade.family", icon: UsersRound },
    { to: "/models", label: "ui_upgrade.models", icon: Cpu },
  ];
</script>
<template>
  <BaseContainer
    ><div class="mb-6 flex items-center justify-between">
      <h1 class="text-2xl font-semibold">{{ $t("ui_upgrade.settings") }}</h1>
      <LanguageQuickSwitch />
    </div>
    <section class="mb-5 max-w-2xl rounded-xl border bg-card p-5">
      <h2 class="mb-3 text-sm font-medium">{{ $t("components.collection.selector.select_collection") }}</h2>
      <CollectionSelector standalone />
    </section>
    <div class="grid max-w-2xl gap-3">
      <NuxtLink
        v-for="link in links"
        :key="link.to"
        :to="link.to"
        class="flex items-center gap-4 rounded-xl border bg-card p-5 hover:border-primary/40"
        ><component :is="link.icon" class="size-5 text-primary" /><span>{{ $t(link.label) }}</span
        ><ChevronRight class="ml-auto size-4 text-muted-foreground"
      /></NuxtLink>
    </div>
    <Button variant="outline" class="mt-8" data-testid="logout-button" @click="logout"
      ><LogOut class="mr-2 size-4" />{{ $t("global.sign_out") }}</Button
    >
  </BaseContainer>
</template>
