<template>
  <DrawerContent :title="$t('quick-action.create-db')">
    <CreateDatabasePrepForm
      ref="form"
      :project-name="projectName"
      :environment-name="environmentName"
      :instance-name="instanceName"
      @dismiss="$emit('dismiss')"
    />
    <template #footer>
      <CreateDatabasePrepButtonGroup :form="form" />
    </template>
  </DrawerContent>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { DrawerContent } from "@/components/v2";
import CreateDatabasePrepButtonGroup from "./CreateDatabasePrepButtonGroup.vue";
import CreateDatabasePrepForm from "./CreateDatabasePrepForm.vue";

defineProps<{
  projectName?: string;
  environmentName?: string;
  instanceName?: string;
}>();

defineEmits<{
  (event: "dismiss"): void;
}>();

const form = ref<InstanceType<typeof CreateDatabasePrepForm>>();
</script>
