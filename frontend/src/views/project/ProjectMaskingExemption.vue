<template>
  <div class="flex flex-col">
    <div class="px-4 py-3">
      <div class="flex flex-row items-center gap-x-2 mb-2">
        <ExemptionSearchBar
          class="flex-1 min-w-0"
          :params="searchParams"
          :project-name="projectName"
          @update:params="searchParams = $event"
        />

        <PermissionGuardWrapper
          v-slot="slotProps"
          :project="project"
          :permissions="[
            'bb.policies.createMaskingExemptionPolicy',
            'bb.policies.updateMaskingExemptionPolicy',
            'bb.databases.list',
            'bb.databaseCatalogs.get',
          ]"
        >
          <NButton
            type="primary"
            :disabled="slotProps.disabled"
            @click="handleGrantClick"
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
      <ExemptionPresetTabs
        :params="searchParams"
        @update:params="searchParams = $event"
      />
    </div>

    <!-- Wide view: side-by-side.
         11rem ≈ global nav (48px) + project selector (44px) + search bar (36px)
         + tabs (40px) + padding (8px). Both panels scroll independently within
         this fixed height. A flex-1 approach would require modifying shared
         layout ancestors (ProjectV1Layout, BodyLayout) which affects all pages. -->
    <template v-if="isWide">
      <div
        class="flex"
        :style="{ height: 'calc(100vh - 11rem)' }"
      >
        <ExemptionMemberList
          class="w-[360px] shrink-0 border-r border-gray-200 overflow-y-auto"
          :members="filteredMembers"
          :disabled="!hasPermission"
          :loading="exemptionsLoading"
          :selected-member-key="selectedMemberKey"
          @select="selectedMemberKey = $event"
          @revoke="handleRevoke"
        />
        <div class="flex-1 min-w-0 overflow-y-auto">
          <ExemptionDetailPanel
            v-if="selectedMemberData"
            :member="selectedMemberData"
            :disabled="!hasPermission"
            :show-database-link="showDatabaseLink"
            :database-filter="activeDatabaseFilter"
            @revoke="(grant) => handleRevoke(selectedMemberData!, grant)"
          />
          <div
            v-else
            class="flex items-center justify-center h-full text-control-placeholder text-sm"
          >
            {{ $t("project.masking-exemption.no-exemptions") }}
          </div>
        </div>
      </div>
    </template>

    <!-- Narrow view: expandable list -->
    <template v-else>
      <ExemptionMemberList
        :members="filteredMembers"
        :disabled="!hasPermission"
        :loading="exemptionsLoading"
        :show-database-link="showDatabaseLink"
        :expandable="true"
        :database-filter="activeDatabaseFilter"
        @revoke="handleRevoke"
      />
    </template>
  </div>

  <FeatureModal
    :open="showFeatureModal"
    :feature="PlanFeature.FEATURE_DATA_MASKING"
    @cancel="showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { ShieldCheckIcon } from "lucide-vue-next";
import { NButton, useDialog } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { FeatureBadge, FeatureModal } from "@/components/FeatureGuard";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import ExemptionDetailPanel from "@/components/SensitiveData/ExemptionDetailPanel.vue";
import ExemptionMemberList from "@/components/SensitiveData/ExemptionMemberList.vue";
import ExemptionPresetTabs from "@/components/SensitiveData/ExemptionPresetTabs.vue";
import ExemptionSearchBar from "@/components/SensitiveData/ExemptionSearchBar.vue";
import { buildMemberSummary } from "@/components/SensitiveData/exemptionDataUtils";
import type {
  ExemptionGrant,
  ExemptionMember,
} from "@/components/SensitiveData/types";
import { useExemptionData } from "@/components/SensitiveData/useExemptionData";
import { useWideScreen } from "@/composables/useWideScreen";
import { PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE } from "@/router/dashboard/projectV1";
import { hasFeature, useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  getValueFromSearchParams,
  hasProjectPermissionV2,
  type SearchParams,
} from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

const { t } = useI18n(); // NOSONAR
const router = useRouter(); // NOSONAR
const $dialog = useDialog(); // NOSONAR

