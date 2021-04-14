<template>
  <div class="py-2">
    <ArchiveBanner v-if="instance.rowStatus == 'ARCHIVED'" />
  </div>
  <div class="px-6 space-y-6">
    <InstanceForm :create="false" :instance="instance" />
    <div
      v-if="hasDataSourceFeature"
      class="py-6 space-y-4 border-t divide-control-border"
    >
      <DataSourceTable :instance="instance" />
    </div>
    <div v-else>
      <div class="text-lg leading-6 font-medium text-main mb-4">Databases</div>
      <DatabaseTable :singleInstance="true" :databaseList="databaseList" />
    </div>
  </div>
</template>

<script lang="ts">
import { computed } from "vue";
import { useStore } from "vuex";
import { idFromSlug } from "../utils";
import ArchiveBanner from "../components/ArchiveBanner.vue";
import DatabaseTable from "../components/DatabaseTable.vue";
import DataSourceTable from "../components/DataSourceTable.vue";
import InstanceForm from "../components/InstanceForm.vue";
import { Instance } from "../types";

export default {
  name: "InstanceDetail",
  components: {
    ArchiveBanner,
    DatabaseTable,
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

    const hasDataSourceFeature = computed(() =>
      store.getters["plan/feature"]("bytebase.data-source")
    );

    const instance = computed(
      (): Instance => {
        return store.getters["instance/instanceById"](
          idFromSlug(props.instanceSlug)
        );
      }
    );

    const databaseList = computed(() => {
      return store.getters["database/databaseListByInstanceId"](
        instance.value.id
      );
    });

    return {
      hasDataSourceFeature,
      instance,
      databaseList,
    };
  },
};
</script>
