<template>
  <div class="w-full mt-4 space-y-4">
    <div class="flex justify-between">
      <div class="textinfolabel max-w-xl">
        {{ $t("settings.access-control.description") }}
      </div>

      <div>
        <button
          class="btn-primary"
          :disabled="!allowAdmin"
          @click="state.showAddRuleModal = true"
        >
          {{ $t("settings.access-control.add-rule") }}
        </button>
      </div>
    </div>

    <div class="relative min-h-[12rem]">
      <BBTable
        :column-list="COLUMN_LIST"
        :data-source="state.policyList"
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
            {{ projectName(databaseOfPolicy(policy).project) }}
          </BBTableCell>
          <BBTableCell class="w-[15%]">
            <div class="flex items-center">
              {{
                environmentName(databaseOfPolicy(policy).instance.environment)
              }}
              <ProtectedEnvironmentIcon
                class="ml-1"
                :environment="databaseOfPolicy(policy).instance.environment"
              />
            </div>
          </BBTableCell>
          <BBTableCell class="w-[15%]">
            <div class="flex flex-row items-center space-x-1">
              <InstanceEngineIcon
                :instance="databaseOfPolicy(policy).instance"
              />
              <span class="flex-1 whitespace-pre-wrap">
                {{ instanceName(databaseOfPolicy(policy).instance) }}
              </span>
            </div>
          </BBTableCell>
          <BBTableCell>
            {{ humanizeTs(policy.updatedTs) }}
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
        v-if="!state.isLoading && state.policyList.length === 0"
        class="w-full flex flex-col py-6 justify-start items-center"
      >
        <heroicons-outline:inbox class="w-12 h-auto text-gray-500" />
        <span class="text-sm leading-6 text-gray-500">{{
          $t("common.no-data")
        }}</span>
      </div>

      <div
        v-if="state.isLoading || state.isUpdating"
        class="absolute w-full h-full inset-0 bg-white/50 flex flex-col items-center justify-center"
      >
        <BBSpin />
      </div>
    </div>
  </div>

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.access-control"
    @cancel="state.showFeatureModal = false"
  />

  <BBModal
    v-if="state.showAddRuleModal"
    :title="$t('settings.access-control.add-rule')"
    @close="state.showAddRuleModal = false"
  >
    <AddRuleForm
      :policy-list="state.policyList"
      @cancel="state.showAddRuleModal = false"
      @add="handleAddRule"
    />
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { NPopconfirm } from "naive-ui";

import {
  featureToRef,
  useCurrentUser,
  useDatabaseStore,
  usePolicyStore,
} from "@/store";
import {
  AccessControlPolicyPayload,
  Database,
  DatabaseId,
  Policy,
  PolicyUpsert,
} from "@/types";
import { BBTableColumn } from "@/bbkit/types";
import { hasWorkspacePermission } from "@/utils";
import AddRuleForm from "@/components/AccessControl/AddRuleForm.vue";
import { uniq } from "lodash-es";

interface LocalState {
  showFeatureModal: boolean;
  showAddRuleModal: boolean;
  isLoading: boolean;
  isUpdating: boolean;
  policyList: Policy[];
}

const { t } = useI18n();
const state = reactive<LocalState>({
  showFeatureModal: false,
  showAddRuleModal: false,
  isLoading: false,
  isUpdating: false,
  policyList: [],
});
const databaseStore = useDatabaseStore();
const policyStore = usePolicyStore();
const hasAccessControlFeature = featureToRef("bb.feature.access-control");

const currentUser = useCurrentUser();
const allowAdmin = computed(() => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-access-control",
    currentUser.value.role
  );
});

const databaseOfPolicy = (policy: Policy) => {
  return databaseStore.getDatabaseById(policy.resourceId as DatabaseId);
};

const prepareList = async () => {
  state.isLoading = true;

  await new Promise((resolve) => setTimeout(resolve, 500));

  const policyList = await policyStore.fetchPolicyListByType(
    "bb.policy.access-control"
  );

  const databaseIdList = uniq(
    policyList.map((policy) => policy.resourceId as DatabaseId)
  );

  Promise.all(
    databaseIdList.map((databaseId) =>
      databaseStore.getOrFetchDatabaseById(databaseId)
    )
  );
  state.policyList = policyList;

  state.isLoading = false;
};

watchEffect(prepareList);

const handleAddRule = async (databaseList: Database[]) => {
  if (!hasAccessControlFeature.value) {
    state.showFeatureModal = true;
    return;
  }

  state.showAddRuleModal = false;
  state.isUpdating = true;
  try {
    for (let i = 0; i < databaseList.length; i++) {
      const database = databaseList[i];
      const payload: AccessControlPolicyPayload = {
        disallowRuleList: [],
      };
      const policyUpsert: PolicyUpsert = {
        inheritFromParent: false,
        payload,
      };
      await policyStore.upsertPolicyByDatabaseAndType({
        databaseId: database.id,
        type: "bb.policy.access-control",
        policyUpsert,
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
    policyStore.deletePolicyByDatabaseAndType({
      databaseId: policy.resourceId,
      type: "bb.policy.access-control",
    });

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
    title: t("common.updated-at"),
    nowrap: true,
  },
  {
    title: t("common.operation"),
    center: true,
    nowrap: true,
  },
]);
</script>
