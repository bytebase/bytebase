<template>
  <NPopselect
    :remote="true"
    :loading="state.loading"
    :multiple="true"
    :value="issue.subscribers"
    :options="options"
    :render-label="renderLabel"
    :scrollable="true"
    :filterable="true"
    trigger="click"
    placement="left"
    :disabled="readonly"
    @search="handleSearch"
    @update:show="onUpdateShow"
    @update:value="onUpdateSubscribers"
  >
    <NButton quaternary>
      <div class="flex items-center gap-x-1">
        <UserAvatar
          v-for="user in subscriberList"
          :key="user.name"
          :user="user"
          size="SMALL"
          class="ml-[-18px] first:ml-0"
        />
        <heroicons:ellipsis-horizontal v-if="!readonly" class="w-5 h-5" />
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
import { useDebounceFn } from "@vueuse/core";
import type { SelectOption, SelectGroupOption } from "naive-ui";
import { NButton, NPopselect, NInput } from "naive-ui";
import { computed, h, nextTick, ref, onMounted, reactive } from "vue";
import { updateIssueSubscribers, useIssueContext } from "@/components/IssueV1";
import UserAvatar from "@/components/User/UserAvatar.vue";
import { useUserStore } from "@/store";
import { unknownUser } from "@/types";
import { State, stateToJSON } from "@/types/proto/v1/common";
import { type User } from "@/types/proto/v1/user_service";
import { UserType, userTypeToJSON } from "@/types/proto/v1/user_service";
import { getDefaultPagination } from "@/utils";
import SubscriberListItem from "./SubscriberListItem.vue";

defineProps<{
  readonly?: boolean;
}>();

type UserSelectOption = SelectOption & {
  user: User;
  value: string;
};

interface LocalState {
  loading: boolean;
  rawUserList: User[];
}

const state = reactive<LocalState>({
  loading: false,
  rawUserList: [],
});

const userStore = useUserStore();
const { issue } = useIssueContext();
const keyword = ref("");
const filterInputRef = ref<InstanceType<typeof NInput>>();

const subscriberList = computed(() => {
  return issue.value.subscribers.map((subscriber) => {
    return userStore.getUserByIdentifier(subscriber) ?? unknownUser();
  });
});

const options = computed(() => {
  const subscribers = new Set(issue.value.subscribers);
  const options = state.rawUserList
    .filter(
      (user) => user.userType === UserType.USER && user.state === State.ACTIVE
    )
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
  try {
    await updateIssueSubscribers(issue.value, subscribers);
    keyword.value = "";
  } finally {
    // Nothing
  }
};

const onUpdateShow = (show: boolean) => {
  if (show) {
    nextTick(() => {
      filterInputRef.value?.focus();
    });
  }
};

// TODO(ed): I found the NPopselect NOT support search.
// We need to use the NSelector instead.
const handleSearch = useDebounceFn(async (search: string) => {
  const filter = [
    `state == "${stateToJSON(State.ACTIVE)}"`,
    `user_type == "${userTypeToJSON(UserType.USER)}"`,
  ];
  if (search) {
    filter.push(`(name.matches("${search}") || email.matches("${search}"))`);
  }

  state.loading = true;

  try {
    const { users } = await userStore.fetchUserList({
      filter: filter.join(" && "),
      pageSize: getDefaultPagination(),
      showDeleted: false,
    });
    state.rawUserList = users;
  } finally {
    state.loading = false;
  }
});

onMounted(async () => {
  await handleSearch("");
});
</script>
