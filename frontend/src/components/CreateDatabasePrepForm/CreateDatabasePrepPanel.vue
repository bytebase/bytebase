<template>
  <DrawerContent :title="$t('quick-action.create-db')">
    <CreateDatabasePrepForm
      ref="form"
      :project-id="projectId"
      :environment="environment"
      :instance="instance"
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
  projectId?: string;
  environment?: string;
  instance?: string;
}>();

defineEmits<{
  (event: "dismiss"): void;
}>();

const form = ref<InstanceType<typeof CreateDatabasePrepForm>>();
</script>
