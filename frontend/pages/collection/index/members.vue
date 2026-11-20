<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
  import { Button } from "@/components/ui/button";
  import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
  import { Switch } from "@/components/ui/switch";
  import MdiDelete from "~icons/mdi/delete";
  import { toast } from "@/components/ui/sonner";
  import type { UserSummary } from "~~/lib/api/types/data-contracts";

  definePageMeta({
    middleware: ["auth"],
  });

  const { t } = useI18n();

  useHead(() => ({ title: `${t("app.name")} | ${t("collection.tabs.members")}` }));

  const api = useUserApi();
  const auth = useAuthContext();
  const confirm = useConfirm();

  const loading = ref(true);
  const members = ref<UserSummary[]>([]);
  const selfPrivileged = ref(false);
  const selfIsOwner = ref(false);
  const error = ref<string | null>(null);
  const removing = ref<Record<string, boolean>>({});
  const updatingPrivilege = ref<Record<string, boolean>>({});

  const currentUserId = computed(() => auth.user?.id ?? "");

  const isLastMember = computed(() => members.value.length <= 1);

  const loadMembers = async () => {
    loading.value = true;
    error.value = null;

    try {
      const res = await api.group.getMembers();
      if (res.error) {
        const msg = t("errors.api_failure") + String(res.error);
        error.value = msg;
        members.value = [];
        selfPrivileged.value = false;
        selfIsOwner.value = false;
        toast.error(msg);
      } else {
        members.value = res.data?.members ?? [];
        selfPrivileged.value = Boolean(res.data?.selfPrivileged);
        selfIsOwner.value = Boolean(res.data?.selfIsOwner);
      }
    } catch (e) {
      const msg = (e as Error).message ?? String(e);
      error.value = msg;
      members.value = [];
      toast.error(msg);
    } finally {
      loading.value = false;
    }
  };

  const handleRemove = async (user: UserSummary) => {
    if (!user?.id) return;

    if (isLastMember.value && user.id === currentUserId.value) {
      toast.error(t("collection.members.cannot_remove_last"));
      return;
    }

    const result = await confirm.open(t("collection.members.remove_confirm"));
    if (result.isCanceled) {
      return;
    }

    removing.value = { ...removing.value, [user.id]: true };

    try {
      const res = await api.group.removeMember(user.id);
      if (res.error) {
        const msg = t("errors.api_failure") + String(res.error);
        toast.error(msg);
      } else {
        members.value = members.value.filter(m => m.id !== user.id);
        toast.success(t("collection.members.removed"));
      }
    } catch (e) {
      const msg = (e as Error).message ?? String(e);
      toast.error(msg);
    } finally {
      removing.value = { ...removing.value, [user.id]: false };
    }
  };

  const handlePrivilegeToggle = async (user: UserSummary, privileged: boolean) => {
    if (!user?.id || user.isOwner) return;
    updatingPrivilege.value = { ...updatingPrivilege.value, [user.id]: true };
    try {
      const res = await api.group.setMemberPrivileged(user.id, privileged);
      if (res.error) {
        toast.error(t("errors.api_failure") + String(res.error));
        return;
      }
      members.value = members.value.map(m => (m.id === user.id ? { ...m, privileged } : m));
      toast.success(t("collection.members.privileged_updated"));
    } catch (e) {
      toast.error((e as Error).message ?? String(e));
    } finally {
      updatingPrivilege.value = { ...updatingPrivilege.value, [user.id]: false };
    }
  };

  onMounted(() => {
    loadMembers();
  });
</script>

<template>
  <div class="space-y-4">
    <div v-if="loading" class="rounded-md border bg-card p-4 text-sm text-muted-foreground">
      {{ $t("global.loading") }}
    </div>

    <div v-else>
      <div v-if="!members.length" class="rounded-md border bg-card p-4 text-sm text-muted-foreground">
        {{ $t("collection.members.empty") }}
      </div>

      <div v-else class="scroll-bg overflow-x-auto rounded-md border bg-card">
        <Table class="min-w-[480px]">
          <TableHeader>
            <TableRow>
              <TableHead>{{ $t("collection.members.name") }}</TableHead>
              <TableHead>{{ $t("collection.members.email") }}</TableHead>
              <TableHead v-if="selfPrivileged">{{ $t("collection.members.role") }}</TableHead>
              <TableHead class="w-40 text-right"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="user in members" :key="user.id">
              <TableCell>
                <div class="flex flex-wrap items-center gap-2">
                  <span>{{ user.name }}</span>
                  <span
                    v-if="selfPrivileged && user.isOwner"
                    class="rounded bg-secondary px-1.5 py-0.5 text-xs text-muted-foreground"
                  >
                    {{ $t("collection.members.owner") }}
                  </span>
                  <span
                    v-else-if="selfPrivileged && user.privileged"
                    class="rounded bg-secondary px-1.5 py-0.5 text-xs text-muted-foreground"
                  >
                    {{ $t("collection.members.privileged") }}
                  </span>
                </div>
              </TableCell>
              <TableCell>{{ user.email }}</TableCell>
              <TableCell v-if="selfPrivileged">
                <div v-if="selfIsOwner && !user.isOwner" class="flex items-center gap-2">
                  <Switch
                    :model-value="Boolean(user.privileged)"
                    :disabled="updatingPrivilege[user.id]"
                    :aria-label="
                      user.privileged
                        ? $t('collection.members.privileged_revoke')
                        : $t('collection.members.privileged_grant')
                    "
                    @update:model-value="v => handlePrivilegeToggle(user, Boolean(v))"
                  />
                </div>
                <span v-else-if="user.isOwner" class="text-sm text-muted-foreground">{{
                  $t("collection.members.owner")
                }}</span>
                <span v-else class="text-sm text-muted-foreground">{{
                  user.privileged ? $t("collection.members.privileged") : $t("collection.members.member")
                }}</span>
              </TableCell>
              <TableCell>
                <div class="ml-auto">
                  <TooltipProvider :delay-duration="0">
                    <Tooltip>
                      <TooltipTrigger as-child>
                        <Button
                          variant="destructive"
                          size="icon"
                          :aria-label="$t('global.delete')"
                          :disabled="removing[user.id] || (isLastMember && user.id === currentUserId)"
                          @click="handleRemove(user)"
                        >
                          <MdiDelete class="size-4" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>
                        {{ $t("global.delete") }}
                      </TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                </div>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>
    </div>
  </div>
</template>
