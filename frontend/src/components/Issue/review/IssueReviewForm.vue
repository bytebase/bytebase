<template>
  <div space-y-3>
    <div class="pt-4 pb-1 flex justify-end gap-x-3">
      <button class="btn-normal" @click="$emit('cancel')">
        {{ $t("common.cancel") }}
      </button>
      <button class="btn-primary" @click="handleConfirm">
        {{ $t("common.approve") }}
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { h, Ref } from "vue";

import { useOverrideSubtitle } from "@/bbkit/BBModal.vue";
import { useIssueLogic } from "../logic";
import { Issue } from "@/types";

const emit = defineEmits<{
  (event: "cancel"): void;
  (event: "confirm", onSuccess: () => void): void;
}>();

const issueContext = useIssueLogic();
const issue = issueContext.issue as Ref<Issue>;

useOverrideSubtitle(() => {
  return h(
    "div",
    {
      class: "mt-1 textinfolabel whitespace-pre-wrap",
    },
    issue.value.name
  );
});

const handleConfirm = (e: MouseEvent) => {
  const button = e.target as HTMLElement;
  const { left, top, width, height } = button.getBoundingClientRect();
  const { innerWidth: winWidth, innerHeight: winHeight } = window;
  const onSuccess = () => {
    // import the effect lib asynchronously
    import("canvas-confetti").then(({ default: confetti }) => {
      // Create a confetti effect from the position of the LGTM button
      confetti({
        particleCount: 100,
        spread: 70,
        origin: {
          x: (left + width / 2) / winWidth,
          y: (top + height / 2) / winHeight,
        },
      });
    });
  };

  emit("confirm", onSuccess);
};
</script>
