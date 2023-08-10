import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watchEffect } from "vue";
import { projectServiceClient } from "@/grpcweb";
import { MaybeRef, ResourceId } from "@/types";
import { DeploymentConfig } from "@/types/proto/v1/project_service";

export const useDeploymentConfigV1Store = defineStore(
  "deploymentConfig_v1",
  () => {
    const deploymentConfigByProjectName = reactive(
      new Map<ResourceId, DeploymentConfig>()
    );

    const fetchDeploymentConfigByProjectName = async (project: ResourceId) => {
      const name = `${project}/deploymentConfig`;
      try {
        const deploymentConfig = await projectServiceClient.getDeploymentConfig(
          {
            name,
          }
        );
        deploymentConfigByProjectName.set(project, deploymentConfig);
        return deploymentConfig;
      } catch {
        return undefined;
      }
    };
    const updatedDeploymentConfigByProjectName = async (
      project: ResourceId,
      config: DeploymentConfig
    ) => {
      const updated = await projectServiceClient.updateDeploymentConfig({
        config,
      });
      deploymentConfigByProjectName.set(project, updated);
      return updated;
    };

    return {
      deploymentConfigByProjectName,
      fetchDeploymentConfigByProjectName,
      updatedDeploymentConfigByProjectName,
    };
  }
);

export const useDeploymentConfigV1ByProject = (
  project: MaybeRef<ResourceId>
) => {
  const store = useDeploymentConfigV1Store();
  const ready = ref(false);
  watchEffect(() => {
    ready.value = false;
    store.fetchDeploymentConfigByProjectName(unref(project)).then(() => {
      ready.value = true;
    });
  });
  const deploymentConfig = computed(() => {
    return store.deploymentConfigByProjectName.get(unref(project));
  });
  return { deploymentConfig, ready };
};
