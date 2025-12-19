<template>
  <RemoteResourceSelector
    v-bind="$attrs"
    :multiple="multiple"
    :value="user"
    :values="users"
    :additional-data="additionalData"
    :custom-label="renderLabel"
    :show-resource-name="false"
    :search="handleSearch"
    :get-option="getOption"
    :filter="filter"
    @update:value="(val) => $emit('update:user', val)"
    @update:values="(val) => $emit('update:users', val)"
  />
</template>

<script lang="tsx" setup>
import { computedAsync } from "@vueuse/core";
import { HighlightLabelText } from "@/components/v2";
import { UserNameCell } from "@/components/v2/Model/cells";
import { type UserFilter, useUserStore } from "@/store";
import { allUsersUser, isValidUserName } from "@/types";
import { type User, UserType } from "@/types/proto-es/v1/user_service_pb";
import { ensureUserFullName } from "@/utils";
import RemoteResourceSelector from "./RemoteResourceSelector.vue";

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
    filter?: (user: User) => boolean;
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
  }
);

const emit = defineEmits<{
  (event: "update:user", value: string | undefined): void;
  (event: "update:users", value: string[]): void;
}>();

const userStore = useUserStore();

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

const additionalData = computedAsync(async () => {
  const data = [];
  if (props.includeAllUsers) {
    data.unshift(allUsersUser());
  }

  let userNames: string[] = [];
  if (props.user) {
    userNames = [props.user];
  } else if (props.users) {
    userNames = props.users;
  }

  for (const email of userNames) {
    if (!email) continue;
    const userName = ensureUserFullName(email);
    if (isValidUserName(userName)) {
      const user = await userStore.getOrFetchUserByIdentifier(userName);
      data.push(user);
    }
  }

  return data;
}, []);

const handleSearch = async (params: {
  search: string;
  pageToken: string;
  pageSize: number;
}) => {
  const { users, nextPageToken } = await userStore.fetchUserList({
    filter: getFilter(params.search),
    pageToken: params.pageToken,
    pageSize: params.pageSize,
  });
  return {
    nextPageToken,
    data: users,
  };
};

const renderLabel = (user: User, keyword: string) => {
  return (
    <UserNameCell
      user={user}
      allowEdit={false}
      link={false}
      size="small"
      keyword={keyword}
      onClickUser={() => {}}
    >
      {{
        suffix: () => (
          <span class="textinfolabel truncate">
            (<HighlightLabelText keyword={keyword} text={user.email} />)
          </span>
        ),
        footer: () => <div />,
      }}
    </UserNameCell>
  );
};

const getOption = (user: User) => ({
  value: user.email,
  label: user.title,
});
</script>
