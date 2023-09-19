<template>
  <ExprEditor
    :expr="template.expr"
    :allow-admin="false"
    :factor-list="getFactorList(source)"
    :factor-support-dropdown="factorSupportDropdown"
    :factor-options-map="getFactorOptionsMap(source)"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import ExprEditor from "@/components/ExprEditor";
import { Risk_Source } from "@/types/proto/v1/risk_service";
import {
  getFactorList,
  getFactorOptionsMap,
  factorSupportDropdown,
} from "../../common/utils";
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
