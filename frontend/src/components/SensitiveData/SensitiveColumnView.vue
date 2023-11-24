<template>
  <div class="w-full space-y-4">
    <FeatureAttentionForInstanceLicense
      v-if="
        hasSensitiveDataFeature &&
        findInstanceWithoutLicense(filteredColumnList) !== undefined
      "
      feature="bb.feature.sensitive-data"
    />
    <div>
      <EnvironmentTabFilter
        :environment="state.selectedEnvironmentName"
        :include-all="true"
        @update:environment="state.selectedEnvironmentName = $event"
      />
    </div>
    <div class="flex justify-between items-center">
      <NInputGroup>
        <ProjectSelect
          :project="state.selectedProjectUid"
          :include-default-project="true"
          :include-all="true"
          :disabled="false"
          @update:project="
            state.selectedProjectUid = $event ?? String(UNKNOWN_ID)
          "
        />
        <InstanceSelect
          class="!w-48"
          :instance="state.selectedInstanceUid"
          :include-all="true"
          :environment="environment?.uid"
          @update:instance="onInstanceSelect($event)"
        />
        <DatabaseSelect
          :include-all="true"
          :project="state.selectedProjectUid"
          :instance="state.selectedInstanceUid"
          :database="state.selectedDatabaseUid"
          @update:database="onDatabaseSelect($event)"
        />
      </NInputGroup>
      <NButton
        type="primary"
        :disabled="
          state.pendingGrantAccessColumn.length === 0 ||
          !hasPermission ||
          !hasSensitiveDataFeature
        "
        @click="onGrantAccessButtonClick"
      >
        {{ $t("settings.sensitive-data.grant-access") }}
      </NButton>
    </div>

    <div class="textinfolabel">
      {{ $t("settings.sensitive-data.description") }}
      <a
        href="https://www.bytebase.com/docs/security/mask-data?source=console"
        class="normal-link inline-flex flex-row items-center"
        target="_blank"
      >
        {{ $t("common.learn-more") }}
        <heroicons-outline:external-link class="w-4 h-4" />
      </a>
    </div>

    <SensitiveColumnTable
      v-if="hasSensitiveDataFeature"
      :row-clickable="true"
      :row-selectable="true"
      :show-operation="hasPermission && hasSensitiveDataFeature"
      :column-list="filteredColumnList"
      :checked-column-index-list="checkedColumnIndexList"
      @click="onRowClick"
      @checked:update="updateCheckedColumnList($event)"
    />

    <NoDataPlaceholder v-else />
  </div>

  <FeatureModal
    feature="bb.feature.sensitive-data"
    :open="state.showFeatureModal"
    :instance="findInstanceWithoutLicense(state.pendingGrantAccessColumn)"
    @cancel="state.showFeatureModal = false"
  />

  <GrantAccessDrawer
    v-if="
      state.showGrantAccessDrawer && state.pendingGrantAccessColumn.length > 0
    "
    :column-list="state.pendingGrantAccessColumn"
    @dismiss="
      () => {
        state.showGrantAccessDrawer = false;
        state.pendingGrantAccessColumn = [];
      }
    "
  />

  <SensitiveColumnDrawer
    v-if="filteredColumnList.length > 0"
    :show="
      state.showSensitiveColumnDrawer &&
      state.pendingGrantAccessColumn.length === 1
    "
    :column="
      state.pendingGrantAccessColumn.length === 1
        ? state.pendingGrantAccessColumn[0]
        : filteredColumnList[0]
    "
    @dismiss="
      () => {
        state.showSensitiveColumnDrawer = false;
        state.pendingGrantAccessColumn = [];
      }
    "
  />
</template>

