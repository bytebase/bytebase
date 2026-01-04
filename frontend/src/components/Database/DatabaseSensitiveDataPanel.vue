<template>
  <div class="w-full flex flex-col gap-y-4">
    <FeatureAttention
      :feature="PlanFeature.FEATURE_DATA_MASKING"
      :instance="database.instanceResource"
    />
    <div
      class="flex flex-col gap-x-2 lg:flex-row gap-y-4 justify-between items-end lg:items-center"
    >
      <SearchBox v-model:value="state.searchText" style="max-width: 100%" />
      <NButton
        v-if="!isMaskingForNoSQL"
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
            :feature="PlanFeature.FEATURE_DATA_MASKING"
            class="text-white"
            :instance="database.instanceResource"
          />
        </template>
        {{ $t("settings.sensitive-data.grant-access") }}
      </NButton>
    </div>

    <SensitiveColumnTable
      :database="database"
      :row-clickable="false"
      :row-selectable="!isMaskingForNoSQL"
      :show-operation="hasUpdateCatalogPermission && hasSensitiveDataFeature"
      :column-list="filteredColumnList"
      v-model:checked-column-list="state.pendingGrantAccessColumn"
      @delete="onColumnRemove"
    />
  </div>

  <FeatureModal
    :feature="PlanFeature.FEATURE_DATA_MASKING"
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
import { create } from "@bufbuild/protobuf";
import { ShieldCheckIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, reactive, watch } from "vue";
import {
  FeatureAttention,
  FeatureBadge,
  FeatureModal,
} from "@/components/FeatureGuard";
import SensitiveColumnTable from "@/components/SensitiveData/components/SensitiveColumnTable.vue";
import GrantAccessDrawer from "@/components/SensitiveData/GrantAccessDrawer.vue";
import type { MaskData } from "@/components/SensitiveData/types";
import { isCurrentColumnException } from "@/components/SensitiveData/utils";
import { SearchBox } from "@/components/v2";
import { featureToRef, useDatabaseCatalog, usePolicyV1Store } from "@/store";
import { type ComposedDatabase } from "@/types";
import {
  type ObjectSchema,
  ObjectSchema_Type,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import {
  MaskingExemptionPolicySchema,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasProjectPermissionV2, instanceV1MaskingForNoSQL } from "@/utils";

const props = defineProps<{
  database: ComposedDatabase;
}>();

interface LocalState {
  searchText: string;
  showFeatureModal: boolean;
  isLoading: boolean;
  sensitiveList: MaskData[];
  pendingGrantAccessColumn: MaskData[];
  showGrantAccessDrawer: boolean;
}

const state = reactive<LocalState>({
  searchText: "",
  showFeatureModal: false,
  isLoading: false,
  sensitiveList: [],
  pendingGrantAccessColumn: [],
  showGrantAccessDrawer: false,
});

const isMaskingForNoSQL = computed(() =>
  instanceV1MaskingForNoSQL(props.database.instanceResource)
);

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

const hasSensitiveDataFeature = featureToRef(PlanFeature.FEATURE_DATA_MASKING);

const databaseCatalog = useDatabaseCatalog(props.database.name, false);

const flattenObjectSchema = (
  parentPath: string,
  objectSchema: ObjectSchema
): {
  column: string;
  semanticTypeId: string;
  target: ObjectSchema;
}[] => {
  switch (objectSchema.type) {
    case ObjectSchema_Type.OBJECT:
      const resp = [];
      if (objectSchema.kind?.case === "structKind") {
        for (const [key, schema] of Object.entries(
          objectSchema.kind.value.properties ?? {}
        )) {
          resp.push(
            ...flattenObjectSchema(
              [parentPath, key].filter((i) => i).join("."),
              schema
            )
          );
        }
      }
      return resp;
    case ObjectSchema_Type.ARRAY:
      if (
        objectSchema.kind?.case === "arrayKind" &&
        objectSchema.kind.value.kind
      ) {
        return flattenObjectSchema(parentPath, objectSchema.kind.value.kind);
      }
      return [];
    default:
      return [
        {
          column: parentPath,
          semanticTypeId: objectSchema.semanticType,
          target: objectSchema,
        },
      ];
  }
};

const updateList = async () => {
  state.isLoading = true;
  const sensitiveList: MaskData[] = [];

  for (const schema of databaseCatalog.value.schemas) {
    for (const table of schema.tables) {
      // Handle table with structured columns
      if (table.kind?.case === "columns") {
        for (const column of table.kind.value.columns ?? []) {
          if (!column.semanticType && !column.classification) {
            continue;
          }
          sensitiveList.push({
            schema: schema.name,
            table: table.name,
            column: column.name,
            semanticTypeId: column.semanticType,
            classificationId: column.classification,
            target: column,
          });
        }
      }

      // Handle table with object schema
      if (table.kind?.case === "objectSchema") {
        const flattenItems = flattenObjectSchema("", table.kind.value);
        sensitiveList.push(
          ...flattenItems.map((item) => ({
            ...item,
            schema: schema.name,
            table: table.name,
            classificationId: "",
            // TODO(ed/zp/danny): can we support classification for object schema?
            disableClassification: true,
          }))
        );
      }

      if (table.classification) {
        sensitiveList.push({
          schema: schema.name,
          table: table.name,
          column: "",
          semanticTypeId: "",
          // TODO(ed/zp/danny): can we support senamtic type for table catalog?
          disableSemanticType: true,
          classificationId: table.classification,
          target: table,
        });
      }
    }
  }

  state.sensitiveList = sensitiveList;
  state.isLoading = false;
};

watch(databaseCatalog, updateList, { immediate: true, deep: true });

const filteredColumnList = computed(() => {
  let list = state.sensitiveList;
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
  await removeMaskingExceptions(sensitiveColumn);
};

const removeMaskingExceptions = async (sensitiveColumn: MaskData) => {
  const policy = await policyStore.getOrFetchPolicyByParentAndType({
    parentPath: props.database.project,
    policyType: PolicyType.MASKING_EXEMPTION,
  });
  if (!policy) {
    return;
  }

  const exceptions = (
    policy.policy?.case === "maskingExemptionPolicy"
      ? policy.policy.value.exemptions
      : []
  ).filter(
    (exception) =>
      !isCurrentColumnException(exception, {
        database: props.database,
        maskData: sensitiveColumn,
      })
  );

  policy.policy = {
    case: "maskingExemptionPolicy",
    value: create(MaskingExemptionPolicySchema, {
      exemptions: exceptions,
    }),
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
</script>
