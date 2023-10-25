import { defineStore } from "pinia";
import { reactive, ref, watchEffect } from "vue";
import { schemaDesignServiceClient } from "@/grpcweb";
import { Engine } from "@/types/proto/v1/common";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import {
  MergeSchemaDesignRequest,
  SchemaDesign,
  SchemaDesign_Type,
  SchemaDesignView,
} from "@/types/proto/v1/schema_design_service";
import {
  getProjectAndSchemaDesignSheetId,
  projectNamePrefix,
  sheetNamePrefix,
} from "./v1/common";

export const useSchemaDesignStore = defineStore("schema_design", () => {
  const schemaDesignMapByName = reactive(new Map<string, SchemaDesign>());
  const getSchemaDesignRequestCacheByName = new Map<
    string,
    Promise<SchemaDesign>
  >();

  // Actions
  const fetchSchemaDesignList = async (projectName: string = "projects/-") => {
    const { schemaDesigns } = await schemaDesignServiceClient.listSchemaDesigns(
      {
        parent: projectName,
        view: SchemaDesignView.SCHEMA_DESIGN_VIEW_FULL,
      }
    );
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
    console.debug("baseline schema", schemaDesign.baselineSchema);
    console.debug("target metadata", schemaDesign.schemaMetadata);
    console.debug("got schema", createdSchemaDesign.schema);
    schemaDesignMapByName.set(createdSchemaDesign.name, createdSchemaDesign);
    return createdSchemaDesign;
  };

  const createSchemaDesignDraft = async (schemaDesign: SchemaDesign) => {
    const [projectName, sheetId] = getProjectAndSchemaDesignSheetId(
      schemaDesign.name
    );
    const projectResourceId = `${projectNamePrefix}${projectName}`;
    const baselineSheetName = `${projectResourceId}/${sheetNamePrefix}${sheetId}`;
    return createSchemaDesign(projectResourceId, {
      ...schemaDesign,
      type: SchemaDesign_Type.PERSONAL_DRAFT,
      baselineSheetName: baselineSheetName,
      protection: {
        // For personal draft, allow force pushes by default.
        allowForcePushes: true,
      },
    });
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
    const updatedSchemaDesign =
      await schemaDesignServiceClient.mergeSchemaDesign(request, {
        silent: true,
      });
    schemaDesignMapByName.set(updatedSchemaDesign.name, updatedSchemaDesign);
  };

  const fetchSchemaDesignByName = async (
    name: string,
    useCache = true,
    silent = false
  ) => {
    if (useCache) {
      const cachedEntity = schemaDesignMapByName.get(name);
      if (cachedEntity) {
        return cachedEntity;
      }

      // Avoid making duplicated requests concurrently
      const cachedRequest = getSchemaDesignRequestCacheByName.get(name);
      if (cachedRequest) {
        return cachedRequest;
      }
    }
    const request = schemaDesignServiceClient.getSchemaDesign(
      {
        name,
      },
      {
        silent,
      }
    );
    request.then((schemaDesign) => {
      schemaDesignMapByName.set(schemaDesign.name, schemaDesign);
    });
    getSchemaDesignRequestCacheByName.set(name, request);
    return request;
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
    fetchSchemaDesignList,
    createSchemaDesign,
    createSchemaDesignDraft,
    updateSchemaDesign,
    mergeSchemaDesign,
    fetchSchemaDesignByName,
    parseSchemaString,
    deleteSchemaDesign,
  };
});

export const useSchemaDesignList = (
  projectName: string | undefined = undefined
) => {
  const store = useSchemaDesignStore();
  const ready = ref(false);
  const schemaDesignList = ref<SchemaDesign[]>([]);

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
