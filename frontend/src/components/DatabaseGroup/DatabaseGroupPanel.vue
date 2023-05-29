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
        :database-group="props.databaseGroup"
      />
      <template #footer>
        <div class="flex items-center justify-end gap-x-2">
          <NButton @click="$emit('close')">{{ $t("common.cancel") }}</NButton>
          <NButton type="primary" @click="doConfirm">
            {{ $t("common.confirm") }}
          </NButton>
        </div>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<script lang="ts" setup>
import { NButton, NDrawer, NDrawerContent } from "naive-ui";
import { computed, ref } from "vue";
import { ComposedProject } from "@/types";
import { DatabaseGroup } from "@/types/proto/v1/project_service";
import DatabaseGroupForm from "./DatabaseGroupForm.vue";
import { useDBGroupStore, useEnvironmentV1Store } from "@/store";
import { stringifyDatabaseGroupExpr } from "@/utils/databaseGroup/cel";
import { Expr } from "@/types/proto/google/type/expr";

const props = defineProps<{
  project: ComposedProject;
  databaseGroup?: DatabaseGroup;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const environmentStore = useEnvironmentV1Store();
const dbGroupStore = useDBGroupStore();
const formRef = ref<InstanceType<typeof DatabaseGroupForm>>();

const isCreating = computed(() => props.databaseGroup === undefined);

const title = computed(() => {
  return isCreating.value ? "Create database group" : "Edit database group";
});

const doConfirm = async () => {
  const formState = formRef.value?.getFormState();
  if (!formState) {
    return;
  }

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
        databasePlaceholder: formState.databasePlaceholder,
        databaseExpr: Expr.fromJSON({
          expression: celString,
        }),
      },
      resourceId
    );
    emit("close");
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
      databasePlaceholder: formState.databasePlaceholder,
      databaseExpr: Expr.fromJSON({
        expression: celString,
      }),
    });
  }
};
</script>
