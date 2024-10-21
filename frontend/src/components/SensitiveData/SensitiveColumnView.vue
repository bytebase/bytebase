<template>
  <div class="w-full space-y-4">
    <FeatureAttentionForInstanceLicense
      v-if="
        hasSensitiveDataFeature &&
        findInstanceWithoutLicense(filteredColumnList) !== undefined
      "
      feature="bb.feature.sensitive-data"
    />
    <EnvironmentTabFilter
      :environment="state.selectedEnvironmentName"
      :include-all="true"
      @update:environment="state.selectedEnvironmentName = $event"
    />
    <div
      class="flex flex-col sm:flex-row gap-y-4 justify-between items-end sm:items-center"
    >
      <NInputGroup>
        <ProjectSelect
          :project-name="state.selectedProjectName"
          :include-default-project="true"
          :include-all="true"
          :disabled="false"
          @update:project-name="
            state.selectedProjectName = $event ?? UNKNOWN_PROJECT_NAME
          "
        />
        <InstanceSelect
          class="!w-48"
          :instance-name="state.selectedInstanceName"
          :include-all="true"
          :environment-name="environment?.name"
          @update:instance-name="onInstanceSelect($event)"
        />
        <DatabaseSelect
          :include-all="true"
          :project-name="state.selectedProjectName"
          :instance-name="state.selectedInstanceName"
          :database-name="state.selectedDatabaseName"
          @update:database-name="onDatabaseSelect($event)"
        />
      </NInputGroup>
      <NTooltip :disabled="selectedProjects.size <= 1">
        <template #trigger>
          <NButton
            type="primary"
            :disabled="
              state.pendingGrantAccessColumn.length === 0 ||
              !hasPermission ||
              !hasSensitiveDataFeature ||
              selectedProjects.size !== 1
            "
            @click="onGrantAccessButtonClick"
          >
            {{ $t("settings.sensitive-data.grant-access") }}
          </NButton>
        </template>
        <span class="textinfolabel">
          {{ $t("database.select-databases-from-same-project") }}
        </span>
      </NTooltip>
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
    :project-name="[...selectedProjects][0]"
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
import { NButton, NInputGroup, NTooltip } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import {
  ProjectSelect,
  InstanceSelect,
  DatabaseSelect,
} from "@/components/v2/Select";
import {
  usePolicyListByResourceTypeAndPolicyType,
  featureToRef,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  pushNotification,
  usePolicyV1Store,
  useSubscriptionV1Store,
  useInstanceResourceByName,
} from "@/store";
import { getPolicyResourceNameAndType } from "@/store/modules/v1/common";
import {
  UNKNOWN_ENVIRONMENT_NAME,
  UNKNOWN_INSTANCE_NAME,
  UNKNOWN_PROJECT_NAME,
  isValidProjectName,
  isValidEnvironmentName,
  isValidInstanceName,
  UNKNOWN_DATABASE_NAME,
  isValidDatabaseName,
} from "@/types";
import { MaskingLevel } from "@/types/proto/v1/common";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto/v1/instance_service";
import {
  PolicyType,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import { autoDatabaseRoute, hasWorkspacePermissionV2 } from "@/utils";
import { FeatureAttentionForInstanceLicense } from "../FeatureGuard";
import FeatureModal from "../FeatureGuard/FeatureModal.vue";
import NoDataPlaceholder from "../misc/NoDataPlaceholder.vue";
import { EnvironmentTabFilter } from "../v2";
import GrantAccessDrawer from "./GrantAccessDrawer.vue";
import SensitiveColumnDrawer from "./SensitiveColumnDrawer.vue";
import SensitiveColumnTable from "./components/SensitiveColumnTable.vue";
import type { SensitiveColumn } from "./types";
import { getMaskDataIdentifier, isCurrentColumnException } from "./utils";

interface LocalState {
  selectedEnvironmentName: string;
  selectedProjectName: string;
  selectedInstanceName: string;
  selectedDatabaseName: string;
  showFeatureModal: boolean;
  isLoading: boolean;
  sensitiveColumnList: SensitiveColumn[];
  pendingGrantAccessColumn: SensitiveColumn[];
  showGrantAccessDrawer: boolean;
  showSensitiveColumnDrawer: boolean;
}

const { t } = useI18n();
const router = useRouter();
const databaseStore = useDatabaseV1Store();
const policyStore = usePolicyV1Store();
const environmentStore = useEnvironmentV1Store();
const subscriptionStore = useSubscriptionV1Store();
const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");
const policyList = usePolicyListByResourceTypeAndPolicyType({
  resourceType: PolicyResourceType.DATABASE,
  policyType: PolicyType.MASKING,
  showDeleted: false,
});

const state = reactive<LocalState>({
  showFeatureModal: false,
  isLoading: false,
  sensitiveColumnList: [],
  selectedEnvironmentName: UNKNOWN_ENVIRONMENT_NAME,
  selectedProjectName: UNKNOWN_PROJECT_NAME,
  selectedInstanceName: UNKNOWN_INSTANCE_NAME,
  selectedDatabaseName: UNKNOWN_DATABASE_NAME,
  pendingGrantAccessColumn: [],
  showGrantAccessDrawer: false,
  showSensitiveColumnDrawer: false,
});

const hasPermission = computed(() => {
  return hasWorkspacePermissionV2("bb.policies.update");
});

const selectedProjects = computed(() => {
  return new Set(
    state.pendingGrantAccessColumn.map((data) => data.database.project)
  );
});

const updateList = async () => {
  state.isLoading = true;
  const distinctDatabaseNameList = uniq(
    policyList.value.map((policy) => {
      const [databaseName, _] = getPolicyResourceNameAndType(policy.name);
      return databaseName;
    })
  );
  // Fetch or get all needed databases
  await Promise.all(
    distinctDatabaseNameList.map((name) =>
      databaseStore.getOrFetchDatabaseByName(name)
    )
  );

  const sensitiveColumnList: SensitiveColumn[] = [];
  for (let i = 0; i < policyList.value.length; i++) {
    const policy = policyList.value[i];
    if (!policy.maskingPolicy) {
      continue;
    }

    const [databaseName, _] = getPolicyResourceNameAndType(policy.name);
    if (!databaseName) {
      continue;
    }
    const database = await databaseStore.getOrFetchDatabaseByName(databaseName);

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

const isMissingLicenseForInstance = (
  instance: Instance | InstanceResource
): boolean => {
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
      const query: Record<string, string> = {
        table: item.maskData.table,
      };
      if (item.maskData.schema != "") {
        query.schema = item.maskData.schema;
      }
      router.push({
        ...autoDatabaseRoute(router, item.database),
        query,
      });
      break;
    }
    case "DELETE":
      await onColumnRemove(item);
      break;
    case "EDIT":
      state.pendingGrantAccessColumn = [item];
      if (isMissingLicenseForInstance(item.database.instanceResource)) {
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
      isValidEnvironmentName(state.selectedEnvironmentName) &&
      column.database.effectiveEnvironment !== state.selectedEnvironmentName
    ) {
      return false;
    }
    if (
      isValidProjectName(state.selectedProjectName) &&
      column.database.project !== state.selectedProjectName
    ) {
      return false;
    }
    if (
      isValidInstanceName(state.selectedInstanceName) &&
      state.selectedInstanceName !== column.database.instance
    ) {
      return false;
    }
    if (
      isValidDatabaseName(state.selectedDatabaseName) &&
      column.database.name !== state.selectedDatabaseName
    ) {
      return false;
    }
    return true;
  });
});

const findInstanceWithoutLicense = (columnList: SensitiveColumn[]) => {
  for (const column of columnList) {
    const instance = useInstanceResourceByName(column.database.instance);
    const missingLicense = isMissingLicenseForInstance(instance);
    if (missingLicense) {
      return instance;
    }
  }
  return;
};

const environment = computed(() => {
  if (!isValidEnvironmentName(state.selectedEnvironmentName)) {
    return;
  }
  return environmentStore.getEnvironmentByName(state.selectedEnvironmentName);
});

const onInstanceSelect = (name: string | undefined) => {
  state.selectedInstanceName = name ?? UNKNOWN_INSTANCE_NAME;
  state.selectedDatabaseName = UNKNOWN_DATABASE_NAME;
};

const onDatabaseSelect = (name: string | undefined) => {
  state.selectedDatabaseName = name ?? UNKNOWN_DATABASE_NAME;
  if (isValidDatabaseName(name)) {
    const database = databaseStore.getDatabaseByName(name);
    state.selectedInstanceName = database.instance;
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
