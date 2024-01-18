import { cloneDeep, uniqueId } from "lodash-es";
import { computed, reactive, ref, unref, watchEffect } from "vue";
import { useDBSchemaV1Store } from "@/store";
import { ComposedDatabase, ComposedProject, MaybeRef } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import {
  DatabaseMetadata,
  DatabaseMetadataView,
} from "@/types/proto/v1/database_service";
import { filterDatabaseMetadata } from "@/utils";

/**
 * Create a "virtual" branch to show the diff between a branch's head and a database's head
 * @param database
 * @param branch
 */
export const useVirtualBranch = (
  project: MaybeRef<ComposedProject>,
  branch: MaybeRef<Branch>,
  database: MaybeRef<ComposedDatabase | undefined>
) => {
  const state = reactive({
    isLoadingDatabaseMetadata: false,
  });
  const databaseHeadMetadata = ref<DatabaseMetadata>();

  const fetchDatabaseHeadMetadata = async (
    db: ComposedDatabase | undefined,
    signal: AbortController["signal"]
  ) => {
    if (!db) {
      databaseHeadMetadata.value = undefined;
      state.isLoadingDatabaseMetadata = false;
      return;
    }

    state.isLoadingDatabaseMetadata = true;
    const metadata = await useDBSchemaV1Store().getOrFetchDatabaseMetadata({
      database: db.name,
      skipCache: true, // ensure using the latest
      view: DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL,
    });
    if (signal.aborted) return;
    databaseHeadMetadata.value = filterDatabaseMetadata(metadata);
    state.isLoadingDatabaseMetadata = false;
  };

  watchEffect((onCancel) => {
    const controller = new AbortController();
    fetchDatabaseHeadMetadata(unref(database), controller.signal);
    onCancel(() => controller.abort());
  });

  const virtualBranch = computed(() => {
    const db = unref(database);
    if (!db) {
      return undefined;
    }
    if (state.isLoadingDatabaseMetadata) {
      return undefined;
    }
    return Branch.fromPartial({
      name: `${unref(project).name}/branches/-${uniqueId()}`,
      engine: db.instanceEntity.engine,
      baselineDatabase: db.name,
      schemaMetadata: cloneDeep(unref(branch).schemaMetadata),
      baselineSchemaMetadata: cloneDeep(databaseHeadMetadata.value),
    });
  });

  const isLoading = computed(() => {
    return state.isLoadingDatabaseMetadata;
  });

  const ready = computed(() => {
    return !state.isLoadingDatabaseMetadata;
  });

  return {
    isLoading,
    ready,
    branch: virtualBranch,
  };
};
