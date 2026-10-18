<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import emojis from "~/lib/data/emoji.json";
  import { Label } from "~/components/ui/label";
  import { Button } from "~/components/ui/button";
  import { availableIcons } from "~/lib/icons";

  const props = defineProps({
    modelValue: {
      type: String,
      required: false,
      default: "",
    },
    label: {
      type: String,
      default: "",
    },
  });

  const emits = defineEmits(["update:modelValue"]);

  const { locale } = useI18n();
  const mode = ref("emoji");
  const groups = [...new Set(emojis.map(e => e.group))];
  const id = useId();

  const value = useVModel(props, "modelValue", emits);

  function selectIcon(iconName: string) {
    if (value.value === iconName) {
      value.value = "";
    } else {
      value.value = iconName;
    }
  }
</script>

<template>
  <div class="flex w-full flex-col gap-1.5">
    <Label :for="id" class="flex w-full px-1">
      <span>{{ label }}</span>
    </Label>
    <div class="flex flex-col gap-1">
      <div class="mb-2 flex gap-2">
        <Button type="button" size="sm" :variant="mode === 'emoji' ? 'default' : 'outline'" @click="mode = 'emoji'">{{
          $t("ui_upgrade.common_emoji")
        }}</Button>
        <Button type="button" size="sm" :variant="mode === 'line' ? 'default' : 'outline'" @click="mode = 'line'">{{
          $t("ui_upgrade.line_icons")
        }}</Button>
      </div>
      <div v-if="mode === 'emoji'" class="max-h-64 space-y-3 overflow-y-auto p-1">
        <section v-for="group in groups" :key="group">
          <h3 class="mb-2 text-xs text-muted-foreground">
            {{ locale.startsWith("zh") ? group : emojis.find(e => e.group === group)?.groupEn }}
          </h3>
          <div class="grid grid-cols-[repeat(auto-fill,minmax(36px,1fr))] gap-1">
            <button
              v-for="entry in emojis.filter(e => e.group === group)"
              :key="entry.emoji"
              type="button"
              class="grid size-9 place-items-center rounded-md border text-xl hover:bg-secondary focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"
              :class="value === 'emoji:' + entry.emoji ? 'border-primary bg-secondary' : 'border-transparent'"
              :aria-label="entry.emoji"
              :aria-pressed="value === 'emoji:' + entry.emoji"
              @click="selectIcon('emoji:' + entry.emoji)"
            >
              {{ entry.emoji }}
            </button>
          </div>
        </section>
      </div>
      <div v-else class="grid w-fit grid-cols-4 gap-1">
        <Button
          v-for="icon in availableIcons"
          :key="icon.name"
          type="button"
          :variant="value === icon.name ? 'default' : 'outline'"
          size="sm"
          class="flex size-10 items-center justify-center p-0"
          :aria-label="`Select ${icon.name} icon`"
          @click="selectIcon(icon.name)"
        >
          <component :is="icon.component" class="size-5" />
        </Button>
      </div>
    </div>
  </div>
</template>
