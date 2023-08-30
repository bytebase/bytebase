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

    <SensitiveColumnTable
      v-if="hasSensitiveDataFeature"
      :row-clickable="true"
      :row-selectable="true"
      :show-operation="true"
      :column-list="filteredColumnList"
      @click="onRowClick"
    />

    <template v-else>
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

  <Drawer
    :show="
      state.showGrantAccessDrawer && state.pendingGrantAccessColumn.length > 0
    "
    @close="state.showGrantAccessDrawer = false"
  >
    <GrantAccessDrawer :column-list="state.pendingGrantAccessColumn" />
  </Drawer>
</template>

<script lang="ts" setup>
import { uniq } from "lodash-es";
import { computed, reactive, watch } from "vue";
import { useRouter } from "vue-router";
import { Drawer } from "@/components/v2";
import { featureToRef, useDatabaseV1Store } from "@/store";
import {
  usePolicyListByResourceTypeAndPolicyType,
  usePolicyV1Store,
} from "@/store/modules/v1/policy";
import { UNKNOWN_ENVIRONMENT_NAME } from "@/types";
import {
  PolicyType,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import { databaseV1Slug } from "@/utils";
import { SensitiveColumn } from "./types";

interface LocalState {
  environment: string;
  showFeatureModal: boolean;
  isLoading: boolean;
  sensitiveColumnList: SensitiveColumn[];
  pendingGrantAccessColumn: SensitiveColumn[];
  showGrantAccessDrawer: boolean;
}

const router = useRouter();
const state = reactive<LocalState>({
  showFeatureModal: false,
  isLoading: false,
  sensitiveColumnList: [],
  environment: UNKNOWN_ENVIRONMENT_NAME,
  pendingGrantAccessColumn: [],
  showGrantAccessDrawer: false,
});
const databaseStore = useDatabaseV1Store();
const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");

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

const onRowClick = (
  item: SensitiveColumn,
  action: "VIEW" | "DELETE" | "EDIT"
) => {
  switch (action) {
    case "VIEW":
      let url = `/db/${databaseV1Slug(item.database)}?table=${
        item.maskData.table
      }`;
      if (item.maskData.schema != "") {
        url += `&schema=${item.maskData.schema}`;
      }
      router.push(url);
      break;
    case "DELETE":
      removeSensitiveColumn(item);
      break;
    case "EDIT":
      state.pendingGrantAccessColumn = [item];
      state.showGrantAccessDrawer = true;
      break;
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
</script>
