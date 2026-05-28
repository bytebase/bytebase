import { shallowReactive } from "vue";

export type KeyType = string | number | boolean;

type RequestCacheEntry<K extends KeyType[], T> = {
  keys: K;
  promise: Promise<T>;
  abortController: AbortController;
};
type EntityCacheEntry<K extends KeyType[], T> = {
  keys: K;
  entity: T;
};
const REQUEST_CACHE = new Map<
  string,
  Map<string, RequestCacheEntry<KeyType[], unknown>>
>();
const ENTITY_CACHE = new Map<
  string,
  Map<string, EntityCacheEntry<KeyType[], unknown>>
>();

type NamespaceSubscription = {
  version: number;
  listeners: Set<() => void>;
};
const SUBSCRIPTIONS = new Map<string, NamespaceSubscription>();

const getSubscription = (namespace: string): NamespaceSubscription => {
  let sub = SUBSCRIPTIONS.get(namespace);
  if (!sub) {
    sub = { version: 0, listeners: new Set() };
    SUBSCRIPTIONS.set(namespace, sub);
  }
  return sub;
};

const notify = (namespace: string) => {
  const sub = getSubscription(namespace);
  sub.version++;
  for (const listener of sub.listeners) {
    listener();
  }
};

export const useCache = <K extends KeyType[], T>(namespace: string) => {
  const requestCacheMap = getRequestCacheMap<K, T>(namespace);
  const entityCacheMap = getEntityCacheMap<K, T>(namespace);

  const trace = (title: string, keys: KeyType[], ...args: unknown[]) => {
    console.debug("cache", namespace, title, JSON.stringify(keys), ...args);
  };

  const invalidateRequest = (keys: K) => {
    const key = getKey(keys);
    const request = requestCacheMap.get(key);
    if (!request) return;
    if (!request.abortController.signal.aborted) {
      request.abortController.abort();
    }
    requestCacheMap.delete(key);
  };

  const getRequest = (keys: K) => {
    const key = getKey(keys);
    trace("getRequest", keys, requestCacheMap.has(key));
    const request = requestCacheMap.get(key);
    if (!request) {
      return undefined;
    }
    if (request.abortController.signal.aborted) {
      invalidateRequest(keys);
      return undefined;
    }
    return request.promise;
  };

  const setRequest = (keys: K, promise: Promise<T>) => {
    invalidateRequest(keys);

    const key = getKey(keys);
    const abortController = new AbortController();
    promise
      .then((entity: T) => {
        if (!abortController.signal.aborted) {
          setEntity(keys, entity);
        }
      })
      .catch(() => undefined)
      .finally(() => {
        invalidateRequest(keys);
      });
    requestCacheMap.set(key, {
      keys,
      promise,
      abortController,
    });
  };

  const getEntity = (keys: K) => {
    const key = getKey(keys);
    return entityCacheMap.get(key)?.entity;
  };

  const setEntity = (keys: K, entity: T) => {
    const key = getKey(keys);
    entityCacheMap.set(key, {
      keys,
      entity,
    });
    notify(namespace);
  };

  const invalidateEntity = (keys: K) => {
    invalidateRequest(keys);
    const key = getKey(keys);
    const existed = entityCacheMap.delete(key);
    if (existed) {
      notify(namespace);
    }
  };

  const clear = () => {
    for (const request of requestCacheMap.values()) {
      if (!request.abortController.signal.aborted) {
        request.abortController.abort();
      }
    }
    const hadEntities = entityCacheMap.size > 0;
    requestCacheMap.clear();
    entityCacheMap.clear();
    if (hadEntities) {
      notify(namespace);
    }
  };

  const subscribe = (listener: () => void) => {
    const sub = getSubscription(namespace);
    sub.listeners.add(listener);
    return () => {
      sub.listeners.delete(listener);
    };
  };

  const getVersion = () => getSubscription(namespace).version;

  return {
    requestCacheMap,
    entityCacheMap,
    getRequest,
    getEntity,
    setRequest,
    setEntity,
    invalidateRequest,
    invalidateEntity,
    clear,
    subscribe,
    getVersion,
  };
};

const getRequestCacheMap = <K extends KeyType[], T>(namespace: string) => {
  const existed = REQUEST_CACHE.get(namespace) as Map<
    string,
    RequestCacheEntry<K, T>
  >;
  if (existed) {
    return existed;
  }
  const created = new Map<string, RequestCacheEntry<K, T>>();
  REQUEST_CACHE.set(namespace, created);
  return created;
};

const getEntityCacheMap = <K extends KeyType[], T>(namespace: string) => {
  const existed = ENTITY_CACHE.get(namespace) as Map<
    string,
    EntityCacheEntry<K, T>
  >;

  if (existed) {
    return existed;
  }
  const created = shallowReactive(new Map<string, EntityCacheEntry<K, T>>());
  ENTITY_CACHE.set(namespace, created);
  return created;
};

const getKey = (keys: KeyType[]) => {
  return JSON.stringify(keys);
};
