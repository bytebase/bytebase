import { computed, reactive, ref, watchEffect } from "vue";
import { defineStore } from "pinia";
import { schemaDesignServiceClient } from "@/grpcweb";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import { Engine } from "@/types/proto/v1/common";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";

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
        parent: `projects/${projectResourceId}`,
        schemaDesign,
      });
    schemaDesignMapByName.set(createdSchemaDesign.name, createdSchemaDesign);
    return schemaDesignMapByName;
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
    updateSchemaDesign,
    fetchSchemaDesignByName,
    getOrFetchSchemaDesignByName,
    getSchemaDesignByName,
    parseSchemaString,
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
    return store.schemaDesignList;
  });
  return { schemaDesignList, ready };
};
