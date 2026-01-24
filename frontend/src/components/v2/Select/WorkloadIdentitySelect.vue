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
import { useWorkloadIdentityStore, workloadIdentityToUser } from "@/store";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";
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
  value?: string | string[] | undefined; // workloadIdentities/{email}
  filter?: (workloadIdentity: WorkloadIdentity) => boolean;
}>();

defineEmits<{
  // the value is workloadIdentities/{email}
  (event: "update:value", value: string[] | string | undefined): void;
}>();

const workloadIdentityStore = useWorkloadIdentityStore();

const hasPermission = computed(() =>
  hasWorkspacePermissionV2("bb.workloadIdentities.list")
);

const getOption = (
  workloadIdentity: WorkloadIdentity
): ResourceSelectOption<WorkloadIdentity> => ({
  resource: workloadIdentity,
  value: workloadIdentity.name,
  label: workloadIdentity.title,
});

const additionalOptions = computedAsync(async () => {
  const options: ResourceSelectOption<WorkloadIdentity>[] = [];

  let names: string[] = [];
  if (Array.isArray(props.value)) {
    names = props.value;
  } else if (props.value) {
    names = [props.value];
  }

  for (const name of names) {
    const workloadIdentity =
      await workloadIdentityStore.getOrFetchWorkloadIdentity(name, true);
    if (workloadIdentity) {
      options.push(getOption(workloadIdentity));
    }
  }

  return options;
}, []);

const handleSearch = async (params: {
  search: string;
  pageToken: string;
  pageSize: number;
}) => {
  const { workloadIdentities, nextPageToken } =
    await workloadIdentityStore.listWorkloadIdentities({
      pageSize: params.pageSize,
      pageToken: params.pageToken,
      showDeleted: false,
      filter: {
        query: params.search.toLowerCase(),
      },
    });

  return {
    nextPageToken,
    options: workloadIdentities.map(getOption),
  };
};

const customLabel = (workloadIdentity: WorkloadIdentity, keyword: string) => {
  const user = workloadIdentityToUser(workloadIdentity);
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
            <HighlightLabelText
              keyword={keyword}
              text={workloadIdentity.email}
            />
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
