<template>
  <div class="w-full mt-4 space-y-4">
    <div class="textinfolabel">
      {{ $t("settings.sensitive-data.description") }}
    </div>

    <BBGrid
      v-if="hasSensitiveDataFeature"
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
          {{ item.schema ? `${item.schema}.${item.table}` : item.table }}
        </div>
        <div class="bb-grid-cell">
          <DatabaseV1Name :database="item.database" :link="false" />
        </div>
        <div class="bb-grid-cell gap-x-1">
          <InstanceV1Name
            :instance="item.database.instanceEntity"
            :link="false"
          />
        </div>
        <div class="bb-grid-cell">
          <EnvironmentV1Name
            :environment="item.database.effectiveEnvironmentEntity"
            :link="false"
          />
        </div>
        <div class="bb-grid-cell">
          <ProjectV1Name :project="item.database.projectEntity" :link="false" />
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

    <template v-else>
      <BBGrid :column-list="COLUMN_LIST" :data-source="[]" class="border" />
      <div class="w-full h-full flex flex-col items-center justify-center">
        <img
          src="../../assets/illustration/no-data.webp"
          class="max-h-[30vh]"
        />
      </div>
    </template>
  </div>

  <FeatureModal
    feature="bb.feature.sensitive-data"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { uniq } from "lodash-es";
import { NPopconfirm } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBGrid, type BBGridColumn } from "@/bbkit";
import {
  DatabaseV1Name,
  EnvironmentV1Name,
  InstanceV1Name,
  ProjectV1Name,
} from "@/components/v2";
import { featureToRef, useCurrentUserV1, useDatabaseV1Store } from "@/store";
import {
  usePolicyListByResourceTypeAndPolicyType,
  usePolicyV1Store,
} from "@/store/modules/v1/policy";
import { ComposedDatabase } from "@/types";
import {
  PolicyType,
  Policy,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import { databaseV1Slug, hasWorkspacePermissionV1 } from "@/utils";

type SensitiveColumn = {
  database: ComposedDatabase;
  policy: Policy;
  schema: string;
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
const databaseStore = useDatabaseV1Store();
const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");

const currentUserV1 = useCurrentUserV1();
const allowAdmin = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sensitive-data",
    currentUserV1.value.userRole
  );
});

const policyList = usePolicyListByResourceTypeAndPolicyType({
  resourceType: PolicyResourceType.DATABASE,
  policyType: PolicyType.SENSITIVE_DATA,
  showDeleted: false,
});

const updateList = async () => {
  state.isLoading = true;
  const distinctDatabaseIdList = uniq(
    policyList.value.map((policy) => policy.resourceUid)
  );
  // Fetch or get all needed databases
  await Promise.all(
    distinctDatabaseIdList.map((databaseId) =>
      databaseStore.getOrFetchDatabaseByUID(databaseId)
    )
  );

  const sensitiveColumnList: SensitiveColumn[] = [];
  for (let i = 0; i < policyList.value.length; i++) {
    const policy = policyList.value[i];
    if (!policy.sensitiveDataPolicy) {
      continue;
    }

    const databaseId = policy.resourceUid;
    const database = await databaseStore.getOrFetchDatabaseByUID(databaseId);

    for (const sensitiveData of policy.sensitiveDataPolicy.sensitiveData) {
      const { schema, table, column } = sensitiveData;
      sensitiveColumnList.push({ database, policy, schema, table, column });
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

  const { table, column } = sensitiveColumn;
  const policy = policyList.value.find(
    (policy) => policy.resourceUid == sensitiveColumn.database.uid
  );
  if (!policy) return;
  const sensitiveData = policy.sensitiveDataPolicy?.sensitiveData;
  if (!sensitiveData) return;

  const index = sensitiveData.findIndex(
    (sensitiveData) =>
      sensitiveData.table === table && sensitiveData.column === column
  );
  if (index >= 0) {
    // mutate the list and the item directly
    // so we don't need to re-fetch the whole list.
    sensitiveData.splice(index, 1);

    usePolicyV1Store().updatePolicy(["payload"], {
      name: policy.name,
      type: PolicyType.SENSITIVE_DATA,
      resourceType: PolicyResourceType.DATABASE,
      sensitiveDataPolicy: {
        sensitiveData,
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
  let url = `/db/${databaseV1Slug(item.database)}?table=${item.table}`;
  if (item.schema != "") {
    url += `&schema=${item.schema}`;
  }
  if (e.ctrlKey || e.metaKey) {
    window.open(url, "_blank");
  } else {
    router.push(url);
  }
};
</script>
