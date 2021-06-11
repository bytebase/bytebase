<template></template>

<script lang="ts">
import { reactive, computed, watchEffect } from "vue";
import { useStore } from "vuex";
import { idFromSlug } from "../utils";
import { VCS } from "../types";

interface LocalState {}

export default {
  name: "SettingWorkspaceVCSDetail",
  props: {
    vcsSlug: {
      required: true,
      type: String,
    },
  },
  components: {},
  setup(props, ctx) {
    const store = useStore();
    const state = reactive<LocalState>({});

    const vcs = computed((): VCS => {
      return store.getters["vcs/vcsById"](idFromSlug(props.vcsSlug));
    });

    const prepareRepositoryList = () => {
      store.dispatch("vcs/fetchRepositoryListByVCS", vcs.value);
    };

    watchEffect(prepareRepositoryList);

    const repositoryList = computed(() =>
      store.getters["vcs/repositoryListByVCSId"](vcs.value.id)
    );

    return {
      state,
      repositoryList,
    };
  },
};
</script>
