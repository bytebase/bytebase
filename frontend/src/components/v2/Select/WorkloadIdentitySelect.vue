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
import {
  useWorkloadIdentityStore,
  workloadIdentityNamePrefix,
  workloadIdentityToUser,
} from "@/store";
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
  value?: string | string[] | undefined; // workloadIdentity fullname
  parent?: string; // e.g. "projects/{project}" for project-scoped identities
  filter?: (wi: WorkloadIdentity) => boolean;
}>();

defineEmits<{
  // the value is workloadIdentity fullname
  (event: "update:value", value: string[] | string | undefined): void;
}>();

const workloadIdentityStore = useWorkloadIdentityStore();

const hasPermission = computed(() =>
  hasWorkspacePermissionV2("bb.workloadIdentities.list")
);

const getOption = (
  wi: WorkloadIdentity
): ResourceSelectOption<WorkloadIdentity> => ({
  resource: wi,
  value: `${workloadIdentityNamePrefix}${wi.email}`,
  label: wi.title,
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
    const wi = await workloadIdentityStore.getOrFetchWorkloadIdentity(
      name,
      true
    );
    options.push(getOption(wi));
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
      parent: props.parent,
      filter: { query: params.search },
      pageToken: params.pageToken,
      pageSize: params.pageSize,
      showDeleted: false,
    });
  return {
    nextPageToken,
    options: workloadIdentities.map(getOption),
  };
};

const customLabel = (wi: WorkloadIdentity, keyword: string) => {
  const user = workloadIdentityToUser(wi);
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
            (<HighlightLabelText keyword={keyword} text={wi.email} />)
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
