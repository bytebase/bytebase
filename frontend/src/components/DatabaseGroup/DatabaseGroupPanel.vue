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
        :database-group="props.databaseGroup"
        :parent-database-group="props.parentDatabaseGroup"
      />
      <template #footer>
        <div class="w-full flex justify-between items-center">
          <NButton v-if="showDeleteButton" text @click="doDelete">
            <template #icon>
              <Trash2Icon class="w-4 h-auto" />
            </template>
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
import { Trash2Icon } from "lucide-vue-next";
import { NButton, useDialog } from "naive-ui";
import type { ClientError } from "nice-grpc-common";
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { Drawer, DrawerContent } from "@/components/v2";
import { buildCELExpr } from "@/plugins/cel/logic";
import {
  PROJECT_V1_ROUTE_DATABASE_GROUPS,
  PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
} from "@/router/dashboard/projectV1";
import { pushNotification, useDBGroupStore } from "@/store";
import type { ComposedDatabaseGroup, ComposedProject } from "@/types";
import { ParsedExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import { Expr } from "@/types/proto/google/type/expr";
import type { DatabaseGroup } from "@/types/proto/v1/project_service";
import { batchConvertParsedExprToCELString } from "@/utils";
import DatabaseGroupForm from "./DatabaseGroupForm.vue";

const props = defineProps<{
  show: boolean;
  project: ComposedProject;
  databaseGroup?: DatabaseGroup;
  parentDatabaseGroup?: ComposedDatabaseGroup;
}>();

const emit = defineEmits<{
  (event: "close"): void;
  (event: "created", databaseGroupName: string): void;
}>();

const { t } = useI18n();
const router = useRouter();
const dialog = useDialog();
const dbGroupStore = useDBGroupStore();
const formRef = ref<InstanceType<typeof DatabaseGroupForm>>();

const isCreating = computed(() => props.databaseGroup === undefined);

const title = computed(() => {
  return isCreating.value
    ? t("database-group.create")
    : t("database-group.edit");
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
  return formState.resourceId && formState.placeholder;
});

const showDeleteButton = computed(() => {
  return !isCreating.value;
});

onMounted(async () => {
  const project = router.currentRoute.value.params.projectId as string;
  if (project && typeof project === "string") {
    await dbGroupStore.getOrFetchDBGroupListByProjectName(
      `projects/${project}`
    );
  } else {
    await dbGroupStore.fetchAllDatabaseGroupList();
  }
});

const doDelete = () => {
  dialog.error({
    title: "Confirm to delete",
    positiveText: t("common.confirm"),
    negativeText: t("common.cancel"),
    onPositiveClick: async () => {
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
    if (isCreating.value) {
      const celStrings = await batchConvertParsedExprToCELString([
        ParsedExpr.fromJSON({
          expr: buildCELExpr(formState.expr),
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
          multitenancy: formState.multitenancy,
        },
        databaseGroupId: resourceId,
      });
      emit("created", resourceId);
    } else {
      const celStrings = await batchConvertParsedExprToCELString([
        ParsedExpr.fromJSON({
          expr: buildCELExpr(formState.expr),
        }),
      ]);
      await dbGroupStore.updateDatabaseGroup({
        ...props.databaseGroup!,
        databasePlaceholder: formState.placeholder,
        databaseExpr: Expr.fromJSON({
          expression: celStrings[0] || "true",
        }),
        multitenancy: formState.multitenancy,
      });
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
