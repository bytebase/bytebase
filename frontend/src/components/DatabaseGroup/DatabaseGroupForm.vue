<template>
  <div class="flex-1 flex flex-col">
    <div class="flex-1 mb-6">
      <div class="w-full">
        <p class="font-medium text-main mb-2">{{ $t("common.name") }}</p>
        <NInput
          v-model:value="state.placeholder"
          :disabled="readonly"
          :maxlength="200"
        />
        <div class="mt-2">
          <ResourceIdField
            ref="resourceIdField"
            editing-class="mt-4"
            resource-type="database-group"
            :readonly="!isCreating"
            :value="state.resourceId"
            :resource-title="state.placeholder"
            :fetch-resource="
              (id) =>
                dbGroupStore.getOrFetchDBGroupByName(
                  `${props.project.name}/${databaseGroupNamePrefix}${id}`,
                  { silent: true, view: DatabaseGroupView.FULL }
                )
            "
          />
        </div>
      </div>
      <NDivider />
      <div class="w-full grid grid-cols-5 gap-x-6">
        <div class="col-span-3">
          <p class="pl-1 font-medium text-main mb-2">
            {{ $t("database-group.condition.self") }}
          </p>
          <ExprEditor
            :expr="state.expr"
            :readonly="readonly"
            :factor-list="FactorList"
            :option-config-map="getDatabaseGroupOptionConfigMap(project.name)"
          />
        </div>
        <div class="col-span-2">
          <MatchedDatabaseView :project="project.name" :expr="state.expr" />
        </div>
      </div>
    </div>

    <div v-if="!isCreating && allowDelete" class="py-6 border-t">
      <BBButtonConfirm
        :type="'DELETE'"
        :button-text="$t('database-group.delete-group')"
        :ok-text="$t('common.delete')"
        :confirm-title="
          $t('database-group.delete-group', { name: databaseGroup?.title })
        "
        :require-confirm="true"
        @confirm="doDelete"
      />
    </div>

    <div v-if="!readonly" class="sticky bottom-0 z-10">
      <div
        class="flex justify-end w-full py-4 border-t border-block-border bg-white gap-x-3"
      >
        <NButton @click="$emit('dismiss')">{{ $t("common.cancel") }}</NButton>
        <NButton type="primary" :disabled="!allowConfirm" @click="doConfirm">
          {{ isCreating ? $t("common.save") : $t("common.confirm") }}
        </NButton>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { cloneDeep, head, isEqual } from "lodash-es";
import { NButton, NDivider, NInput } from "naive-ui";
import { computed, reactive, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBButtonConfirm } from "@/bbkit";
import ExprEditor from "@/components/ExprEditor";
import type { ConditionGroupExpr } from "@/plugins/cel";
import {
  buildCELExpr,
  emptySimpleExpr,
  resolveCELExpr,
  validateSimpleExpr,
  wrapAsGroup,
} from "@/plugins/cel";
import {
  PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_GROUPS,
} from "@/router/dashboard/projectV1";
import { pushNotification, useDBGroupStore } from "@/store";
import {
  databaseGroupNamePrefix,
  getProjectNameAndDatabaseGroupName,
} from "@/store/modules/v1/common";
import { isValidDatabaseGroupName } from "@/types";
import type { Expr as CELExpr } from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import {
  type DatabaseGroup,
  DatabaseGroupSchema,
  DatabaseGroupView,
} from "@/types/proto-es/v1/database_group_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  batchConvertCELStringToParsedExpr,
  batchConvertParsedExprToCELString,
  hasProjectPermissionV2,
} from "@/utils";
import { ResourceIdField } from "../v2";
import MatchedDatabaseView from "./MatchedDatabaseView.vue";
import { FactorList, getDatabaseGroupOptionConfigMap } from "./utils";

const props = defineProps<{
  readonly: boolean;
  project: Project;
  databaseGroup?: DatabaseGroup;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
  (event: "created", databaseGroupName: string): void;
}>();

