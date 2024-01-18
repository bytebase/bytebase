<template>
  <Drawer
    :show="show"
    width="auto"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <DrawerContent
      :title="title"
      :closable="true"
      class="w-[64rem] max-w-[100vw] relative"
    >
      <DatabaseGroupForm
        ref="formRef"
        :project="project"
        :resource-type="resourceType"
        :database-group="props.databaseGroup"
        :parent-database-group="props.parentDatabaseGroup"
      />
      <template #footer>
        <div class="w-full flex justify-between items-center">
          <NButton v-if="showDeleteButton" type="error" @click="doDelete">
            {{ $t("common.delete") }}
          </NButton>
          <div class="flex flex-row justify-end items-center gap-x-2">
            <NButton @click="$emit('close')">{{ $t("common.cancel") }}</NButton>
            <NButton
              type="primary"
              :disabled="!allowConfirm"
              @click="doConfirm"
            >
              {{ isCreating ? $t("common.save") : $t("common.confirm") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { NButton, useDialog } from "naive-ui";
import { ClientError } from "nice-grpc-common";
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { Drawer, DrawerContent } from "@/components/v2";
import { buildCELExpr } from "@/plugins/cel/logic";
import {
  PROJECT_V1_ROUTE_DATABASE_GROUPS,
  PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_GROUP_TABLE_GROUP_DETAIL,
} from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useDBGroupStore,
  useEnvironmentV1Store,
} from "@/store";
import {
  getProjectNameAndDatabaseGroupName,
  getProjectNameAndDatabaseGroupNameAndSchemaGroupName,
} from "@/store/modules/v1/common";
import {
  ComposedDatabaseGroup,
  ComposedProject,
  ComposedSchemaGroup,
} from "@/types";
import { ParsedExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import { Expr } from "@/types/proto/google/type/expr";
import { DatabaseGroup, SchemaGroup } from "@/types/proto/v1/project_service";
import { batchConvertParsedExprToCELString } from "@/utils";
import { buildDatabaseGroupExpr } from "@/utils/databaseGroup/cel";
import DatabaseGroupForm from "./DatabaseGroupForm.vue";
import { ResourceType } from "./common/ExprEditor/context";

const props = defineProps<{
  show: boolean;
  project: ComposedProject;
  resourceType: ResourceType;
  databaseGroup?: DatabaseGroup | SchemaGroup;
  parentDatabaseGroup?: ComposedDatabaseGroup;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const dialog = useDialog();
const environmentStore = useEnvironmentV1Store();
const dbGroupStore = useDBGroupStore();
const formRef = ref<InstanceType<typeof DatabaseGroupForm>>();

const isCreating = computed(() => props.databaseGroup === undefined);

const title = computed(() => {
  if (props.resourceType === "DATABASE_GROUP") {
    return isCreating.value
      ? t("database-group.create")
      : t("database-group.edit");
  } else if (props.resourceType === "SCHEMA_GROUP") {
    return isCreating.value
      ? t("database-group.table-group.create")
      : t("database-group.table-group.edit");
  } else {
    throw new Error("Unknown resource type");
  }
});

const allowConfirm = computed(() => {
  if (!formRef.value) {
    return true;
  }

  if (!isCreating.value) {
    return true;
  }

  const formState = formRef.value.getFormState();
  if (formState.existMatchedUnactivateInstance) {
    return false;
  }
  if (props.resourceType === "DATABASE_GROUP") {
    return (
      formState.resourceId && formState.placeholder && formState.environmentId
    );
  } else if (props.resourceType === "SCHEMA_GROUP") {
    return (
      formState.resourceId &&
      formState.placeholder &&
      formState.environmentId &&
      formState.selectedDatabaseGroupId
    );
  }
  return false;
});

const showDeleteButton = computed(() => {
  return !isCreating.value;
});

const allowDelete = computed(() => {
  return (
    props.resourceType === "SCHEMA_GROUP" ||
    dbGroupStore.getSchemaGroupListByDBGroupName(props.databaseGroup!.name)
      .length === 0
  );
});

onMounted(async () => {
  await dbGroupStore.fetchAllDatabaseGroupList();
});

const doDelete = () => {
  if (!allowDelete.value) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: "You need to delete related table groups first.",
    });
    return;
  }

  dialog.error({
    title: "Confirm to delete",
    positiveText: t("common.confirm"),
    negativeText: t("common.cancel"),
    onPositiveClick: async () => {
      if (props.resourceType === "DATABASE_GROUP") {
        const databaseGroup = props.databaseGroup as DatabaseGroup;
        await dbGroupStore.deleteDatabaseGroup(databaseGroup.name);
        if (
          router.currentRoute.value.name ===
          PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL
        ) {
          router.replace({
            name: PROJECT_V1_ROUTE_DATABASE_GROUPS,
          });
        }
      } else if (props.resourceType === "SCHEMA_GROUP") {
        const schemaGroup = props.databaseGroup as ComposedSchemaGroup;
        const schemaGroupName = schemaGroup.name;
        await dbGroupStore.deleteSchemaGroup(schemaGroupName);
        if (
          router.currentRoute.value.name ===
          PROJECT_V1_ROUTE_DATABASE_GROUP_TABLE_GROUP_DETAIL
        ) {
          const [, databaseGroupName] =
            getProjectNameAndDatabaseGroupNameAndSchemaGroupName(
              schemaGroupName
            );
          router.replace({
            name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
            params: {
              databaseGroupName,
            },
          });
        }
      }
      emit("close");
    },
  });
};

