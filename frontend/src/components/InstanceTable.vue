<template>
  <BBTable
    :columnList="state.columnList"
    :dataSource="instanceList"
    :showHeader="true"
    @click-row="clickInstance"
  >
    <template v-slot:body="{ rowData: instance }">
      <BBTableCell :leftPadding="4" class="w-24">
        {{ environmentName(instance.attributes.environmentId) }}
      </BBTableCell>
      <BBTableCell class="w-48">
        {{ instance.attributes.name }}
      </BBTableCell>
      <BBTableCell class="w-64">
        <template v-if="instance.attributes.port"
          >{{ instance.attributes.host }}:{{
            instance.attributes.port
          }}</template
        ><template v-else>{{ instance.attributes.host }}</template>
      </BBTableCell>
      <BBTableCell class="w-2">
        <button
          v-if="instance.attributes.externalLink?.trim().length != 0"
          class="btn-icon"
          @click.stop="
            window.open(urlfy(instance.attributes.externalLink), '_blank')
          "
        >
          <svg
            class="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
            ></path>
          </svg>
        </button>
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { reactive, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { BBTableColumn } from "../bbkit/types";
import { humanize, urlfy, instanceSlug } from "../utils";
import { EnvironmentId, Instance } from "../types";

interface LocalState {
  columnList: BBTableColumn[];
  dataSource: Object[];
}

export default {
  name: "InstanceTable",
  components: {},
  props: {
    instanceList: {
      required: true,
      type: Object as PropType<Instance[]>,
    },
  },
  setup(props, ctx) {
    const store = useStore();

    const state = reactive<LocalState>({
      columnList: [
        {
          title: "Environment",
        },
        {
          title: "Name",
        },
        {
          title: "Host:Port",
        },
        {
          title: "Link",
        },
      ],
      dataSource: [],
    });

    const router = useRouter();

    const clickInstance = function (section: number, row: number) {
      const instance = props.instanceList[row];
      router.push(
        `/instance/${instanceSlug(
          environmentName(instance.attributes.environmentId),
          instance.attributes.name,
          instance.id
        )}`
      );
    };

    const environmentName = function (id: EnvironmentId) {
      return store.getters["environment/environmentById"](id)?.attributes.name;
    };

    return {
      state,
      humanize,
      urlfy,
      clickInstance,
      environmentName,
    };
  },
};
</script>
