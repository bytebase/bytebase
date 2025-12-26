<template>
  <template v-if="users.length > 0">
    <!-- Show tooltip with text when there are more than 3 users -->
    <NPopover
      v-if="users.length > 3"
      placement="bottom-end"
      :show-arrow="false"
    >
      <template #trigger>
        <span class="text-xs text-gray-600 cursor-pointer hover:text-blue-600">
          {{ triggerText }}
        </span>
      </template>
      <div class="max-w-xs">
        <div class="flex flex-col gap-y-1 max-h-64 overflow-y-auto">
          <UserNameCell
            v-for="user in users"
            :key="user.name"
            :user="user"
            size="small"
            :show-mfa-enabled="false"
            :show-source="false"
            :allow-edit="false"
          />
        </div>
      </div>
    </NPopover>
    <!-- Show user avatars when 3 or fewer users -->
    <div v-else class="flex flex-col items-start gap-1">
      <UserNameCell
        v-for="user in users"
        :key="user.name"
        :user="user"
        size="tiny"
        :show-mfa-enabled="false"
        :show-source="false"
        :allow-edit="false"
        :show-email="false"
      />
    </div>
  </template>
</template>

<script setup lang="ts">
import { NPopover } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { UserNameCell } from "@/components/v2/Model/cells";
import type { User as UserType } from "@/types/proto-es/v1/user_service_pb";

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
