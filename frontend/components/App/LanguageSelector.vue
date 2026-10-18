<script setup lang="ts">
  import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
  import { Label } from "@/components/ui/label";
  import { useViewPreferences } from "~~/composables/use-preferences";

  const preferences = useViewPreferences();

  defineProps({
    expanded: {
      type: Boolean,
      default: true,
    },
  });

  function setLanguage(lang: string) {
    preferences.value.language = lang;
  }
</script>

<template>
  <div class="w-full" :class="{ 'p-5 pt-0': expanded }">
    <Label v-if="expanded" for="language"> {{ $t("profile.language") }} </Label>
    <Select
      id="language"
      v-model="$i18n.locale"
      @update:model-value="
        event => {
          setLanguage(event as string);
        }
      "
    >
      <SelectTrigger>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem v-for="lang in $i18n.availableLocales" :key="lang" :value="lang">
          {{ $t(`languages.${lang}`) }} ({{ $t(`languages.${lang}`, 1, { locale: lang }) }})
        </SelectItem>
      </SelectContent>
    </Select>
  </div>
</template>
