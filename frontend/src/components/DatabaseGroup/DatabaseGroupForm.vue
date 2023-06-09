<template>
  <div class="w-full">
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
            :resource-type="formatedResourceType"
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
            (environmentId) => {
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
            (id) => {
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
          :project="project"
          :environment-id="state.environmentId || ''"
          :database-group="databaseGroup as DatabaseGroup"
          :expr="state.expr"
        />
        <MatchedTableView
          v-if="resourceType === 'SCHEMA_GROUP'"
          :project="project"
          :schema-group="databaseGroup as SchemaGroup"
          :database-group-name="selectedDatabaseGroupName || ''"
          :expr="state.expr"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Status } from "nice-grpc-web";
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  ConditionGroupExpr,
  emptySimpleExpr,
  wrapAsGroup,
} from "@/plugins/cel";
import ExprEditor from "./common/ExprEditor";
import { ResourceType } from "./common/ExprEditor/context";
import { DatabaseGroup, SchemaGroup } from "@/types/proto/v1/project_service";
import {
  ComposedDatabaseGroup,
  ComposedProject,
  ResourceId,
  ValidatedMessage,
} from "@/types";
import { convertCELStringToExpr } from "@/utils/databaseGroup/cel";
import { useDBGroupStore } from "@/store";
import { getErrorCode } from "@/utils/grpcweb";
import EnvironmentSelect from "../EnvironmentSelect.vue";
import MatchedDatabaseView from "./MatchedDatabaseView.vue";
import MatchedTableView from "./MatchedTableView.vue";
import DatabaseGroupSelect from "./DatabaseGroupSelect.vue";
import {
  databaseGroupNamePrefix,
  getProjectNameAndDatabaseGroupName,
  getProjectNameAndDatabaseGroupNameAndSchemaGroupName,
  schemaGroupNamePrefix,
} from "@/store/modules/v1/common";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { ResourceIdField } from "../v2";

const props = defineProps<{
  project: ComposedProject;
  resourceType: ResourceType;
  databaseGroup?: DatabaseGroup | SchemaGroup;
  parentDatabaseGroup?: ComposedDatabaseGroup;
}>();

type LocalState = {
  resourceId: string;
  placeholder: string;
  environmentId?: string;
  selectedDatabaseGroupId?: string;
  expr: ConditionGroupExpr;
};

const { t } = useI18n();
const dbGroupStore = useDBGroupStore();
const state = reactive<LocalState>({
  resourceId: "",
  placeholder: "",
  expr: wrapAsGroup(emptySimpleExpr("_||_"), "_||_"),
});
const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();
const selectedDatabaseGroupName = computed(() => {
  const [, databaseGroupName] = getProjectNameAndDatabaseGroupName(
    state.selectedDatabaseGroupId || ""
  );
  return databaseGroupName;
});
const formatedResourceType = computed(() =>
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
  if (props.parentDatabaseGroup) {
    state.environmentId = props.parentDatabaseGroup.environment.uid;
    state.selectedDatabaseGroupId = props.parentDatabaseGroup.name;
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
      `${props.project.name}/${databaseGroupNamePrefix}${resourceId}`
    );
  } else if (props.resourceType === "SCHEMA_GROUP") {
    if (state.selectedDatabaseGroupId) {
      request = dbGroupStore.getOrFetchSchemaGroupByName(
        `${state.selectedDatabaseGroupId}/${schemaGroupNamePrefix}${resourceId}`
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

defineExpose({
  getFormState: () => {
    return {
      ...state,
      resourceId: resourceIdField.value?.resourceId || "",
    };
  },
});
</script>
