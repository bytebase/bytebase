import { defineStore } from "pinia";
import { ref, unref, watchEffect } from "vue";
import { branchServiceClient } from "@/grpcweb";
import { MaybeRef } from "@/types";
import {
  MergeBranchRequest,
  Branch,
  BranchView,
  RebaseBranchRequest,
} from "@/types/proto/v1/branch_service";
import { useCache } from "../cache";

type BranchCacheKey = [string /* name */, BranchView];

export const useBranchStore = defineStore("branch", () => {
  const cacheByName = useCache<BranchCacheKey, Branch>("bb.branch.by-name");

  // Cache utils
  const setBranchCache = (branch: Branch, view: BranchView) => {
    if (view === BranchView.BRANCH_VIEW_FULL) {
      // A FULL view branch should override its BASIC view
      cacheByName.invalidateEntity([branch.name, BranchView.BRANCH_VIEW_BASIC]);
    }
    cacheByName.setEntity([branch.name, view], branch);
  };

  // Actions
  const fetchBranchList = async (projectName: string) => {
    const { branches } = await branchServiceClient.listBranches({
      parent: projectName,
      view: BranchView.BRANCH_VIEW_BASIC,
    });
    branches.forEach((branch) =>
      setBranchCache(branch, BranchView.BRANCH_VIEW_BASIC)
    );
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
    setBranchCache(createdBranch, BranchView.BRANCH_VIEW_FULL);
    return createdBranch;
  };

  const updateBranch = async (branch: Branch, updateMask: string[]) => {
    const updatedBranch = await branchServiceClient.updateBranch({
      branch,
      updateMask,
    });
    setBranchCache(updatedBranch, BranchView.BRANCH_VIEW_FULL);
    return updatedBranch;
  };

  const mergeBranch = async (request: MergeBranchRequest) => {
    const branch = await branchServiceClient.mergeBranch(request, {
      silent: true,
    });
    setBranchCache(branch, BranchView.BRANCH_VIEW_FULL);
    return branch;
  };

  const rebaseBranch = async (request: RebaseBranchRequest) => {
    const response = await branchServiceClient.rebaseBranch(request, {
      silent: true,
    });
    if (response.branch) {
      setBranchCache(response.branch, BranchView.BRANCH_VIEW_FULL);
    }
    return response;
  };

  const fetchBranchByName = async (
    name: string,
    useCache = true,
    silent = false
  ) => {
    if (useCache) {
      const cachedEntity = cacheByName.getEntity([
        name,
        BranchView.BRANCH_VIEW_FULL,
      ]);
      if (cachedEntity) {
        return cachedEntity;
      }

      // Avoid making duplicated requests concurrently.
      const cachedRequest = cacheByName.getRequest([
        name,
        BranchView.BRANCH_VIEW_FULL,
      ]);
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
    cacheByName.setRequest([name, BranchView.BRANCH_VIEW_FULL], request);
    return request;
  };

  /**
   *
   * @param name
   * @param view default undefined to any (FULL -> BASIC)
   * @returns
   */
  const getBranchByName = (name: string, view?: BranchView) => {
    if (view === undefined) {
      return (
        cacheByName.getEntity([name, BranchView.BRANCH_VIEW_FULL]) ??
        cacheByName.getEntity([name, BranchView.BRANCH_VIEW_BASIC])
      );
    }
    return cacheByName.getEntity([name, view]);
  };

  const deleteBranch = async (name: string, force = false) => {
    await branchServiceClient.deleteBranch(
      {
        name,
        force,
      },
      {
        silent: true,
      }
    );
    cacheByName.invalidateEntity([name, BranchView.BRANCH_VIEW_FULL]);
    cacheByName.invalidateEntity([name, BranchView.BRANCH_VIEW_BASIC]);
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
