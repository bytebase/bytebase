-- Enhanced schema with comprehensive bit and range types
CREATE TABLE public.system_flags (
    flag_id SERIAL PRIMARY KEY,
    flag_name VARCHAR(50) NOT NULL,
    bit_flags BIT(16) NOT NULL DEFAULT B'0000000000000000', -- Expanded from BIT(8)
    permissions BIT(32) DEFAULT B'00000000000000000000000000000000', -- Expanded from BIT(16) to BIT(32)
    feature_flags BIT(32) DEFAULT B'00000000000000000000000000000000', -- New bit column
    access_mask BIT(64),                                    -- New fixed bit column
    system_state BIT(128)                                   -- New fixed bit column
);

CREATE TABLE public.bookings (
    booking_id SERIAL PRIMARY KEY,
    booking_period TSRANGE NOT NULL,       -- Keep as TSRANGE for compatibility
    price_range NUMRANGE,
    discount_range NUMRANGE DEFAULT '[0,100)'::NUMRANGE, -- New range with default
    age_range INT4RANGE,                   -- New integer range
    duration_range INT8RANGE,              -- New bigint range
    booking_dates DATERANGE NOT NULL,      -- New date range
    valid_periods TSTZRANGE[]              -- Array of timestamp ranges
);

CREATE TABLE public.scheduling (
    schedule_id SERIAL PRIMARY KEY,
    schedule_name VARCHAR(100) NOT NULL,
    working_hours TSRANGE[] NOT NULL DEFAULT ARRAY[]::TSRANGE[],
    available_dates DATERANGE[] DEFAULT ARRAY[]::DATERANGE[],
    capacity_ranges INT4RANGE[] DEFAULT ARRAY[]::INT4RANGE[],
    budget_ranges NUMRANGE[] DEFAULT ARRAY[]::NUMRANGE[],
    holiday_periods DATERANGE NOT NULL DEFAULT '[2000-01-01,2000-01-02)'::DATERANGE
);

CREATE TABLE public.permissions_matrix (
    permission_id SERIAL PRIMARY KEY,
    user_role VARCHAR(50) NOT NULL,
    read_permissions BIT(64) NOT NULL DEFAULT B'0000000000000000000000000000000000000000000000000000000000000000',
    write_permissions BIT(64) NOT NULL DEFAULT B'0000000000000000000000000000000000000000000000000000000000000000',
    admin_flags BIT(128) DEFAULT B'00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000',
    feature_access BIT(8) DEFAULT B'11111111'
);

-- Add specialized indexes for range operations
CREATE INDEX idx_bookings_period ON public.bookings USING GIST(booking_period);
CREATE INDEX idx_bookings_price ON public.bookings USING GIST(price_range);
CREATE INDEX idx_bookings_dates ON public.bookings USING GIST(booking_dates);
CREATE INDEX idx_scheduling_hours ON public.scheduling USING GIN(working_hours);
CREATE INDEX idx_scheduling_dates ON public.scheduling USING GIN(available_dates);