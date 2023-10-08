type RequestFn = () => Promise<any>;

export class ResourceComposer {
  requests: Record<string, RequestFn> = {};
  collect(resource: string, request: RequestFn) {
    this.requests[resource] = request;
  }
  compose(method: "all" | "allSettled" = "all") {
    const distinctRequests: Promise<void>[] = [];
    for (const resource in this.requests) {
      distinctRequests.push(this.requests[resource]());
    }
    return method === "all"
      ? Promise.all(distinctRequests)
      : Promise.allSettled(distinctRequests);
  }
}
