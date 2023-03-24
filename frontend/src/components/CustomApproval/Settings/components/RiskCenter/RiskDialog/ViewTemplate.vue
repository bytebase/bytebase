<template>
  <ExprEditor
    :expr="template.expr"
    :risk-source="source"
    :allow-admin="false"
    :allow-high-level-factors="false"
  />
</template>

<script lang="ts" setup>
import { computed, h } from "vue";

import { useOverrideSubtitle } from "@/bbkit/BBModal.vue";
import { Risk_Source } from "@/types/proto/v1/risk_service";
import { ExprEditor } from "../../common";
import { type RuleTemplate, titleOfTemplate } from "./template";

const props = defineProps<{
  template: RuleTemplate;
}>();

useOverrideSubtitle(() => {
  return h(
    "div",
    {
      class:
        "text-xs text-control-light mt-1 whitespace-pre-wrap overflow-hidden",
    },
    titleOfTemplate(props.template)
  );
});

const source = computed(() => {
  const { source } = props.template;
  if (source === Risk_Source.SOURCE_UNSPECIFIED) return Risk_Source.DDL;
  return source;
});
</script>
