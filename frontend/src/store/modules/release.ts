import { head } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watch } from "vue";
import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { releaseServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import type { MaybeRef, ComposedRelease, Pagination } from "@/types";
import { isValidReleaseName, unknownRelease, unknownUser } from "@/types";
import type { Release } from "@/types/proto-es/v1/release_service_pb";
import { ReleaseSchema } from "@/types/proto-es/v1/release_service_pb";
import { State } from "@/types/proto-es/v1/common_pb";
import { 
  GetReleaseRequestSchema,
  ListReleasesRequestSchema,
  UpdateReleaseRequestSchema,
  DeleteReleaseRequestSchema,
  UndeleteReleaseRequestSchema
} from "@/types/proto-es/v1/release_service_pb";
import { DEFAULT_PAGE_SIZE } from "./common";
import { useUserStore } from "./user";
import { useProjectV1Store, batchGetOrFetchProjects } from "./v1";
import { getProjectNameReleaseId, projectNamePrefix } from "./v1/common";

export const useReleaseStore = defineStore("release", () => {
  const releaseMapByName = reactive(new Map<string, ComposedRelease>());

  const releaseList = computed(() => {
    return Array.from(releaseMapByName.values());
  });

  const fetchReleasesByProject = async (
    project: string,
    pagination?: Pagination,
    showDeleted?: boolean
  ) => {
    const request = create(ListReleasesRequestSchema, {
      parent: project,
      pageSize: pagination?.pageSize || DEFAULT_PAGE_SIZE,
      pageToken: pagination?.pageToken || "",
      showDeleted: Boolean(showDeleted),
    });
    const resp = await releaseServiceClientConnect.listReleases(request);
    const composedReleaseList = await batchComposeRelease(resp.releases);
    composedReleaseList.forEach((release) => {
      releaseMapByName.set(release.name, release);
    });
    return {
      releases: composedReleaseList,
      nextPageToken: resp.nextPageToken,
    };
  };

  const fetchReleaseByName = async (name: string, silent = false) => {
    const request = create(GetReleaseRequestSchema, { name });
    const response = await releaseServiceClientConnect.getRelease(request, {
      contextValues: createContextValues().set(silentContextKey, silent),
    });
    const [composedRelease] = await batchComposeRelease([response]);
    releaseMapByName.set(composedRelease.name, composedRelease);
    return composedRelease;
  };

  const getReleasesByProject = (project: string) => {
    return releaseList.value.filter((release) => release.project === project);
  };

  const getReleaseByName = (name: string) => {
    return releaseMapByName.get(name) ?? unknownRelease();
  };

  const updateRelase = async (
    release: Partial<Release>,
    updateMask: string[]
  ) => {
    const fullRelease = {
      ...create(ReleaseSchema, {}),
      ...release,
    };
    
    const request = create(UpdateReleaseRequestSchema, {
      release: fullRelease,
      updateMask: { paths: updateMask },
    });
    const resp = await releaseServiceClientConnect.updateRelease(request);
    const composedRelease = await batchComposeRelease([resp]);
    releaseMapByName.set(composedRelease[0].name, composedRelease[0]);
    return composedRelease[0];
  };

  const deleteRelease = async (name: string) => {
    const request = create(DeleteReleaseRequestSchema, { name });
    await releaseServiceClientConnect.deleteRelease(request);
    if (releaseMapByName.get(name)) {
      releaseMapByName.get(name)!.state = State.DELETED;
    }
  };

  const undeleteRelease = async (name: string) => {
    const request = create(UndeleteReleaseRequestSchema, { name });
    const response = await releaseServiceClientConnect.undeleteRelease(request);
    const composedRelease = await batchComposeRelease([response]);
    releaseMapByName.set(composedRelease[0].name, composedRelease[0]);
  };

  return {
    releaseList,
    fetchReleasesByProject,
    fetchReleaseByName,
    getReleasesByProject,
    getReleaseByName,
    updateRelase,
    deleteRelease,
    undeleteRelease,
  };
});

export const useReleaseByName = (name: MaybeRef<string>) => {
  const store = useReleaseStore();
  const ready = ref(true);
  watch(
    () => unref(name),
    async (name) => {
      if (!isValidReleaseName(name)) {
        return;
      }

      const cachedRelease = store.getReleaseByName(name);
      if (!isValidReleaseName(cachedRelease.name)) {
        ready.value = false;
        await store.fetchReleaseByName(name);
        ready.value = true;
      }
    },
    { immediate: true }
  );
  const release = computed(() => store.getReleaseByName(unref(name)));

  return {
    release,
    ready,
  };
};

export const batchComposeRelease = async (releaseList: Release[]): Promise<ComposedRelease[]> => {
  const userStore = useUserStore();
  await userStore.batchGetUsers(releaseList.map((release) => release.creator));

  const composedReleaseList = releaseList.map((release) => {
    const composed = release as ComposedRelease;
    composed.project = `${projectNamePrefix}${head(getProjectNameReleaseId(release.name))}`;
    composed.creatorEntity =
      userStore.getUserByIdentifier(composed.creator) ?? unknownUser();
    return composed;
  });

  await batchGetOrFetchProjects(
    composedReleaseList.map((release) => release.project)
  );

  const projectV1Store = useProjectV1Store();
  return composedReleaseList.map((release) => {
    release.projectEntity = projectV1Store.getProjectByName(release.project);
    return release;
  });
};
