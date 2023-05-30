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
      />
      <template #footer>
        <div class="w-full flex justify-between items-center">
          <div>
            <NButton v-if="!isCreating" type="error" @click="doDelete">{{
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
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { ComposedProject } from "@/types";
import { DatabaseGroup, SchemaGroup } from "@/types/proto/v1/project_service";
import { Expr } from "@/types/proto/google/type/expr";
import { stringifyDatabaseGroupExpr } from "@/utils/databaseGroup/cel";
import { useDBGroupStore, useEnvironmentV1Store } from "@/store";
import { ResourceType } from "./common/ExprEditor/context";
import DatabaseGroupForm from "./DatabaseGroupForm.vue";

const props = defineProps<{
  project: ComposedProject;
  resourceType: ResourceType;
  databaseGroup?: DatabaseGroup | SchemaGroup;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const dialog = useDialog();
const environmentStore = useEnvironmentV1Store();
const dbGroupStore = useDBGroupStore();
const formRef = ref<InstanceType<typeof DatabaseGroupForm>>();

const isCreating = computed(() => props.databaseGroup === undefined);

const title = computed(() => {
  if (props.resourceType === "DATABASE_GROUP") {
    return isCreating.value ? "Create database group" : "Edit database group";
  } else if (props.resourceType === "SCHEMA_GROUP") {
    return isCreating.value ? "Create table group" : "Edit table group";
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
  if (props.resourceType === "DATABASE_GROUP") {
    return formState.resourceId && formState.environmentId;
  } else if (props.resourceType === "SCHEMA_GROUP") {
    return (
      formState.resourceId &&
      formState.environmentId &&
      formState.selectedDatabaseGroupId
    );
  }
  return false;
});

const doDelete = () => {
  dialog.error({
    title: "Confirm to delete",
    positiveText: t("common.confirm"),
    negativeText: t("common.cancel"),
    onPositiveClick: async () => {
      if (props.resourceType === "DATABASE_GROUP") {
        await dbGroupStore.deleteDatabaseGroup(
          props.databaseGroup as DatabaseGroup
        );
      } else if (props.resourceType === "SCHEMA_GROUP") {
        await dbGroupStore.deleteSchemaGroup(
          props.databaseGroup as SchemaGroup
        );
      } else {
        throw new Error("Unknown resource type");
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

  if (props.resourceType === "DATABASE_GROUP") {
    if (isCreating.value) {
      const environment = environmentStore.getEnvironmentByUID(
        formState.environmentId || ""
      );
      const celString = stringifyDatabaseGroupExpr({
        environmentId: environment.name,
        conditionGroupExpr: formState.expr,
      });
      const resourceId = formState.resourceId;
      await dbGroupStore.createDatabaseGroup(
        props.project.name,
        {
          name: `${props.project.name}/databaseGroups/${resourceId}`,
          databasePlaceholder: resourceId,
          databaseExpr: Expr.fromJSON({
            expression: celString,
          }),
        },
        resourceId
      );
    } else {
      const environment = environmentStore.getEnvironmentByUID(
        formState.environmentId || ""
      );
      const celString = stringifyDatabaseGroupExpr({
        environmentId: environment.name,
        conditionGroupExpr: formState.expr,
      });
      await dbGroupStore.updateDatabaseGroup({
        ...props.databaseGroup!,
        databasePlaceholder: "",
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

      const environment = environmentStore.getEnvironmentByUID(
        formState.environmentId || ""
      );
      const celString = stringifyDatabaseGroupExpr({
        environmentId: environment.name,
        conditionGroupExpr: formState.expr,
      });
      const resourceId = formState.resourceId;
      await dbGroupStore.createSchemaGroup(
        formState.selectedDatabaseGroupId,
        {
          name: `${formState.selectedDatabaseGroupId}/schemaGroups/${resourceId}`,
          tablePlaceholder: resourceId,
          tableExpr: Expr.fromJSON({
            expression: celString,
          }),
        },
        resourceId
      );
    } else {
      const environment = environmentStore.getEnvironmentByUID(
        formState.environmentId || ""
      );
      const celString = stringifyDatabaseGroupExpr({
        environmentId: environment.name,
        conditionGroupExpr: formState.expr,
      });
      await dbGroupStore.updateSchemaGroup({
        ...props.databaseGroup!,
        tablePlaceholder: "",
        tableExpr: Expr.fromJSON({
          expression: celString,
        }),
      });
    }
  } else {
    throw new Error("Unknown resource type");
  }

  emit("close");
};
</script>
