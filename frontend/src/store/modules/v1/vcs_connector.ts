import { defineStore } from "pinia";
import { reactive } from "vue";
import { vcsConnectorServiceClient, vcsProviderServiceClient } from "@/grpcweb";
import { useCache } from "@/store/cache";
import { getVCSConnectorId } from "@/store/modules/v1/common";
import type { VCSConnector } from "@/types/proto/v1/vcs_connector_service";
import { vcsConnectorPrefix, projectNamePrefix } from "./common";

type VCSConnectorCacheKey = [string /* /projects/{project}/vcsConnectors/ */];

export const useVCSConnectorStore = defineStore("vcs_connector_v1", () => {
  const cacheByName = useCache<VCSConnectorCacheKey, VCSConnector[]>(
    "bb.project-vcs-connector.by-name"
  );
  const repositoryMapByVCS = reactive(new Map<string, VCSConnector[]>());

  const fetchConnectorsInProvider = async (
    vcsName: string
  ): Promise<VCSConnector[]> => {
    const resp = await vcsProviderServiceClient.listVCSConnectorsInProvider({
      name: vcsName,
    });

    repositoryMapByVCS.set(vcsName, resp.vcsConnectors);
    return resp.vcsConnectors;
  };

  const getConnectorsInProvider = (vcsName: string): VCSConnector[] => {
    return repositoryMapByVCS.get(vcsName) || [];
  };

  const fetchConnectorList = async (project: string) => {
    const resp = await vcsConnectorServiceClient.listVCSConnectors({
      parent: project,
    });
    return resp.vcsConnectors;
  };

  const getOrFetchConnectors = async (project: string) => {
    const cacheName = `${project}/${vcsConnectorPrefix}`;
    const cachedRequest = cacheByName.getRequest([cacheName]);
    if (cachedRequest) {
      return cachedRequest;
    }
    const cachedEntity = cacheByName.getEntity([cacheName]);
    if (cachedEntity) {
      return cachedEntity;
    }
    const request = fetchConnectorList(project);
    cacheByName.setRequest([cacheName], request);
    return request;
  };

  const getConnectorList = (project: string) => {
    const cacheName = `${project}/${vcsConnectorPrefix}`;
    return cacheByName.getEntity([cacheName]) ?? [];
  };

  const getConnector = (project: string, vcsConnectorId: string) => {
    const name = `${project}/${vcsConnectorPrefix}${vcsConnectorId}`;
    const cached = getConnectorList(project);
    return cached.find((item) => item.name === name);
  };

  const getConnectorByName = (connectorName: string) => {
    const { projectId, vcsConnectorId } = getVCSConnectorId(connectorName);
    return getConnector(`${projectNamePrefix}${projectId}`, vcsConnectorId);
  };

  const setCache = (project: string, connector: VCSConnector) => {
    const cached = getConnectorList(project);
    cached.push(connector);
    cacheByName.setEntity([`${project}/${vcsConnectorPrefix}`], cached);
  };

  const deleteCache = (name: string) => {
    const { projectId } = getVCSConnectorId(name);
    const project = `${projectNamePrefix}${projectId}`;
    const cached = getConnectorList(project);

    const index = cached.findIndex((item) => item.name === name);
    if (index >= 0) {
      cached.splice(index, 1);
    }
    cacheByName.setEntity([`${project}/${vcsConnectorPrefix}`], cached);
  };

  const getOrFetchConnector = async (
    project: string,
    vcsConnectorId: string
  ) => {
    const name = `${project}/${vcsConnectorPrefix}${vcsConnectorId}`;
    const entity = getConnector(project, vcsConnectorId);
    if (entity) {
      return entity;
    }

    const connector = await vcsConnectorServiceClient.getVCSConnector({ name });
    setCache(project, connector);
    return connector;
  };

  const getOrFetchConnectorByName = async (connectorName: string) => {
    const { projectId, vcsConnectorId } = getVCSConnectorId(connectorName);
    return getOrFetchConnector(
      `${projectNamePrefix}${projectId}`,
      vcsConnectorId
    );
  };

  const createConnector = async (
    project: string,
    vcsConnectorId: string,
    connector: Partial<VCSConnector>
  ) => {
    const newConnector = await vcsConnectorServiceClient.createVCSConnector({
      parent: project,
      vcsConnector: connector,
      vcsConnectorId,
    });
    setCache(project, newConnector);
    return newConnector;
  };

  const updateConnector = async (
    vcsConnector: Partial<VCSConnector>,
    updateMask: string[]
  ) => {
    const updatedConnector = await vcsConnectorServiceClient.updateVCSConnector(
      {
        vcsConnector,
        updateMask,
      }
    );

    deleteCache(updatedConnector.name);

    const { projectId } = getVCSConnectorId(updatedConnector.name);
    setCache(`${projectNamePrefix}${projectId}`, updatedConnector);
    return updatedConnector;
  };

  const deleteConnector = async (name: string) => {
    await vcsConnectorServiceClient.deleteVCSConnector({ name });

    deleteCache(name);
  };

  return {
    getOrFetchConnectors,
    getConnectorList,
    getOrFetchConnector,
    getOrFetchConnectorByName,
    getConnector,
    getConnectorByName,
    createConnector,
    deleteConnector,
    updateConnector,
    fetchConnectorsInProvider,
    getConnectorsInProvider,
  };
});
