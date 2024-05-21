<template>
  <NPopover>
    <template #trigger>
      <NButton size="tiny" text>
        <template #icon>
          <FileClockIcon class="w-4 h-4" />
        </template>
      </NButton>
    </template>
    <template #default>
      <div class="flex flex-col items-stretch gap-2 text-sm">
        <div class="flex justify-between gap-4">
          <div>
            {{ $t("branch.last-update.self") }}
          </div>

          <div class="flex justify-end gap-1">
            <UserAvatar :user="user" size="TINY" />
            {{ user?.title }}
          </div>
        </div>
        <div v-if="updateTime" class="flex justify-end gap-1 text-xs">
          {{ dayjs(updateTime).format("YYYY-MM-DD HH:mm:ss UTCZZ") }}
        </div>
      </div>
    </template>
  </NPopover>
</template>

<script setup lang="ts">
import { FileClockIcon } from "lucide-vue-next";
import { NButton, NPopover } from "naive-ui";
import { computed } from "vue";
import UserAvatar from "@/components/User/UserAvatar.vue";
import { extractUserEmail, useUserStore } from "@/store";

const props = defineProps<{
  updater: string; // Format: users/{email}
  updateTime: Date | undefined;
}>();

const user = computed(() => {
  const email = extractUserEmail(props.updater);
  return useUserStore().getUserByEmail(email);
});
</script>
