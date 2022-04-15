<template>
  <BBTable
    :column-list="state.columnList"
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
        {{ environmentNameFromId(instance.environment.id) }}
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
import { reactive, PropType, defineComponent } from "vue";
import { useRouter } from "vue-router";
import { BBTableColumn } from "../bbkit/types";
import InstanceEngineIcon from "./InstanceEngineIcon.vue";
import { urlfy, instanceSlug } from "../utils";
import { EnvironmentId, Instance } from "../types";
import { useI18n } from "vue-i18n";
import { useEnvironmentStore } from "@/store";

interface LocalState {
  columnList: BBTableColumn[];
  dataSource: any[];
}

export default defineComponent({
  name: "InstanceTable",
  components: { InstanceEngineIcon },
  props: {
    instanceList: {
      required: true,
      type: Object as PropType<Instance[]>,
    },
  },
  setup(props) {
    const { t } = useI18n();

    const state = reactive<LocalState>({
      columnList: [
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
      ],
      dataSource: [],
    });

    const router = useRouter();
    const envStore = useEnvironmentStore();

    const clickInstance = function (section: number, row: number) {
      const instance = props.instanceList[row];
      router.push(`/instance/${instanceSlug(instance)}`);
    };

    const environmentNameFromId = function (id: EnvironmentId) {
      return envStore.getEnvironmentNameById(id);
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
      state,
      urlfy,
      clickInstance,
      environmentNameFromId,
      instanceLink,
    };
  },
});
</script>