const doConfirm = async () => {
  const formState = formRef.value?.getFormState();
  if (!formState) {
    return;
  }

  try {
    if (props.resourceType === "DATABASE_GROUP") {
      if (isCreating.value) {
        const environment = environmentStore.getEnvironmentByUID(
          formState.environmentId || ""
        );
        const celStrings = await batchConvertParsedExprToCELString([
          ParsedExpr.fromJSON({
            expr: buildCELExpr(
              buildDatabaseGroupExpr({
                environmentId: environment.name,
                conditionGroupExpr: formState.expr,
              })
            ),
          }),
        ]);
        const resourceId = formState.resourceId;
        await dbGroupStore.createDatabaseGroup({
          projectName: props.project.name,
          databaseGroup: {
            name: `${props.project.name}/databaseGroups/${resourceId}`,
            databasePlaceholder: formState.placeholder,
            databaseExpr: Expr.fromJSON({
              expression: celStrings[0] || "true",
            }),
          },
          databaseGroupId: resourceId,
        });
        router.push({
          name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
          params: {
            databaseGroupName: resourceId,
          },
        });
      } else {
        const environment = environmentStore.getEnvironmentByUID(
          formState.environmentId || ""
        );
        const celStrings = await batchConvertParsedExprToCELString([
          ParsedExpr.fromJSON({
            expr: buildCELExpr(
              buildDatabaseGroupExpr({
                environmentId: environment.name,
                conditionGroupExpr: formState.expr,
              })
            ),
          }),
        ]);
        await dbGroupStore.updateDatabaseGroup({
          ...props.databaseGroup!,
          databasePlaceholder: formState.placeholder,
          databaseExpr: Expr.fromJSON({
            expression: celStrings[0],
          }),
        });
      }
    } else if (props.resourceType === "SCHEMA_GROUP") {
      if (isCreating.value) {
        if (!formState.selectedDatabaseGroupId) {
          return;
        }

        let tableExpression = "true";
        if (buildCELExpr(formState.expr)) {
          const celStrings = await batchConvertParsedExprToCELString([
            ParsedExpr.fromJSON({
              expr: buildCELExpr(formState.expr),
            }),
          ]);
          tableExpression = celStrings[0] || "true";
        }

        const resourceId = formState.resourceId;
        await dbGroupStore.createSchemaGroup({
          dbGroupName: formState.selectedDatabaseGroupId,
          schemaGroup: {
            name: `${formState.selectedDatabaseGroupId}/schemaGroups/${resourceId}`,
            tablePlaceholder: formState.placeholder,
            tableExpr: Expr.fromJSON({
              expression: tableExpression,
            }),
          },
          schemaGroupId: resourceId,
        });
        const [, databaseGroupName] = getProjectNameAndDatabaseGroupName(
          formState.selectedDatabaseGroupId
        );
        router.push({
          name: PROJECT_V1_ROUTE_DATABASE_GROUP_TABLE_GROUP_DETAIL,
          params: {
            databaseGroupName,
            schemaGroupName: resourceId,
          },
        });
      } else {
        const celStrings = await batchConvertParsedExprToCELString([
          ParsedExpr.fromJSON({
            expr: buildCELExpr(formState.expr),
          }),
        ]);
        await dbGroupStore.updateSchemaGroup({
          ...props.databaseGroup!,
          tablePlaceholder: formState.placeholder,
          tableExpr: Expr.fromJSON({
            expression: celStrings[0],
          }),
        });
      }
    }
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: isCreating.value ? t("common.created") : t("common.updated"),
    });
    emit("close");
  } catch (error) {
    console.error(error);
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Request error occurred`,
      description: (error as ClientError).details,
    });
    return;
  }
};
</script>
