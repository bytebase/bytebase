<template>
  <RemoteResourceSelector
    v-bind="$attrs"
    :multiple="multiple"
    :disabled="disabled"
    :size="size"
    :value="value"
    :tag="true"
    :remote="true"
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
import { useI18n } from "vue-i18n";
import { HighlightLabelText } from "@/components/v2";
import { UserNameCell } from "@/components/v2/Model/cells";
import { getUserFullNameByType, useUserStore } from "@/store";
import { type User } from "@/types/proto-es/v1/user_service_pb";
import RemoteResourceSelector from "./RemoteResourceSelector/index.vue";
import type {
  ResourceSelectOption,
  SelectSize,
} from "./RemoteResourceSelector/types";
import {
  getRenderLabelFunc,
  getRenderTagFunc,
  searchUsersWithFallback,
  type UserResource,
} from "./RemoteResourceSelector/utils";

const props = defineProps<{
  multiple?: boolean;
  disabled?: boolean;
  size?: SelectSize;
  value?: string | string[] | undefined; // user fullname
  projectName?: string;
  filter?: (user: User) => boolean;
  allowArbitraryEmail?: boolean;
}>();

defineEmits<{
  // the value is user fullname
  (event: "update:value", value: string[] | string | undefined): void;
}>();

const userStore = useUserStore();
const { t } = useI18n();

const getOption = (user: User): ResourceSelectOption<UserResource> => ({
  resource: user,
  value: getUserFullNameByType(user),
  label: user.title,
});

const additionalOptions = computedAsync(async () => {
  const options: ResourceSelectOption<UserResource>[] = [];

  let userEmails: string[] = [];
  if (Array.isArray(props.value)) {
    userEmails = props.value;
  } else if (props.value) {
    userEmails = [props.value];
  }

  // Ensure users are fetched into store
  await userStore.batchGetOrFetchUsers(userEmails);

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
  const { users, nextPageToken } = await searchUsersWithFallback({
    ...params,
    project: props.projectName,
    allowArbitraryEmail: props.allowArbitraryEmail,
  });
  return {
    nextPageToken,
    options: users.map(getOption),
  };
};

const customLabel = (user: UserResource, keyword: string) => {
  if (user.isExternal) {
    return (
      <div class="flex items-center shrink gap-x-2 py-0.5">
        <HighlightLabelText keyword={keyword} text={user.email} />
        <span class="text-xs text-gray-400 whitespace-nowrap">
          {t("settings.members.not-a-member")}
        </span>
      </div>
    );
  }

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
