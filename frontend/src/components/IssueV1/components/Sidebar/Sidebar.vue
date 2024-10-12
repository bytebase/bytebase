<template>
  <div class="flex flex-col gap-y-3 py-2 px-3">
    <VCSInfo ref="vcsInfoRef" />
    <ReleaseInfo ref="releaseInfoRef" />
    <ReviewSection ref="reviewSectionRef" />
    <IssueLabels />

    <div v-if="isFirstSectionShown" class="border-t -mx-3" />

    <EarliestAllowedTime />
    <PreBackupSection />
    <GhostSection />
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import EarliestAllowedTime from "./EarliestAllowedTime.vue";
import GhostSection from "./GhostSection";
import IssueLabels from "./IssueLabels.vue";
import PreBackupSection from "./PreBackupSection";
import ReleaseInfo from "./ReleaseInfo.vue";
import ReviewSection from "./ReviewSection";
import VCSInfo from "./VCSInfo.vue";

const vcsInfoRef = ref<InstanceType<typeof VCSInfo>>();
const releaseInfoRef = ref<InstanceType<typeof ReleaseInfo>>();
const reviewSectionRef = ref<InstanceType<typeof ReviewSection>>();

const isFirstSectionShown = computed(() => {
  return [vcsInfoRef, releaseInfoRef, reviewSectionRef].some(
    (elemRef) => elemRef.value?.shown
  );
});
</script>
