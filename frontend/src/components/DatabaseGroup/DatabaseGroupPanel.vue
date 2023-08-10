<template>
  <NDrawer
    :show="true"
    width="auto"
    :auto-focus="false"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <NDrawerContent
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
          <div>
            <NButton v-if="showDeleteButton" type="error" @click="doDelete">{{
              $t("common.delete")
            }}</NButton>
          </div>
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
    </NDrawerContent>
  </NDrawer>
</template>

<script lang="ts" setup>
import { NButton, NDrawer, NDrawerContent, useDialog } from "naive-ui";
import { ClientError } from "nice-grpc-common";
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { buildCELExpr } from "@/plugins/cel/logic";
import {
  pushNotification,
  useDBGroupStore,
  useEnvironmentV1Store,
} from "@/store";
import { getProjectNameAndDatabaseGroupNameAndSchemaGroupName } from "@/store/modules/v1/common";
import {
  ComposedDatabaseGroup,
  ComposedProject,
  ComposedSchemaGroup,
} from "@/types";
import { ParsedExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import { Expr } from "@/types/proto/google/type/expr";
import { DatabaseGroup, SchemaGroup } from "@/types/proto/v1/project_service";
import { convertParsedExprToCELString, projectV1Slug } from "@/utils";
import { buildDatabaseGroupExpr } from "@/utils/databaseGroup/cel";
import DatabaseGroupForm from "./DatabaseGroupForm.vue";
import { ResourceType } from "./common/ExprEditor/context";

const props = defineProps<{
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
          router.currentRoute.value.name === "workspace.database-group.detail"
        ) {
          router.replace({
            name: "workspace.project.detail",
            params: {
              projectSlug: projectV1Slug(props.project),
            },
            hash: "#database-groups",
          });
        }
      } else if (props.resourceType === "SCHEMA_GROUP") {
        const schemaGroup = props.databaseGroup as ComposedSchemaGroup;
        const schemaGroupName = schemaGroup.name;
        await dbGroupStore.deleteSchemaGroup(schemaGroupName);
        if (
          router.currentRoute.value.name ===
          "workspace.database-group.table-group.detail"
        ) {
          const [projectName, databaseGroupName] =
            getProjectNameAndDatabaseGroupNameAndSchemaGroupName(
              schemaGroupName
            );
          // TODO(steven): prevent `Cannot use 'in' operator to search for 'path' in undefined` error in vue-router.
          window.location.href = `/projects/${projectName}/database-groups/${databaseGroupName}`;
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
        const celString = await convertParsedExprToCELString(
          ParsedExpr.fromJSON({
            expr: buildCELExpr(
              buildDatabaseGroupExpr({
                environmentId: environment.name,
                conditionGroupExpr: formState.expr,
              })
            ),
          })
        );
        const resourceId = formState.resourceId;
        await dbGroupStore.createDatabaseGroup({
          projectName: props.project.name,
          databaseGroup: {
            name: `${props.project.name}/databaseGroups/${resourceId}`,
            databasePlaceholder: formState.placeholder,
            databaseExpr: Expr.fromJSON({
              expression: celString || "true",
            }),
          },
          databaseGroupId: resourceId,
        });
      } else {
        const environment = environmentStore.getEnvironmentByUID(
          formState.environmentId || ""
        );
        const celString = await convertParsedExprToCELString(
          ParsedExpr.fromJSON({
            expr: buildCELExpr(
              buildDatabaseGroupExpr({
                environmentId: environment.name,
                conditionGroupExpr: formState.expr,
              })
            ),
          })
        );
        await dbGroupStore.updateDatabaseGroup({
          ...props.databaseGroup!,
          databasePlaceholder: formState.placeholder,
          databaseExpr: Expr.fromJSON({
            expression: celString,
          }),
        });
      }
    } else if (props.resourceType === "SCHEMA_GROUP") {
      if (isCreating.value) {
        if (!formState.selectedDatabaseGroupId) {
          return;
        }

        const celString = await convertParsedExprToCELString(
          ParsedExpr.fromJSON({
            expr: buildCELExpr(formState.expr),
          })
        );
        const resourceId = formState.resourceId;
        await dbGroupStore.createSchemaGroup({
          dbGroupName: formState.selectedDatabaseGroupId,
          schemaGroup: {
            name: `${formState.selectedDatabaseGroupId}/schemaGroups/${resourceId}`,
            tablePlaceholder: formState.placeholder,
            tableExpr: Expr.fromJSON({
              expression: celString || "true",
            }),
          },
          schemaGroupId: resourceId,
        });
      } else {
        const celString = await convertParsedExprToCELString(
          ParsedExpr.fromJSON({
            expr: buildCELExpr(formState.expr),
          })
        );
        await dbGroupStore.updateSchemaGroup({
          ...props.databaseGroup!,
          tablePlaceholder: formState.placeholder,
          tableExpr: Expr.fromJSON({
            expression: celString,
          }),
        });
      }
    }
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

  emit("close");
};
</script>
