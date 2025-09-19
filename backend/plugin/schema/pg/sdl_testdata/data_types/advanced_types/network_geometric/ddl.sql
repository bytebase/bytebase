CREATE SEQUENCE "public"."geographic_features_feature_id_seq" START WITH 1 INCREMENT BY 1;
CREATE SEQUENCE "public"."network_topology_topology_id_seq" START WITH 1 INCREMENT BY 1;
CREATE TABLE "public"."geographic_features" (
    "feature_id" integer NOT NULL DEFAULT nextval('public.geographic_features_feature_id_seq'::regclass),
    "feature_name" character varying(200) NOT NULL,
    "center_point" point NOT NULL,
    "bounding_box" box NOT NULL,
    "area_polygon" polygon,
    "coverage_circle" circle,
    "primary_line" lseg,
    "reference_paths" _path,
    "key_points" _point NOT NULL DEFAULT ARRAY[]::point[]
);
ALTER TABLE "public"."geographic_features" ADD CONSTRAINT "geographic_features_pkey" PRIMARY KEY (feature_id);
CREATE INDEX idx_features_center ON public.geographic_features USING gist (center_point);
CREATE INDEX idx_features_polygon ON public.geographic_features USING gist (area_polygon);

ALTER TABLE "public"."locations" ADD COLUMN "boundary_box" box;
ALTER TABLE "public"."locations" ADD COLUMN "center_point" point;
ALTER TABLE "public"."locations" ADD COLUMN "coverage_polygon" polygon;
ALTER TABLE "public"."locations" ADD COLUMN "route_path" path;
ALTER TABLE "public"."locations" ADD COLUMN "service_area" circle;
ALTER TABLE "public"."locations" ALTER COLUMN "coordinates" SET NOT NULL;
CREATE INDEX idx_locations_boundary ON public.locations USING gist (boundary_box);
CREATE INDEX idx_locations_coordinates ON public.locations USING gist (coordinates);

ALTER TABLE "public"."network_devices" ADD COLUMN "backup_mac" macaddr;
ALTER TABLE "public"."network_devices" ADD COLUMN "gateway" inet;
ALTER TABLE "public"."network_devices" ADD COLUMN "network_range" cidr;
ALTER TABLE "public"."network_devices" ADD COLUMN "subnet_mask" inet;
ALTER TABLE "public"."network_devices" ALTER COLUMN "ip_address" SET NOT NULL;
ALTER TABLE "public"."network_devices" ADD CONSTRAINT "network_devices_mac_address_key" UNIQUE (mac_address);

CREATE TABLE "public"."network_topology" (
    "topology_id" integer NOT NULL DEFAULT nextval('public.network_topology_topology_id_seq'::regclass),
    "network_name" character varying(100) NOT NULL,
    "network_cidr" cidr NOT NULL,
    "gateway_ip" inet NOT NULL,
    "dns_servers" _inet DEFAULT ARRAY[]::inet[],
    "device_macs" _macaddr DEFAULT ARRAY[]::macaddr[],
    "subnet_ranges" _cidr
);
ALTER TABLE "public"."network_topology" ADD CONSTRAINT "network_topology_pkey" PRIMARY KEY (topology_id);
ALTER TABLE "public"."network_topology" ADD CONSTRAINT "network_topology_network_cidr_key" UNIQUE (network_cidr);

