import Emittery from "emittery";
import type { InjectionKey, Ref } from "vue";
import { computed, inject, provide, ref, watchEffect } from "vue";
import { useRoute } from "vue-router";
import {
  useChangelistStore,
  useCurrentUserV1,
  useProjectV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { ComposedProject, Permission } from "@/types";
import { unknownChangelist, unknownProject } from "@/types";
import type {
  Changelist,
  Changelist_Change as Change,
} from "@/types/proto/v1/changelist_service";
import {
  extractUserResourceName,
  hasPermissionToCreateChangeDatabaseIssueInProject,
  hasProjectPermissionV2,
} from "@/utils";

export type ChangelistDetailEvents = Emittery<{
  "reorder-cancel": undefined;
  "reorder-confirm": undefined;
}>;

export type ChangelistDetailContext = {
  changelist: Ref<Changelist>;
  project: Ref<ComposedProject>;
  allowEdit: Ref<boolean>;
  allowDelete: Ref<boolean>;
  allowApply: Ref<boolean>;
  reorderMode: Ref<boolean>;
  selectedChanges: Ref<Change[]>;
  showAddChangePanel: Ref<boolean>;
  showApplyToDatabasePanel: Ref<boolean>;
  isUpdating: Ref<boolean>;

  events: ChangelistDetailEvents;
};

export const KEY = Symbol(
  "bb.changelist.detail"
) as InjectionKey<ChangelistDetailContext>;

export const useChangelistDetailContext = () => {
  return inject(KEY)!;
};

export const provideChangelistDetailContext = () => {
  const me = useCurrentUserV1();
  const route = useRoute();
  const projectV1Store = useProjectV1Store();

  const project = computed(() => {
    const projectId = route.params.projectId as string;
    if (!projectId) {
      return unknownProject();
    }

    return projectV1Store.getProjectByName(`${projectNamePrefix}${projectId}`);
  });

  const name = computed(() => {
    return `${project.value.name}/changelists/${route.params.changelistName}`;
  });

  watchEffect(async () => {
    await useChangelistStore().fetchChangelistByName(name.value);
  });

  const changelist = computed(() => {
    return (
      useChangelistStore().getChangelistByName(name.value) ??
      unknownChangelist()
    );
  });

  const checkPermission = (permission: Permission): boolean => {
    return (
      hasProjectPermissionV2(project.value, permission) ||
      extractUserResourceName(changelist.value.creator) === me.value.email
    );
  };

  const allowDelete = computed(() => {
    return checkPermission("bb.changelists.delete");
  });
  const allowApply = computed(() => {
    return hasPermissionToCreateChangeDatabaseIssueInProject(project.value);
  });
  const allowEdit = computed(() => {
    return checkPermission("bb.changelists.update");
  });

  const context: ChangelistDetailContext = {
    changelist,
    project,
    allowEdit,
    allowDelete,
    allowApply,
    reorderMode: ref(false),
    selectedChanges: ref([]),
    showAddChangePanel: ref(false),
    showApplyToDatabasePanel: ref(false),
    isUpdating: ref(false),

    events: new Emittery(),
  };

  provide(KEY, context);

  return context;
};
