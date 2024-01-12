import { defineStore } from "pinia";
import { reactive, ref, unref, watchEffect } from "vue";
import { branchServiceClient } from "@/grpcweb";
import { MaybeRef } from "@/types";
import {
  MergeBranchRequest,
  Branch,
  BranchView,
  RebaseBranchRequest,
} from "@/types/proto/v1/branch_service";

export const useBranchStore = defineStore("schema_design", () => {
  const branchMapByName = reactive(new Map<string, Branch>());
  const getBranchRequestCacheByName = new Map<string, Promise<Branch>>();

  // Actions
  const fetchBranchList = async (projectName: string) => {
    const { branches } = await branchServiceClient.listBranches({
      parent: projectName,
      view: BranchView.BRANCH_VIEW_BASIC,
    });
    for (const branch of branches) {
      branchMapByName.set(branch.name, branch);
    }
    return branches;
  };

  const createBranch = async (
    projectName: string,
    branchId: string,
    branch: Branch
  ) => {
    const createdBranch = await branchServiceClient.createBranch({
      parent: projectName,
      branchId: branchId,
      branch,
    });
    branchMapByName.set(createdBranch.name, createdBranch);
    return createdBranch;
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
    const branch = await branchServiceClient.mergeBranch(request, {
      silent: true,
    });
    branchMapByName.set(branch.name, branch);
    return branch;
  };

  const rebaseBranch = async (request: RebaseBranchRequest) => {
    const response = await branchServiceClient.rebaseBranch(request, {
      silent: true,
    });
    if (response.branch) {
      branchMapByName.set(response.branch.name, response.branch);
    }
    return response;
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

      // Avoid making duplicated requests concurrently.
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
      force: true,
    });
    branchMapByName.delete(name);
  };

  return {
    fetchBranchList,
    createBranch,
    updateBranch,
    mergeBranch,
    rebaseBranch,
    fetchBranchByName,
    getBranchByName,
    deleteBranch,
  };
});

export const useBranchListByProject = (project: MaybeRef<string>) => {
  const store = useBranchStore();
  const ready = ref(false);
  const branchList = ref<Branch[]>([]);

  watchEffect(() => {
    ready.value = false;
    branchList.value = [];
    store.fetchBranchList(unref(project)).then((response) => {
      ready.value = true;
      branchList.value = response;
    });
  });

  return { branchList, ready };
};
