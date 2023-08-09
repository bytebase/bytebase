<template>
  <ExprEditor
    :expr="template.expr"
    :risk-source="source"
    :allow-admin="false"
    :allow-high-level-factors="false"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { Risk_Source } from "@/types/proto/v1/risk_service";
import { ExprEditor } from "../../common";
import { type RuleTemplate } from "./template";

const props = defineProps<{
  template: RuleTemplate;
}>();

const source = computed(() => {
  const { source } = props.template;
  if (source === Risk_Source.SOURCE_UNSPECIFIED) return Risk_Source.DDL;
  return source;
});
</script>
