<template>
  <NPopover :disabled="!tooltip">
    <template #trigger>
      <div class="flex flex-row justify-start items-center">
        <div v-if="showProject">
          <ProjectV1Name :project="database.projectEntity" :link="false" />
        </div>

        <heroicons:chevron-right
          v-if="showProject && showArrow"
          class="text-control-light"
        />

        <div
          v-if="showInstance || showEngineIcon"
          class="flex flex-row items-center gap-x-1"
        >
          <InstanceV1EngineIcon
            v-if="showEngineIcon"
            :instance="database.instanceResource"
          />
          <InstanceV1Name
            v-if="showInstance"
            :instance="database.instanceResource"
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
          <DatabaseV1Name
            :database="database"
            :link="false"
            :show-icon="false"
            :keyword="keyword"
          />
        </div>
      </div>
    </template>
    <template #default>
      <InstanceV1Name
        v-if="tooltip === 'instance'"
        :instance="database.instanceResource"
        :link="false"
      />
    </template>
  </NPopover>
</template>

<script setup lang="ts">
import { NPopover } from "naive-ui";
import type { ComposedDatabase } from "@/types";
import DatabaseV1Name from "./DatabaseV1Name.vue";
import EnvironmentV1Name from "./EnvironmentV1Name.vue";
import { InstanceV1EngineIcon, InstanceV1Name } from "./Instance";
import ProjectV1Name from "./ProjectV1Name.vue";

withDefaults(
  defineProps<{
    database: ComposedDatabase;
    showProject?: boolean;
    showEngineIcon?: boolean;
    showInstance?: boolean;
    showArrow?: boolean;
    showEnvironment?: boolean;
    showProductionEnvironmentIcon?: boolean;
    tooltip?: "instance" | undefined;
    keyword?: string;
  }>(),
  {
    showProject: false,
    showEngineIcon: true,
    showInstance: true,
    showArrow: true,
    showEnvironment: true,
    showProductionEnvironmentIcon: true,
    tooltip: undefined,
    keyword: "",
  }
);
</script>
