import { defineStore } from "pinia";
import { computed, reactive, ref, watchEffect } from "vue";
import { schemaDesignServiceClient } from "@/grpcweb";
import { Engine } from "@/types/proto/v1/common";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import {
  MergeSchemaDesignRequest,
  SchemaDesign,
  SchemaDesign_Type,
} from "@/types/proto/v1/schema_design_service";
import {
  getProjectAndSchemaDesignSheetId,
  projectNamePrefix,
  sheetNamePrefix,
} from "./v1/common";

export const useSchemaDesignStore = defineStore("schema_design", () => {
  const schemaDesignMapByName = reactive(new Map<string, SchemaDesign>());

  // Getters
  const schemaDesignList = computed(() => {
    const list = Array.from(schemaDesignMapByName.values());
    return list;
  });

  // Actions
  const fetchSchemaDesignList = async () => {
    const { schemaDesigns } = await schemaDesignServiceClient.listSchemaDesigns(
      {
        parent: "projects/-",
      }
    );
    // Clear the cache and re-populate it.
    schemaDesignMapByName.clear();
    for (const schemaDesign of schemaDesigns) {
      schemaDesignMapByName.set(schemaDesign.name, schemaDesign);
    }
    return schemaDesigns;
  };

  const createSchemaDesign = async (
    projectResourceId: string,
    schemaDesign: SchemaDesign
  ) => {
    const createdSchemaDesign =
      await schemaDesignServiceClient.createSchemaDesign({
        parent: projectResourceId,
        schemaDesign,
      });
    schemaDesignMapByName.set(createdSchemaDesign.name, createdSchemaDesign);
    return createdSchemaDesign;
  };

  const createSchemaDesignDraft = async (schemaDesign: SchemaDesign) => {
    const [projectName, sheetId] = getProjectAndSchemaDesignSheetId(
      schemaDesign.name
    );
    const baselineSheetName = `${projectNamePrefix}${projectName}/${sheetNamePrefix}${sheetId}`;
    const createdSchemaDesign =
      await schemaDesignServiceClient.createSchemaDesign({
        parent: `${projectNamePrefix}${projectName}`,
        schemaDesign: {
          ...schemaDesign,
          type: SchemaDesign_Type.PERSONAL_DRAFT,
          baselineSheetName: baselineSheetName,
        },
      });
    schemaDesignMapByName.set(createdSchemaDesign.name, createdSchemaDesign);
    return createdSchemaDesign;
  };

  const updateSchemaDesign = async (
    schemaDesign: SchemaDesign,
    updateMask: string[]
  ) => {
    const updatedSchemaDesign =
      await schemaDesignServiceClient.updateSchemaDesign({
        schemaDesign,
        updateMask,
      });
    schemaDesignMapByName.set(updatedSchemaDesign.name, updatedSchemaDesign);
    return updatedSchemaDesign;
  };

  const mergeSchemaDesign = async (request: MergeSchemaDesignRequest) => {
    await schemaDesignServiceClient.mergeSchemaDesign(request, {
      silent: true,
    });
    // Re-fetch schema design list to refresh the cache.
    await fetchSchemaDesignList();
  };

  const fetchSchemaDesignByName = async (name: string, silent = false) => {
    const schemaDesign = await schemaDesignServiceClient.getSchemaDesign(
      {
        name,
      },
      {
        silent,
      }
    );
    schemaDesignMapByName.set(schemaDesign.name, schemaDesign);
    return schemaDesign;
  };

  const getSchemaDesignByName = (name: string) => {
    return schemaDesignMapByName.get(name) ?? SchemaDesign.fromPartial({});
  };

  const getOrFetchSchemaDesignByName = async (name: string, silent = false) => {
    const cached = schemaDesignMapByName.get(name);
    if (cached) {
      return cached;
    }
    await fetchSchemaDesignByName(name, silent);
    return getSchemaDesignByName(name);
  };

  const deleteSchemaDesign = async (name: string) => {
    await schemaDesignServiceClient.deleteSchemaDesign({
      name,
    });
    schemaDesignMapByName.delete(name);
  };

  // Util functions
  const parseSchemaString = async (
    schema: string,
    engine: Engine
  ): Promise<DatabaseMetadata> => {
    try {
      const { schemaMetadata } =
        await schemaDesignServiceClient.parseSchemaString(
          {
            schemaString: schema,
            engine,
          },
          {
            silent: true,
          }
        );
      return schemaMetadata || DatabaseMetadata.fromPartial({});
    } catch (error) {
      return DatabaseMetadata.fromPartial({});
    }
  };

  return {
    schemaDesignList,
    fetchSchemaDesignList,
    createSchemaDesign,
    createSchemaDesignDraft,
    updateSchemaDesign,
    mergeSchemaDesign,
    fetchSchemaDesignByName,
    getOrFetchSchemaDesignByName,
    getSchemaDesignByName,
    parseSchemaString,
    deleteSchemaDesign,
  };
});

export const useSchemaDesignList = () => {
  const store = useSchemaDesignStore();
  const ready = ref(false);

  watchEffect(() => {
    ready.value = false;
    store.fetchSchemaDesignList().then(() => {
      ready.value = true;
    });
  });

  const schemaDesignList = computed(() => {
    // Only return main branch schema designs in the list.
    return store.schemaDesignList.filter((schemaDesign) => {
      return schemaDesign.type === SchemaDesign_Type.MAIN_BRANCH;
    });
  });

  return { schemaDesignList, ready };
};
