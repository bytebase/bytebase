<template>
  <BBTable
    :columnList="state.columnList"
    :dataSource="instanceList"
    :showHeader="true"
    @click-row="clickInstance"
  >
    <template v-slot:body="{ rowData: instance }">
      <BBTableCell :leftPadding="4" class="w-36">
        {{ environmentName(instance.attributes.environmentId) }}
      </BBTableCell>
      <BBTableCell class="w-96">
        {{ instance.attributes.name }}</span>
      </BBTableCell>
      <BBTableCell class="w-24">
        <template v-if="instance.attributes.port"
          >{{ instance.attributes.host }}:{{
            instance.attributes.port
          }}</template
        ><template v-else>{{ instance.attributes.host }}</template>
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { reactive, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { BBTableColumn } from "../bbkit/types";
import { humanize } from "../utils";
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
      ],
      dataSource: [],
    });

    const router = useRouter();

    const clickInstance = function (section: number, row: number) {
      router.push(`/instance/${props.instanceList[row].id}`);
    };

    const environmentName = function (id: EnvironmentId) {
      return store.getters["environment/environmentById"](id)?.attributes.name;
    }

    return {
      state,
      humanize,
      clickInstance,
      environmentName,
    };
  },
};
</script>
