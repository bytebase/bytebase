<template>
  <div class="flex items-center flex-wrap gap-1">
    <InstanceV1Name
      :instance="instanceResource"
      :plain="true"
      :link="link"
      :text-class="link ? 'hover:underline' : ''"
    >
      <template
        v-if="
          instanceResource.environment !==
          database.effectiveEnvironment
        "
        #prefix
      >
        <EnvironmentV1Name
          :environment="instanceEnvironment"
          :plain="true"
          :show-icon="false"
          :link="link"
          :text-class="`text-control-light ${link ? 'hover:underline' : ''}`"
          :null-environment-placeholder="'Null'"
        />
      </template>
    </InstanceV1Name>

    <heroicons-outline:chevron-right class="text-control-light" />

    <div class="flex items-center gap-x-1">
      <heroicons-outline:database />

      <template v-if="database">
        <EnvironmentV1Name
          :environment="effectiveEnvironment"
          :plain="true"
          :show-icon="false"
          :link="link"
          :text-class="`text-control-light ${link ? 'hover:underline' : ''}`"
          :null-environment-placeholder="'Null'"
        />

        <DatabaseV1Name
          :database="database"
          :plain="true"
          :link="link"
          :text-class="link ? 'hover:underline' : ''"
        />
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import {
  DatabaseV1Name,
  EnvironmentV1Name,
  InstanceV1Name,
} from "@/components/v2";
import { useEnvironmentV1Store } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { getDatabaseEnvironment, getInstanceResource } from "@/utils";

const props = withDefaults(
  defineProps<{
    database: Database;
    link?: boolean;
  }>(),
  {
    link: true,
  }
);

const instanceResource = computed(() => getInstanceResource(props.database));
const effectiveEnvironment = computed(() =>
  getDatabaseEnvironment(props.database)
);

const instanceEnvironment = computed(() => {
  return useEnvironmentV1Store().getEnvironmentByName(
    instanceResource.value.environment ?? ""
  );
});
</script>
