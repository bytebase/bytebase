<template>
  <div class="w-full mt-4 space-y-4">
    <EnvironmentTabFilter
      :environment="state.environment"
      :include-all="true"
      @update:environment="state.environment = $event"
    />

    <div class="textinfolabel">
      {{ $t("settings.sensitive-data.description") }}
    </div>

    <BBGrid
      v-if="hasSensitiveDataFeature"
      :column-list="COLUMN_LIST"
      :data-source="filteredColumnList"
      class="border"
      @click-row="clickRow"
    >
      <template #item="{ item }: { item: SensitiveColumn }">
        <div class="bb-grid-cell">
          {{ getMaskingLevelText(item.maskData.maskingLevel) }}
        </div>
        <div class="bb-grid-cell">
          {{ item.maskData.column }}
        </div>
        <div class="bb-grid-cell">
          {{
            item.maskData.schema
              ? `${item.maskData.schema}.${item.maskData.table}`
              : item.maskData.table
          }}
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
import { ComposedDatabase, UNKNOWN_ENVIRONMENT_NAME } from "@/types";
import { MaskingLevel, maskingLevelToJSON } from "@/types/proto/v1/common";
import {
  PolicyType,
  MaskData,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import { databaseV1Slug, hasWorkspacePermissionV1 } from "@/utils";

type SensitiveColumn = {
  database: ComposedDatabase;
  maskData: MaskData;
};

interface LocalState {
  environment: string;
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
  environment: UNKNOWN_ENVIRONMENT_NAME,
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
  policyType: PolicyType.MASKING,
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
    if (!policy.maskingPolicy) {
      continue;
    }

    const databaseId = policy.resourceUid;
    const database = await databaseStore.getOrFetchDatabaseByUID(databaseId);

    for (const maskData of policy.maskingPolicy.maskData) {
      sensitiveColumnList.push({ database, maskData });
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

  const { table, column } = sensitiveColumn.maskData;
  const policy = policyList.value.find(
    (policy) => policy.resourceUid == sensitiveColumn.database.uid
  );
  if (!policy) return;
  const maskData = policy.maskingPolicy?.maskData;
  if (!maskData) return;

  const index = maskData.findIndex(
    (sensitiveData) =>
      sensitiveData.table === table && sensitiveData.column === column
  );
  if (index >= 0) {
    // mutate the list and the item directly
    // so we don't need to re-fetch the whole list.
    maskData.splice(index, 1);

    usePolicyV1Store().updatePolicy(["payload"], {
      name: policy.name,
      type: PolicyType.MASKING,
      resourceType: PolicyResourceType.DATABASE,
      maskingPolicy: {
        maskData,
      },
    });
  }
  updateList();
};

const COLUMN_LIST = computed((): BBGridColumn[] => [
  {
    title: t("settings.sensitive-data.masking-level.self"),
    width: "minmax(auto, 1fr)",
  },
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
  let url = `/db/${databaseV1Slug(item.database)}?table=${item.maskData.table}`;
  if (item.maskData.schema != "") {
    url += `&schema=${item.maskData.schema}`;
  }
  if (e.ctrlKey || e.metaKey) {
    window.open(url, "_blank");
  } else {
    router.push(url);
  }
};

const filteredColumnList = computed(() => {
  return state.sensitiveColumnList.filter((column) => {
    if (state.environment === UNKNOWN_ENVIRONMENT_NAME) {
      return true;
    }
    return (
      column.database.effectiveEnvironmentEntity.name === state.environment
    );
  });
});

const getMaskingLevelText = (maskingLevel: MaskingLevel) => {
  let level = maskingLevelToJSON(maskingLevel);
  if (maskingLevel === MaskingLevel.MASKING_LEVEL_UNSPECIFIED) {
    level = maskingLevelToJSON(MaskingLevel.FULL);
  }
  return t(`settings.sensitive-data.masking-level.${level.toLowerCase()}`);
};
</script>
