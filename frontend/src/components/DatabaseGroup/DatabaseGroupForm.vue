<template>
  <div class="w-full">
    <div class="w-full grid grid-cols-3 gap-x-6 pb-6 mb-4 border-b">
      <div>
        <p class="text-lg mb-2">Name</p>
        <input
          v-model="state.databasePlaceholder"
          required
          type="text"
          class="textfield w-full"
        />
        <ResourceIdField
          ref="resourceIdField"
          resource-type="database-group"
          :readonly="!isCreating"
          :value="state.resourceId"
          :resource-title="state.databasePlaceholder"
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
      <div>
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
    </div>
    <div class="w-full grid grid-cols-5 gap-x-6">
      <div class="col-span-3">
        <p class="pl-1 text-lg mb-2">Condition</p>
        <ExprEditor
          :expr="state.expr"
          :allow-admin="true"
          :allow-high-level-factors="false"
        />
      </div>
      <div class="col-span-2">
        <p class="text-lg mb-2">Database</p>
        <MatchedDatabaseView
          :project="project"
          :environment-id="state.environmentId || ''"
          :expr="state.expr"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref } from "vue";
import { ConditionGroupExpr, resolveCELExpr, wrapAsGroup } from "@/plugins/cel";

import { Expr as CELExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import ExprEditor from "./common/ExprEditor";
import { ResourceType } from "./common/ExprEditor/context";
import { DatabaseGroup } from "@/types/proto/v1/project_service";
import { ComposedProject, ResourceId, ValidatedMessage } from "@/types";
import EnvironmentSelect from "../EnvironmentSelect.vue";
import MatchedDatabaseView from "./MatchedDatabaseView.vue";
import { convertDatabaseGroupExprFromCEL } from "@/utils/databaseGroup/cel";
import { useDBGroupStore, useEnvironmentV1Store } from "@/store";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { useI18n } from "vue-i18n";
import { getErrorCode } from "@/utils/grpcweb";
import { Status } from "nice-grpc-web";

const props = defineProps<{
  project: ComposedProject;
  databaseGroup?: DatabaseGroup;
}>();

type LocalState = {
  resourceId: string;
  databasePlaceholder: string;
  environmentId?: string;
  expr: ConditionGroupExpr;
  resourceType: ResourceType;
};

const { t } = useI18n();
const environmentStore = useEnvironmentV1Store();
const dbGroupStore = useDBGroupStore();
const state = reactive<LocalState>({
  resourceId: "",
  databasePlaceholder: props.databaseGroup?.databasePlaceholder ?? "",
  expr: wrapAsGroup(resolveCELExpr(CELExpr.fromJSON({}))),
  resourceType: "DATABASE_GROUP",
});
const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();

const isCreating = computed(() => props.databaseGroup === undefined);

onMounted(async () => {
  const databaseGroup = props.databaseGroup;
  if (!databaseGroup) {
    return;
  }

  const convertResult = await convertDatabaseGroupExprFromCEL(
    databaseGroup.databaseExpr?.expression ?? ""
  );
  const environment = environmentStore.getEnvironmentByName(
    convertResult.environmentId
  );
  state.resourceId = databaseGroup.name.split("/").pop() || "";
  state.databasePlaceholder = databaseGroup.databasePlaceholder;
  state.environmentId = environment?.uid;
  state.expr = convertResult.conditionGroupExpr;
});

const validateResourceId = async (
  resourceId: ResourceId
): Promise<ValidatedMessage[]> => {
  if (!resourceId) {
    return [];
  }

  try {
    const dbGroup = await dbGroupStore.getOrFetchDBGroupById(
      `${props.project.name}/databaseGroups/${resourceId}`
    );
    if (dbGroup) {
      return [
        {
          type: "error",
          message: t("resource-id.validation.duplicated", {
            resource: t("resource.environment"),
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
