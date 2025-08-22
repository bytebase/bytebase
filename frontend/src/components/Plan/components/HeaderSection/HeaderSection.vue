<template>
  <div class="py-2 px-2 sm:px-4">
    <div class="flex flex-row items-center justify-between gap-2">
      <NTag v-if="showDraftTag" round>
        <template #icon>
          <CircleDotDashedIcon class="w-4 h-4" />
        </template>
        {{ $t("common.draft") }}
      </NTag>
      <TitleInput />
      <div class="flex flex-row items-center justify-end">
        <Actions />
      </div>
    </div>
    <div v-if="showDescriptionInput" class="flex flex-col mt-2">
      <label class="block text-sm text-control mb-1">
        {{ $t("plan.description.add-a-description") }}
      </label>
      <DescriptionInput />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { CircleDotDashedIcon } from "lucide-vue-next";
import { NTag } from "naive-ui";
import { computed } from "vue";
import { usePlanContext } from "../../logic";
import Actions from "./Actions";
import DescriptionInput from "./DescriptionInput.vue";
import TitleInput from "./TitleInput.vue";

const { isCreating, plan } = usePlanContext();

const showDraftTag = computed(() => {
  return !isCreating.value && !plan.value.issue && !plan.value.rollout;
});

const showDescriptionInput = computed(() => {
  return !plan.value.issue;
});
</script>
