<template>
  <div class="flex flex-row justify-start items-center gap-1 whitespace-nowrap">
    <div
      class="flex flex-row gap-x-1 mr-1 justify-start items-center border border-gray-200 rounded-md p-0.5 px-2"
    >
      <UserAvatar size="TINY" :user="updater" />
      <span>{{ updater.title }}</span>
    </div>
    <span>{{ updateTimeStr }}</span>
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { computed } from "vue";
import { useUserStore } from "@/store";
import { getUserEmailFromIdentifier } from "@/store/modules/v1/common";
import { User } from "@/types/proto/v1/auth_service";
import { Branch } from "@/types/proto/v1/branch_service";

const props = defineProps<{
  branch: Branch;
}>();

const userStore = useUserStore();

const updater = computed(() => {
  const email = getUserEmailFromIdentifier(props.branch.updater);
  return userStore.getUserByEmail(email) as User;
});

const updateTimeStr = computed(() => {
  return dayjs
    .duration((props.branch.updateTime ?? new Date()).getTime() - Date.now())
    .humanize(true);
});
</script>
