import { defineStore } from "pinia";
import { instanceServiceClient } from "@/grpcweb";
import { ResourceId } from "@/types";
import { Instance } from "@/types/proto/v1/instance_service";

interface InstanceState {
  instanceMapByName: Map<ResourceId, Instance>;
}

export const useInstanceV1Store = defineStore("instance_v1", {
  state: (): InstanceState => ({
    instanceMapByName: new Map(),
  }),
  getters: {
    instanceList(state) {
      return Array.from(state.instanceMapByName.values());
    },
  },
  actions: {
    async fetchInstances(showDeleted = false) {
      const { instances } = await instanceServiceClient.listInstances({
        showDeleted,
      });
      for (const instance of instances) {
        this.instanceMapByName.set(instance.name, instance);
      }
      return instances;
    },
    async createInstance(instance: Partial<Instance>) {
      const createdInstance = await instanceServiceClient.createInstance({
        instance,
        instanceId: instance.name,
      });
      this.instanceMapByName.set(createdInstance.name, createdInstance);
      return createdInstance;
    },
    async getOrFetchInstanceByName(name: string) {
      const cachedData = this.instanceMapByName.get(name);
      if (cachedData) {
        return cachedData;
      }
      const instance = await instanceServiceClient.getInstance({
        name,
      });
      this.instanceMapByName.set(instance.name, instance);
      return instance;
    },
    getInstanceByName(name: string) {
      return this.instanceMapByName.get(name);
    },
  },
});
