<template>
  <div>
    <div class="flex flex-col items-center justify-center md:flex-row">
      <h1 class="text-xl md:text-3xl font-semibold flex-1">{{ guide.name }}</h1>

      <button type="button" class="btn-cancel mr-4">Remove</button>
      <button type="button" class="btn-primary">Edit</button>
    </div>
    <div class="flex flex-wrap gap-x-3 my-5">
      <span>Environments:</span>
      <BBBadge
        v-for="envId in guide.environmentList"
        :key="envId"
        :text="environmentNameFromId(envId)"
        :can-remove="false"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { idFromSlug, environmentName } from "../utils";
import { Environment, EnvironmentId, DatabaseSchemaGuide } from "../types";
import { useEnvironmentStore, useSchemaSystemStore } from "@/store";

const props = defineProps({
  schemaGuideSlug: {
    required: true,
    type: String,
  },
});

const store = useSchemaSystemStore();

const environmentNameFromId = function (id: EnvironmentId) {
  const env = useEnvironmentStore().getEnvironmentById(id);

  return environmentName(env);
};

const guide = computed((): DatabaseSchemaGuide => {
  return store.getGuideById(idFromSlug(props.schemaGuideSlug));
});
</script>
