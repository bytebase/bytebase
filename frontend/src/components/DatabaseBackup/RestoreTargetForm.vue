<template>
  <div class="flex flex-col gap-y-2">
    <label class="textlabel">
      {{ $t("database.pitr.restore-to") }}
    </label>
    <div class="flex items-center gap-x-6 gap-y-2 flex-wrap">
      <NRadio
        v-for="value in targets"
        :key="value"
        :checked="state.target === value"
        @update:checked="check(value, $event)"
      >
        <template v-if="value === 'IN-PLACE'">
          {{ $t("database.pitr.restore-to-in-place") }}
        </template>
        <template v-if="value === 'NEW'">
          {{ $t("database.pitr.restore-to-new-db") }}
        </template>
      </NRadio>
    </div>
    <div
      v-if="state.target === 'IN-PLACE'"
      class="flex items-center gap-2 text-error"
    >
      <heroicons-outline:exclamation-circle class="w-4 h-4" />
      <span class="whitespace-nowrap text-sm">
        {{ $t("database.pitr.will-overwrite-current-database") }}
      </span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NRadio } from "naive-ui";
import { computed } from "vue";
import { PropType, reactive, watch } from "vue";

export type RestoreTarget = "IN-PLACE" | "NEW";

type LocalState = {
  target: RestoreTarget;
};

const props = defineProps({
  target: {
    type: String as PropType<RestoreTarget>,
    required: true,
  },
  first: {
    type: String as PropType<RestoreTarget>,
    default: "IN-PLACE",
  },
});

const emit = defineEmits<{
  (event: "change", target: RestoreTarget): void;
}>();

const state = reactive<LocalState>({
  target: props.target,
});

const targets = computed((): RestoreTarget[] => {
  return props.first === "IN-PLACE" ? ["IN-PLACE", "NEW"] : ["NEW", "IN-PLACE"];
});

const check = (value: RestoreTarget, on: boolean) => {
  if (on) {
    emit("change", value);
  }
};

watch(
  () => props.target,
  (target) => {
    state.target = target;
  }
);
</script>
