<template>
  <ResourceSelect
    v-bind="$attrs"
    :remote="true"
    :loading="state.loading"
    :multiple="multiple"
    :value="user"
    :values="users"
    :size="size"
    :options="options"
    :custom-label="renderLabel"
    :filter="filterByEmail"
    :show-resource-name="false"
    @search="handleSearch"
    @update:value="(val) => $emit('update:user', val)"
    @update:values="(val) => $emit('update:users', val)"
  />
</template>

<script lang="tsx" setup>
import { useDebounceFn } from "@vueuse/core";
import { computed, onMounted, reactive, watch } from "vue";
import { UserNameCell } from "@/components/v2/Model/cells";
import { type UserFilter, useUserStore } from "@/store";
import { allUsersUser, DEBOUNCE_SEARCH_DELAY, isValidUserName } from "@/types";
import { type User, UserType } from "@/types/proto-es/v1/user_service_pb";
import { ensureUserFullName, getDefaultPagination } from "@/utils";
import ResourceSelect from "./ResourceSelect.vue";

const props = withDefaults(
  defineProps<{
    multiple?: boolean;
    user?: string;
    users?: string[];
    projectName?: string;
    // allUsers is a special user that represents all users in the project.
    includeAllUsers?: boolean;
    includeSystemBot?: boolean;
    includeServiceAccount?: boolean;
    includeWorkloadIdentity?: boolean;
    autoReset?: boolean;
    filter?: (user: User, index: number) => boolean;
    size?: "tiny" | "small" | "medium" | "large";
  }>(),
  {
    multiple: false,
    user: undefined,
    users: undefined,
    projectName: undefined,
    includeAllUsers: false,
    includeSystemBot: false,
    includeServiceAccount: false,
    includeWorkloadIdentity: false,
    autoReset: true,
    filter: (_1: User, _2: number) => true,
    size: "medium",
  }
);

const emit = defineEmits<{
  (event: "update:user", value: string | undefined): void;
  (event: "update:users", value: string[]): void;
}>();

interface LocalState {
  loading: boolean;
  rawUserList: User[];
  // Track if initial fetch has been done to avoid redundant API calls
  initialized: boolean;
}

const userStore = useUserStore();
const state = reactive<LocalState>({
  loading: false,
  rawUserList: [],
  initialized: false,
});

const getFilter = (search: string): UserFilter => {
  const filter = [];
  if (search) {
    filter.push(`(name.matches("${search}") || email.matches("${search}"))`);
  }
  const allowedType = [UserType.USER];
  if (props.includeServiceAccount) {
    allowedType.push(UserType.SERVICE_ACCOUNT);
  }
  if (props.includeSystemBot) {
    allowedType.push(UserType.SYSTEM_BOT);
  }
  if (props.includeWorkloadIdentity) {
    allowedType.push(UserType.WORKLOAD_IDENTITY);
  }

  return {
    query: search,
    project: props.projectName,
    types: allowedType,
  };
};

const initSelectedUsers = async (userEmails: string[]) => {
  for (const email of userEmails) {
    if (!email) continue;
    const userName = ensureUserFullName(email);
    if (isValidUserName(userName)) {
      const user = await userStore.getOrFetchUserByIdentifier(userName);
      if (!state.rawUserList.find((u) => u.name === user.name)) {
        state.rawUserList.unshift(user);
      }
    }
  }
};

const searchUsers = async (search: string) => {
  const { users } = await userStore.fetchUserList({
    filter: getFilter(search),
    pageSize: getDefaultPagination(),
  });
  return users.filter(props.filter);
};

const handleSearch = useDebounceFn(async (search: string) => {
  // Skip if no search term and already initialized (lazy loading optimization)
  if (!search && state.initialized) {
    return;
  }

  state.loading = true;
  try {
    const users = await searchUsers(search);
    state.rawUserList = users;
    if (!search) {
      state.initialized = true;
      if (props.includeAllUsers) {
        state.rawUserList.unshift(allUsersUser());
      }
      if (props.user) {
        await initSelectedUsers([props.user]);
      }
      if (props.users) {
        await initSelectedUsers(props.users);
      }
    }
  } finally {
    state.loading = false;
  }
}, DEBOUNCE_SEARCH_DELAY);

// Only fetch the selected user(s) on mount, not the entire user list.
// The full list will be fetched lazily when dropdown is opened.
onMounted(async () => {
  let users: string[] = [];
  if (props.user) {
    users = [props.user];
  } else if (props.users) {
    users = props.users;
  }
  await initSelectedUsers(users);
});

const renderLabel = (user: User) => {
  return (
    <UserNameCell
      user={user}
      allowEdit={false}
      size="small"
      onClickUser={() => {}}
    >
      {{
        suffix: () => (
          <span class="textinfolabel truncate">{`(${user.email})`}</span>
        ),
        footer: () => <div />,
      }}
    </UserNameCell>
  );
};

const options = computed(() => {
  return state.rawUserList.map((user) => {
    return {
      resource: user,
      value: user.email,
      label: user.title,
    };
  });
});

const filterByEmail = (pattern: string, user: User) => {
  return user.email.includes(pattern);
};

// The user list might change if props change, and the previous selected id
// might not exist in the new list. In such case, we need to invalidate the selection
// and emit the event.
const resetInvalidSelection = () => {
  if (!props.autoReset || props.multiple) {
    return;
  }
  if (state.loading) {
    return;
  }
  // Don't reset selection before the full user list has been fetched
  if (!state.initialized) {
    return;
  }
  if (
    props.user &&
    !state.rawUserList.find((user) => user.email === props.user)
  ) {
    emit("update:user", undefined);
  }
};

watch(
  [() => state.loading, () => props.user, state.rawUserList],
  resetInvalidSelection,
  {
    immediate: true,
  }
);
</script>
