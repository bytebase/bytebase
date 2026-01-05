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
import { type UserFilter, userNamePrefix, useUserStore } from "@/store";
import { allUsersUser } from "@/types";
import { type User, UserType } from "@/types/proto-es/v1/user_service_pb";
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
  value?: string | string[] | undefined; // email or emails
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
    options.push(getOption(allUsersUser()));
  }

  let userEmails: string[] = [];
  if (Array.isArray(props.value)) {
    userEmails = props.value;
  } else if (props.value) {
    userEmails = [props.value];
  }

  // Ensure users are fetched into store
  await userStore.batchGetOrFetchUsers(
    userEmails.map((email) => `${userNamePrefix}${email}`)
  );

  // Get all users from store
  for (const email of userEmails) {
    const user = userStore.getUserByIdentifier(email);
    if (user) {
      options.push(getOption(user));
    }
  }

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
      showMfaEnabled={false}
      showSource={false}
      showEmail={false}
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
