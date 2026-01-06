import { create } from "@bufbuild/protobuf";
import { defineStore } from "pinia";
import { reactive } from "vue";
import { databaseServiceClientConnect } from "@/connect";
import { useCache } from "@/store/cache";
import { UNKNOWN_ID } from "@/types";
import {
  type Changelog,
  ChangelogView,
  type GetChangelogRequest,
  GetChangelogRequestSchema,
  type ListChangelogsRequest,
  ListChangelogsRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { extractChangelogUID } from "@/utils/v1/changelog";

type CacheKeyType = [string /* name */, ChangelogView];

export const useChangelogStore = defineStore("changelog", () => {
  const cache = useCache<CacheKeyType, Changelog>("bb.changelog.by-name");
  const changelogsMapByDatabase = reactive(new Map<string, Changelog[]>());

  const upsertChangelogsMap = async (
    parent: string,
    changelogs: Changelog[]
  ) => {
    changelogsMapByDatabase.set(parent, changelogs);
    changelogs.forEach((changelog) => {
      cache.setEntity([changelog.name, ChangelogView.BASIC], changelog);
    });
  };

  const clearCache = (parent: string) => {
    changelogsMapByDatabase.delete(parent);
  };

  const fetchChangelogList = async (params: Partial<ListChangelogsRequest>) => {
    const { parent } = params;
    if (!parent) throw new Error('"parent" field is required');
    const request = create(ListChangelogsRequestSchema, {
      parent: params.parent,
      pageSize: params.pageSize,
      pageToken: params.pageToken,
      view: params.view,
      filter: params.filter,
    });
    const response = await databaseServiceClientConnect.listChangelogs(request);
    const changelogs = response.changelogs;
    const { nextPageToken } = response;
    await upsertChangelogsMap(parent, changelogs);
    return { changelogs, nextPageToken };
  };
  const getOrFetchChangelogListOfDatabase = async (
    databaseName: string,
    pageSize: number,
    view = ChangelogView.BASIC
  ) => {
    if (changelogsMapByDatabase.has(databaseName)) {
      return changelogsMapByDatabase.get(databaseName) ?? [];
    }
    const { changelogs } = await fetchChangelogList({
      parent: databaseName,
      pageSize,
      view,
    });
    return changelogs;
  };
  const changelogListByDatabase = (name: string) => {
    return changelogsMapByDatabase.get(name) ?? [];
  };
  const fetchChangelog = async (params: Partial<GetChangelogRequest>) => {
    const request = create(GetChangelogRequestSchema, {
      name: params.name,
      view: params.view,
    });
    const changelog = await databaseServiceClientConnect.getChangelog(request);
    cache.setEntity(
      [changelog.name, params.view ?? ChangelogView.BASIC],
      changelog
    );
    return changelog;
  };
  const getOrFetchChangelogByName = async (
    name: string,
    view: ChangelogView = ChangelogView.BASIC
  ) => {
    const uid = extractChangelogUID(name);
    if (!uid || uid === String(UNKNOWN_ID)) {
      return undefined;
    }
    const entity = cache.getEntity([name, view]);
    if (entity) {
      return entity;
    }
    const request = cache.getRequest([name, view]);
    if (request) {
      return request;
    }
    const promise = fetchChangelog({ name, view });
    cache.setRequest([name, view], promise);
    return promise;
  };
  /**
   *
   * @param name
   * @param view default undefined to any view (full -> basic)
   * @returns
   */
  const getChangelogByName = (
    name: string,
    view: ChangelogView | undefined = undefined
  ) => {
    if (view === undefined) {
      return (
        cache.getEntity([name, ChangelogView.FULL]) ??
        cache.getEntity([name, ChangelogView.BASIC])
      );
    }
    return cache.getEntity([name, view]);
  };

  /**
   * Fetches the previous changelog for a given changelog with FULL view.
   * Returns the previous changelog's schema as the "before" state for diff comparison.
   * @param changelogName - The name of the current changelog
   * @returns The previous changelog with full schema, or undefined if this is the first changelog
   */
  const fetchPreviousChangelog = async (
    changelogName: string
  ): Promise<Changelog | undefined> => {
    // Extract database name from changelog name
    // Format: instances/{instance}/databases/{database}/changelogs/{changelog}
    const parts = changelogName.split("/changelogs/");
    if (parts.length !== 2) {
      return undefined;
    }
    const databaseName = parts[0];
    const currentChangelogUID = extractChangelogUID(changelogName);
    if (!currentChangelogUID || currentChangelogUID === String(UNKNOWN_ID)) {
      return undefined;
    }

    // Fetch changelogs with BASIC view to find the previous one efficiently
    const { changelogs } = await fetchChangelogList({
      parent: databaseName,
      pageSize: 1000, // Fetch enough to find the previous one
      view: ChangelogView.BASIC, // Use BASIC view for listing - much lighter
    });

    // Find the current changelog index
    const currentIndex = changelogs.findIndex(
      (cl) => extractChangelogUID(cl.name) === currentChangelogUID
    );

    // If not found or it's the last one (oldest), there's no previous
    if (currentIndex === -1 || currentIndex === changelogs.length - 1) {
      return undefined;
    }

    // Get the previous changelog name
    const previousChangelogName = changelogs[currentIndex + 1].name;

    // Fetch the previous changelog with FULL view to get the schema
    return await getOrFetchChangelogByName(
      previousChangelogName,
      ChangelogView.FULL
    );
  };

  return {
    clearCache,
    fetchChangelogList,
    getOrFetchChangelogListOfDatabase,
    changelogListByDatabase,
    fetchChangelog,
    getOrFetchChangelogByName,
    getChangelogByName,
    fetchPreviousChangelog,
  };
});
