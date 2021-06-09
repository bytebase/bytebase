<template>
  <div v-for="(vcs, index) in vcsList" :key="index">
    <button
      type="button"
      class="btn-normal items-center space-x-2"
      @click.prevent="selectVCS(vcs)"
    >
      <template v-if="vcs.type.startsWith('GITLAB')">
        <img class="h-6 w-auto" src="../assets/gitlab-logo.svg" />
      </template>
      <span>{{ vcs.name }}</span>
    </button>
  </div>
</template>

<script lang="ts">
import { reactive } from "@vue/reactivity";
import { useStore } from "vuex";
import { computed, PropType, watchEffect } from "@vue/runtime-core";
import { ProjectRepoConfig, VCS } from "../types";

interface LocalState {}

export default {
  name: "RepositoryVCSPanel",
  emits: ["select-vcs"],
  props: {
    config: {
      required: true,
      type: Object as PropType<ProjectRepoConfig>,
    },
  },
  components: {},
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({});

    const prepareVCSList = () => {
      store.dispatch("vcs/fetchVCSList");
    };

    watchEffect(prepareVCSList);

    const vcsList = computed(() => {
      return store.getters["vcs/vcsList"]();
    });

    const selectVCS = (vcs: VCS) => {
      emit("select-vcs", vcs);
    };

    return {
      state,
      vcsList,
      selectVCS,
    };
  },
};
</script>
