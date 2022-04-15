import axios from "axios";
import {
  empty,
  unknown,
  SchemaGuideId,
  EMPTY_ID,
  Environment,
  DatabaseSchemaGuide,
  DatabaseSchemaGuideCreate,
  DatabaseSchemaGuidePatch,
} from "../../types";
import { defineStore } from "pinia";

interface SchemaSystemState {
  guideList: DatabaseSchemaGuide[];
}

export const useSchemaSystemStore = defineStore("schemaSystem", {
  state: (): SchemaSystemState => ({
    guideList: [],
  }),
  actions: {
    availableEnvironments(environmentList: Environment[]): Environment[] {
      const envMap = environmentList.reduce((map, env) => {
        map.set(env.id, env);
        return map;
      }, new Map<number, Environment>());

      for (const guideline of this.guideList) {
        for (const envId of guideline.environmentList) {
          if (envMap.has(envId)) {
            envMap.delete(envId);
          }
        }
      }

      return [...envMap.values()];
    },
    addGuideline(guideline: DatabaseSchemaGuideCreate) {
      const mock = empty("SCHEMA_GUIDE") as DatabaseSchemaGuide;
      this.guideList.push({
        ...guideline,
        id: this.guideList.length + 1,
        creator: mock.creator,
        updater: mock.updater,
        createdTs: new Date().getTime() / 1000,
        updatedTs: new Date().getTime() / 1000,
      });
    },
    removeGuideline(id: SchemaGuideId) {
      const index = this.guideList.findIndex((g) => g.id === id);
      if (index < 0) {
        return;
      }
      this.guideList = [
        ...this.guideList.slice(0, index),
        ...this.guideList.slice(index + 1),
      ];
    },
    updateGuideline(id: SchemaGuideId, guideline: DatabaseSchemaGuidePatch) {
      const index = this.guideList.findIndex((g) => g.id === id);
      if (index < 0) {
        return;
      }
      this.guideList = [
        ...this.guideList.slice(0, index),
        {
          ...this.guideList[index],
          ...guideline,
        },
        ...this.guideList.slice(index + 1),
      ];
    },
    getGuideById(id: SchemaGuideId): DatabaseSchemaGuide {
      if (id == EMPTY_ID) {
        return empty("SCHEMA_GUIDE") as DatabaseSchemaGuide;
      }

      return (
        this.guideList.find((g) => g.id === id) ||
        (unknown("SCHEMA_GUIDE") as DatabaseSchemaGuide)
      );
    },

    async fetchGuideList() {
      throw new Error("function haven't implement yet");
    },
    async fetchGuideById(id: SchemaGuideId) {
      throw new Error(`guide ${id} not found`);
    },
  },
});
