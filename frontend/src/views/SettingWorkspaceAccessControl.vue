<template>
  <div class="w-full mt-4 space-y-4">
    <FeatureAttention
      v-if="!hasAccessControlFeature"
      feature="bb.feature.access-control"
    />
    <div class="flex justify-between">
      <i18n-t
        tag="div"
        keypath="settings.access-control.description"
        class="textinfolabel"
      >
        <template #link>
          <LearnMoreLink
            url="https://www.bytebase.com/docs/security/data-access-control"
          />
        </template>
      </i18n-t>

      <div>
        <button
          class="btn-primary whitespace-nowrap"
          :disabled="!allowAdmin || !hasAccessControlFeature"
          @click="state.showAddRuleModal = true"
        >
          {{ $t("settings.access-control.add-rule") }}
        </button>
      </div>
    </div>

    <div v-if="hasAccessControlFeature" class="relative min-h-[12rem]">
      <BBTable
        :column-list="COLUMN_LIST"
        :data-source="activePolicyList"
        :show-header="true"
        :left-bordered="true"
        :right-bordered="true"
        :row-clickable="false"
      >
        <template #body="{ rowData: policy }: { rowData: Policy }">
          <BBTableCell class="w-[25%]" :left-padding="4">
            <DatabaseV1Name
              :database="databaseOfPolicy(policy)"
              :link="false"
            />
          </BBTableCell>
          <BBTableCell class="w-[15%]">
            <ProjectV1Name
              :project="databaseOfPolicy(policy).projectEntity"
              :link="false"
            />
          </BBTableCell>
          <BBTableCell class="w-[15%]">
            <EnvironmentV1Name
              :environment="
                databaseOfPolicy(policy).instanceEntity.environmentEntity
              "
              :link="false"
            />
          </BBTableCell>
          <BBTableCell class="w-[15%]">
            <InstanceV1Name
              :instance="databaseOfPolicy(policy).instanceEntity"
              :link="false"
            />
          </BBTableCell>
          <BBTableCell>
            <div class="flex items-center justify-center">
              <NPopconfirm @positive-click="handleRemove(policy)">
                <template #trigger>
                  <button
                    :disabled="!allowAdmin"
                    class="w-5 h-5 p-0.5 bg-white hover:bg-control-bg-hover rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
                  >
                    <heroicons-outline:trash />
                  </button>
                </template>

                <div class="whitespace-nowrap">
                  {{ $t("settings.access-control.remove-policy-tips") }}
                </div>
              </NPopconfirm>
            </div>
          </BBTableCell>
        </template>
      </BBTable>

      <div
        v-if="!state.isLoading && activePolicyList.length === 0"
        class="w-full flex flex-col py-6 justify-start items-center"
      >
        <heroicons-outline:inbox class="w-12 h-auto text-gray-500" />
        <span class="text-sm leading-6 text-gray-500">{{
          $t("common.no-data")
        }}</span>
      </div>

      <template v-if="inactivePolicyList.length > 0">
        <div class="text-lg mt-6 mb-1">
          {{ $t("settings.access-control.inactive-rules") }}
        </div>
        <div class="textinfolabel max-w-xl mb-4">
          {{ $t("settings.access-control.inactive-rules-description") }}
        </div>
        <BBTable
          :column-list="COLUMN_LIST"
          :data-source="inactivePolicyList"
          :show-header="true"
          :left-bordered="true"
          :right-bordered="true"
          :row-clickable="false"
        >
          <template #body="{ rowData: policy }: { rowData: Policy }">
            <BBTableCell class="w-[25%]" :left-padding="4">
              <div class="flex items-center space-x-2">
                <span>{{ databaseOfPolicy(policy).name }}</span>
              </div>
            </BBTableCell>
            <BBTableCell class="w-[15%]">
              <ProjectV1Name
                :project="databaseOfPolicy(policy).projectEntity"
                :link="false"
              />
            </BBTableCell>
            <BBTableCell class="w-[15%]">
              <EnvironmentV1Name
                :environment="
                  databaseOfPolicy(policy).instanceEntity.environmentEntity
                "
                :link="false"
              />
            </BBTableCell>
            <BBTableCell class="w-[15%]">
              <InstanceV1Name
                :instance="databaseOfPolicy(policy).instanceEntity"
                :link="false"
              />
            </BBTableCell>
            <BBTableCell>
              <div class="flex items-center justify-center">
                <NPopconfirm @positive-click="handleRemove(policy)">
                  <template #trigger>
                    <button
                      :disabled="!allowAdmin"
                      class="w-5 h-5 p-0.5 bg-white hover:bg-control-bg-hover rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
                    >
                      <heroicons-outline:trash />
                    </button>
                  </template>

                  <div class="whitespace-nowrap">
                    {{ $t("settings.access-control.remove-policy-tips") }}
                  </div>
                </NPopconfirm>
              </div>
            </BBTableCell>
          </template>
        </BBTable>
      </template>

      <div
        v-if="state.isLoading || state.isUpdating"
        class="absolute w-full h-full inset-0 bg-white/50 flex flex-col items-center justify-center"
      >
        <BBSpin />
      </div>
    </div>

    <template v-else>
      <BBTable
        :column-list="COLUMN_LIST"
        :data-source="[]"
        :show-header="true"
        :left-bordered="true"
        :right-bordered="true"
        :row-clickable="false"
      />
      <div class="w-full h-full flex flex-col items-center justify-center">
        <img src="../assets/illustration/no-data.webp" class="max-h-[30vh]" />
      </div>
    </template>
  </div>

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.access-control"
    @cancel="state.showFeatureModal = false"
  />

  <Drawer
    :show="state.showAddRuleModal"
    @close="state.showAddRuleModal = false"
  >
    <AddRuleForm
      :policy-list="state.policyList"
      :database-list="state.databaseList"
      @cancel="state.showAddRuleModal = false"
      @add="handleAddRule"
    />
  </Drawer>
