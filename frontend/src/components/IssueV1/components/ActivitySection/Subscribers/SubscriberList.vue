<template>
  <NPopselect
    :multiple="true"
    :value="issue.subscribers"
    :options="options"
    :render-label="renderLabel"
    :scrollable="true"
    trigger="click"
    placement="left"
    @update:show="onUpdateShow"
    @update:value="onUpdateSubscribers"
  >
    <NButton
      quaternary
      :style="{
        '--n-padding':
          subscriberList.length === 0 ? '0 8px 0 8px' : '0 8px 0 12px',
      }"
    >
      <div class="flex items-center gap-x-1">
        <UserAvatar
          v-for="user in subscriberList"
          :key="user.name"
          :user="user"
          size="SMALL"
          class="ml-[-18px] first:ml-0"
        />
        <heroicons:ellipsis-horizontal class="w-5 h-5" />
      </div>
    </NButton>

    <template v-if="false" #action>
      <NInput
        ref="filterInputRef"
        v-model:value="keyword"
        :placeholder="$t('common.search-user')"
      />
    </template>
  </NPopselect>
</template>

<script setup lang="ts">
import { computed, h, nextTick, ref } from "vue";
import {
  NButton,
  NPopselect,
  NInput,
  SelectOption,
  SelectGroupOption,
} from "naive-ui";

import { useIssueContext } from "@/components/IssueV1";
import { useUserStore } from "@/store";
import { unknownUser } from "@/types";
import { extractUserResourceName } from "@/utils";
import UserAvatar from "@/components/User/UserAvatar.vue";
import { User, UserType } from "@/types/proto/v1/auth_service";
import SubscriberListItem from "./SubscriberListItem.vue";

type UserSelectOption = SelectOption & {
  user: User;
  value: string;
};

const userStore = useUserStore();
const { issue } = useIssueContext();
const keyword = ref("");
const filterInputRef = ref<InstanceType<typeof NInput>>();

const subscriberList = computed(() => {
  return issue.value.subscribers.map((subscriber) => {
    const email = extractUserResourceName(subscriber);
    return userStore.getUserByEmail(email) ?? unknownUser();
  });
});

const options = computed(() => {
  const subscribers = new Set(issue.value.subscribers);
  const options = userStore.userList
    .filter((user) => user.userType === UserType.USER)
    .map<UserSelectOption>((user) => ({
      user,
      value: `users/${user.email}`,
      label: user.title,
    }));

  const subscribersOptions = options.filter((option) =>
    subscribers.has(option.value)
  );
  const nonsubscribersOptions = options.filter(
    (option) => !subscribers.has(option.value)
  );
  const groups: SelectGroupOption[] = [];
  if (subscribersOptions.length > 0) {
    groups.push({
      type: "group",
      key: "subscribers",
      children: subscribersOptions,
      render() {
        return null;
      },
    });
  }
  if (nonsubscribersOptions.length > 0) {
    groups.push({
      type: "group",
      key: "nonsubscribers",
      children: nonsubscribersOptions,
      render() {
        if (subscribersOptions.length > 0) {
          return h("hr", { class: "my-1" });
        }
        return null;
      },
    });
  }

  return groups;
});

const renderLabel = (option: SelectOption | SelectGroupOption) => {
  if (option.type === "group") {
    return null;
  }

  const { user } = option as UserSelectOption;

  return h(SubscriberListItem, { user });
};

const onUpdateSubscribers = async (subscribers: string[]) => {
  issue.value.subscribers = subscribers;
  keyword.value = "";
};

const onUpdateShow = (show: boolean) => {
  if (show) {
    nextTick(() => {
      filterInputRef.value?.focus();
    });
  }
};
</script>
