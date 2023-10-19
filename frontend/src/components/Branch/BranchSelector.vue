<template>
  <NSelect
    v-bind="$attrs"
    :value="branch"
    :options="options"
    :placeholder="$t('database.select-branch')"
    :filterable="true"
    :clearable="clearable"
    :filter="filterByName"
    class="bb-branch-select"
    @update:value="$emit('update:branch', $event)"
  />
</template>

<script lang="ts" setup>
import { NSelect, SelectOption } from "naive-ui";
import { computed } from "vue";
import { useSchemaDesignList } from "@/store";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";

interface BranchSelectOption extends SelectOption {
  value: string;
  branch: SchemaDesign;
}

const props = withDefaults(
  defineProps<{
    branch?: string | undefined;
    filter?: (branch: SchemaDesign, index: number) => boolean;
    clearable?: boolean;
  }>(),
  {
    branch: undefined,
    clearable: true,
    filter: () => true,
  }
);

defineEmits<{
  (event: "update:branch", name: string | undefined): void;
}>();

const { schemaDesignList: branchList } = useSchemaDesignList();

const combinedBranchList = computed(() => {
  let list = branchList.value;
  if (props.filter) {
    list = list.filter(props.filter);
  }
  return list;
});

const options = computed(() => {
  return combinedBranchList.value.map<BranchSelectOption>((branch) => {
    return {
      branch,
      value: branch.name,
      label: branch.title,
    };
  });
});

const filterByName = (pattern: string, option: SelectOption) => {
  const { branch } = option as BranchSelectOption;
  pattern = pattern.toLowerCase();
  return (
    branch.name.toLowerCase().includes(pattern) ||
    branch.title.toLowerCase().includes(pattern)
  );
};
</script>

<style lang="postcss" scoped>
.bb-branch-select :deep(.n-base-selection-input:focus) {
  @apply !ring-0;
}
</style>
