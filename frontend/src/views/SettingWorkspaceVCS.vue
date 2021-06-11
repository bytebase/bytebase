<template>
  <div class="space-y-4 divide-y divide-block-border">
    <div class="flex items-center justify-end">
      <button
        type="button"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        @click.prevent="addVCSProvider"
      >
        Add a Git provider
      </button>
    </div>
    <div class="pt-4">
      <template v-if="vcsList.length > 0">
        <template v-for="(vcs, index) in vcsList" :key="index">
          <VCSCard :vcs="vcs" />
        </template>
      </template>
      <template v-else>
        <VCSSetupWizard />
      </template>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, watchEffect } from "@vue/runtime-core";
import { reactive } from "@vue/reactivity";
import { useRouter } from "vue-router";
import VCSCard from "../components/VCSCard.vue";
import VCSSetupWizard from "../components/VCSSetupWizard.vue";
import { useStore } from "vuex";

interface LocalState {}

export default {
  name: "SettingWorkspaceVCS",
  props: {},
  components: {
    VCSCard,
    VCSSetupWizard,
  },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();
    const state = reactive<LocalState>({});

    const prepareVCSList = () => {
      store.dispatch("vcs/fetchVCSList");
    };

    watchEffect(prepareVCSList);

    const vcsList = computed(() => {
      return store.getters["vcs/vcsList"]();
    });

    const addVCSProvider = () => {
      router.push({
        name: "setting.workspace.version-control.create",
      });
    };

    return {
      state,
      vcsList,
      addVCSProvider,
    };
  },
};
</script>
