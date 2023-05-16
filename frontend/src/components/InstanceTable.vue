<template>
  <BBGrid
    :column-list="columnList"
    :data-source="instanceList"
    class="mt-2 border-y"
    @click-row="clickInstance"
  >
    <template #item="{ item: instance }">
      <div class="bb-grid-cell justify-center !px-2">
        <InstanceEngineIcon :instance="instance" />
      </div>
      <div class="bb-grid-cell">
        {{ instanceName(instance) }}
      </div>
      <div class="bb-grid-cell">
        <EnvironmentName :environment="instance.environment" :link="false" />
      </div>
      <div class="bb-grid-cell">
        <template v-if="instance.port"
          >{{ instance.host }}:{{ instance.port }}</template
        ><template v-else>{{ instance.host }}</template>
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

<script lang="ts">
import { PropType, defineComponent, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { urlfy, instanceSlug } from "@/utils";
import { Instance } from "@/types";
import InstanceEngineIcon from "./InstanceEngineIcon.vue";
import { BBGridColumn } from "@/bbkit";
import { EnvironmentName } from "@/components/v2";

export default defineComponent({
  name: "InstanceTable",
  components: { InstanceEngineIcon, EnvironmentName },
  props: {
    instanceList: {
      required: true,
      type: Object as PropType<Instance[]>,
    },
  },
  setup(props) {
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
      instance: Instance,
      section: number,
      row: number,
      e: MouseEvent
    ) => {
      const url = `/instance/${instanceSlug(instance)}`;
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        router.push(url);
      }
    };

    return {
      columnList,
      urlfy,
      clickInstance,
    };
  },
});
</script>
