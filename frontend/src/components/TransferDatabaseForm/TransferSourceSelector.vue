<template>
  <div v-if="project.id != DEFAULT_PROJECT_ID" class="textlabel">
    <div v-if="state.transferSource == 'DEFAULT'" class="textinfolabel mb-2">
      {{ $t("quick-action.default-db-hint") }}
    </div>
    <div class="radio-set-row">
      <label class="radio">
        <input
          v-model="state.transferSource"
          tabindex="-1"
          type="radio"
          class="btn"
          value="DEFAULT"
        />
        <span class="label">
          {{ $t("quick-action.from-default-project") }}
        </span>
      </label>
      <label class="radio">
        <input
          v-model="state.transferSource"
          tabindex="-1"
          type="radio"
          class="btn"
          value="OTHER"
        />
        <span class="label">
          {{ $t("quick-action.from-other-projects") }}
        </span>
      </label>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PropType, reactive, watch } from "vue";
import { TransferSource } from "./utils";
import { Project, DEFAULT_PROJECT_ID } from "@/types";

interface LocalState {
  transferSource: TransferSource;
}

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
  transferSource: {
    type: String as PropType<TransferSource>,
    required: true,
  },
});

const emit = defineEmits<{
  (event: "change", src: TransferSource): void;
}>();

const state = reactive<LocalState>({
  transferSource: props.transferSource,
});

watch(
  () => props.transferSource,
  (src) => (state.transferSource = src)
);

watch(
  () => state.transferSource,
  (src) => {
    if (src !== props.transferSource) {
      emit("change", src);
    }
  }
);
</script>
