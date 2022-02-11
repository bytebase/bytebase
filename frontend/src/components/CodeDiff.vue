<template>
  <a
    :id="elId"
    :href="'#' + elId"
    class="flex items-center text-lg text-main mt-6 hover:underline capitalize"
  >
    {{ title }}
    <button tabindex="-1" class="btn-icon ml-1" @click.prevent="copyNewCode">
      <heroicons-outline:clipboard class="w-6 h-6" />
    </button>
  </a>
  <div class="flex flex-row items-center space-x-2 mt-2">
    <BBSwitch
      v-if="oldCode !== newCode"
      :label="switcherLabel"
      :value="state.showDiff"
      @toggle="
              (on: any) => {
                state.showDiff = on;
              }
            "
    />
    <div class="textinfolabel">
      {{ state.showDiff ? infoSwitchOnDiff : infoSwitchOffDiff }}
    </div>
    <div v-if="oldCode === newCode" class="text-sm font-normal text-accent">
      ({{ infoNoDiff }})
    </div>
  </div>
  <v-code-diff
    v-if="state.showDiff"
    class="mt-4 w-full"
    :old-string="oldCode"
    :new-string="newCode"
    output-format="side-by-side"
  />
  <div v-else v-highlight class="border mt-2 px-2 whitespace-pre-wrap w-full">
    {{ newCode }}
  </div>
</template>

<script lang="ts">
import { defineComponent, reactive } from "vue";
import { CodeDiff as VCodeDiff } from "v-code-diff";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import { useStore } from "vuex";

interface LocalState {
  showDiff: boolean;
}

export default defineComponent({
  name: "CodeDiff",
  components: { VCodeDiff },
  props: {
    elId: { required: true, type: String },
    title: { required: true, type: String },
    switcherLabel: { required: true, type: String },
    infoSwitchOnDiff: { required: true, type: String },
    infoSwitchOffDiff: { required: true, type: String },
    infoNoDiff: { required: true, type: String },
    oldCode: { required: true, type: String },
    newCode: { required: true, type: String },
  },
  setup(props) {
    const store = useStore();

    const state = reactive<LocalState>({
      showDiff: props.oldCode !== props.newCode,
    });

    const copyNewCode = () => {
      toClipboard(props.newCode).then(() => {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "INFO",
          title: `Schema copied to clipboard.`,
        });
      });
    };

    return {
      state,
      copyNewCode,
    };
  },
});
</script>
