<template>
  <div class="shrink-0">
    <NTabs
      :value="activePreset"
      type="line"
      size="small"
      @update:value="selectPreset"
    >
      <NTab
        v-for="preset in presets"
        :key="preset.id"
        :name="preset.id"
      >
        {{ preset.label }}
      </NTab>
    </NTabs>
  </div>
</template>

<script lang="ts" setup>
import { NTab, NTabs } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { SearchParams } from "@/utils";
import { getValueFromSearchParams, upsertScope } from "@/utils";

const props = defineProps<{
  params: SearchParams;
}>();

const emit = defineEmits<{
  (e: "update:params", params: SearchParams): void;
}>();

const { t } = useI18n(); // NOSONAR

const presets = computed(() => [
  { id: "active", label: t("common.active") },
  { id: "expired", label: t("sql-editor.expired") },
  { id: "all", label: t("common.all") },
]);

const activePreset = computed(() => {
  const status = getValueFromSearchParams(props.params, "status");
  return status || "active";
});

const selectPreset = (id: string) => {
  const updated = upsertScope({
    params: props.params,
    scopes: { id: "status", value: id },
  });
  emit("update:params", updated);
};
</script>
