<template>
  <div class="flex flex-row gap-x-2 items-center">
    <ProjectV1Name
      :project="project"
      :link="link"
      tag="div"
      :keyword="keyword"
    />

    <NTag v-if="project.state === State.DELETED" size="small" type="default">
      {{ $t("common.archived") }}
    </NTag>

    <slot name="suffix">
      {{ suffix }}
    </slot>
  </div>
</template>

<script setup lang="ts">
import { NTag } from "naive-ui";
import { ProjectV1Name } from "@/components/v2";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

withDefaults(
  defineProps<{
    project: Project;
    link?: boolean;
    keyword?: string;
    suffix?: string;
  }>(),
  {
    link: false,
    keyword: undefined,
    suffix: "",
  }
);
</script>
