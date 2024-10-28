<template>
  <div class="space-y-4">
    <div
      class="flex flex-col sm:flex-row gap-y-4 justify-between items-end sm:items-center"
    >
      <NInputGroup style="width: auto">
        <DatabaseSelect
          style="width: 12rem"
          :include-all="false"
          :clearable="true"
          :project-name="project.name"
          :database-name="state.selectedDatabaseName"
        />
        <MaskingLevelDropdown
          v-model:level="state.selectedMaskLevel"
          style="width: 12rem"
          :clearable="true"
          :level-list="[MaskingLevel.PARTIAL, MaskingLevel.NONE]"
        />
        <MaskingActionDropdown
          v-model:action="state.selectedAction"
          style="width: 12rem"
          :clearable="true"
          :action-list="[Action.EXPORT, Action.QUERY]"
        />
      </NInputGroup>

      <SearchBox
        ref="searchField"
        v-model:value="state.searchText"
        :placeholder="$t('settings.members.search-member')"
      />
    </div>
    <MaskingExceptionUserTable
      size="medium"
      :disabled="false"
      :project="project.name"
      :show-database-column="true"
      :filter-exception="filterException"
    />
  </div>
</template>

<script lang="tsx" setup>
import { NInputGroup } from "naive-ui";
import { computed, reactive } from "vue";
import MaskingExceptionUserTable from "@/components/SensitiveData/MaskingExceptionUserTable.vue";
import MaskingActionDropdown from "@/components/SensitiveData/components/MaskingActionDropdown.vue";
import MaskingLevelDropdown from "@/components/SensitiveData/components/MaskingLevelDropdown.vue";
import { SearchBox, DatabaseSelect } from "@/components/v2";
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { MaskingLevel } from "@/types/proto/v1/common";
import type { MaskingExceptionPolicy_MaskingException } from "@/types/proto/v1/org_policy_service";
import { MaskingExceptionPolicy_MaskingException_Action as Action } from "@/types/proto/v1/org_policy_service";
import { extractDatabaseResourceName } from "@/utils";

interface LocalState {
  searchText: string;
  selectedEnvironmentName?: string;
  selectedDatabaseName?: string;
  selectedMaskLevel?: MaskingLevel;
  selectedAction?: Action;
}

const props = defineProps<{
  projectId: string;
}>();

const state = reactive<LocalState>({
  searchText: "",
});

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const filterException = (
  exception: MaskingExceptionPolicy_MaskingException
): boolean => {
  if (
    state.selectedMaskLevel &&
    exception.maskingLevel !== state.selectedMaskLevel
  ) {
    return false;
  }
  if (state.selectedAction && exception.action !== state.selectedAction) {
    return false;
  }
  if (
    state.searchText.trim() &&
    !exception.member.toLowerCase().includes(state.searchText.trim())
  ) {
    return false;
  }
  if (state.selectedDatabaseName) {
    const { instanceName, databaseName } = extractDatabaseResourceName(
      state.selectedDatabaseName
    );
    const expression = [
      `resource.instance_id == "${instanceName}"`,
      `resource.database_name == "${databaseName}"`,
    ].join(" && ");
    return exception.condition?.expression?.includes(expression) ?? false;
  }

  return true;
};
</script>
