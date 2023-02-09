<template>
  <div class="w-full mt-4 space-y-4">
    <div class="textinfolabel">
      {{ $t("settings.sensitive-data.description") }}
    </div>

    <BBGrid
      :column-list="COLUMN_LIST"
      :data-source="state.sensitiveColumnList"
      class="border"
      @click-row="clickRow"
    >
      <template #item="{ item }: { item: SensitiveColumn }">
        <div class="bb-grid-cell">
          {{ item.column }}
        </div>
        <div class="bb-grid-cell">
          {{ item.table }}
        </div>
        <div class="bb-grid-cell">
          {{ item.database.name }}
        </div>
        <div class="bb-grid-cell gap-x-1">
          <InstanceEngineIcon :instance="item.database.instance" />
          <span class="flex-1 whitespace-pre-wrap">
            {{ instanceName(item.database.instance) }}
          </span>
        </div>
        <div class="bb-grid-cell">
          {{ environmentName(item.database.instance.environment) }}
          <ProductionEnvironmentIcon
            class="ml-1 w-4 h-4"
            :environment="item.database.instance.environment"
          />
        </div>
        <div class="bb-grid-cell">
          {{ projectName(item.database.project) }}
        </div>
        <div class="bb-grid-cell justify-center !px-2">
          <NPopconfirm @positive-click="removeSensitiveColumn(item)">
            <template #trigger>
              <button
                :disabled="!allowAdmin"
                class="w-5 h-5 p-0.5 bg-white hover:bg-control-bg-hover rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
                @click.stop=""
              >
                <heroicons-outline:trash />
              </button>
            </template>

            <div class="whitespace-nowrap">
              {{ $t("settings.sensitive-data.remove-sensitive-column-tips") }}
            </div>
          </NPopconfirm>
        </div>
      </template>
    </BBGrid>
  </div>

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.sensitive-data"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { NPopconfirm } from "naive-ui";
import { uniq } from "lodash-es";
import { useRouter } from "vue-router";

import {
  featureToRef,
  useCurrentUser,
  useDatabaseStore,
  usePolicyListByResourceTypeAndPolicyType,
  usePolicyStore,
} from "@/store";
import { Database, Policy, SensitiveDataPolicyPayload } from "@/types";
import { BBGridColumn } from "@/bbkit/types";
import { databaseSlug, hasWorkspacePermission } from "@/utils";
import { BBGrid } from "@/bbkit";

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
const router = useRouter();
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

const policyList = usePolicyListByResourceTypeAndPolicyType({
  resourceType: "database",
  policyType: "bb.policy.sensitive-data",
});

const updateList = async () => {
  state.isLoading = true;
  const distinctDatabaseIdList = uniq(
    policyList.value.map((policy) => policy.resourceId)
  );
  // Fetch or get all needed databases
  await Promise.all(
    distinctDatabaseIdList.map((databaseId) =>
      databaseStore.getOrFetchDatabaseById(databaseId)
    )
  );

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

watch(policyList, updateList, { immediate: true });

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

const COLUMN_LIST = computed((): BBGridColumn[] => [
  {
    title: t("database.column"),
    width: "minmax(auto, 1fr)",
  },
  {
    title: t("common.table"),
    width: "minmax(auto, 1fr)",
  },
  {
    title: t("common.database"),
    width: "minmax(auto, 1fr)",
  },
  {
    title: t("common.instance"),
    width: "minmax(auto, 1fr)",
  },
  {
    title: t("common.environment"),
    width: "minmax(auto, 1fr)",
  },
  {
    title: t("common.project"),
    width: "minmax(auto, 1fr)",
  },
  {
    title: t("common.operation"),
    width: "minmax(auto, 6rem)",
    class: "justify-center !px-2",
  },
]);

const clickRow = (
  item: SensitiveColumn,
  section: number,
  row: number,
  e: MouseEvent
) => {
  const url = `/db/${databaseSlug(item.database)}/table/${item.table}`;
  if (e.ctrlKey || e.metaKey) {
    window.open(url, "_blank");
  } else {
    router.push(url);
  }
};
</script>
