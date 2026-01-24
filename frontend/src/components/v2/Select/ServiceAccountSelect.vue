<template>
  <RemoteResourceSelector
    v-bind="$attrs"
    :multiple="multiple"
    :disabled="disabled"
    :size="size"
    :value="value"
    :tag="!hasPermission"
    :remote="hasPermission"
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
import { serviceAccountToUser, useServiceAccountStore } from "@/store";
import type { ServiceAccount } from "@/types/proto-es/v1/service_account_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
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
  value?: string | string[] | undefined; // serviceAccounts/{email}
  filter?: (serviceAccount: ServiceAccount) => boolean;
}>();

defineEmits<{
  // the value is serviceAccounts/{email}
  (event: "update:value", value: string[] | string | undefined): void;
}>();

const serviceAccountStore = useServiceAccountStore();

const hasPermission = computed(() =>
  hasWorkspacePermissionV2("bb.serviceAccounts.list")
);

const getOption = (
  serviceAccount: ServiceAccount
): ResourceSelectOption<ServiceAccount> => ({
  resource: serviceAccount,
  value: serviceAccount.name,
  label: serviceAccount.title,
});

const additionalOptions = computedAsync(async () => {
  const options: ResourceSelectOption<ServiceAccount>[] = [];

  let names: string[] = [];
  if (Array.isArray(props.value)) {
    names = props.value;
  } else if (props.value) {
    names = [props.value];
  }

  for (const name of names) {
    const serviceAccount = await serviceAccountStore.getOrFetchServiceAccount(
      name,
      true
    );
    if (serviceAccount) {
      options.push(getOption(serviceAccount));
    }
  }

  return options;
}, []);

const handleSearch = async (params: {
  search: string;
  pageToken: string;
  pageSize: number;
}) => {
  const { serviceAccounts, nextPageToken } =
    await serviceAccountStore.listServiceAccounts({
      pageSize: params.pageSize,
      pageToken: params.pageToken,
      showDeleted: false,
      filter: {
        query: params.search.toLowerCase(),
      },
    });

  return {
    nextPageToken,
    options: serviceAccounts.map(getOption),
  };
};

const customLabel = (serviceAccount: ServiceAccount, keyword: string) => {
  const user = serviceAccountToUser(serviceAccount);
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
            (
            <HighlightLabelText keyword={keyword} text={serviceAccount.email} />
            )
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
