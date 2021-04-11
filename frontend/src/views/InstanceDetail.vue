<template>
  <div class="py-2">
    <ArchiveBanner v-if="instance.rowStatus == 'ARCHIVED'" />
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
import ArchiveBanner from "../components/ArchiveBanner.vue";
import DataSourceTable from "../components/DataSourceTable.vue";
import InstanceForm from "../components/InstanceForm.vue";

export default {
  name: "InstanceDetail",
  components: {
    ArchiveBanner,
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