type LocalState = {
  resourceId: string;
  placeholder: string;
  selectedDatabaseGroupId?: string;
  expr: ConditionGroupExpr;
};

const { t } = useI18n();
const dbGroupStore = useDBGroupStore();
const state = reactive<LocalState>({
  resourceId: "",
  placeholder: "",
  expr: wrapAsGroup(emptySimpleExpr()),
});
const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();
const router = useRouter();

const isCreating = computed(() => props.databaseGroup === undefined);

const allowDelete = computed(() => {
  return hasProjectPermissionV2(props.project, "bb.databaseGroups.delete");
});

watchEffect(async () => {
  const databaseGroup = props.databaseGroup;
  if (!databaseGroup || !isValidDatabaseGroupName(databaseGroup.name)) {
    return;
  }

  const databaseGroupEntity = databaseGroup;
  const [, databaseGroupName] = getProjectNameAndDatabaseGroupName(
    databaseGroup.name
  );
  state.resourceId = databaseGroupName;
  state.placeholder = databaseGroupEntity.title;

  if (databaseGroup.databaseExpr?.expression) {
    // Convert CEL expression to simple expression
    const expressions = [databaseGroup.databaseExpr.expression];
    const exprList = await batchConvertCELStringToParsedExpr(expressions);
    if (exprList.length > 0) {
      const simpleExpr = resolveCELExpr(exprList[0]);
      state.expr = cloneDeep(wrapAsGroup(simpleExpr));
    }
  }
});

const doDelete = async () => {
  const databaseGroup = props.databaseGroup as DatabaseGroup;
  await dbGroupStore.deleteDatabaseGroup(databaseGroup.name);
  if (
    router.currentRoute.value.name === PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL
  ) {
    router.replace({
      name: PROJECT_V1_ROUTE_DATABASE_GROUPS,
    });
  }
  emit("dismiss");
};

const allowConfirm = computed(() => {
  return (
    resourceIdField.value?.resourceId &&
    state.placeholder &&
    validateSimpleExpr(state.expr)
  );
});

const doConfirm = async () => {
  const formState = {
    ...state,
    resourceId: resourceIdField.value?.resourceId || "",
  };
  if (!formState || !allowConfirm.value) {
    return;
  }

  let celExpr: CELExpr | undefined = undefined;
  try {
    celExpr = await buildCELExpr(formState.expr);
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `CEL expression error occurred`,
      description: (error as Error).message,
    });
    return;
  }

  const celStrings = await batchConvertParsedExprToCELString([celExpr!]);
  const celString = head(celStrings);
  if (!celString) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `CEL expression error occurred`,
      description: "CEL expression is empty",
    });
    return;
  }

  if (isCreating.value) {
    const resourceId = formState.resourceId;
    await dbGroupStore.createDatabaseGroup({
      projectName: props.project.name,
      databaseGroup: create(DatabaseGroupSchema, {
        name: `${props.project.name}/databaseGroups/${resourceId}`,
        title: formState.placeholder,
        databaseExpr: create(ExprSchema, {
          expression: celString,
        }),
      }),
      databaseGroupId: resourceId,
    });
    emit("created", resourceId);
  } else {
    if (!props.databaseGroup) {
      return;
    }

    const updateMask: string[] = [];
    if (!isEqual(props.databaseGroup.title, formState.placeholder)) {
      updateMask.push("title");
    }
    if (
      !isEqual(
        props.databaseGroup.databaseExpr,
        create(ExprSchema, {
          expression: celString,
        })
      )
    ) {
      updateMask.push("database_expr");
    }
    await dbGroupStore.updateDatabaseGroup(
      {
        ...props.databaseGroup,
        title: formState.placeholder,
        databaseExpr: create(ExprSchema, {
          expression: celString,
        }),
      },
      updateMask
    );
  }

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: isCreating.value ? t("common.created") : t("common.updated"),
  });
  emit("dismiss");
};
</script>
