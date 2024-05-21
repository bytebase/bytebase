<template>
  <div class="flex flex-col gap-y-3 py-2 px-3">
    <VCSInfo ref="vcsInfoRef" />
    <RollbackFromTips ref="rollbackFromTipsRef" />
    <ReviewSection ref="reviewSectionRef" />
    <IssueLabels />

    <div v-if="isFirstSectionShown" class="border-t -mx-3" />

    <EarliestAllowedTime />
    <PreBackupSection />
    <RollbackSection />
    <GhostSection />
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import EarliestAllowedTime from "./EarliestAllowedTime.vue";
import GhostSection from "./GhostSection";
import IssueLabels from "./IssueLabels.vue";
import PreBackupSection from "./PreBackupSection";
import ReviewSection from "./ReviewSection";
import RollbackFromTips from "./RollbackFromTips.vue";
import RollbackSection from "./RollbackSection";
import VCSInfo from "./VCSInfo.vue";

const vcsInfoRef = ref<InstanceType<typeof VCSInfo>>();
const rollbackFromTipsRef = ref<InstanceType<typeof RollbackFromTips>>();
const reviewSectionRef = ref<InstanceType<typeof ReviewSection>>();

const isFirstSectionShown = computed(() => {
  return [vcsInfoRef, rollbackFromTipsRef, reviewSectionRef].some(
    (elemRef) => elemRef.value?.shown
  );
});
</script>
