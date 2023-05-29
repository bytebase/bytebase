<template>
  <div class="w-full">
    <div class="w-full grid grid-cols-3 gap-x-6 pb-6 mb-4 border-b">
      <div>
        <p class="text-lg mb-2">Name</p>
        <ResourceIdInput
          ref="resourceIdInput"
          :value="state.resourceId"
          :resource-type="resourceIdType"
          :readonly="!isCreating"
          :validate="validateResourceId"
        />
      </div>
      <div>
        <p class="text-lg mb-2">Environment</p>
        <EnvironmentSelect
          :selected-id="state.environmentId"
          @select-environment-id="
            (environmentId) => {
              state.environmentId = environmentId;
            }
          "
        />
      </div>
      <div v-if="resourceType === 'DATABASE_GROUP'">
        <p class="text-lg mb-2">Project</p>
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
        <p class="text-lg mb-2">Database group</p>
        <DatabaseGroupSelect
          :project-id="project.name"
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
        <p class="pl-1 text-lg mb-2">Condition</p>
        <ExprEditor
          :expr="state.expr"
          :allow-admin="true"
          :resource-type="resourceType"
        />
      </div>
      <div class="col-span-2">
        <p class="text-lg mb-2">{{ matchedSectionTitle }}</p>
        <MatchedDatabaseView
          v-if="resourceType === 'DATABASE_GROUP'"
          :project="project"
          :environment-id="state.environmentId || ''"
          :expr="state.expr"
        />
        <MatchedTableView
          v-if="resourceType === 'SCHEMA_GROUP'"
          :project="project"
          :environment-id="state.environmentId || ''"
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
import { ConditionGroupExpr, resolveCELExpr, wrapAsGroup } from "@/plugins/cel";
import { Expr as CELExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import ExprEditor from "./common/ExprEditor";
import { ResourceType } from "./common/ExprEditor/context";
import { DatabaseGroup, SchemaGroup } from "@/types/proto/v1/project_service";
import { ComposedProject, ResourceId, ValidatedMessage } from "@/types";
import { convertDatabaseGroupExprFromCEL } from "@/utils/databaseGroup/cel";
import { useDBGroupStore, useEnvironmentV1Store } from "@/store";
import { getErrorCode } from "@/utils/grpcweb";
import EnvironmentSelect from "../EnvironmentSelect.vue";
import MatchedDatabaseView from "./MatchedDatabaseView.vue";
import MatchedTableView from "./MatchedTableView.vue";
import ResourceIdInput from "../v2/Form/ResourceIdInput.vue";
import DatabaseGroupSelect from "./DatabaseGroupSelect.vue";

const props = defineProps<{
  project: ComposedProject;
  resourceType: ResourceType;
  databaseGroup?: DatabaseGroup | SchemaGroup;
}>();

type LocalState = {
  resourceId: string;
  environmentId?: string;
  selectedDatabaseGroupId?: string;
  expr: ConditionGroupExpr;
};

const { t } = useI18n();
const environmentStore = useEnvironmentV1Store();
const dbGroupStore = useDBGroupStore();
const state = reactive<LocalState>({
  resourceId: "",
  expr: wrapAsGroup(resolveCELExpr(CELExpr.fromJSON({}))),
});
const resourceIdInput = ref<InstanceType<typeof ResourceIdInput>>();

const isCreating = computed(() => props.databaseGroup === undefined);

const resourceIdType = computed(() =>
  props.resourceType === "DATABASE_GROUP" ? "database-group" : "schema-group"
);

const matchedSectionTitle = computed(() =>
  props.resourceType === "DATABASE_GROUP" ? "Database" : "Table"
);

onMounted(async () => {
  const databaseGroup = props.databaseGroup;
  if (!databaseGroup) {
    return;
  }

  let expression = "";
  if (props.resourceType === "DATABASE_GROUP") {
    expression =
      (databaseGroup as DatabaseGroup).databaseExpr?.expression ?? "";
  } else {
    expression = (databaseGroup as SchemaGroup).tableExpr?.expression ?? "";
  }
  const convertResult = await convertDatabaseGroupExprFromCEL(expression);
  const environment = environmentStore.getEnvironmentByName(
    convertResult.environmentId
  );
  state.resourceId = databaseGroup.name.split("/").pop() || "";
  state.environmentId = environment?.uid;
  state.expr = convertResult.conditionGroupExpr;
});

const validateResourceId = async (
  resourceId: ResourceId
): Promise<ValidatedMessage[]> => {
  if (!resourceId) {
    return [];
  }

  let request = undefined;
  if (props.resourceType === "DATABASE_GROUP") {
    request = dbGroupStore.getOrFetchDBGroupById(
      `${props.project.name}/databaseGroups/${resourceId}`
    );
  } else if (props.resourceType === "SCHEMA_GROUP") {
    if (state.selectedDatabaseGroupId) {
      request = dbGroupStore.getOrFetchSchemaGroupById(
        `${state.selectedDatabaseGroupId}/schemaGroups/${resourceId}`
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
      resourceId: resourceIdInput.value?.resourceId || "",
    };
  },
});
</script>
