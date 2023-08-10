<template>
  <div class="w-full">
    <FeatureAttentionForInstanceLicense
      v-if="existMatchedUnactivateInstance"
      custom-class="mb-5"
      :style="`WARN`"
      feature="bb.feature.database-grouping"
    />
    <div class="w-full grid grid-cols-3 gap-x-6 pb-6 mb-4 border-b">
      <div>
        <p class="text-lg mb-2">{{ $t("common.name") }}</p>
        <input
          v-model="state.placeholder"
          required
          type="text"
          class="textfield w-full"
        />
        <div class="mt-2">
          <ResourceIdField
            ref="resourceIdField"
            :resource-type="formattedResourceType"
            :readonly="!isCreating"
            :value="state.resourceId"
            :resource-title="state.placeholder"
            :validate="validateResourceId"
          />
        </div>
      </div>
      <div>
        <p class="text-lg mb-2">{{ $t("common.environment") }}</p>
        <EnvironmentSelect
          :disabled="!isCreating || disableEditDatabaseGroupFields"
          :selected-id="state.environmentId"
          @select-environment-id="
            (environmentId: any) => {
              state.environmentId = environmentId;
            }
          "
        />
      </div>
      <div v-if="resourceType === 'DATABASE_GROUP'">
        <p class="text-lg mb-2">{{ $t("common.project") }}</p>
        <input
          required
          type="text"
          readonly
          disabled
          :value="project.title"
          class="textfield w-full"
        />
      </div>
      <div v-if="resourceType === 'SCHEMA_GROUP'">
        <p class="text-lg mb-2">{{ $t("database-group.self") }}</p>
        <DatabaseGroupSelect
          :disabled="!isCreating || disableEditDatabaseGroupFields"
          :project-id="project.name"
          :environment-id="state.environmentId || ''"
          :selected-id="state.selectedDatabaseGroupId"
          @select-database-group-id="
            (id: any) => {
              state.selectedDatabaseGroupId = id;
            }
          "
        />
      </div>
    </div>
    <div class="w-full grid grid-cols-5 gap-x-6">
      <div class="col-span-3">
        <p class="pl-1 text-lg mb-2">
          {{ $t("database-group.condition.self") }}
        </p>
        <ExprEditor
          :expr="state.expr"
          :allow-admin="true"
          :resource-type="resourceType"
        />
      </div>
      <div class="col-span-2">
        <MatchedDatabaseView
          v-if="resourceType === 'DATABASE_GROUP'"
          :loading="state.isRequesting"
          :matched-database-list="matchedDatabaseList"
          :unmatched-database-list="unmatchedDatabaseList"
        />
        <MatchedTableView
          v-if="resourceType === 'SCHEMA_GROUP'"
          :loading="state.isRequesting"
          :matched-table-list="matchedTableList"
          :unmatched-table-list="unmatchedTableList"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import { head } from "lodash-es";
import { Status } from "nice-grpc-web";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  ConditionGroupExpr,
  emptySimpleExpr,
  wrapAsGroup,
} from "@/plugins/cel";
import { useDBGroupStore, useSubscriptionV1Store } from "@/store";
import {
  databaseGroupNamePrefix,
  getProjectNameAndDatabaseGroupName,
  getProjectNameAndDatabaseGroupNameAndSchemaGroupName,
  schemaGroupNamePrefix,
} from "@/store/modules/v1/common";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  ComposedSchemaGroupTable,
  ComposedDatabase,
  ComposedDatabaseGroup,
  ComposedProject,
  ResourceId,
  ValidatedMessage,
} from "@/types";
import { DatabaseGroup, SchemaGroup } from "@/types/proto/v1/project_service";
import { convertCELStringToExpr } from "@/utils/databaseGroup/cel";
import { getErrorCode } from "@/utils/grpcweb";
import EnvironmentSelect from "../EnvironmentSelect.vue";
import { ResourceIdField } from "../v2";
import DatabaseGroupSelect from "./DatabaseGroupSelect.vue";
import MatchedDatabaseView from "./MatchedDatabaseView.vue";
import MatchedTableView from "./MatchedTableView.vue";
import ExprEditor from "./common/ExprEditor";
import { ResourceType } from "./common/ExprEditor/context";

