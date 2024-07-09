<template>
  <Drawer :show="show" @close="$emit('close')">
    <DrawerContent
      :title="$t('sql-review.attach-resource.self')"
      class="w-[40rem] max-w-[100vw] relative"
    >
      <template #default>
        <div class="space-y-6">
          <p class="textinfolabel">
            {{ $t("sql-review.attach-resource.label") }}
          </p>
          <div>
            <div class="textlabel mb-1">
              {{ $t("common.environment") }}
            </div>
            <p class="textinfolabel">
              {{ $t("sql-review.attach-resource.label-environment") }}
            </p>
            <EnvironmentSelect
              class="mt-3"
              required
              :environments="environments"
              :use-resource-id="true"
              :multiple="true"
              @update:environments="onResourcesChange($event, projects)"
            />
          </div>
          <NDivider />
          <div>
            <div class="textlabel mb-1">
              {{ $t("common.project") }}
            </div>
            <p class="textinfolabel">
              {{ $t("sql-review.attach-resource.label-project") }}
            </p>
            <ProjectSelect
              class="mt-3"
              style="width: 100%"
              required
              :projects="projects"
              :use-resource-id="true"
              :multiple="true"
              @update:projects="onResourcesChange($event, environments)"
            />
          </div>
        </div>
      </template>
      <template #footer>
        <div class="flex items-center justify-end gap-x-3">
          <NButton @click="$emit('close')">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            :disabled="!hasPermission"
            type="primary"
            @click="upsertReviewResource"
          >
            {{ $t("common.confirm") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { NDivider } from "naive-ui";
import { watch, ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import { Drawer, DrawerContent, ProjectSelect } from "@/components/v2";
import { useSQLReviewStore, pushNotification, useCurrentUserV1 } from "@/store";
import {
  environmentNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import type { SQLReviewPolicy } from "@/types";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  show: boolean;
  review: SQLReviewPolicy;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const me = useCurrentUserV1();
const resources = ref<string[]>([]);
const sqlReviewStore = useSQLReviewStore();
const { t } = useI18n();

watch(
  () => props.review.resources,
  () => {
    resources.value = [...props.review.resources];
  },
  { immediate: true, deep: true }
);

const hasPermission = computed(() => {
  return hasWorkspacePermissionV2(me.value, "bb.policies.update");
});

const environments = computed(() =>
  resources.value.filter((resource) =>
    resource.startsWith(environmentNamePrefix)
  )
);
const projects = computed(() =>
  resources.value.filter((resource) => resource.startsWith(projectNamePrefix))
);

const onResourcesChange = (
  newResources: string[],
  otherResources: string[]
) => {
  resources.value = [...newResources, ...otherResources];
};

const upsertReviewResource = async () => {
  await sqlReviewStore.upsertReviewConfigTag({
    oldResources: props.review.resources,
    newResources: resources.value,
    review: props.review.id,
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("sql-review.policy-updated"),
  });
  emit("close");
};
</script>
