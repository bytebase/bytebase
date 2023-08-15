<template>
  <TabFilter
    :value="selected"
    :items="items"
    @update:value="$emit('update:engine', $event)"
  >
    <template #label="{ item }">
      <div class="flex items-center justify-center gap-x-1">
        <RuleEngineIcon v-if="item.value !== UNKNOWN_ID" :engine="item.value" />
        {{ item.label }}
      </div>
    </template>
  </TabFilter>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { UNKNOWN_ID, SchemaRuleEngineType } from "@/types";
import { engineFromJSON } from "@/types/proto/v1/common";
import { engineNameV1 } from "@/utils";

const props = defineProps<{
  selected: string;
  engineList: SchemaRuleEngineType[];
  individualEngineList: SchemaRuleEngineType[];
}>();

defineEmits<{
  (event: "update:engine", id: string): void;
}>();

const { t } = useI18n();

const items = computed(() => {
  const resp = props.individualEngineList.map((engine) => {
    return {
      value: `${engine}`,
      label: engineNameV1(engineFromJSON(engine)),
    };
  });
  if (props.individualEngineList.length < props.engineList.length) {
    resp.unshift({
      value: `${UNKNOWN_ID}`,
      label: t("sql-review.other-engines"),
    });
  }
  return resp;
});
</script>
