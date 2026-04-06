<template>
  <Drawer :show="show" :auto-focus="true" @close="$emit('dismiss')">
    <DataExportPrepForm
      :show="show"
      :project-name="projectName"
      :seed="seed"
      @dismiss="$emit('dismiss')"
    />
  </Drawer>
</template>

<script lang="ts" setup>
import { Drawer } from "@/components/v2";
import DataExportPrepForm from "./DataExportPrepForm.vue";
import type { DataExportPrepSeed } from "./types";

defineProps<{
  show: boolean;
  projectName: string;
  seed?: DataExportPrepSeed;
}>();

defineEmits<{
  (event: "dismiss"): void;
}>();
</script>
