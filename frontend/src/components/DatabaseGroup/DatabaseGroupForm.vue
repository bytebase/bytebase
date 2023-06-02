<template>
  <div class="w-full">
    <div class="w-full grid grid-cols-3 gap-x-6 pb-6 mb-4 border-b">
      <div>
        <p class="text-lg mb-2">{{ $t("common.name") }}</p>
        <ResourceIdInput
          ref="resourceIdInput"
          :value="state.resourceId"
          :resource-type="resourceIdType"
          :readonly="!isCreating"
          :validate="validateResourceId"
        />
      </div>
      <div>
        <p class="text-lg mb-2">{{ $t("common.environment") }}</p>
        <EnvironmentSelect
          :disabled="!isCreating"
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
          :disabled="!isCreating"
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
import { ConditionGroupExpr, resolveCELExpr, wrapAsGroup } from "@/plugins/cel";
import { Expr as CELExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import ExprEditor from "./common/ExprEditor";
import { ResourceType } from "./common/ExprEditor/context";
import { DatabaseGroup, SchemaGroup } from "@/types/proto/v1/project_service";
import { ComposedProject, ResourceId, ValidatedMessage } from "@/types";
import { convertCELStringToExpr } from "@/utils/databaseGroup/cel";
import { useDBGroupStore } from "@/store";
import { getErrorCode } from "@/utils/grpcweb";
import EnvironmentSelect from "../EnvironmentSelect.vue";
import MatchedDatabaseView from "./MatchedDatabaseView.vue";
import MatchedTableView from "./MatchedTableView.vue";
import ResourceIdInput from "../v2/Form/ResourceIdInput.vue";
import DatabaseGroupSelect from "./DatabaseGroupSelect.vue";
import {
  databaseGroupNamePrefix,
  getProjectNameAndDatabaseGroupName,
  getProjectNameAndDatabaseGroupNameAndSchemaGroupName,
} from "@/store/modules/v1/common";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { useDebounceFn } from "@vueuse/core";

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
const dbGroupStore = useDBGroupStore();
const state = reactive<LocalState>({
  resourceId: "",
  expr: wrapAsGroup(resolveCELExpr(CELExpr.fromJSON({}))),
});
const resourceIdInput = ref<InstanceType<typeof ResourceIdInput>>();
const selectedDatabaseGroupName = computed(() => {
  const [, databaseGroupName] = getProjectNameAndDatabaseGroupName(
    state.selectedDatabaseGroupId || ""
  );
  return databaseGroupName;
});

const isCreating = computed(() => props.databaseGroup === undefined);

const resourceIdType = computed(() =>
  props.resourceType === "DATABASE_GROUP" ? "database-group" : "schema-group"
);

onMounted(async () => {
  const databaseGroup = props.databaseGroup;
  if (!databaseGroup) {
    return;
  }

  if (props.resourceType === "DATABASE_GROUP") {
    const [, databaseGroupName] = getProjectNameAndDatabaseGroupName(
      databaseGroup.name
    );
    state.resourceId = databaseGroupName;
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
    const schemaGroup = databaseGroup as SchemaGroup;
    const expression = schemaGroup.tableExpr?.expression ?? "";
    const [projectName, databaseGroupName, schemaGroupName] =
      getProjectNameAndDatabaseGroupNameAndSchemaGroupName(schemaGroup.name);
    state.resourceId = schemaGroupName;
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

const validateResourceId = useDebounceFn(
  async (resourceId: ResourceId): Promise<ValidatedMessage[]> => {
    if (!resourceId) {
      return [];
    }

    let request = undefined;
    if (props.resourceType === "DATABASE_GROUP") {
      request = dbGroupStore.getOrFetchDBGroupByName(
        `${props.project.name}/databaseGroups/${resourceId}`
      );
    } else if (props.resourceType === "SCHEMA_GROUP") {
      if (state.selectedDatabaseGroupId) {
        request = dbGroupStore.getOrFetchSchemaGroupByName(
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
  },
  500
);

defineExpose({
  getFormState: () => {
    return {
      ...state,
      resourceId: resourceIdInput.value?.resourceId || "",
    };
  },
});
</script>
