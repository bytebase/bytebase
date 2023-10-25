<template>
  <Drawer :show="showCreatePanel" @close="showCreatePanel = false">
    <DrawerContent :title="$t('changelist.new')" class="w-[40rem] relative">
      <template #default>
        <div
          class="grid items-center gap-y-4 gap-x-4"
          style="grid-template-columns: minmax(6rem, auto) 1fr"
        >
          <div class="contents">
            <div class="textlabel">
              {{ $t("common.project") }}
              <span class="ml-0.5 text-error">*</span>
            </div>
            <div>
              <ProjectSelect
                v-model:project="projectUID"
                :include-all="false"
                :disabled="disableProjectSelect"
                style="width: 14rem"
              />
            </div>
          </div>
          <div class="contents">
            <div class="textlabel">
              {{ $t("changelist.name") }}
              <span class="ml-0.5 text-error">*</span>
            </div>
            <div>
              <NInput
                v-model:value="title"
                :placeholder="$t('changelist.name-placeholder')"
                style="width: 14rem"
              />
            </div>
          </div>
          <div class="contents">
            <div class="col-span-2">
              <ResourceIdField
                ref="resourceIdField"
                v-model:value="resourceId"
                resource-type="changelist"
                :resource-title="title"
                :validate="validateResourceId"
              />
            </div>
          </div>
        </div>

        <div
          v-if="isLoading"
          v-zindexable="{ enabled: true }"
          class="absolute bg-white/50 inset-0 flex flex-col items-center justify-center"
        >
          <BBSpin />
        </div>
      </template>

      <template #footer>
        <div class="flex items-center justify-end gap-x-3">
          <NButton>{{ $t("common.cancel") }}</NButton>
          <NTooltip :disabled="errors.length === 0">
            <template #trigger>
              <NButton
                type="primary"
                tag="div"
                :disabled="errors.length > 0"
                @click="doCreate"
              >
                {{ $t("common.add") }}
              </NButton>
            </template>
            <template #default>
              <ErrorList :errors="errors" />
            </template>
          </NTooltip>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { asyncComputed } from "@vueuse/core";
import { NInput, NTooltip } from "naive-ui";
import { Status } from "nice-grpc-common";
import { zindexable as vZindexable } from "vdirs";
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import ErrorList from "@/components/misc/ErrorList.vue";
import {
  Drawer,
  DrawerContent,
  ProjectSelect,
  ResourceIdField,
} from "@/components/v2";
import {
  pushNotification,
  useChangelistStore,
  useProjectV1Store,
} from "@/store";
import { ResourceId, UNKNOWN_ID, ValidatedMessage } from "@/types";
import { Changelist } from "@/types/proto/v1/changelist_service";
import { getErrorCode } from "@/utils/grpcweb";
import { useChangelistDashboardContext } from "./context";

const props = defineProps<{
  projectUid?: string;
  disableProjectSelect?: boolean;
}>();

const router = useRouter();
const { t } = useI18n();
const { showCreatePanel, events } = useChangelistDashboardContext();

const title = ref("");
const projectUID = ref<string | undefined>(props.projectUid);
const isLoading = ref(false);
const resourceId = ref("");
const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();

const errors = asyncComputed(() => {
  const errors: string[] = [];
  if (!projectUID.value || projectUID.value === String(UNKNOWN_ID)) {
    errors.push(t("changelist.error.project-is-required"));
  }
  if (!title.value.trim()) {
    errors.push(t("changelist.error.name-is-required"));
  }
  if (resourceIdField.value && !resourceIdField.value.isValidated) {
    errors.push(t("changelist.error.invalid-resource-id"));
  }

  return errors;
}, []);

const validateResourceId = async (
  resourceId: ResourceId
): Promise<ValidatedMessage[]> => {
  if (!resourceId) {
    return [];
  }

  if (!projectUID.value) return [];
  const project = useProjectV1Store().getProjectByUID(projectUID.value);

  try {
    const name = `${project.name}/changelists/${resourceId}`;
    const maybeExistedChangelist =
      await useChangelistStore().getOrFetchChangelistByName(
        name,
        true /* silent */
      );
    if (maybeExistedChangelist) {
      return [
        {
          type: "error",
          message: t("resource-id.validation.duplicated", {
            resource: t("resource.changelist"),
          }),
        },
      ];
    }
  } catch (error) {
    if (getErrorCode(error) !== Status.NOT_FOUND) {
      throw error;
    }
  }
  return [];
};

const doCreate = async () => {
  if (errors.value.length > 0) return;
  if (!resourceIdField.value) return;

  isLoading.value = true;
  try {
    const project = useProjectV1Store().getProjectByUID(projectUID.value!);

    const created = await useChangelistStore().createChangelist({
      parent: project.name,
      changelist: Changelist.fromPartial({
        description: title.value,
      }),
      changelistId: resourceId.value,
    });
    showCreatePanel.value = false;
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.created"),
    });
    router.push(created.name);
    events.emit("refresh");
  } finally {
    isLoading.value = false;
  }
};

const reset = () => {
  title.value = "";
  projectUID.value = undefined;
};

watch(showCreatePanel, (show) => {
  if (show) {
    reset();
  }
});
</script>
