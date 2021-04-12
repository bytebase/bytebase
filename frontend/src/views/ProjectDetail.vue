<template>
  <div class="py-2">
    <ArchiveBanner v-if="project.rowStatus == 'ARCHIVED'" />
  </div>
  <div class="px-6 space-y-6">
    {{ project.name }}
    <!-- <div class="py-6 space-y-4 border-t divide-control-border">
      <DatabaseTable :instance="instance" />
    </div> -->
  </div>
</template>

<script lang="ts">
import { computed } from "vue";
import { useStore } from "vuex";
import { idFromSlug } from "../utils";
import ArchiveBanner from "../components/ArchiveBanner.vue";
import DatabaseTable from "../components/DatabaseTable.vue";

export default {
  name: "ProjectDetail",
  components: {
    ArchiveBanner,
    DatabaseTable,
  },
  props: {
    projectSlug: {
      required: true,
      type: String,
    },
  },
  setup(props, { emit }) {
    const store = useStore();

    const project = computed(() => {
      return store.getters["project/projectById"](
        idFromSlug(props.projectSlug)
      );
    });

    return {
      project,
    };
  },
};
</script>
