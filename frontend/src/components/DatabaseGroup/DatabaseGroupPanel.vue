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
      />
      <template #footer>
        <div class="w-full flex justify-between items-center">
          <div>
            <NButton v-if="showDeleteButton" text @click="doDelete">
              <template #icon>
                <Trash2Icon class="w-4 h-auto" />
              </template>
              {{ $t("common.delete") }}
            </NButton>
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
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { head, isEqual } from "lodash-es";
import { Trash2Icon } from "lucide-vue-next";
import { NButton, useDialog } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { Drawer, DrawerContent } from "@/components/v2";
import { buildCELExpr, validateSimpleExpr } from "@/plugins/cel/logic";
import {
  PROJECT_V1_ROUTE_DATABASE_GROUPS,
  PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
} from "@/router/dashboard/projectV1";
import { pushNotification, useDBGroupStore } from "@/store";
import type { ComposedDatabaseGroup, ComposedProject } from "@/types";
import { Expr as CELExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import { Expr } from "@/types/proto/google/type/expr";
import type { DatabaseGroup } from "@/types/proto/v1/database_group_service";
import { batchConvertParsedExprToCELString } from "@/utils";
import DatabaseGroupForm from "./DatabaseGroupForm.vue";

const props = defineProps<{
  show: boolean;
  project: ComposedProject;
  databaseGroup?: ComposedDatabaseGroup;
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
    return false;
  }

  const formState = formRef.value?.getFormState();
  if (formState.existMatchedUnactivateInstance) {
    return false;
  }
  return (
    formState.resourceId &&
    formState.placeholder &&
    validateSimpleExpr(formState.expr)
  );
});

const showDeleteButton = computed(() => {
  return !isCreating.value;
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
      databaseGroup: {
        name: `${props.project.name}/databaseGroups/${resourceId}`,
        databasePlaceholder: formState.placeholder,
        databaseExpr: Expr.fromPartial({
          expression: celString,
        }),
        multitenancy: formState.multitenancy,
      },
      databaseGroupId: resourceId,
    });
    emit("created", resourceId);
  } else {
    if (!props.databaseGroup) {
      return;
    }

    const updateMask: string[] = [];
    if (
      !isEqual(props.databaseGroup.databasePlaceholder, formState.placeholder)
    ) {
      updateMask.push("database_placeholder");
    }
    if (
      !isEqual(
        props.databaseGroup.databaseExpr,
        Expr.fromPartial({
          expression: celString,
        })
      )
    ) {
      updateMask.push("database_expr");
    }
    if (!isEqual(props.databaseGroup.multitenancy, formState.multitenancy)) {
      updateMask.push("multitenancy");
    }
    await dbGroupStore.updateDatabaseGroup(
      {
        ...props.databaseGroup!,
        databasePlaceholder: formState.placeholder,
        databaseExpr: Expr.fromPartial({
          expression: celString,
        }),
        multitenancy: formState.multitenancy,
      },
      updateMask
    );
  }

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: isCreating.value ? t("common.created") : t("common.updated"),
  });
  emit("close");
};
</script>
