import { defineStore } from "pinia";
import axios from "axios";
import {
  ResourceObject,
  ProjectId,
  DeploymentState,
  DeploymentConfig,
  EMPTY_ID,
  empty,
  unknown,
  UNKNOWN_ID,
  DeploymentConfigPatch,
} from "@/types";
import { getPrincipalFromIncludedList } from "./principal";

function convert(
  deployment: ResourceObject,
  includedList: ResourceObject[]
): DeploymentConfig {
  if (parseInt(deployment.id, 10) === EMPTY_ID) {
    return empty("DEPLOYMENT_CONFIG") as DeploymentConfig;
  }

  let schedule: DeploymentConfig["schedule"] = (
    unknown("DEPLOYMENT_CONFIG") as DeploymentConfig
  ).schedule;
  try {
    schedule = JSON.parse(deployment.attributes.payload as string);
    schedule.deployments.forEach((deployment) => {
      deployment.spec.selector.matchExpressions.forEach((rule) => {
        if (!rule.values) {
          rule.values = []; // empty values polyfill
        }
      });
    });
  } catch {
    // nothing
  }

  return {
    ...(deployment.attributes as Omit<
      DeploymentConfig,
      "id" | "schedule" | "creator" | "updater"
    >),
    id: parseInt(deployment.id, 10),
    creator: getPrincipalFromIncludedList(
      deployment.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      deployment.relationships!.updater.data,
      includedList
    ),
    schedule,
  };
}

export const useDeploymentStore = defineStore("deployment", {
  state: (): DeploymentState => ({
    deploymentConfigByProjectId: new Map(),
  }),
  actions: {
    getDeploymentConfigByProjectId(projectId: ProjectId): DeploymentConfig {
      if (projectId == EMPTY_ID) {
        return empty("DEPLOYMENT_CONFIG") as DeploymentConfig;
      }

      return (
        this.deploymentConfigByProjectId.get(projectId) ||
        (unknown("DEPLOYMENT_CONFIG") as DeploymentConfig)
      );
    },
    setDeploymentConfigByProjectId({
      projectId,
      deploymentConfig,
    }: {
      projectId: ProjectId;
      deploymentConfig: DeploymentConfig;
    }) {
      this.deploymentConfigByProjectId.set(projectId, deploymentConfig);
    },
    async fetchDeploymentConfigByProjectId(projectId: ProjectId) {
      const data = (await axios.get(`/api/project/${projectId}/deployment`))
        .data;

      const deploymentConfig = convert(data.data, data.included);
      const { id } = deploymentConfig;
      if (id !== EMPTY_ID && id !== UNKNOWN_ID) {
        this.setDeploymentConfigByProjectId({
          projectId,
          deploymentConfig,
        });
      }
      return this.getDeploymentConfigByProjectId(projectId);
    },
    async patchDeploymentConfigByProjectId({
      projectId,
      deploymentConfigPatch,
    }: {
      projectId: ProjectId;
      deploymentConfigPatch: DeploymentConfigPatch;
    }) {
      const data = (
        await axios.patch(`/api/project/${projectId}/deployment`, {
          data: {
            type: "deploymentConfigPatch",
            attributes: deploymentConfigPatch,
          },
        })
      ).data;
      const updatedDeploymentConfig = convert(data.data, data.included);
      this.setDeploymentConfigByProjectId({
        projectId,
        deploymentConfig: updatedDeploymentConfig,
      });
      return updatedDeploymentConfig;
    },
  },
});
