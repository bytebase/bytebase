<template>
  <BBGrid
    :column-list="columnList"
    :data-source="instanceList"
    class="mt-2 border-y"
    @click-row="clickInstance"
  >
    <template #item="{ item: instance }: InstanceRow">
      <div class="bb-grid-cell justify-center !px-2">
        <InstanceV1EngineIcon :instance="instance" />
      </div>
      <div class="bb-grid-cell">
        {{ instanceV1Name(instance) }}
      </div>
      <div class="bb-grid-cell">
        <EnvironmentV1Name
          :environment="instance.environmentEntity"
          :link="false"
        />
      </div>
      <div class="bb-grid-cell">
        {{ hostPortOfInstanceV1(instance) }}
      </div>
      <div class="bb-grid-cell hidden sm:flex">
        <button
          v-if="instance.externalLink?.trim().length != 0"
          class="btn-icon"
          @click.stop="window.open(urlfy(instance.externalLink), '_blank')"
        >
          <heroicons-outline:external-link class="w-4 h-4" />
        </button>
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";

import {
  urlfy,
  instanceV1Slug,
  instanceV1Name,
  hostPortOfInstanceV1,
} from "@/utils";
import { ComposedInstance } from "@/types";
import InstanceV1EngineIcon from "../InstanceV1EngineIcon.vue";
import EnvironmentV1Name from "../../EnvironmentV1Name.vue";
import { BBGrid, BBGridColumn, BBGridRow } from "@/bbkit";

export type InstanceRow = BBGridRow<ComposedInstance>;

defineProps<{
  instanceList: ComposedInstance[];
}>();

const { t } = useI18n();

const router = useRouter();

const columnList = computed((): BBGridColumn[] => {
  return [
    {
      title: "",
      width: "minmax(auto, 4rem)",
    },
    {
      title: t("common.name"),
      width: "minmax(auto, 3fr)",
    },
    {
      title: t("common.environment"),
      width: "minmax(auto, 1fr)",
    },
    {
      title: t("common.Address"),
      width: "minmax(auto, 2fr)",
    },
    {
      title: t("instance.external-link"),
      width: { sm: "1fr" },
      class: "hidden sm:flex",
    },
  ];
});

const clickInstance = (
  instance: ComposedInstance,
  section: number,
  row: number,
  e: MouseEvent
) => {
  const url = `/instance/${instanceV1Slug(instance)}`;
  if (e.ctrlKey || e.metaKey) {
    window.open(url, "_blank");
  } else {
    router.push(url);
  }
};
</script>
