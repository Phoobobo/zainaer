<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->
<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { toast } from "@/components/ui/sonner";
  import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
  import { Button } from "@/components/ui/button";
  import AppFilingFooter from "~/components/App/FilingFooter.vue";
  import FormTextField from "~/components/Form/TextField.vue";

  definePageMeta({ layout: "empty", middleware: ["auth"] });
  const { t } = useI18n();
  const route = useRoute();
  const api = useUserApi();
  const collections = useCollections();
  const inviteCode = ref(typeof route.query.token === "string" ? route.query.token : "");
  const loading = ref(false);
  const errorMessage = ref("");
  useHead(() => ({ title: `${t("app.name")} | ${t("components.collection.join_modal.title")}` }));

  async function join() {
    if (loading.value) return;
    let token = inviteCode.value.trim();
    if (token.includes("?")) {
      try {
        token = new URL(token).searchParams.get("token") || "";
      } catch {
        token = "";
      }
    }
    if (!/^[A-Z2-7]{26}$/.test(token)) {
      errorMessage.value = t("components.collection.join_modal.toast.please_enter_valid_join_code");
      return;
    }
    loading.value = true;
    errorMessage.value = "";
    try {
      const { data, error, status } = await api.group.acceptInvitation(token);
      if (error || !data?.id) {
        const reason =
          status === 404
            ? "invite_not_found"
            : status === 410
              ? "invite_unavailable"
              : status === 409
                ? "already_member"
                : "join_failed";
        errorMessage.value = t(`components.collection.join_modal.toast.${reason}`);
        return;
      }
      collections.set(data.id);
      toast.success(t("components.collection.join_modal.toast.join_success"));
      window.location.replace("/items");
    } catch {
      errorMessage.value = t("components.collection.join_modal.toast.join_failed");
    } finally {
      loading.value = false;
    }
  }
</script>

<template>
  <div class="mx-auto w-full max-w-lg px-4 py-8">
    <Card>
      <CardHeader
        ><CardTitle>{{ $t("components.collection.join_modal.title") }}</CardTitle></CardHeader
      >
      <CardContent>
        <form class="flex flex-col gap-4" data-testid="invitation-join-form" @submit.prevent="join">
          <FormTextField
            v-model="inviteCode"
            :label="$t('components.collection.join_modal.invite_code_label')"
            :required="true"
          />
          <p v-if="errorMessage" role="alert" class="text-sm text-destructive">{{ errorMessage }}</p>
          <Button :disabled="loading" type="submit">{{ $t("components.collection.join_modal.join") }}</Button>
          <NuxtLink to="/items" class="text-center text-sm text-muted-foreground underline">{{
            $t("global.cancel")
          }}</NuxtLink>
        </form>
      </CardContent>
    </Card>
    <AppFilingFooter />
  </div>
</template>