const props = defineProps<{
  project: ComposedProject;
  resourceType: ResourceType;
  databaseGroup?: DatabaseGroup | SchemaGroup;
  parentDatabaseGroup?: ComposedDatabaseGroup;
}>();

type LocalState = {
  isRequesting: boolean;
  resourceId: string;
  placeholder: string;
  environmentId?: string;
  selectedDatabaseGroupId?: string;
  expr: ConditionGroupExpr;
};

const { t } = useI18n();
const dbGroupStore = useDBGroupStore();
const subscriptionV1Store = useSubscriptionV1Store();
const state = reactive<LocalState>({
  isRequesting: false,
  resourceId: "",
  placeholder: "",
  expr: wrapAsGroup(emptySimpleExpr()),
});
const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();
const selectedDatabaseGroupName = computed(() => {
  const [, databaseGroupName] = getProjectNameAndDatabaseGroupName(
    state.selectedDatabaseGroupId || ""
  );
  return databaseGroupName;
});
const formattedResourceType = computed(() =>
  props.resourceType === "DATABASE_GROUP" ? "database-group" : "schema-group"
);

const isCreating = computed(() => props.databaseGroup === undefined);

const disableEditDatabaseGroupFields = computed(() => {
  return (
    props.resourceType === "SCHEMA_GROUP" &&
    props.parentDatabaseGroup !== undefined
  );
});

const resourceIdType = computed(() =>
  props.resourceType === "DATABASE_GROUP" ? "database-group" : "schema-group"
);

onMounted(async () => {
  if (isCreating.value && props.resourceType === "SCHEMA_GROUP") {
    if (props.parentDatabaseGroup) {
      state.environmentId = props.parentDatabaseGroup.environment.uid;
      state.selectedDatabaseGroupId = props.parentDatabaseGroup.name;
    } else {
      const dbGroup = head(dbGroupStore.getAllDatabaseGroupList());
      if (dbGroup) {
        state.environmentId = dbGroup.environment.uid;
        state.selectedDatabaseGroupId = dbGroup.name;
      }
    }
    return;
  }

  const databaseGroup = props.databaseGroup;
  if (!databaseGroup) {
    return;
  }

  if (props.resourceType === "DATABASE_GROUP") {
    const databaseGroupEntity = databaseGroup as DatabaseGroup;
    const [, databaseGroupName] = getProjectNameAndDatabaseGroupName(
      databaseGroup.name
    );
    state.resourceId = databaseGroupName;
    state.placeholder = databaseGroupEntity.databasePlaceholder;
    const composedDatabaseGroup = await dbGroupStore.getOrFetchDBGroupByName(
      databaseGroup.name
    );
    if (composedDatabaseGroup.environment) {
      state.environmentId = composedDatabaseGroup.environment.uid;
    }
    if (composedDatabaseGroup.simpleExpr) {
      state.expr = composedDatabaseGroup.simpleExpr;
    }
  } else {
    const schemaGroupEntity = databaseGroup as SchemaGroup;
    const expression = schemaGroupEntity.tableExpr?.expression ?? "";
    const [projectName, databaseGroupName, schemaGroupName] =
      getProjectNameAndDatabaseGroupNameAndSchemaGroupName(
        schemaGroupEntity.name
      );
    state.resourceId = schemaGroupName;
    state.placeholder = schemaGroupEntity.tablePlaceholder;
    state.selectedDatabaseGroupId = `${projectNamePrefix}${projectName}/${databaseGroupNamePrefix}${databaseGroupName}`;
    const expr = await convertCELStringToExpr(expression);
    state.expr = expr;

    // Fetch related database group environment.
    const relatedDatabaseGroup = await dbGroupStore.getOrFetchDBGroupByName(
      `${projectNamePrefix}${projectName}/${databaseGroupNamePrefix}${databaseGroupName}`
    );
    if (relatedDatabaseGroup.environment) {
      state.environmentId = relatedDatabaseGroup.environment.uid;
    }
  }
});

