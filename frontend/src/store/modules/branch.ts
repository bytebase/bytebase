import { defineStore } from "pinia";
import { reactive, ref, watchEffect } from "vue";
import { branchServiceClient } from "@/grpcweb";
import {
  MergeBranchRequest,
  Branch,
  BranchView,
} from "@/types/proto/v1/branch_service";
import {
  getProjectAndBranchId,
  projectNamePrefix,
  branchNamePrefix,
} from "./v1/common";

export const useBranchStore = defineStore("schema_design", () => {
  const branchMapByName = reactive(new Map<string, Branch>());
  const getBranchRequestCacheByName = new Map<string, Promise<Branch>>();

  // Actions
  const fetchBranchList = async (projectName: string = "projects/-") => {
    const { branches } = await branchServiceClient.listBranches({
      parent: projectName,
      view: BranchView.BRANCH_VIEW_BASIC,
    });
    return branches;
  };

  const createBranch = async (projectResourceId: string, branch: Branch) => {
    const createdBranch = await branchServiceClient.createBranch({
      parent: projectResourceId,
      branch,
    });
    console.debug("baseline schema", branch.baselineSchema);
    console.debug("target metadata", branch.schemaMetadata);
    console.debug("got schema", createdBranch.schema);
    branchMapByName.set(createdBranch.name, createdBranch);
    return createdBranch;
  };

  const createBranchDraft = async (branch: Branch) => {
    const [projectName, branchId] = getProjectAndBranchId(branch.name);
    const projectResourceId = `${projectNamePrefix}${projectName}`;
    const parentBranch = `${projectResourceId}/${branchNamePrefix}${branchId}`;
    return createBranch(projectResourceId, {
      ...branch,
      parentBranch: parentBranch,
    });
  };

  const updateBranch = async (branch: Branch, updateMask: string[]) => {
    const updatedBranch = await branchServiceClient.updateBranch({
      branch,
      updateMask,
    });
    branchMapByName.set(updatedBranch.name, updatedBranch);
    return updatedBranch;
  };

  const mergeBranch = async (request: MergeBranchRequest) => {
    const updatedBranch = await branchServiceClient.mergeBranch(request, {
      silent: true,
    });
    branchMapByName.set(updatedBranch.name, updatedBranch);
  };

  const fetchBranchByName = async (
    name: string,
    useCache = true,
    silent = false
  ) => {
    if (useCache) {
      const cachedEntity = branchMapByName.get(name);
      if (cachedEntity) {
        return cachedEntity;
      }

      // Avoid making duplicated requests concurrently
      const cachedRequest = getBranchRequestCacheByName.get(name);
      if (cachedRequest) {
        return cachedRequest;
      }
    }
    const request = branchServiceClient.getBranch(
      {
        name,
      },
      {
        silent,
      }
    );
    request.then((branch) => {
      branchMapByName.set(branch.name, branch);
    });
    getBranchRequestCacheByName.set(name, request);
    return request;
  };

  const getBranchByName = (name: string) => {
    return branchMapByName.get(name);
  };

  const deleteBranch = async (name: string) => {
    await branchServiceClient.deleteBranch({
      name,
    });
    branchMapByName.delete(name);
  };

  return {
    fetchBranchList,
    createBranch,
    createBranchDraft,
    updateBranch,
    mergeBranch,
    fetchBranchByName,
    getBranchByName,
    deleteBranch,
  };
});

export const useBranchList = (projectName: string | undefined = undefined) => {
  const store = useBranchStore();
  const ready = ref(false);
  const branchList = ref<Branch[]>([]);

  watchEffect(() => {
    ready.value = false;
    branchList.value = [];
    store.fetchBranchList(projectName).then((response) => {
      ready.value = true;
      branchList.value = response;
    });
  });

  return { branchList, ready };
};
