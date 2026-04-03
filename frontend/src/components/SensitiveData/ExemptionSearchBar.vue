<template>
  <AdvancedSearch
    v-model:params="localParams"
    :scope-options="scopeOptions"
    :placeholder="$t('issue.advanced-search.self')"
    :autofocus="false"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import AdvancedSearch from "@/components/AdvancedSearch";
import type { SearchParams } from "@/utils";
import { useExemptionSearchScopeOptions } from "./useExemptionSearchScopeOptions";

const props = defineProps<{
  params: SearchParams;
  projectName: string;
}>();

const emit = defineEmits<{
  (e: "update:params", params: SearchParams): void;
}>();

const projectNameRef = computed(() => props.projectName);
const scopeOptions = useExemptionSearchScopeOptions(projectNameRef); // NOSONAR

const localParams = computed({
  get: () => props.params,
  set: (val) => emit("update:params", val),
});
</script>
