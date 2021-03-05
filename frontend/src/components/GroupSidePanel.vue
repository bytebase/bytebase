<template>
  <div class="mt-1 space-y-1" role="group" aria-labelledby="groups-headline">
    <button
      @click.prevent="toggleExpand"
      class="outline-item mt-1 group w-full flex items-center pr-2 py-2"
    >
      <svg
        v-if="expandState"
        class="mr-2 h-5 w-5 transform rotate-90 group-hover:text-control-light-hover group-focus:text-control-light-hover transition-colors ease-in-out duration-150"
        viewBox="0 0 20 20"
      >
        <path d="M6 6L14 10L6 14V6Z" fill="currentColor" />
      </svg>
      <svg
        v-else
        class="mr-2 h-5 w-5 transform group-hover:text-control-light-hover group-focus:text-control-light-hover transition-colors ease-in-out duration-150"
        viewBox="0 0 20 20"
      >
        <path d="M6 6L14 10L6 14V6Z" fill="currentColor" />
      </svg>
      {{ group.attributes.name }}
    </button>
    <!-- Expandable link section, show/hide based on state. -->
    <div v-if="expandState" class="mt-1 space-y-1">
      <router-link
        :to="`/${group.attributes.slug}`"
        class="outline-item group w-full flex items-center pl-10 pr-2 py-1"
      >
        Dashboard
      </router-link>
      <router-link
        :to="`/${group.attributes.slug}/setting`"
        class="outline-item group w-full flex items-center pl-10 pr-2 py-1"
      >
        Setting
      </router-link>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, PropType } from "vue";
import { useStore } from "vuex";
import { Group, GroupId } from "../types";

export default {
  name: "GroupSidePanel",
  props: {
    group: {
      required: true,
      type: Object as PropType<Group>,
    },
  },
  setup(props, ctx) {
    const store = useStore();

    const groupId: GroupId = props.group.id;

    const expandState = computed(() =>
      store.getters["uistate/expandStateByGroup"](groupId)
    );

    const toggleExpand = () => {
      const newState = !expandState.value;
      store
        .dispatch("uistate/saveExpandStateByGroup", {
          groupId,
          expand: newState,
        })
        .catch((error) => {
          console.log(error);
          return;
        });
    };

    return {
      expandState,
      toggleExpand,
    };
  },
};
</script>
