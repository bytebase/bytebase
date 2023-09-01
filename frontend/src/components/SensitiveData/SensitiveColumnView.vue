<template>
  <div class="w-full mt-4 space-y-4">
    <div class="flex justify-between">
      <EnvironmentTabFilter
        :environment="state.environment"
        :include-all="true"
        @update:environment="state.environment = $event"
      />
      <NButton
        type="primary"
        :disabled="
          state.pendingGrantAccessColumnIndex.length === 0 || !hasPermission
        "
        @click="state.showGrantAccessDrawer = true"
      >
        {{ $t("settings.sensitive-data.grant-access") }}
      </NButton>
    </div>

    <div class="textinfolabel">
      {{ $t("settings.sensitive-data.description") }}
    </div>

    <SensitiveColumnTable
      v-if="hasSensitiveDataFeature"
      :row-clickable="true"
      :row-selectable="true"
      :show-operation="hasPermission && hasSensitiveDataFeature"
      :column-list="filteredColumnList"
      :checked-column-index-list="state.pendingGrantAccessColumnIndex"
      @click="onRowClick"
      @checked:update="state.pendingGrantAccessColumnIndex = $event"
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

  <GrantAccessDrawer
    :show="
      state.showGrantAccessDrawer &&
      state.pendingGrantAccessColumnIndex.length > 0
    "
    :column-list="
      state.pendingGrantAccessColumnIndex.map((i) => filteredColumnList[i])
    "
    @dismiss="
      () => {
        state.showGrantAccessDrawer = false;
        state.pendingGrantAccessColumnIndex = [];
      }
    "
  />

  <SensitiveColumnDrawer
    v-if="filteredColumnList.length > 0"
    :show="
      state.showSensitiveColumnDrawer &&
      state.pendingGrantAccessColumnIndex.length === 1
    "
    :column="
      filteredColumnList[state.pendingGrantAccessColumnIndex[0]] ??
      filteredColumnList[0]
    "
    @dismiss="
      () => {
        state.showSensitiveColumnDrawer = false;
        state.pendingGrantAccessColumnIndex = [];
      }
    "
  />
</template>

<script lang="ts" setup>
import { uniq } from "lodash-es";
import { computed, reactive, watch } from "vue";
import { useRouter } from "vue-router";
import {
  usePolicyListByResourceTypeAndPolicyType,
  usePolicyV1Store,
  featureToRef,
  useDatabaseV1Store,
  useCurrentUserV1,
} from "@/store";
import { UNKNOWN_ENVIRONMENT_NAME } from "@/types";
import {
  PolicyType,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import { databaseV1Slug, hasWorkspacePermissionV1 } from "@/utils";
import { SensitiveColumn } from "./types";
import { getMaskDataIdentifier, isCurrentColumnException } from "./utils";

interface LocalState {
  environment: string;
  showFeatureModal: boolean;
  isLoading: boolean;
  sensitiveColumnList: SensitiveColumn[];
  pendingGrantAccessColumnIndex: number[];
  showGrantAccessDrawer: boolean;
  showSensitiveColumnDrawer: boolean;
}

const router = useRouter();
const state = reactive<LocalState>({
  showFeatureModal: false,
  isLoading: false,
  sensitiveColumnList: [],
  environment: UNKNOWN_ENVIRONMENT_NAME,
  pendingGrantAccessColumnIndex: [],
  showGrantAccessDrawer: false,
  showSensitiveColumnDrawer: false,
});
const databaseStore = useDatabaseV1Store();
const policyStore = usePolicyV1Store();
const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");

const policyList = usePolicyListByResourceTypeAndPolicyType({
  resourceType: PolicyResourceType.DATABASE,
  policyType: PolicyType.MASKING,
  showDeleted: false,
});

const currentUserV1 = useCurrentUserV1();
const hasPermission = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sensitive-data",
    currentUserV1.value.userRole
  );
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

const removeSensitiveColumn = async (sensitiveColumn: SensitiveColumn) => {
  const policy = policyList.value.find(
    (policy) => policy.resourceUid == sensitiveColumn.database.uid
  );
  if (!policy) return;
  const maskData = policy.maskingPolicy?.maskData;
  if (!maskData) return;

  const index = maskData.findIndex(
    (sensitiveData) =>
      getMaskDataIdentifier(sensitiveData) ===
      getMaskDataIdentifier(sensitiveColumn.maskData)
  );
  if (index >= 0) {
    // mutate the list and the item directly
    // so we don't need to re-fetch the whole list.
    maskData.splice(index, 1);

    await policyStore.updatePolicy(["payload"], {
      name: policy.name,
      type: PolicyType.MASKING,
      resourceType: PolicyResourceType.DATABASE,
      maskingPolicy: {
        maskData,
      },
    });
    await removeMaskingExceptions(sensitiveColumn);
  }
  updateList();
};

const removeMaskingExceptions = async (sensitiveColumn: SensitiveColumn) => {
  const policy = await policyStore.getOrFetchPolicyByParentAndType({
    parentPath: sensitiveColumn.database.name,
    policyType: PolicyType.MASKING_EXCEPTION,
  });
  if (!policy) {
    return;
  }

  const exceptions = (
    policy.maskingExceptionPolicy?.maskingExceptions ?? []
  ).filter(
    (exception) =>
      !isCurrentColumnException(exception, sensitiveColumn.maskData)
  );

  policy.maskingExceptionPolicy = {
    ...(policy.maskingExceptionPolicy ?? {}),
    maskingExceptions: exceptions,
  };
  await policyStore.updatePolicy(["payload"], policy);
};

const onRowClick = async (
  item: SensitiveColumn,
  row: number,
  action: "VIEW" | "DELETE" | "EDIT"
) => {
  switch (action) {
    case "VIEW": {
      let url = `/db/${databaseV1Slug(item.database)}?table=${
        item.maskData.table
      }`;
      if (item.maskData.schema != "") {
        url += `&schema=${item.maskData.schema}`;
      }
      router.push(url);
      break;
    }
    case "DELETE":
      await removeSensitiveColumn(item);
      break;
    case "EDIT":
      state.pendingGrantAccessColumnIndex = [row];
      state.showSensitiveColumnDrawer = true;
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
