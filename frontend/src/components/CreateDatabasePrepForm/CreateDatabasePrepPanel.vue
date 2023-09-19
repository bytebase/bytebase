<template>
  <DrawerContent :title="$t('quick-action.create-db')">
    <CreateDatabasePrepForm
      ref="form"
      :project-id="projectId"
      :environment-id="environmentId"
      :instance-id="instanceId"
      :backup="backup"
      @dismiss="$emit('dismiss')"
    />
    <template #footer>
      <CreateDatabasePrepButtonGroup :form="form" />
    </template>
  </DrawerContent>
</template>

<script setup lang="ts">
import { PropType, ref } from "vue";
import { DrawerContent } from "@/components/v2";
import { Backup } from "@/types/proto/v1/database_service";
import CreateDatabasePrepButtonGroup from "./CreateDatabasePrepButtonGroup.vue";
import CreateDatabasePrepForm from "./CreateDatabasePrepForm.vue";

defineProps({
  projectId: {
    type: String,
    default: undefined,
  },
  environmentId: {
    type: String,
    default: undefined,
  },
  instanceId: {
    type: String,
    default: undefined,
  },
  backup: {
    type: Object as PropType<Backup>,
    default: undefined,
  },
});

defineEmits<{
  (event: "dismiss"): void;
}>();

const form = ref<InstanceType<typeof CreateDatabasePrepForm>>();
</script>
