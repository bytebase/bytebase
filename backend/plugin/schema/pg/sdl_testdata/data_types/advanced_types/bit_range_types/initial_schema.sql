-- Initial schema with basic bit and range types
CREATE TABLE public.system_flags (
    flag_id SERIAL PRIMARY KEY,
    flag_name VARCHAR(50) NOT NULL,
    bit_flags BIT(8),
    permissions BIT(16)
);

CREATE TABLE public.bookings (
    booking_id SERIAL PRIMARY KEY,
    booking_period TSRANGE,
    price_range NUMRANGE
);