<template>
  <div class="flex flex-col gap-y-4">
    <div
      class="flex flex-col lg:flex-row gap-y-4 justify-between items-end lg:items-center gap-x-2"
    >
      <DatabaseSelect
        style="max-width: max-content"
        :placeholder="$t('database.select')"
        :project-name="project.name"
        :show-instance="false"
        v-model:value="state.selectedDatabaseName"
      />

      <div class="flex-1 flex flex-row items-center justify-end gap-x-2">
        <SearchBox
          ref="searchField"
          v-model:value="state.searchText"
          style="max-width: 100%"
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
                name: PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE,
              });
            }
          "
        >
          <template #icon>
            <ShieldCheckIcon class="w-4" />
            <FeatureBadge
              :feature="PlanFeature.FEATURE_DATA_MASKING"
              class="text-white"
            />
          </template>
          {{ $t("project.masking-exemption.grant-exemption") }}
        </NButton>
      </div>
    </div>
    <MaskingExceptionUserTable
      size="medium"
      :disabled="false"
      :project="project"
      :show-database-column="true"
      :filter-access-user="filterAccessUser"
    />
  </div>

  <FeatureModal
    :open="state.showFeatureModal"
    :feature="PlanFeature.FEATURE_DATA_MASKING"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="tsx" setup>
import { ShieldCheckIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import { FeatureBadge, FeatureModal } from "@/components/FeatureGuard";
import MaskingExceptionUserTable from "@/components/SensitiveData/MaskingExceptionUserTable.vue";
import { type AccessUser } from "@/components/SensitiveData/types";
import { DatabaseSelect, SearchBox } from "@/components/v2";
import { PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE } from "@/router/dashboard/projectV1";
import { hasFeature, useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasProjectPermissionV2 } from "@/utils";

interface LocalState {
  searchText: string;
  showFeatureModal: boolean;
  selectedDatabaseName?: string;
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
  return hasFeature(PlanFeature.FEATURE_DATA_MASKING);
});

const allowCreate = computed(() => {
  return (
    hasProjectPermissionV2(project.value, "bb.databases.list") &&
    hasProjectPermissionV2(project.value, "bb.policies.create")
  );
});

const filterAccessUser = (user: AccessUser): boolean => {
  if (
    state.searchText.trim() &&
    !user.key.toLowerCase().includes(state.searchText.trim())
  ) {
    return false;
  }
  if (state.selectedDatabaseName) {
    if (!user.databaseResource) {
      return true;
    }
    return (
      user.databaseResource.databaseFullName === state.selectedDatabaseName
    );
  }

  return true;
};
</script>
