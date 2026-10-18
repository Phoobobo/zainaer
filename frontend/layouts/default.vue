<template>
  <div id="app">
    <ModalConfirm /><OutdatedModal v-if="status" :status="status" />
    <EntityCreateModal /><TagCreateModal /><AppQuickMenuModal :actions="quickMenuActions" />
    <CollectionCreateModal /><CollectionJoinModal /><CollectionInviteCreateModal />
    <SidebarProvider :default-open="sidebarState">
      <Sidebar collapsible="icon">
        <SidebarHeader class="px-4 py-8">
          <NuxtLink to="/items" class="flex items-center gap-3" :aria-label="$t('app.name')">
            <AppLogo class="size-9 shrink-0 text-primary" /><span
              class="text-xl font-semibold group-data-[collapsible=icon]:hidden"
              >{{ $t("app.name") }}</span
            >
          </NuxtLink>
        </SidebarHeader>
        <SidebarContent
          ><SidebarGroup
            ><SidebarMenu>
              <SidebarMenuItem v-for="n in primaryNav" :key="n.to">
                <SidebarMenuLink
                  :href="n.to"
                  :tooltip="n.name.value"
                  :class="{ 'bg-secondary text-primary': n.active.value }"
                  :aria-current="n.active.value ? 'page' : undefined"
                >
                  <component :is="n.icon" /><span>{{ n.name.value }}</span>
                </SidebarMenuLink>
              </SidebarMenuItem>
            </SidebarMenu></SidebarGroup
          ></SidebarContent
        >
        <SidebarFooter class="gap-4 p-3">
          <SidebarMenuLink
            href="/settings"
            :tooltip="$t('ui_upgrade.settings')"
            :class="{ 'bg-secondary text-primary': settingsActive }"
            ><MdiCog /><span>{{ $t("ui_upgrade.settings") }}</span></SidebarMenuLink
          >
          <div class="border-t pt-4"><CollectionSelector /></div>
        </SidebarFooter>
        <SidebarRail />
      </Sidebar>
      <SidebarInset class="min-h-dvh min-w-0 max-w-full bg-background">
        <header
          class="zainaer-header sticky top-0 z-30 flex h-16 items-center gap-2 bg-card/95 px-3 backdrop-blur sm:px-6"
        >
          <SidebarTrigger variant="ghost" />
          <span class="min-w-0 flex-1 truncate text-sm text-muted-foreground">{{ currentPage }}</span>
          <LanguageQuickSwitch class="hidden sm:flex" />
          <TextAssistant />
          <Button
            class="h-9 gap-1.5 rounded-lg px-3"
            data-testid="floating-create-button"
            @click="openDialog(DialogID.CreateEntity, { params: { baseType: 'item' } })"
            ><MdiPlus class="size-4" />{{ $t("global.add") }}</Button
          >
        </header>
        <main class="min-w-0 flex-1 pb-20 md:pb-4"><slot /></main>
        <nav class="mobile-navigation" :aria-label="$t('app.name')">
          <NuxtLink
            v-for="n in primaryNav"
            :key="n.to"
            :to="n.to"
            :class="{ active: n.active.value }"
            :aria-current="n.active.value ? 'page' : undefined"
            ><component :is="n.icon" /><span>{{ n.name.value }}</span></NuxtLink
          >
          <NuxtLink to="/settings" :class="{ active: settingsActive }"
            ><MdiCog /><span>{{ $t("ui_upgrade.settings") }}</span></NuxtLink
          >
        </nav>
      </SidebarInset>
    </SidebarProvider>
  </div>
</template>
<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import {
    Sidebar,
    SidebarContent,
    SidebarFooter,
    SidebarGroup,
    SidebarHeader,
    SidebarInset,
    SidebarMenu,
    SidebarMenuItem,
    SidebarMenuLink,
    SidebarProvider,
    SidebarRail,
    SidebarTrigger,
  } from "@/components/ui/sidebar";
  import { Button } from "@/components/ui/button";
  import { useDialog } from "~/components/ui/dialog-provider";
  import { DialogID } from "~/components/ui/dialog-provider/utils";
  import ModalConfirm from "~/components/ModalConfirm.vue";
  import OutdatedModal from "~/components/App/OutdatedModal.vue";
  import EntityCreateModal from "~/components/Entity/CreateModal.vue";
  import TagCreateModal from "~/components/Tag/CreateModal.vue";
  import AppQuickMenuModal from "~/components/App/QuickMenuModal.vue";
  import AppLogo from "~/components/App/Logo.vue";
  import TextAssistant from "~/components/App/TextAssistant.vue";
  import LanguageQuickSwitch from "~/components/App/LanguageQuickSwitch.vue";
  import CollectionSelector from "~/components/Collection/Selector.vue";
  import CollectionCreateModal from "~/components/Collection/CreateModal.vue";
  import CollectionJoinModal from "~/components/Collection/JoinModal.vue";
  import CollectionInviteCreateModal from "~/components/Collection/InviteCreateModal.vue";
  import MdiMagnify from "~icons/mdi/magnify";
  import MdiFileTree from "~icons/mdi/file-tree";
  import MdiTagMultiple from "~icons/mdi/tag-multiple";
  import MdiCog from "~icons/mdi/cog";
  import MdiPlus from "~icons/mdi/plus";
  const { t } = useI18n();
  const { openDialog } = useDialog();
  const route = useRoute();
  const router = useRouter();
  const path = computed(() => route.path.replace(/\/$/, ""));
  const sidebarState = useCookie("sidebar:state", { readonly: true, decode: value => value !== "false" });
  const pubApi = usePublicApi();
  const { data: status } = useAsyncData(async () => (await pubApi.status()).data);
  const primaryNav = [
    {
      to: "/items",
      icon: MdiMagnify,
      name: computed(() => t("ui_upgrade.search")),
      active: computed(() => path.value === "/items" || path.value === "/home" || path.value.startsWith("/item/")),
    },
    {
      to: "/locations",
      icon: MdiFileTree,
      name: computed(() => t("ui_upgrade.storage")),
      active: computed(() => path.value.startsWith("/location")),
    },
    {
      to: "/tags",
      icon: MdiTagMultiple,
      name: computed(() => t("ui_upgrade.tags")),
      active: computed(() => path.value.startsWith("/tag")),
    },
  ];
  const settingsActive = computed(
    () => path.value === "/settings" || path.value === "/profile" || path.value.startsWith("/collection")
  );
  const currentPage = computed(() => primaryNav.find(n => n.active.value)?.name.value || t("ui_upgrade.settings"));
  const quickMenuActions = reactive(primaryNav.map(n => ({ text: n.name, href: n.to, type: "navigate" as const })));
  const tagStore = useTagStore();
  const locationStore = useLocationStore();
  const entityTypes = useEntityTypeStore();
  tagStore.ensureAllTagsFetched();
  locationStore.ensureLocationsFetched();
  entityTypes.ensureFetched();
  onMounted(() => {
    locationStore.refreshParents();
    locationStore.refreshTree();
    if (typeof route.query.token === "string") {
      const token = route.query.token;
      const query = { ...route.query };
      delete query.token;
      router.replace({ query });
      openDialog(DialogID.JoinCollection, { params: { inviteCode: token } });
    }
  });
  onServerEvent(ServerEvent.TagMutation, () => tagStore.refresh());
  onServerEvent(ServerEvent.EntityMutation, () => {
    locationStore.refreshChildren();
    locationStore.refreshParents();
    locationStore.refreshTree();
  });
</script>
