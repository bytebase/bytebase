<template>
  <div class="py-4 flex flex-col">
    <div class="px-4 pb-2">
      <div
        class="flex flex-row justify-between items-center gap-x-2"
      >
      <div class="flex-1">
        <SearchBox
          ref="searchField"
          v-model:value="state.searchText"
          style="max-width: 100%"
          :placeholder="$t('settings.members.search-member')"
        />
      </div>
        <PermissionGuardWrapper
          v-slot="slotProps"
          :project="project"
          :permissions="[
            'bb.policies.createMaskingExemptionPolicy',
            'bb.policies.updateMaskingExemptionPolicy',
            'bb.databases.list',
            'bb.databaseCatalogs.get'
          ]"
        >
          <NButton
            type="primary"
            :disabled="slotProps.disabled"
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
        </PermissionGuardWrapper>
      </div>
    </div>
    <MaskingExceptionUserTable
      size="medium"
      :disabled="!hasPermission"
      :project="project"
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
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import MaskingExceptionUserTable from "@/components/SensitiveData/MaskingExceptionUserTable.vue";
import { type AccessUser } from "@/components/SensitiveData/types";
import { SearchBox } from "@/components/v2";
import { PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE } from "@/router/dashboard/projectV1";
import { hasFeature, useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasProjectPermissionV2 } from "@/utils";

interface LocalState {
  searchText: string;
  showFeatureModal: boolean;
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

const hasPermission = computed(() =>
  hasProjectPermissionV2(
    project.value,
    "bb.policies.updateMaskingExemptionPolicy"
  )
);

const hasSensitiveDataFeature = computed(() => {
  return hasFeature(PlanFeature.FEATURE_DATA_MASKING);
});

const filterAccessUser = (user: AccessUser): boolean => {
  if (
    state.searchText.trim() &&
    !user.key.toLowerCase().includes(state.searchText.trim())
  ) {
    return false;
  }
  return true;
};
</script>
