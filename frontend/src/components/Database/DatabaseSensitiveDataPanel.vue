<template>
  <div class="w-full space-y-4">
    <FeatureAttentionForInstanceLicense
      v-if="hasSensitiveDataFeature && isMissingLicenseForInstance"
      feature="bb.feature.sensitive-data"
    />
    <div
      class="flex flex-col space-x-2 lg:flex-row gap-y-4 justify-between items-end lg:items-center"
    >
      <SearchBox v-model:value="state.searchText" style="max-width: 100%" />
      <div class="flex items-center space-x-2">
        <NButton
          type="primary"
          :disabled="
            state.pendingGrantAccessColumn.length === 0 ||
            !hasPolicyPermission ||
            !hasGetCatalogPermission
          "
          @click="onGrantAccessButtonClick"
        >
          <template #icon>
            <ShieldCheckIcon v-if="hasSensitiveDataFeature" class="w-4" />
            <FeatureBadge
              v-else
              feature="bb.feature.sensitive-data"
              custom-class="text-white"
            />
          </template>
          {{ $t("settings.sensitive-data.grant-access") }}
        </NButton>
      </div>
    </div>

    <SensitiveColumnTable
      :database="database"
      :row-clickable="false"
      :row-selectable="true"
      :show-operation="hasUpdateCatalogPermission && hasSensitiveDataFeature"
      :column-list="filteredColumnList"
      :checked-column-index-list="checkedColumnIndexList"
      @delete="onColumnRemove"
      @checked:update="updateCheckedColumnList($event)"
    />
  </div>

  <FeatureModal
    feature="bb.feature.sensitive-data"
    :open="state.showFeatureModal"
    :instance="database.instanceResource"
    @cancel="state.showFeatureModal = false"
  />

  <GrantAccessDrawer
    v-if="
      state.showGrantAccessDrawer && state.pendingGrantAccessColumn.length > 0
    "
    :column-list="
      state.pendingGrantAccessColumn.map((maskData) => ({
        database,
        maskData,
      }))
    "
    :project-name="database.project"
    @dismiss="
      () => {
        state.showGrantAccessDrawer = false;
        state.pendingGrantAccessColumn = [];
      }
    "
  />
</template>

<script lang="tsx" setup>
import { ShieldCheckIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { updateColumnCatalog } from "@/components/ColumnDataTable/utils";
import {
  FeatureModal,
  FeatureBadge,
  FeatureAttentionForInstanceLicense,
} from "@/components/FeatureGuard";
import GrantAccessDrawer from "@/components/SensitiveData/GrantAccessDrawer.vue";
import SensitiveColumnTable from "@/components/SensitiveData/components/SensitiveColumnTable.vue";
import type { MaskData } from "@/components/SensitiveData/types";
import { isCurrentColumnException } from "@/components/SensitiveData/utils";
import { SearchBox } from "@/components/v2";
import {
  featureToRef,
  usePolicyV1Store,
  useSubscriptionV1Store,
  useDatabaseCatalog,
} from "@/store";
import { type ComposedDatabase } from "@/types";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import { hasProjectPermissionV2 } from "@/utils";

const props = defineProps<{
  database: ComposedDatabase;
}>();

interface LocalState {
  searchText: string;
  showFeatureModal: boolean;
  isLoading: boolean;
  sensitiveColumnList: MaskData[];
  pendingGrantAccessColumn: MaskData[];
  showGrantAccessDrawer: boolean;
}

const state = reactive<LocalState>({
  searchText: "",
  showFeatureModal: false,
  isLoading: false,
  sensitiveColumnList: [],
  pendingGrantAccessColumn: [],
  showGrantAccessDrawer: false,
});

const hasUpdateCatalogPermission = computed(() => {
  return hasProjectPermissionV2(
    props.database.projectEntity,
    "bb.databaseCatalogs.update"
  );
});

const hasGetCatalogPermission = computed(() => {
  return hasProjectPermissionV2(
    props.database.projectEntity,
    "bb.databaseCatalogs.get"
  );
});

const hasPolicyPermission = computed(() => {
  return hasProjectPermissionV2(
    props.database.projectEntity,
    "bb.policies.update"
  );
});

const policyStore = usePolicyV1Store();
const subscriptionStore = useSubscriptionV1Store();

const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");

const isMissingLicenseForInstance = computed(() =>
  subscriptionStore.instanceMissingLicense(
    "bb.feature.sensitive-data",
    props.database.instanceResource
  )
);

const databaseCatalog = useDatabaseCatalog(props.database.name, false);

const updateList = async () => {
  state.isLoading = true;
  const sensitiveColumnList: MaskData[] = [];

  for (const schema of databaseCatalog.value.schemas) {
    for (const table of schema.tables) {
      for (const column of table.columns?.columns ?? []) {
        if (!column.semanticType && !column.classification) {
          continue;
        }
        sensitiveColumnList.push({
          schema: schema.name,
          table: table.name,
          column: column.name,
          semanticTypeId: column.semanticType,
          classificationId: column.classification,
        });
      }
    }
  }

  state.sensitiveColumnList = sensitiveColumnList;
  state.isLoading = false;
};

watch(databaseCatalog, updateList, { immediate: true, deep: true });

const filteredColumnList = computed(() => {
  let list = state.sensitiveColumnList;
  const searchText = state.searchText.trim().toLowerCase();
  if (searchText) {
    list = list.filter(
      (item) =>
        item.column.includes(searchText) ||
        item.table.includes(searchText) ||
        item.schema.includes(searchText)
    );
  }
  return list;
});

const removeSensitiveColumn = async (sensitiveColumn: MaskData) => {
  await updateColumnCatalog({
    database: props.database.name,
    schema: sensitiveColumn.schema,
    table: sensitiveColumn.table,
    column: sensitiveColumn.column,
    columnCatalog: {
      classification: "",
      semanticType: "",
    },
    notification: "common.removed",
  });
  await removeMaskingExceptions(sensitiveColumn);
};

const removeMaskingExceptions = async (sensitiveColumn: MaskData) => {
  const policy = await policyStore.getOrFetchPolicyByParentAndType({
    parentPath: props.database.project,
    policyType: PolicyType.MASKING_EXCEPTION,
  });
  if (!policy) {
    return;
  }

  const exceptions = (
    policy.maskingExceptionPolicy?.maskingExceptions ?? []
  ).filter(
    (exception) =>
      !isCurrentColumnException(exception, {
        database: props.database,
        maskData: sensitiveColumn,
      })
  );

  policy.maskingExceptionPolicy = {
    ...(policy.maskingExceptionPolicy ?? {}),
    maskingExceptions: exceptions,
  };
  await policyStore.upsertPolicy({
    parentPath: props.database.project,
    policy,
  });
};

const onColumnRemove = async (column: MaskData) => {
  await removeSensitiveColumn(column);
};

const onGrantAccessButtonClick = () => {
  if (!hasSensitiveDataFeature.value) {
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
        col.table === column.table &&
        col.schema === column.schema &&
        col.column === column.column
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
