<template>
  <div class="mx-4 space-y-6 divide-y divide-block-border">
    <div class="grid gap-y-6 gap-x-4 grid-cols-4">
      <div class="col-span-3 col-start-2">
        <label class="textlabel">
          {{ $t("database.pitr.restore-to") }}
        </label>
        <div class="flex items-center gap-6 textlabel py-1">
          <label class="flex items-center gap-2">
            <input
              type="radio"
              :checked="state.target === 'NEW'"
              @input="$emit('change', 'NEW')"
            />
            <span>{{ $t("database.pitr.restore-to-new-db") }}</span>
          </label>
          <label class="flex items-center">
            <input
              type="radio"
              :checked="state.target === 'IN-PLACE'"
              @input="$emit('change', 'IN-PLACE')"
            />
            <span class="ml-2 flex items-center">
              {{ $t("database.pitr.restore-to-in-place") }}

              <FeatureBadge
                feature="bb.feature.pitr"
                class="text-accent ml-1"
              />
            </span>
          </label>
        </div>
        <div
          v-if="state.target === 'IN-PLACE'"
          class="flex items-center gap-2 text-error mt-2"
        >
          <heroicons-outline:exclamation-circle class="w-4 h-4" />
          <span class="whitespace-nowrap text-sm">
            {{ $t("database.pitr.will-overwrite-current-database") }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
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
});

defineEmits<{
  (event: "change", target: RestoreTarget): void;
}>();

const state = reactive<LocalState>({
  target: props.target,
});

watch(
  () => props.target,
  (target) => {
    state.target = target;
  }
);
</script>
