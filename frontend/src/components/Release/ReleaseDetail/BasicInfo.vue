<template>
  <div class="flex flex-row items-center pl-4 gap-4">
    <div class="flex items-center gap-1">
      <BBAvatar size="MINI" :username="creatorTitle" />
      <span class="textlabel truncate">{{ creatorTitle }}</span>
    </div>
    <div class="flex items-center gap-1">
      <Clock4Icon class="w-4 h-auto textinfolabel" />
      <span class="textlabel">{{
        humanizeDate(getDateForPbTimestampProtoEs(release.createTime))
      }}</span>
    </div>
    <div
      v-if="vcsSource && vcsSource?.vcsType !== VCSType.VCS_TYPE_UNSPECIFIED"
      class="flex flex-row items-center gap-1"
    >
      <VCSIcon custom-class="h-4" :type="vcsSource.vcsType" />
      <EllipsisText>
        <a :href="vcsSource.url" target="_blank" class="normal-link !text-sm">
          {{ beautifyUrl(vcsSource.url) }}
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
import { useUserStore } from "@/store";
import { getDateForPbTimestampProtoEs } from "@/types";
import { VCSType } from "@/types/proto-es/v1/common_pb";
import { humanizeDate } from "@/utils";
import { useReleaseDetailContext } from "./context";

const { release } = useReleaseDetailContext();
const userStore = useUserStore();

const vcsSource = computed(() => release.value.vcsSource);

const creatorTitle = computed(() => {
  const user = userStore.getUserByIdentifier(release.value.creator);
  return user?.title || release.value.creator;
});

const beautifyUrl = (url: string) => {
  // Prevent URL parsing error when url is invalid.
  try {
    const parsedUrl = new URL(url);
    return parsedUrl.pathname.length > 0
      ? parsedUrl.pathname.substring(1)
      : parsedUrl.pathname;
  } catch {
    return url;
  }
};
</script>
