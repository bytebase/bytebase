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
    :render-label="renderLabel"
    @update:value="$emit('update:branch', $event)"
  />
</template>

<script lang="ts" setup>
import { NSelect, SelectOption, SelectRenderLabel } from "naive-ui";
import { computed, h } from "vue";
import {
  useDatabaseV1Store,
  useProjectV1Store,
  useSchemaDesignList,
} from "@/store";
import {
  getProjectAndSchemaDesignSheetId,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { UNKNOWN_ID } from "@/types";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import { InstanceV1EngineIcon } from "../v2";

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
const databaseStore = useDatabaseV1Store();

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

const renderLabel: SelectRenderLabel = (option) => {
  const { branch } = option as BranchSelectOption;
  if (!branch) {
    return;
  }

  const children = [h("div", {}, [branch.title])];
  const database = databaseStore.getDatabaseByName(branch.baselineDatabase);
  if (database.uid !== String(UNKNOWN_ID)) {
    // prefix engine icon
    children.unshift(
      h(InstanceV1EngineIcon, {
        class: "mr-1",
        instance: database.instanceEntity,
      })
    );
  }
  return h(
    "div",
    {
      class: "w-full flex flex-row justify-start items-center truncate",
    },
    children
  );
};
</script>
