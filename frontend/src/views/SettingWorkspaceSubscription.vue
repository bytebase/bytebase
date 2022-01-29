<template>
  <div class="mx-auto">
    <div class="textinfolabel">
      {{ $t("subscription.description") }}
      <span class="text-accent">{{
        $t("subscription.description-highlight")
      }}</span>
    </div>
    <div class="w-full mt-5 flex flex-col">
      <textarea
        id="license"
        v-model="state.license"
        type="text"
        name="license"
        :placeholder="$t('subscription.sensitive-placeholder')"
        class="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full sm:text-sm border-gray-300 rounded-md"
      />
      <button
        type="button"
        :class="[
          disabled ? 'cursor-not-allowed' : '',
          'btn-primary inline-flex justify-center ml-auto mt-3',
        ]"
        target="_blank"
        href="https://hub.bytebase.com"
        @click="uploadLicense"
      >
        {{ $t("subscription.upload-license") }}
      </button>
    </div>
    <div class="sm:flex sm:flex-col sm:align-center pt-5 mt-5 border-t">
      <div class="textinfolabel">
        {{ $t("subscription.plan-compare") }}
      </div>
      <PricingTable />
    </div>
  </div>
</template>

<script lang="ts">
import { computed, ref, reactive } from "vue";
import { useStore } from "vuex";
import PricingTable from "../components/PricingTable.vue";

interface LocalState {
  loading: boolean;
  license: string;
}

export default {
  name: "SettingWorkspaceSubscription",
  components: {
    PricingTable,
  },
  setup() {
    const store = useStore();

    const state = reactive<LocalState>({
      loading: false,
      license: "",
    });

    const disabled = computed((): boolean => {
      return state.loading || !state.license;
    });

    const uploadLicense = async () => {
      if (disabled.value) return;
      state.loading = true;

      try {
        await store.dispatch("subscription/patchSubscription", state.license);
      } finally {
        state.loading = false;
      }
    };

    return {
      state,
      disabled,
      uploadLicense,
    };
  },
};
</script>