const validateResourceId = async (
  resourceId: ResourceId
): Promise<ValidatedMessage[]> => {
  if (!resourceId) {
    return [];
  }

  let request = undefined;
  if (props.resourceType === "DATABASE_GROUP") {
    request = dbGroupStore.getOrFetchDBGroupByName(
      `${props.project.name}/${databaseGroupNamePrefix}${resourceId}`,
      true /* silent */
    );
  } else if (props.resourceType === "SCHEMA_GROUP") {
    if (state.selectedDatabaseGroupId) {
      request = dbGroupStore.getOrFetchSchemaGroupByName(
        `${state.selectedDatabaseGroupId}/${schemaGroupNamePrefix}${resourceId}`,
        true /* silent */
      );
    }
  }

  if (!request) {
    return [];
  }

  try {
    const data = await request;
    if (data) {
      return [
        {
          type: "error",
          message: t("resource-id.validation.duplicated", {
            resource: t(`resource.${resourceIdType.value}`),
          }),
        },
      ];
    }
  } catch (error) {
    if (getErrorCode(error) !== Status.NOT_FOUND) {
      throw error;
    }
  }

  return [];
};

const matchedDatabaseList = ref<ComposedDatabase[]>([]);
const unmatchedDatabaseList = ref<ComposedDatabase[]>([]);
const updateDatabaseMatchingState = useDebounceFn(async () => {
  if (props.resourceType !== "DATABASE_GROUP") {
    return;
  }
  if (!state.environmentId) {
    return;
  }

  state.isRequesting = true;
  const result = await dbGroupStore.fetchDatabaseGroupMatchList({
    projectName: props.project.name,
    environmentId: state.environmentId,
    expr: state.expr,
  });

  matchedDatabaseList.value = result.matchedDatabaseList;
  unmatchedDatabaseList.value = result.unmatchedDatabaseList;
  state.isRequesting = false;
}, 500);

watch(
  [() => props.project.name, () => state.environmentId, () => state.expr],
  updateDatabaseMatchingState,
  {
    immediate: true,
    deep: true,
  }
);

const matchedTableList = ref<ComposedSchemaGroupTable[]>([]);
const unmatchedTableList = ref<ComposedSchemaGroupTable[]>([]);
const updateTableMatchingState = useDebounceFn(async () => {
  if (props.resourceType !== "SCHEMA_GROUP") {
    return;
  }
  if (!selectedDatabaseGroupName.value) {
    return;
  }

  state.isRequesting = true;
  const result = await dbGroupStore.fetchSchemaGroupMatchList({
    projectName: props.project.name,
    databaseGroupName: selectedDatabaseGroupName.value,
    expr: state.expr,
  });

  matchedTableList.value = result.matchedTableList;
  unmatchedTableList.value = result.unmatchedTableList;
  state.isRequesting = false;
}, 500);

watch(
  [
    () => props.project.name,
    () => selectedDatabaseGroupName.value,
    () => state.expr,
  ],
  updateTableMatchingState,
  {
    immediate: true,
    deep: true,
  }
);

const existMatchedUnactivateInstance = computed(() => {
  if (props.resourceType === "DATABASE_GROUP") {
    return matchedDatabaseList.value.some(
      (database) =>
        !subscriptionV1Store.hasInstanceFeature(
          "bb.feature.database-grouping",
          database.instanceEntity
        )
    );
  } else {
    return matchedTableList.value.some(
      (tb) =>
        !subscriptionV1Store.hasInstanceFeature(
          "bb.feature.database-grouping",
          tb.databaseEntity.instanceEntity
        )
    );
  }
});

defineExpose({
  getFormState: () => {
    return {
      ...state,
      resourceId: resourceIdField.value?.resourceId || "",
      existMatchedUnactivateInstance: existMatchedUnactivateInstance.value,
    };
  },
});
</script>
