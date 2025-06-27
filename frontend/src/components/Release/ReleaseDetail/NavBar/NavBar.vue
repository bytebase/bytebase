<template>
  <ArchiveBanner v-if="release.state === convertStateToOld(State.DELETED)" />
  <div class="w-full flex flex-row items-center justify-between gap-x-4">
    <div class="flex-1 p-0.5 overflow-hidden truncate">
      <TitleEditor />
    </div>
    <div class="flex items-center justify-end gap-x-3">
      <ApplyToDatabaseButton v-if="release.state === convertStateToOld(State.ACTIVE)" />
    </div>
  </div>
</template>

<script setup lang="ts">
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import { State } from "@/types/proto-es/v1/common_pb";
import { useReleaseDetailContext } from "../context";
import { convertStateToOld } from "@/utils/v1/common-conversions";
import ApplyToDatabaseButton from "./ApplyToDatabaseButton.vue";
import TitleEditor from "./TitleEditor.vue";

const { release } = useReleaseDetailContext();
</script>
