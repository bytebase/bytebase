<template>
  <!-- Navigation -->
  <nav class="px-2">
    <BookmarkListSidePanel />
    <!-- <div class="mt-4">
      <ProjectListSidePanel />
    </div>
    <div class="mt-4">
      <GroupListSidePanel />
    </div> -->
  </nav>
</template>

<script lang="ts">
import { onMounted, reactive } from "vue";
import { useStore } from "vuex";
import BookmarkListSidePanel from "../components/BookmarkListSidePanel.vue";
// import GroupListSidePanel from "../components/GroupListSidePanel.vue";
// import ProjectListSidePanel from "../components/ProjectListSidePanel.vue";

interface LocalState {
  hasUnreadMessage: boolean;
}

export default {
  name: "DashboardSidebar",
  props: {},
  components: {
    BookmarkListSidePanel,
    // GroupListSidePanel,
    // ProjectListSidePanel,
  },
  setup(props, ctx) {
    const store = useStore();

    const state = reactive<LocalState>({
      hasUnreadMessage: Math.random() > 0.5,
    });

    const restoreExpandState = () => {
      store.dispatch("uistate/restoreExpandState").catch((error) => {
        console.log(error);
      });
    };

    onMounted(restoreExpandState);

    return {
      state,
    };
  },
};
</script>
