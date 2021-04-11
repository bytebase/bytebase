<template>
  <div
    v-if="instance.rowStatus == 'ARCHIVED'"
    class="h-10 w-full text-2xl font-bold bg-gray-700 text-white flex justify-center items-center"
  >
    Archived
  </div>
  <div class="px-6 space-y-6">
    <InstanceForm :create="false" :instance="instance" />
    <div class="py-6 space-y-4 border-t divide-control-border">
      <DataSourceTable :instance="instance" />
    </div>
  </div>
</template>

<script lang="ts">
import { computed } from "vue";
import { useStore } from "vuex";
import { idFromSlug } from "../utils";
import DataSourceTable from "../components/DataSourceTable.vue";
import InstanceForm from "../components/InstanceForm.vue";

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

    const instance = computed(() => {
      return store.getters["instance/instanceById"](
        idFromSlug(props.instanceSlug)
      );
    });

    return {
      instance,
    };
  },
};
</script>
