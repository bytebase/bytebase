<template>
  <BBModal :title="$t('database.transfer-project')" @close="$emit('cancel')">
    <div class="w-112 flex flex-col items-center gap-y-4">
      <div class="col-span-1 flex flex-col gap-y-2 w-64">
        <label for="user" class="textlabel">
          {{ $t("common.project") }}
        </label>
        <!-- Only allow to transfer database to the project having OWNER role -->
        <ProjectSelect
          style="width: 100%"
          :project="targetProject.uid"
          :allowed-project-role-list="
            hasWorkspaceManageProjectPermission ? [] : [PresetRoleType.OWNER]
          "
          :include-default-project="allowTransferToDefaultProject"
          @update:project="handleSelectProject"
        />
      </div>
      <SelectDatabaseLabel
        v-model:labels="editingLabels"
        :database="database"
        :target-project-id="targetProject.uid"
      />
    </div>
    <div>
      <div
        class="w-full pt-4 mt-6 flex justify-end gap-x-4 border-t border-block-border"
      >
        <NButton @click.prevent="$emit('cancel')">
          {{ $t("common.cancel") }}
        </NButton>
        <!--
              We are not allowed to transfer a db either its labels are not valid
              or transferring into its project itself.
            -->
        <NButton type="primary" :disabled="!allowTransfer" @click="doTransfer">
          {{ $t("common.transfer") }}
        </NButton>
      </div>
    </div>

    <div
      v-if="transferring"
      class="flex items-center justify-center absolute inset-0 bg-white/50 rounded-lg"
    >
      <BBSpin />
    </div>
  </BBModal>
</template>

<script setup lang="ts">
import { cloneDeep } from "lodash-es";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { ProjectSelect } from "@/components/v2";
import {
  pushNotification,
  useCurrentUserV1,
  useDatabaseV1Store,
  useGracefulRequest,
  useProjectV1Store,
} from "@/store";
import {
  ComposedDatabase,
  DEFAULT_PROJECT_V1_NAME,
  PresetRoleType,
  UNKNOWN_ID,
} from "@/types";
import { TenantMode } from "@/types/proto/v1/project_service";
import { hasWorkspacePermissionV1 } from "@/utils";
import SelectDatabaseLabel from "./SelectDatabaseLabel.vue";

const props = defineProps<{
  database: ComposedDatabase;
}>();

const emit = defineEmits<{
  (event: "cancel"): void;
  (event: "updated", updated: ComposedDatabase): void;
}>();

const { t } = useI18n();
const currentUserV1 = useCurrentUserV1();
const transferring = ref(false);
const sourceProject = computed(() => props.database.projectEntity);
const targetProject = ref(sourceProject.value);
const editingLabels = ref({ ...props.database.labels });

const allowTransfer = computed(() => {
  return sourceProject.value.name !== targetProject.value.name;
});

const hasWorkspaceManageProjectPermission = computed(() =>
  hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-project",
    currentUserV1.value.userRole
  )
);

const allowTransferToDefaultProject = computed(() => {
  if (props.database.project === DEFAULT_PROJECT_V1_NAME) {
    return true;
  }

  // Allow to transfer a database to DEFAULT project only if the current user
  // can manage all projects.
  // AKA DBA or workspace owner.
  return hasWorkspaceManageProjectPermission.value;
});

const handleSelectProject = (uid: string | undefined) => {
  if (!uid || uid === String(UNKNOWN_ID)) return;
  targetProject.value = useProjectV1Store().getProjectByUID(uid);
};

const doTransfer = async () => {
  // updateProject(state.currentProjectId, labels);
  // state.showTransferDatabaseModal = false;
  transferring.value = true;
  try {
    await useGracefulRequest(async () => {
      const target = targetProject.value;
      const databasePatch = cloneDeep(props.database);
      databasePatch.project = target.name;
      const updateMask = ["project"];
      if (target.tenantMode === TenantMode.TENANT_MODE_ENABLED) {
        databasePatch.labels = { ...editingLabels.value };
        updateMask.push("labels");
      }
      const updated = await useDatabaseV1Store().updateDatabase({
        database: databasePatch,
        updateMask,
      });

      await new Promise((r) => setTimeout(r, 1000));
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t(
          "database.successfully-transferred-updateddatabase-name-to-project-updateddatabase-project-name",
          [updated.databaseName, updated.projectEntity.title]
        ),
      });
      emit("updated", updated);
    });
  } finally {
    transferring.value = false;
  }
};

watch(
  () => props.database.labels,
  (labels) => (editingLabels.value = { ...labels }),
  { deep: true }
);
watch(targetProject, (tar) => {
  if (tar.tenantMode !== TenantMode.TENANT_MODE_ENABLED) {
    // Restore dirty value if need not to edit labels any more.
    editingLabels.value = { ...props.database.labels };
  }
});
watch(sourceProject, (src) => {
  targetProject.value = src;
});
</script>
