<template>
  <BBTable
    :column-list="columnList"
    :data-source="instanceList"
    :show-header="true"
    :left-bordered="false"
    :right-bordered="false"
    @click-row="clickInstance"
  >
    <template #body="{ rowData: instance }">
      <BBTableCell :left-padding="4" class="w-4">
        <InstanceEngineIcon :instance="instance" />
      </BBTableCell>
      <BBTableCell class="w-32">
        {{ instanceName(instance) }}
      </BBTableCell>
      <BBTableCell class="w-16">
        <div class="flex items-center gap-x-1">
          {{ environmentNameFromId(instance.environment.id) }}
          <ProtectedEnvironmentIcon :environment="instance.environment" />
        </div>
      </BBTableCell>
      <BBTableCell class="w-48">
        <template v-if="instance.port"
          >{{ instance.host }}:{{ instance.port }}</template
        ><template v-else>{{ instance.host }}</template>
      </BBTableCell>
      <BBTableCell class="w-4">
        <button
          v-if="instance.externalLink?.trim().length != 0"
          class="btn-icon"
          @click.stop="window.open(urlfy(instanceLink(instance)), '_blank')"
        >
          <heroicons-outline:external-link class="w-4 h-4" />
        </button>
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(instance.createdTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType, defineComponent, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { urlfy, instanceSlug, environmentName } from "@/utils";
import { EnvironmentId, Instance } from "@/types";
import { useEnvironmentStore } from "@/store";
import InstanceEngineIcon from "./InstanceEngineIcon.vue";
import ProtectedEnvironmentIcon from "./Environment/ProtectedEnvironmentIcon.vue";

export default defineComponent({
  name: "InstanceTable",
  components: { InstanceEngineIcon, ProtectedEnvironmentIcon },
  props: {
    instanceList: {
      required: true,
      type: Object as PropType<Instance[]>,
    },
  },
  setup(props) {
    const { t } = useI18n();

    const router = useRouter();

    const columnList = computed(() => {
      return [
        {
          title: "",
        },
        {
          title: t("common.name"),
        },
        {
          title: t("common.environment"),
        },
        {
          title: t("common.Address"),
        },
        {
          title: t("instance.external-link"),
        },
        {
          title: t("common.created-at"),
        },
      ];
    });

    const clickInstance = (section: number, row: number, e: MouseEvent) => {
      const instance = props.instanceList[row];
      const url = `/instance/${instanceSlug(instance)}`;
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        router.push(url);
      }
    };

    const environmentNameFromId = (id: EnvironmentId) => {
      const env = useEnvironmentStore().getEnvironmentById(id);
      return environmentName(env);
    };

    const instanceLink = (instance: Instance): string => {
      if (instance.engine == "SNOWFLAKE") {
        if (instance.host) {
          return `https://${
            instance.host.split("@")[0]
          }.snowflakecomputing.com/console`;
        }
      }
      return instance.host;
    };

    return {
      columnList,
      urlfy,
      clickInstance,
      environmentNameFromId,
      instanceLink,
    };
  },
});
</script>
