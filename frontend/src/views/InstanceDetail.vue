<template>
  <div class="px-6 space-y-6">
    <InstanceForm :create="false" :instance="state.instance" />
    <div class="py-6 space-y-4 border-t divide-control-border">
      <DataSourceTable :instance="state.instance" />
    </div>
  </div>
</template>

<script lang="ts">
import { reactive } from "vue";
import { useStore } from "vuex";
import { idFromSlug } from "../utils";
import DataSourceTable from "../components/DataSourceTable.vue";
import InstanceForm from "../components/InstanceForm.vue";

interface LocalState {}

export default {
  name: "InstanceDetail",
  components: {
    DataSourceTable,
    InstanceForm,
  },
  props: {
    instanceSlug: {
      required: true,
      type: String,
    },
  },
  setup(props, { emit }) {
    const store = useStore();

    const state = reactive<LocalState>({
      // Instance is already fetched remotely during routing, so we can just
      // use store.getters here.
      instance: store.getters["instance/instanceById"](
        idFromSlug(props.instanceSlug)
      ),
    });

    return {
      state,
    };
  },
};
</script>
