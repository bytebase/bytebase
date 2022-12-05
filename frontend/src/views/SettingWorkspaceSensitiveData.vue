<template>
  <div class="w-full mt-4 space-y-4">
    <div class="textinfolabel">
      {{ $t("settings.sensitive-data.description") }}
    </div>

    <BBTable
      :column-list="COLUMN_LIST"
      :data-source="state.sensitiveColumnList"
      :show-header="true"
      :left-bordered="true"
      :right-bordered="true"
      :row-clickable="false"
    >
      <template #body="{ rowData: item }: { rowData: SensitiveColumn }">
        <BBTableCell :left-padding="4" class="w-[15%]">
          {{ item.column }}
        </BBTableCell>
        <BBTableCell class="w-[15%]">
          {{ item.table }}
        </BBTableCell>
        <BBTableCell class="w-[15%]">
          <div class="flex items-center space-x-2">
            <span>{{ item.database.name }}</span>
          </div>
        </BBTableCell>
        <BBTableCell class="w-[15%]">
          <div class="flex flex-row items-center space-x-1">
            <InstanceEngineIcon :instance="item.database.instance" />
            <span class="flex-1 whitespace-pre-wrap">
              {{ instanceName(item.database.instance) }}
            </span>
          </div>
        </BBTableCell>
        <BBTableCell class="w-[10%]">
          <div class="flex items-center">
            {{ environmentName(item.database.instance.environment) }}
            <ProtectedEnvironmentIcon
              class="ml-1"
              :environment="item.database.instance.environment"
            />
          </div>
        </BBTableCell>
        <BBTableCell class="w-[15%]">
          {{ projectName(item.database.project) }}
        </BBTableCell>
        <BBTableCell>
          {{ humanizeTs(item.policy.updatedTs) }}
        </BBTableCell>
        <BBTableCell>
          <div class="flex items-center justify-center">
            <NPopconfirm @positive-click="removeSensitiveColumn(item)">
              <template #trigger>
                <button
                  :disabled="!allowAdmin"
                  class="w-5 h-5 p-0.5 bg-white hover:bg-control-bg-hover rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
                >
                  <heroicons-outline:trash />
                </button>
              </template>

              <div class="whitespace-nowrap">
                {{ $t("settings.sensitive-data.remove-sensitive-column-tips") }}
              </div>
            </NPopconfirm>
          </div>
        </BBTableCell>
      </template>
    </BBTable>
  </div>

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.sensitive-data"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { NPopconfirm } from "naive-ui";

import {
  featureToRef,
  useCurrentUser,
  useDatabaseStore,
  usePolicyListByResourceTypeAndPolicyType,
  usePolicyStore,
} from "@/store";
import { Database, Policy, SensitiveDataPolicyPayload } from "@/types";
import { BBTableColumn } from "@/bbkit/types";
import { hasWorkspacePermission } from "@/utils";

type SensitiveColumn = {
  database: Database;
  policy: Policy;
  table: string;
  column: string;
};
interface LocalState {
  showFeatureModal: boolean;
  isLoading: boolean;
  sensitiveColumnList: SensitiveColumn[];
}

const { t } = useI18n();
const state = reactive<LocalState>({
  showFeatureModal: false,
  isLoading: false,
  sensitiveColumnList: [],
});
const databaseStore = useDatabaseStore();
const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");

const currentUser = useCurrentUser();
const allowAdmin = computed(() => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-sensitive-data",
    currentUser.value.role
  );
});

const policyList = usePolicyListByResourceTypeAndPolicyType(
  computed(() => ({
    resourceType: "database",
    policyType: "bb.policy.sensitive-data",
  }))
);

const updateList = async () => {
  state.isLoading = true;
  const sensitiveColumnList: SensitiveColumn[] = [];
  for (let i = 0; i < policyList.value.length; i++) {
    const policy = policyList.value[i];
    const payload = policy.payload as SensitiveDataPolicyPayload;

    const databaseId = policy.resourceId;
    const database = await databaseStore.getOrFetchDatabaseById(databaseId);

    for (let j = 0; j < payload.sensitiveDataList.length; j++) {
      const { table, column } = payload.sensitiveDataList[j];
      sensitiveColumnList.push({ database, policy, table, column });
    }
  }
  state.sensitiveColumnList = sensitiveColumnList;
  state.isLoading = false;
};

watchEffect(updateList);

const removeSensitiveColumn = (sensitiveColumn: SensitiveColumn) => {
  if (!hasSensitiveDataFeature.value) {
    state.showFeatureModal = true;
    return;
  }

  const { database, table, column } = sensitiveColumn;
  const policy = policyList.value.find(
    (policy) => policy.resourceId === sensitiveColumn.database.id
  );
  if (!policy) return;

  const payload = policy.payload as SensitiveDataPolicyPayload;
  const index = payload.sensitiveDataList.findIndex(
    (sensitiveData) =>
      sensitiveData.table === table && sensitiveData.column === column
  );
  if (index >= 0) {
    // mutate the list and the item directly
    // so we don't need to re-fetch the whole list.
    payload.sensitiveDataList.splice(index, 1);

    usePolicyStore().upsertPolicyByDatabaseAndType({
      databaseId: database.id,
      type: "bb.policy.sensitive-data",
      policyUpsert: {
        payload,
      },
    });
  }
  updateList();
};

const COLUMN_LIST = computed((): BBTableColumn[] => [
  {
    title: t("database.column"),
  },
  {
    title: t("common.table"),
  },
  {
    title: t("common.database"),
  },
  {
    title: t("common.instance"),
  },
  {
    title: t("common.environment"),
  },
  {
    title: t("common.project"),
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
