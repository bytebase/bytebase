<template>
  <BBModal
    :close-on-esc="true"
    :mask-closable="true"
    :trap-focus="false"
    :show="show"
    :title="$t('project.select')"
    class="w-[48rem] max-w-full h-auto max-h-full"
    @close="$emit('dismiss')"
  >
    <ProjectSwitchContent />
  </BBModal>
</template>

<script lang="ts" setup>
import { BBModal } from "@/bbkit";
import ProjectSwitchContent from "./ProjectSwitchContent.vue";

defineProps<{
  show: boolean;
}>();

defineEmits<{
  (event: "dismiss"): void;
}>();
</script>