</template>

<script lang="ts" setup>
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { NPopconfirm } from "naive-ui";
import { uniq } from "lodash-es";

import { featureToRef, useCurrentUserV1, useDatabaseV1Store } from "@/store";
import { ComposedDatabase, DEFAULT_PROJECT_V1_NAME } from "@/types";
import { BBTableColumn } from "@/bbkit/types";
import { hasWorkspacePermissionV1 } from "@/utils";
import { Drawer } from "@/components/v2";
import AddRuleForm from "@/components/AccessControl/AddRuleForm.vue";
import { usePolicyV1Store } from "@/store/modules/v1/policy";
import {
  Policy,
  PolicyType,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import { EnvironmentTier } from "@/types/proto/v1/environment_service";
import {
  InstanceV1Name,
  ProjectV1Name,
  EnvironmentV1Name,
} from "@/components/v2";

interface LocalState {
  showFeatureModal: boolean;
  showAddRuleModal: boolean;
  isLoading: boolean;
  isUpdating: boolean;
  policyList: Policy[];
  databaseList: ComposedDatabase[];
}

const { t } = useI18n();
const state = reactive<LocalState>({
  showFeatureModal: false,
  showAddRuleModal: false,
  isLoading: false,
  isUpdating: false,
  policyList: [],
  databaseList: [],
});
const databaseStore = useDatabaseV1Store();
const policyStore = usePolicyV1Store();
const hasAccessControlFeature = featureToRef("bb.feature.access-control");

const currentUserV1 = useCurrentUserV1();
const allowAdmin = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-access-control",
    currentUserV1.value.userRole
  );
});

const databaseOfPolicy = (policy: Policy) => {
  return databaseStore.getDatabaseByUID(policy.resourceUid);
};

const activePolicyList = computed(() => {
  return state.policyList.filter(
    (policy) =>
      databaseOfPolicy(policy).instanceEntity.environmentEntity.tier ===
      EnvironmentTier.PROTECTED
  );
});

const inactivePolicyList = computed(() => {
  return state.policyList.filter(
    (policy) =>
      databaseOfPolicy(policy).instanceEntity.environmentEntity.tier ===
      EnvironmentTier.UNPROTECTED
  );
});

const prepareList = async () => {
  state.isLoading = true;

  const policyList = await policyStore.fetchPolicies({
    policyType: PolicyType.ACCESS_CONTROL,
    resourceType: PolicyResourceType.DATABASE,
  });

  const allDatabaseList = await databaseStore.fetchDatabaseList({
    parent: "instances/-",
  });
  state.databaseList = allDatabaseList
    .filter(
      (db) =>
        db.instanceEntity.environmentEntity.tier === EnvironmentTier.PROTECTED
    )
    .filter((db) => db.project !== DEFAULT_PROJECT_V1_NAME);

  // For some policy related databases that are not in the state.databaseList,
  // fetch them.
  const databaseIdList = uniq(
    policyList
      .map((policy) => policy.resourceUid)
      .filter((databaseId) =>
        state.databaseList.findIndex((db) => db.uid == databaseId)
      )
  );

  Promise.all(
    databaseIdList.map((databaseId) =>
      databaseStore.getOrFetchDatabaseByUID(databaseId)
    )
  );

  state.policyList = policyList;

  state.isLoading = false;
};

watchEffect(prepareList);

const handleAddRule = async (databaseList: ComposedDatabase[]) => {
  if (!hasAccessControlFeature.value) {
    state.showFeatureModal = true;
    return;
  }

  state.showAddRuleModal = false;
  state.isUpdating = true;
  try {
    for (let i = 0; i < databaseList.length; i++) {
      const database = databaseList[i];
      await policyStore.upsertPolicy({
        parentPath: database.name,
        updateMask: ["payload", "inherit_from_parent"],
        policy: {
          type: PolicyType.ACCESS_CONTROL,
          inheritFromParent: false,
          accessControlPolicy: {
            disallowRules: [],
          },
        },
      });
    }
  } finally {
    state.isUpdating = false;
  }
  prepareList();
};

const handleRemove = async (policy: Policy) => {
  if (!hasAccessControlFeature.value) {
    state.showFeatureModal = true;
    return;
  }

  state.isUpdating = true;
  try {
    await policyStore.deletePolicy(policy.name);

    prepareList();
  } finally {
    state.isUpdating = false;
  }
};

const COLUMN_LIST = computed((): BBTableColumn[] => [
  {
    title: t("common.database"),
  },
  {
    title: t("common.project"),
  },
  {
    title: t("common.environment"),
  },
  {
    title: t("common.instance"),
  },
  {
    title: t("common.operation"),
    center: true,
    nowrap: true,
  },
]);
</script>