const projectName = computed(() => `${projectNamePrefix}${props.projectId}`);
const { project } = useProjectByName(projectName); // NOSONAR

const isWide = useWideScreen(1024); // NOSONAR
const showFeatureModal = ref(false);

const showDatabaseLink = computed(() =>
  hasProjectPermissionV2(project.value, "bb.databases.get")
);

const hasPermission = computed(() =>
  hasProjectPermissionV2(
    project.value,
    "bb.policies.updateMaskingExemptionPolicy"
  )
);

const hasSensitiveDataFeature = computed(() =>
  hasFeature(PlanFeature.FEATURE_DATA_MASKING)
);

const searchParams = ref<SearchParams>({
  query: "",
  scopes: [{ id: "status", value: "active" }],
});

const {
  members,
  loading: exemptionsLoading,
  revokeGrant,
} = useExemptionData(projectName); // NOSONAR

const activeDatabaseFilter = computed(
  () => getValueFromSearchParams(searchParams.value, "database") || undefined
);

const isGrantActive = (g: ExemptionGrant): boolean =>
  !g.expirationTimestamp || g.expirationTimestamp > Date.now();

// Rebuild an ExemptionMember with a subset of grants and recalculated summaries.
const withFilteredGrants = (
  m: ExemptionMember,
  grants: ExemptionGrant[]
): ExemptionMember => ({
  ...m,
  grants,
  ...buildMemberSummary(grants),
});

const filteredMembers = computed(() => {
  let result = members.value;

  // Free-text query: filter members by typed text (matches member identifier)
  const query = searchParams.value.query.trim().toLowerCase();
  if (query) {
    result = result.filter((m) => m.member.toLowerCase().includes(query));
  }

  const memberScope = getValueFromSearchParams(searchParams.value, "member");
  if (memberScope) {
    result = result.filter((m) =>
      m.member.toLowerCase().includes(memberScope.toLowerCase())
    );
  }

  // Status filter first: narrow grants before applying database filter.
  const statusScope = getValueFromSearchParams(searchParams.value, "status");
  if (statusScope === "active") {
    result = result
      .map((m) => withFilteredGrants(m, m.grants.filter(isGrantActive)))
      .filter((m) => m.grants.length > 0);
  } else if (statusScope === "expired") {
    result = result
      .map((m) =>
        withFilteredGrants(
          m,
          m.grants.filter((g) => !isGrantActive(g))
        )
      )
      .filter((m) => m.grants.length > 0);
  }

  // Database filter: prune grants at grant level (like status filter).
  // Global grants (no databaseResources) match any database filter.
  const dbScope = activeDatabaseFilter.value;
  if (dbScope) {
    const matchesDb = (g: ExemptionGrant) =>
      !g.databaseResources ||
      g.databaseResources.length === 0 ||
      g.databaseResources.some((r) => r.databaseFullName === dbScope);
    result = result
      .map((m) => withFilteredGrants(m, m.grants.filter(matchesDb)))
      .filter((m) => m.grants.length > 0);
  }

  return result;
});

const selectedMemberKey = ref("");

const selectedMemberData = computed(() =>
  filteredMembers.value.find((m) => m.member === selectedMemberKey.value)
);

const handleGrantClick = () => {
  if (!hasSensitiveDataFeature.value) {
    showFeatureModal.value = true;
    return;
  }
  router.push({
    name: PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE,
  });
};

const stripMemberPrefix = (raw: string): string => {
  const idx = raw.indexOf(":");
  return idx >= 0 ? raw.substring(idx + 1) : raw;
};

const handleRevoke = (member: ExemptionMember, grant: ExemptionGrant) => {
  $dialog.warning({
    title: t("common.warning"),
    content: t("project.masking-exemption.revoke-exemption-title", {
      member: stripMemberPrefix(member.member),
    }),
    negativeText: t("common.cancel"),
    positiveText: t("common.confirm"),
    onPositiveClick: async () => {
      await revokeGrant(member, grant);
    },
  });
};
</script>
