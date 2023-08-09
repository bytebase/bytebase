<template>
  <div class="space-y-4">
    <div class="textinfolabel">
      {{ $t("gitops.setting.description") }}
      <a
        class="text-accent hover:opacity-80"
        href="https://www.bytebase.com/docs/administration/sso/overview?source=console"
        >{{ $t("gitops.setting.description-highlight") }}</a
      >
    </div>
    <div v-if="vcsList.length > 0" class="flex items-center justify-end">
      <button
        type="button"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        @click.prevent="addVCSProvider"
      >
        {{ $t("gitops.setting.add-git-provider.self") }}
      </button>
    </div>
    <div class="pt-4 border-t">
      <div v-if="vcsList.length > 0" class="space-y-6">
        <template v-for="(vcs, index) in vcsList" :key="index">
          <VCSCard :vcs="vcs" />
        </template>
      </div>
      <template v-else>
        <VCSSetupWizard />
      </template>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, computed, watchEffect, defineComponent } from "vue";
import { useRouter } from "vue-router";
import { useVCSV1Store } from "@/store";
import VCSCard from "../components/VCSCard.vue";
import VCSSetupWizard from "../components/VCSSetupWizard.vue";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

export default defineComponent({
  name: "SettingWorkspaceVCS",
  components: {
    VCSCard,
    VCSSetupWizard,
  },
  setup() {
    const vcsV1Store = useVCSV1Store();
    const router = useRouter();
    const state = reactive<LocalState>({});

    const prepareVCSList = () => {
      vcsV1Store.fetchVCSList();
    };

    watchEffect(prepareVCSList);

    const vcsList = computed(() => {
      return vcsV1Store.getVCSList();
    });

    const addVCSProvider = () => {
      router.push({
        name: "setting.workspace.gitops.create",
      });
    };

    return {
      state,
      vcsList,
      addVCSProvider,
    };
  },
});
</script>
