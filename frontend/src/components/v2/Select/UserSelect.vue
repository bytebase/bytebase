<template>
  <RemoteResourceSelector
    v-bind="$attrs"
    :multiple="multiple"
    :disabled="disabled"
    :size="size"
    :value="value"
    :additional-options="additionalOptions"
    :render-label="renderLabel"
    :render-tag="renderTag"
    :search="handleSearch"
    :filter="filter"
    @update:value="(val) => $emit('update:value', val)"
  />
</template>

<script lang="tsx" setup>
import { computedAsync } from "@vueuse/core";
import { computed } from "vue";
import { HighlightLabelText } from "@/components/v2";
import { UserNameCell } from "@/components/v2/Model/cells";
import { type UserFilter, useUserStore } from "@/store";
import { allUsersUser, isValidUserName } from "@/types";
import { type User, UserType } from "@/types/proto-es/v1/user_service_pb";
import { ensureUserFullName } from "@/utils";
import RemoteResourceSelector from "./RemoteResourceSelector/index.vue";
import type {
  ResourceSelectOption,
  SelectSize,
} from "./RemoteResourceSelector/types";
import {
  getRenderLabelFunc,
  getRenderTagFunc,
} from "./RemoteResourceSelector/utils";

const props = defineProps<{
  multiple?: boolean;
  disabled?: boolean;
  size?: SelectSize;
  value?: string | string[] | undefined;
  projectName?: string;
  // allUsers is a special user that represents all users in the project.
  includeAllUsers?: boolean;
  includeSystemBot?: boolean;
  includeServiceAccount?: boolean;
  includeWorkloadIdentity?: boolean;
  filter?: (user: User) => boolean;
}>();

defineEmits<{
  (event: "update:value", value: string[] | string | undefined): void;
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

const getOption = (user: User): ResourceSelectOption<User> => ({
  resource: user,
  value: user.email,
  label: user.title,
});

const additionalOptions = computedAsync(async () => {
  const options: ResourceSelectOption<User>[] = [];
  if (props.includeAllUsers) {
    options.unshift(getOption(allUsersUser()));
  }

  let userNames: string[] = [];
  if (Array.isArray(props.value)) {
    userNames = props.value;
  } else if (props.value) {
    userNames = [props.value];
  }

  const users = await userStore.batchGetUsers(userNames);
  options.push(...users.map(getOption));

  return options;
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
    options: users.map(getOption),
  };
};

const customLabel = (user: User, keyword: string) => {
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

const renderLabel = computed(() => {
  return getRenderLabelFunc({
    multiple: props.multiple,
    customLabel,
    showResourceName: false,
  });
});

const renderTag = computed(() => {
  return getRenderTagFunc({
    multiple: props.multiple,
    disabled: props.disabled,
    size: props.size,
    customLabel,
  });
});
</script>
