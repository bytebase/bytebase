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
        :project="project"
        :database-group="props.databaseGroup"
      />
      <template #footer>
        <div class="flex items-center justify-end gap-x-2">
          <NButton @click="$emit('close')">{{ $t("common.cancel") }}</NButton>
          <NButton type="primary">
            {{ $t("common.confirm") }}
          </NButton>
        </div>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<script lang="ts" setup>
import { NButton, NDrawer, NDrawerContent } from "naive-ui";
import { computed } from "vue";
import { ComposedProject } from "@/types";
import { DatabaseGroup } from "@/types/proto/v1/project_service";
import DatabaseGroupForm from "./DatabaseGroupForm.vue";

const props = defineProps<{
  project: ComposedProject;
  databaseGroup?: DatabaseGroup;
}>();

defineEmits<{
  (event: "close"): void;
}>();

const isCreating = computed(() => props.databaseGroup === undefined);

const title = computed(() => {
  return isCreating.value ? "Create database group" : "Edit database group";
});
</script>
