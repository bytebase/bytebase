export default function configurePrincipal(route) {
  route.get("/principal/:id");

  route.get("/principal");

  route.post("/principal", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("principal-new");
    const principal = schema.principals.findBy({ email: attrs.email });
    if (principal) {
      return principal;
    }

    const ts = Date.now();
    return schema.principals.create({
      ...attrs,
      creatorId: attrs.creatorId,
      createdTs: ts,
      updaterId: attrs.creatorId,
      updatedTs: ts,
    });
  });

  route.patch("/principal/:principalId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("principal-patch");
    const principal = schema.principals.find(request.params.principalId);

    if (!principal) {
      return new Response(
        404,
        {},
        {
          errors: "Principal id " + request.params.principalId + " not found",
        }
      );
    }

    const ts = Date.now();
    return principal.update({
      ...attrs,
      updatedTs: ts,
    });
  });
}
