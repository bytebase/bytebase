<template>
  <div>
    <h2 class="text-gray-500 text-xs font-medium uppercase tracking-wide">
      Pinned Projects
    </h2>
    <ul class="grid grid-cols-1 gap-4 grid-cols-4 mt-3">
      <li class="relative col-span-1 flex shadow-sm rounded-md">
        <PinnedProjectCard />
      </li>

      <!-- More project cards... -->
    </ul>
  </div>
</template>

<script lang="ts">
import PinnedProjectCard from "./PinnedProjectCard.vue";

export default {
  name: "PinnedProject",
  components: {
    PinnedProjectCard,
  },
};
</script>
