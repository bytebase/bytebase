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
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { ComposedProject } from "@/types";
import { DatabaseGroup, SchemaGroup } from "@/types/proto/v1/project_service";
import { Expr } from "@/types/proto/google/type/expr";
import { stringifyDatabaseGroupExpr } from "@/utils/databaseGroup/cel";
import {
  pushNotification,
  useDBGroupStore,
  useEnvironmentV1Store,
} from "@/store";
import { ResourceType } from "./common/ExprEditor/context";
import DatabaseGroupForm from "./DatabaseGroupForm.vue";
import { convertToCELString } from "@/plugins/cel/logic";

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
        await dbGroupStore.deleteDatabaseGroup(props.databaseGroup!.name);
      } else if (props.resourceType === "SCHEMA_GROUP") {
        await dbGroupStore.deleteSchemaGroup(props.databaseGroup!.name);
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

      const celString = convertToCELString(formState.expr);
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
      const celString = convertToCELString(formState.expr);
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
