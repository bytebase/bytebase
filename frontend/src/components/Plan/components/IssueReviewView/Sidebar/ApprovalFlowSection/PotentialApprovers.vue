<template>
  <template v-if="users.length > 0">
    <!-- Show tooltip with text when there are more than 3 users -->
    <NPopover
      v-if="users.length > 3"
      placement="bottom-start"
      :show-arrow="false"
    >
      <template #trigger>
        <span class="text-xs text-gray-600 cursor-pointer hover:text-blue-600">
          {{ triggerText }}
        </span>
      </template>
      <div class="max-w-xs">
        <div class="space-y-1 max-h-64 overflow-y-auto">
          <ApprovalCandidate
            v-for="user in users"
            :key="user.email"
            :candidate="`users/${user.email}`"
          />
        </div>
      </div>
    </NPopover>
    <!-- Show user avatars when 3 or fewer users -->
    <div v-else class="flex items-center gap-1">
      <UserView
        v-for="user in users"
        :key="user.email"
        :user="user"
        size="tiny"
      />
    </div>
  </template>
</template>

<script setup lang="ts">
import { NPopover } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import UserView from "@/components/User/UserView.vue";
import type { User as UserType } from "@/types/proto/v1/user_service";
import ApprovalCandidate from "./ApprovalCandidate.vue";

const props = defineProps<{
  users: UserType[];
}>();

const { t } = useI18n();

const triggerText = computed(() => {
  if (props.users.length === 0) return "";

  const displayUsers = props.users.slice(0, 3);
  const remainingCount = props.users.length - displayUsers.length;
  const names = displayUsers
    .map((user) => user.title || user.email.split("@")[0])
    .join(", ");

  if (remainingCount === 0) {
    return names;
  } else {
    return t("custom-approval.issue-review.and-n-other-users", {
      names,
      count: remainingCount,
    });
  }
});
</script>
