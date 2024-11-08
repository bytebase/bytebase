<template>
  <div class="flex flex-row items-center pl-4 gap-4">
    <div class="flex items-center gap-1">
      <BBAvatar size="MINI" :username="release.creatorEntity.title" />
      <span class="textlabel truncate">{{ release.creatorEntity.title }}</span>
    </div>
    <div class="flex items-center gap-1">
      <Clock4Icon class="w-4 h-auto textinfolabel" />
      <span class="textlabel">{{
        humanizeDate(getDateForPbTimestamp(release.createTime))
      }}</span>
    </div>
    <div
      v-if="vcsSource && vcsSource?.vcsType !== VCSType.VCS_TYPE_UNSPECIFIED"
      class="flex flex-row items-center gap-1"
    >
      <VCSIcon custom-class="h-4" :type="vcsSource.vcsType" />
      <EllipsisText>
        <a
          :href="vcsSource.pullRequestUrl"
          target="_blank"
          class="normal-link !text-sm"
        >
          {{ beautifyPullRequestUrl(vcsSource.pullRequestUrl) }}
        </a>
      </EllipsisText>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Clock4Icon } from "lucide-vue-next";
import { computed } from "vue";
import { BBAvatar } from "@/bbkit";
import EllipsisText from "@/components/EllipsisText.vue";
import VCSIcon from "@/components/VCS/VCSIcon.vue";
import { getDateForPbTimestamp } from "@/types";
import { VCSType } from "@/types/proto/v1/common";
import { humanizeDate } from "@/utils";
import { useReleaseDetailContext } from "./context";

const { release } = useReleaseDetailContext();

const vcsSource = computed(() => release.value.vcsSource);

const beautifyPullRequestUrl = (pullRequestUrl: string) => {
  // Prevent URL parsing error when pullRequestUrl is invalid.
  try {
    const parsedUrl = new URL(pullRequestUrl);
    return parsedUrl.pathname.length > 0
      ? parsedUrl.pathname.substring(1)
      : parsedUrl.pathname;
  } catch {
    return pullRequestUrl;
  }
};
</script>
