-- Enhanced schema with comprehensive network and geometric types
CREATE TABLE public.network_devices (
    device_id SERIAL PRIMARY KEY,
    device_name VARCHAR(100) NOT NULL,
    ip_address INET NOT NULL,              -- Made required
    network_range CIDR,                    -- New network range column
    mac_address MACADDR UNIQUE,            -- Made unique
    backup_mac MACADDR,                    -- New MAC address column
    subnet_mask INET,                      -- New subnet information
    gateway INET                           -- New gateway address
);

CREATE TABLE public.locations (
    location_id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    coordinates POINT NOT NULL,            -- Made required
    center_point POINT,                    -- New point column
    boundary_box BOX,                      -- New box geometric type
    service_area CIRCLE,                   -- New circle type
    coverage_polygon POLYGON,              -- New polygon type
    route_path PATH                        -- New path type
);

CREATE TABLE public.geographic_features (
    feature_id SERIAL PRIMARY KEY,
    feature_name VARCHAR(200) NOT NULL,
    center_point POINT NOT NULL,
    bounding_box BOX NOT NULL,
    area_polygon POLYGON,
    coverage_circle CIRCLE,
    primary_line LSEG,                     -- Line segment
    reference_paths PATH[],                -- Array of paths
    key_points POINT[] NOT NULL DEFAULT ARRAY[]::point[]
);

CREATE TABLE public.network_topology (
    topology_id SERIAL PRIMARY KEY,
    network_name VARCHAR(100) NOT NULL,
    network_cidr CIDR NOT NULL UNIQUE,
    gateway_ip INET NOT NULL,
    dns_servers INET[] DEFAULT ARRAY[]::inet[],
    device_macs MACADDR[] DEFAULT ARRAY[]::macaddr[],
    subnet_ranges CIDR[]
);

-- Add specialized indexes (GIST indexes for geometric types)
CREATE INDEX idx_locations_coordinates ON public.locations USING GIST(coordinates);
CREATE INDEX idx_locations_boundary ON public.locations USING GIST(boundary_box);
CREATE INDEX idx_features_center ON public.geographic_features USING GIST(center_point);
CREATE INDEX idx_features_polygon ON public.geographic_features USING GIST(area_polygon);