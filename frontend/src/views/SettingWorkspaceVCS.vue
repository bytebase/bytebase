<template>
  <div class="space-y-4">
    <div class="textinfolabel">
      {{ $t("version-control.setting.description") }}
      <span class="text-accent">{{
        $t("version-control.setting.description-highlight")
      }}</span>
    </div>
    <div class="flex items-center justify-end">
      <button
        type="button"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        @click.prevent="addVCSProvider"
      >
        {{ $t("version-control.setting.add-git-provider.self") }}
      </button>
    </div>
    <div class="pt-4 border-t">
      <FeatureAttention
        v-if="!has3rdPartyLoginFeature && vcsList.length > 0"
        custom-class="mb-5"
        feature="bb.feature.3rd-party-auth"
        :description="
          $t('subscription.features.bb-feature-3rd-party-auth.desc')
        "
      />
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
import VCSCard from "../components/VCSCard.vue";
import VCSSetupWizard from "../components/VCSSetupWizard.vue";
import { featureToRef, useVcsStore } from "@/store";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

export default defineComponent({
  name: "SettingWorkspaceVCS",
  components: {
    VCSCard,
    VCSSetupWizard,
  },
  setup() {
    const vcsStore = useVcsStore();
    const router = useRouter();
    const state = reactive<LocalState>({});

    const prepareVCSList = () => {
      vcsStore.fetchVCSList();
    };

    watchEffect(prepareVCSList);

    const vcsList = computed(() => {
      return vcsStore.getVcsList();
    });

    const addVCSProvider = () => {
      router.push({
        name: "setting.workspace.version-control.create",
      });
    };

    const has3rdPartyLoginFeature = featureToRef("bb.feature.3rd-party-auth");

    return {
      state,
      vcsList,
      addVCSProvider,
      has3rdPartyLoginFeature,
    };
  },
});
</script>
