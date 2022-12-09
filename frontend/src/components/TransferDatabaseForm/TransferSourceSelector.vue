<template>
  <div class="textlabel">
    <div v-if="state.transferSource == 'DEFAULT'" class="textinfolabel mb-2">
      {{ $t("quick-action.unassigned-db-hint") }}
    </div>
    <div class="flex items-center justify-between">
      <div class="radio-set-row">
        <template v-if="project.id != DEFAULT_PROJECT_ID">
          <label class="radio">
            <input
              v-model="state.transferSource"
              tabindex="-1"
              type="radio"
              class="btn"
              value="DEFAULT"
            />
            <span class="label">
              {{ $t("quick-action.from-unassigned-databases") }}
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
              {{ $t("quick-action.from-projects") }}
            </span>
          </label>
        </template>
      </div>
      <div>
        <BBTableSearch
          class="m-px"
          :value="searchText"
          :placeholder="$t('database.search-database')"
          @change-text="(text: string) => $emit('search-text-change', text)"
        />
      </div>
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
  searchText: {
    type: String,
    default: "",
  },
});

const emit = defineEmits<{
  (event: "change", src: TransferSource): void;
  (event: "search-text-change", searchText: string): void;
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
