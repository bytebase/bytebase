<template>
  <div class="flex flex-row justify-start items-center gap-x-1">
    <div
      v-if="showInstance || showEngineIcon"
      class="flex flex-row items-center gap-x-1"
    >
      <InstanceV1EngineIcon
        v-if="showEngineIcon"
        :instance="database.instanceEntity"
      />
      <InstanceV1Name
        v-if="showInstance"
        :instance="database.instanceEntity"
        :icon="false"
        :link="false"
      />
    </div>

    <heroicons:chevron-right
      v-if="(showInstance || showEngineIcon) && showArrow"
      class="text-control-light"
    />

    <div class="flex flex-row items-center gap-x-1">
      <EnvironmentV1Name
        v-if="showEnvironment"
        :environment="database.effectiveEnvironmentEntity"
        :link="false"
        :show-icon="showProductionEnvironmentIcon"
        text-class="text-control-light"
      />
      <DatabaseV1Name :database="database" :link="false" :show-icon="false" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ComposedDatabase } from "@/types";
import DatabaseV1Name from "./DatabaseV1Name.vue";
import EnvironmentV1Name from "./EnvironmentV1Name.vue";
import { InstanceV1EngineIcon, InstanceV1Name } from "./Instance";

withDefaults(
  defineProps<{
    database: ComposedDatabase;
    showEngineIcon?: boolean;
    showInstance?: boolean;
    showArrow?: boolean;
    showEnvironment?: boolean;
    showProductionEnvironmentIcon?: boolean;
  }>(),
  {
    showEngineIcon: true,
    showInstance: true,
    showArrow: true,
    showEnvironment: true,
    showProductionEnvironmentIcon: true,
  }
);
</script>
