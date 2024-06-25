<template>
  <div class="flex flex-col gap-y-2">
    <div class="textlabel flex items-center space-x-1">
      <label>
        {{ $t("environment.access-control.title") }}
      </label>
      <FeatureBadge feature="bb.feature.access-control" />
    </div>
    <div>
      <div class="inline-flex items-center gap-x-2">
        <Switch
          :value="disableCopyDataPolicy"
          :text="true"
          :disabled="!allowEditDisableCopyData"
          @update:value="upsertPolicy"
        />
        <span class="textlabel">{{
          $t("environment.access-control.disable-copy-data-from-sql-editor")
        }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  hasFeature,
  pushNotification,
  usePolicyV1Store,
  useCurrentUserV1,
} from "@/store";
import {
  PolicyResourceType,
  PolicyType,
} from "@/types/proto/v1/org_policy_service";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  resource: string;
  allowEdit: boolean;
}>();

const policyStore = usePolicyV1Store();
const me = useCurrentUserV1();
const { t } = useI18n();

const disableCopyDataPolicy = computed(() => {
  return policyStore.getPolicyByParentAndType({
    parentPath: props.resource,
    policyType: PolicyType.DISABLE_COPY_DATA,
  })?.disableCopyDataPolicy?.active;
});

const allowEditDisableCopyData = computed(() => {
  return (
    props.allowEdit &&
    hasWorkspacePermissionV2(me.value, "bb.policies.update") &&
    hasFeature("bb.feature.access-control")
  );
});

const upsertPolicy = async (on: boolean) => {
  await policyStore.createPolicy(props.resource, {
    type: PolicyType.DISABLE_COPY_DATA,
    resourceType: PolicyResourceType.ENVIRONMENT,
    disableCopyDataPolicy: {
      active: on,
    },
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};
</script>
