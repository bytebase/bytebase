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
import { useProjectV1Store, useSchemaDesignList } from "@/store";
import {
  getProjectAndSchemaDesignSheetId,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";

interface BranchSelectOption extends SelectOption {
  value: string;
  branch: SchemaDesign;
}

const props = defineProps<{
  project?: string;
  branch?: string;
  clearable?: boolean;
  filter?: (branch: SchemaDesign, index: number) => boolean;
}>();

defineEmits<{
  (event: "update:branch", name: string | undefined): void;
}>();

const { schemaDesignList: branchList } = useSchemaDesignList();
const projectStore = useProjectV1Store();

const combinedBranchList = computed(() => {
  let list = branchList.value;
  if (props.project) {
    const project = projectStore.getProjectByUID(props.project);
    if (project) {
      list = list.filter((branch) => {
        const [projectName] = getProjectAndSchemaDesignSheetId(branch.name);
        return project.name === `${projectNamePrefix}${projectName}`;
      });
    }
  }
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
