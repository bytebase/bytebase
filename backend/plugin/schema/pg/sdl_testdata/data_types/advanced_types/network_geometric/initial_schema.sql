-- Initial schema with basic network and geometric types
CREATE TABLE public.network_devices (
    device_id SERIAL PRIMARY KEY,
    device_name VARCHAR(100) NOT NULL,
    ip_address INET,
    mac_address MACADDR
);

CREATE TABLE public.locations (
    location_id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    coordinates POINT
);