<template>
  <div class="space-y-4">
    <div
      class="flex flex-col lg:flex-row gap-y-4 justify-between items-end lg:items-center gap-x-2"
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

      <div class="flex-1 flex flex-row items-center justify-end gap-x-2">
        <SearchBox
          ref="searchField"
          style="max-width: 100%"
          v-model:value="state.searchText"
          :placeholder="$t('settings.members.search-member')"
        />
        <NButton
          v-if="allowCreate"
          type="primary"
          @click="
            () => {
              if (!hasSensitiveDataFeature) {
                state.showFeatureModal = true;
                return;
              }
              router.push({
                name: PROJECT_V1_ROUTE_MASKING_ACCESS_CREATE,
              });
            }
          "
        >
          <template #icon>
            <ShieldCheckIcon v-if="hasSensitiveDataFeature" class="w-4" />
            <FeatureBadge
              v-else
              feature="bb.feature.sensitive-data"
              custom-class="text-white"
            />
          </template>
          {{ $t("project.masking-access.grant-access") }}
        </NButton>
      </div>
    </div>
    <MaskingExceptionUserTable
      size="medium"
      :disabled="false"
      :project="project.name"
      :show-database-column="true"
      :filter-exception="filterException"
    />
  </div>

  <FeatureModal
    :open="state.showFeatureModal"
    feature="bb.feature.sensitive-data"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="tsx" setup>
import { ShieldCheckIcon } from "lucide-vue-next";
import { NInputGroup, NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import { FeatureModal, FeatureBadge } from "@/components/FeatureGuard";
import MaskingExceptionUserTable from "@/components/SensitiveData/MaskingExceptionUserTable.vue";
import MaskingActionDropdown from "@/components/SensitiveData/components/MaskingActionDropdown.vue";
import MaskingLevelDropdown from "@/components/SensitiveData/components/MaskingLevelDropdown.vue";
import { SearchBox, DatabaseSelect } from "@/components/v2";
import { PROJECT_V1_ROUTE_MASKING_ACCESS_CREATE } from "@/router/dashboard/projectV1";
import { useProjectByName, hasFeature } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { MaskingLevel } from "@/types/proto/v1/common";
import type { MaskingExceptionPolicy_MaskingException } from "@/types/proto/v1/org_policy_service";
import { MaskingExceptionPolicy_MaskingException_Action as Action } from "@/types/proto/v1/org_policy_service";
import { extractDatabaseResourceName, hasProjectPermissionV2 } from "@/utils";

interface LocalState {
  searchText: string;
  showFeatureModal: boolean;
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
  showFeatureModal: false,
});
const router = useRouter();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const hasSensitiveDataFeature = computed(() => {
  return hasFeature("bb.feature.sensitive-data");
});

const allowCreate = computed(() => {
  return (
    hasProjectPermissionV2(project.value, "bb.databases.list") &&
    hasProjectPermissionV2(project.value, "bb.policies.create")
  );
});

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
