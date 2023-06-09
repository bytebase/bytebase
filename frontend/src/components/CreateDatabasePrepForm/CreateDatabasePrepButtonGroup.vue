<template>
  <div v-if="form" class="flex justify-end gap-x-3">
    <NButton @click.prevent="form?.cancel">
      {{ $t("common.cancel") }}
    </NButton>
    <NButton
      type="primary"
      :disabled="!form?.allowCreate"
      @click.prevent="form?.create"
    >
      {{ $t("common.create") }}
    </NButton>
  </div>
</template>

<script setup lang="ts">
import CreateDatabasePrepForm from "./CreateDatabasePrepForm.vue";

defineProps<{
  form?: InstanceType<typeof CreateDatabasePrepForm>;
}>();
</script>
