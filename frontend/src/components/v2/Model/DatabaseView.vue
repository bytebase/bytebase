<template>
  <div
    v-if="database && instance"
    class="flex flex-row justify-start items-center"
  >
    <InstanceV1EngineIcon :instance="instance" />
    <span class="text-sm ml-0.5 text-gray-500">{{
      database.databaseName
    }}</span>
  </div>
</template>

<script lang="ts" setup>
import { useDatabaseV1Store, useInstanceV1Store } from "@/store";
import { ComposedDatabase } from "@/types";
import { onMounted, ref } from "vue";
import { InstanceV1EngineIcon } from "./Instance";
import { Instance } from "@/types/proto/v1/instance_service";

const props = defineProps<{
  databaseName: string;
}>();

const databaseStore = useDatabaseV1Store();
const instanceStore = useInstanceV1Store();
const database = ref<ComposedDatabase>();
const instance = ref<Instance>();

onMounted(async () => {
  database.value = await databaseStore.getOrFetchDatabaseByName(
    props.databaseName
  );
  instance.value = await instanceStore.getOrFetchInstanceByName(
    database.value.instance
  );
});
</script>
