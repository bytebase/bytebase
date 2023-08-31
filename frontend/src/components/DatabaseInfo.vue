<template>
  <div class="flex items-center flex-wrap gap-1" @click.stop.prevent>
    <InstanceV1Name
      :instance="database.instanceEntity"
      :plain="true"
      text-class="hover:underline"
    >
      <template
        v-if="
          database.instanceEntity.environment !== database.effectiveEnvironment
        "
        #prefix
      >
        <EnvironmentV1Name
          :environment="database.instanceEntity.environmentEntity"
          :plain="true"
          :show-icon="false"
          text-class="hover:underline text-control-light"
        />
      </template>
    </InstanceV1Name>

    <heroicons-outline:chevron-right class="text-control-light" />

    <div class="flex items-center gap-x-1">
      <heroicons-outline:database />

      <template v-if="database">
        <EnvironmentV1Name
          :environment="database.effectiveEnvironmentEntity"
          :plain="true"
          :show-icon="false"
          text-class="hover:underline text-control-light"
        />

        <DatabaseV1Name
          :database="database"
          :plain="true"
          class="hover:underline"
        />
      </template>

      <SQLEditorButtonV1 v-if="showSQLEditorButton" :database="database" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { SQLEditorButtonV1 } from "@/components/DatabaseDetail";
import { DatabaseV1Name, InstanceV1Name } from "@/components/v2";
import { ComposedDatabase } from "@/types";

defineProps<{
  database: ComposedDatabase;
  showSQLEditorButton?: boolean;
}>();
</script>
