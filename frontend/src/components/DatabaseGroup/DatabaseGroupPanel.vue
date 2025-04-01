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
        :project="project"
        :database-group="props.databaseGroup"
        @dismiss="() => emit('close')"
        @created="
          (databaseGroupName: string) => emit('created', databaseGroupName)
        "
      />
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Drawer, DrawerContent } from "@/components/v2";
import type { ComposedDatabaseGroup, ComposedProject } from "@/types";
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

const isCreating = computed(() => props.databaseGroup === undefined);

const title = computed(() => {
  return isCreating.value
    ? t("database-group.create")
    : t("database-group.edit");
});
</script>
