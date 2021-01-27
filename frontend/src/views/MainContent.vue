<template>
  <div class="bg-normal py-4 sm:px-6">
    <div class="flex -ml-4 justify-between flex-wrap sm:flex-no-wrap">
      <!-- Breadcrumb section -->
      <div class="ml-4 mt-2" aria-label="Breadcrumb">
        <Breadcrumb />
      </div>
      <div class="ml-4 mt-2 flex-shrink-0">
        <span class="inline-flex rounded-md shadow-sm">
          <router-view name="actionbar"></router-view>
        </span>
      </div>
    </div>
  </div>
  <div class="mt-2">
    <router-view name="content"></router-view>
  </div>
</template>

<script lang="ts">
import Breadcrumb from "../components/Breadcrumb.vue";

export default {
  name: "MainContent",
  components: {
    Breadcrumb,
  },
  props: {},
  setup(props, ctx) {},
};
</script>
