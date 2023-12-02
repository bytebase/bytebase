import { defineStore } from "pinia";
import { reactive, ref, watchEffect } from "vue";
import { branchServiceClient } from "@/grpcweb";
import {
  MergeBranchRequest,
  Branch,
  BranchView,
} from "@/types/proto/v1/branch_service";
import {
  getProjectAndSchemaDesignSheetId,
  projectNamePrefix,
  sheetNamePrefix,
} from "./v1/common";

export const useSchemaDesignStore = defineStore("schema_design", () => {
  const branchMapByName = reactive(new Map<string, Branch>());
  const getBranchRequestCacheByName = new Map<string, Promise<Branch>>();

  // Actions
  const fetchSchemaDesignList = async (projectName: string = "projects/-") => {
    const { branches } = await branchServiceClient.listBranches({
      parent: projectName,
      view: BranchView.BRANCH_VIEW_BASIC,
    });
    return branches;
  };

  const createSchemaDesign = async (
    projectResourceId: string,
    branch: Branch
  ) => {
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

  const createSchemaDesignDraft = async (branch: Branch) => {
    const [projectName, sheetId] = getProjectAndSchemaDesignSheetId(
      branch.name
    );
    const projectResourceId = `${projectNamePrefix}${projectName}`;
    const parentBranch = `${projectResourceId}/${sheetNamePrefix}${sheetId}`;
    return createSchemaDesign(projectResourceId, {
      ...branch,
      parentBranch: parentBranch,
    });
  };

  const updateSchemaDesign = async (branch: Branch, updateMask: string[]) => {
    const updatedBranch = await branchServiceClient.updateBranch({
      branch,
      updateMask,
    });
    branchMapByName.set(updatedBranch.name, updatedBranch);
    return updatedBranch;
  };

  const mergeSchemaDesign = async (request: MergeBranchRequest) => {
    const updatedBranch = await branchServiceClient.mergeBranch(request, {
      silent: true,
    });
    branchMapByName.set(updatedBranch.name, updatedBranch);
  };

  const fetchSchemaDesignByName = async (
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

  const getSchemaDesignByName = (name: string) => {
    return branchMapByName.get(name);
  };

  const deleteSchemaDesign = async (name: string) => {
    await branchServiceClient.deleteBranch({
      name,
    });
    branchMapByName.delete(name);
  };

  return {
    fetchSchemaDesignList,
    createSchemaDesign,
    createSchemaDesignDraft,
    updateSchemaDesign,
    mergeSchemaDesign,
    fetchSchemaDesignByName,
    getSchemaDesignByName,
    deleteSchemaDesign,
  };
});

export const useSchemaDesignList = (
  projectName: string | undefined = undefined
) => {
  const store = useSchemaDesignStore();
  const ready = ref(false);
  const schemaDesignList = ref<Branch[]>([]);

  watchEffect(() => {
    ready.value = false;
    schemaDesignList.value = [];
    store.fetchSchemaDesignList(projectName).then((response) => {
      ready.value = true;
      schemaDesignList.value = response;
    });
  });

  return { schemaDesignList, ready };
};
