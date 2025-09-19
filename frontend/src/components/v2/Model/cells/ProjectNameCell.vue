<template>
  <div class="flex flex-row space-x-2 items-center">
    <ProjectV1Name
      :project="project"
      :link="link"
      tag="div"
      :keyword="keyword"
    />

    <slot name="suffix">
      {{ suffix }}
    </slot>

    <NTooltip v-if="project.state === State.DELETED">
      <template #trigger>
        <heroicons-outline:archive class="w-4 h-4 text-control" />
      </template>
      <span class="whitespace-nowrap">
        {{ $t("common.archived") }}
      </span>
    </NTooltip>
  </div>
</template>

<script setup lang="ts">
import { NTooltip } from "naive-ui";
import { ProjectV1Name } from "@/components/v2";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Mode } from "../DatabaseV1Table/DatabaseV1Table.vue";

withDefaults(
  defineProps<{
    project: Project;
    mode?: Mode;
    link?: boolean;
    keyword?: string;
    suffix?: string;
  }>(),
  {
    mode: "ALL",
    link: false,
    keyword: undefined,
    suffix: "",
  }
);
</script>