<script lang="ts" setup>
import { uniq } from "lodash-es";
import { NInputGroup } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import NoData from "@/components/misc/NoData.vue";
import {
  ProjectSelect,
  InstanceSelect,
  DatabaseSelect,
} from "@/components/v2/Select";
import {
  usePolicyListByResourceTypeAndPolicyType,
  featureToRef,
  useDatabaseV1Store,
  useCurrentUserV1,
  useEnvironmentV1Store,
  useInstanceV1Store,
  pushNotification,
  usePolicyV1Store,
  useSubscriptionV1Store,
} from "@/store";
import {
  UNKNOWN_ID,
  UNKNOWN_ENVIRONMENT_NAME,
  ComposedInstance,
} from "@/types";
import { MaskingLevel } from "@/types/proto/v1/common";
import {
  PolicyType,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import { databaseV1Slug, hasWorkspacePermissionV1 } from "@/utils";
import FeatureModal from "../FeatureGuard/FeatureModal.vue";
import GrantAccessDrawer from "./GrantAccessDrawer.vue";
import SensitiveColumnDrawer from "./SensitiveColumnDrawer.vue";
import SensitiveColumnTable from "./components/SensitiveColumnTable.vue";
import { SensitiveColumn } from "./types";
import { getMaskDataIdentifier, isCurrentColumnException } from "./utils";

interface LocalState {
  selectedEnvironmentName: string;
  selectedProjectUid: string;
  selectedInstanceUid: string;
  selectedDatabaseUid: string;
  showFeatureModal: boolean;
  isLoading: boolean;
  sensitiveColumnList: SensitiveColumn[];
  pendingGrantAccessColumn: SensitiveColumn[];
  showGrantAccessDrawer: boolean;
  showSensitiveColumnDrawer: boolean;
}

const { t } = useI18n();
const router = useRouter();
const state = reactive<LocalState>({
  showFeatureModal: false,
  isLoading: false,
  sensitiveColumnList: [],
  selectedEnvironmentName: UNKNOWN_ENVIRONMENT_NAME,
  selectedProjectUid: String(UNKNOWN_ID),
  selectedInstanceUid: String(UNKNOWN_ID),
  selectedDatabaseUid: String(UNKNOWN_ID),
  pendingGrantAccessColumn: [],
  showGrantAccessDrawer: false,
  showSensitiveColumnDrawer: false,
});
const databaseStore = useDatabaseV1Store();
const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");
const policyStore = usePolicyV1Store();
const environmentStore = useEnvironmentV1Store();
const subscriptionStore = useSubscriptionV1Store();
const instanceV1Store = useInstanceV1Store();

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

watch(policyList, updateList, { immediate: true, deep: true });

const removeSensitiveColumn = async (sensitiveColumn: SensitiveColumn) => {
  const policy = await policyStore.getOrFetchPolicyByParentAndType({
    parentPath: sensitiveColumn.database.name,
    policyType: PolicyType.MASKING,
  });
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
};

const removeMaskingExceptions = async (sensitiveColumn: SensitiveColumn) => {
  const policy = await policyStore.getOrFetchPolicyByParentAndType({
    parentPath: sensitiveColumn.database.project,
    policyType: PolicyType.MASKING_EXCEPTION,
  });
  if (!policy) {
    return;
  }

  const exceptions = (
    policy.maskingExceptionPolicy?.maskingExceptions ?? []
  ).filter(
    (exception) => !isCurrentColumnException(exception, sensitiveColumn)
  );

  policy.maskingExceptionPolicy = {
    ...(policy.maskingExceptionPolicy ?? {}),
    maskingExceptions: exceptions,
  };
  await policyStore.updatePolicy(["payload"], policy);
};

const onColumnRemove = async (column: SensitiveColumn) => {
  await removeSensitiveColumn(column);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const isMissingLicenseForInstance = (instance: ComposedInstance): boolean => {
  return subscriptionStore.instanceMissingLicense(
    "bb.feature.sensitive-data",
    instance
  );
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
      await onColumnRemove(item);
      break;
    case "EDIT":
      state.pendingGrantAccessColumn = [item];
      if (isMissingLicenseForInstance(item.database.instanceEntity)) {
        state.showFeatureModal = true;
        return;
      }
      state.showSensitiveColumnDrawer = true;
      break;
  }
};

const filteredColumnList = computed(() => {
  return state.sensitiveColumnList.filter((column) => {
    if (
      column.maskData.maskingLevel === MaskingLevel.NONE ||
      column.maskData.maskingLevel === MaskingLevel.MASKING_LEVEL_UNSPECIFIED
    ) {
      return false;
    }
    if (
      state.selectedEnvironmentName !== UNKNOWN_ENVIRONMENT_NAME &&
      column.database.effectiveEnvironmentEntity.name !==
        state.selectedEnvironmentName
    ) {
      return false;
    }
    if (
      state.selectedProjectUid !== String(UNKNOWN_ID) &&
      column.database.projectEntity.uid !== state.selectedProjectUid
    ) {
      return false;
    }
    if (
      state.selectedInstanceUid !== String(UNKNOWN_ID) &&
      column.database.instanceEntity.uid !== state.selectedInstanceUid
    ) {
      return false;
    }
    if (
      state.selectedDatabaseUid !== String(UNKNOWN_ID) &&
      column.database.uid !== state.selectedDatabaseUid
    ) {
      return false;
    }
    return true;
  });
});

const findInstanceWithoutLicense = (columnList: SensitiveColumn[]) => {
  for (const column of columnList) {
    const instance = instanceV1Store.getInstanceByName(
      column.database.instance
    );
    const missingLicense = isMissingLicenseForInstance(instance);
    if (missingLicense) {
      return instance;
    }
  }
  return;
};

const environment = computed(() => {
  if (state.selectedEnvironmentName === UNKNOWN_ENVIRONMENT_NAME) {
    return;
  }
  return environmentStore.getEnvironmentByName(state.selectedEnvironmentName);
});

const onInstanceSelect = (instanceUid: string | undefined) => {
  state.selectedInstanceUid = instanceUid ?? String(UNKNOWN_ID);
  state.selectedDatabaseUid = String(UNKNOWN_ID);
};

const onDatabaseSelect = (databaseUid: string | undefined) => {
  state.selectedDatabaseUid = databaseUid ?? String(UNKNOWN_ID);
  if (databaseUid) {
    const database = databaseStore.getDatabaseByUID(databaseUid);
    state.selectedInstanceUid = database.instanceEntity.uid;
  }
};

const onGrantAccessButtonClick = () => {
  const instance = findInstanceWithoutLicense(state.pendingGrantAccessColumn);
  if (instance) {
    state.showFeatureModal = true;
    return;
  }
  state.showGrantAccessDrawer = true;
};

const checkedColumnIndexList = computed(() => {
  const resp = [];
  for (const column of state.pendingGrantAccessColumn) {
    const index = filteredColumnList.value.findIndex((col) => {
      return (
        col.database.name === column.database.name &&
        col.maskData.table === column.maskData.table &&
        col.maskData.schema === column.maskData.schema &&
        col.maskData.column === column.maskData.column
      );
    });
    if (index >= 0) {
      resp.push(index);
    }
  }
  return resp;
});

const updateCheckedColumnList = (indexes: number[]) => {
  state.pendingGrantAccessColumn = [];
  for (const index of indexes) {
    const col = filteredColumnList.value[index];
    if (col) {
      state.pendingGrantAccessColumn.push(col);
    }
  }
};
</script>
