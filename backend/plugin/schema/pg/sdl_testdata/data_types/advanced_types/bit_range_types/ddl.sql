CREATE SEQUENCE "public"."permissions_matrix_permission_id_seq" START WITH 1 INCREMENT BY 1;
CREATE SEQUENCE "public"."scheduling_schedule_id_seq" START WITH 1 INCREMENT BY 1;
ALTER TABLE "public"."bookings" ADD COLUMN "age_range" int4range;
ALTER TABLE "public"."bookings" ADD COLUMN "booking_dates" daterange NOT NULL;
ALTER TABLE "public"."bookings" ADD COLUMN "discount_range" numrange DEFAULT '[0,100)'::NUMRANGE;
ALTER TABLE "public"."bookings" ADD COLUMN "duration_range" int8range;
ALTER TABLE "public"."bookings" ADD COLUMN "valid_periods" _tstzrange;
ALTER TABLE "public"."bookings" ALTER COLUMN "booking_period" SET NOT NULL;
CREATE INDEX idx_bookings_dates ON public.bookings USING gist (booking_dates);
CREATE INDEX idx_bookings_period ON public.bookings USING gist (booking_period);
CREATE INDEX idx_bookings_price ON public.bookings USING gist (price_range);

CREATE TABLE "public"."permissions_matrix" (
    "permission_id" integer NOT NULL DEFAULT nextval('public.permissions_matrix_permission_id_seq'::regclass),
    "user_role" character varying(50) NOT NULL,
    "read_permissions" bit(64) NOT NULL DEFAULT B'0000000000000000000000000000000000000000000000000000000000000000',
    "write_permissions" bit(64) NOT NULL DEFAULT B'0000000000000000000000000000000000000000000000000000000000000000',
    "admin_flags" bit(128) DEFAULT B'00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000',
    "feature_access" bit(8) DEFAULT B'11111111'
);
ALTER TABLE "public"."permissions_matrix" ADD CONSTRAINT "permissions_matrix_pkey" PRIMARY KEY (permission_id);

CREATE TABLE "public"."scheduling" (
    "schedule_id" integer NOT NULL DEFAULT nextval('public.scheduling_schedule_id_seq'::regclass),
    "schedule_name" character varying(100) NOT NULL,
    "working_hours" _tsrange NOT NULL DEFAULT ARRAY[]::TSRANGE[],
    "available_dates" _daterange DEFAULT ARRAY[]::DATERANGE[],
    "capacity_ranges" _int4range DEFAULT ARRAY[]::INT4RANGE[],
    "budget_ranges" _numrange DEFAULT ARRAY[]::NUMRANGE[],
    "holiday_periods" daterange NOT NULL DEFAULT '[2000-01-01,2000-01-02)'::DATERANGE
);
ALTER TABLE "public"."scheduling" ADD CONSTRAINT "scheduling_pkey" PRIMARY KEY (schedule_id);
CREATE INDEX idx_scheduling_hours ON public.scheduling USING gin (working_hours);
CREATE INDEX idx_scheduling_dates ON public.scheduling USING gin (available_dates);

ALTER TABLE "public"."system_flags" ADD COLUMN "access_mask" bit(64);
ALTER TABLE "public"."system_flags" ADD COLUMN "feature_flags" bit(32) DEFAULT B'00000000000000000000000000000000';
ALTER TABLE "public"."system_flags" ADD COLUMN "system_state" bit(128);
ALTER TABLE "public"."system_flags" ALTER COLUMN "bit_flags" TYPE bit(16);
ALTER TABLE "public"."system_flags" ALTER COLUMN "bit_flags" SET NOT NULL;
ALTER TABLE "public"."system_flags" ALTER COLUMN "bit_flags" SET DEFAULT B'0000000000000000';
ALTER TABLE "public"."system_flags" ALTER COLUMN "permissions" TYPE bit(32);
ALTER TABLE "public"."system_flags" ALTER COLUMN "permissions" SET DEFAULT B'00000000000000000000000000000000';

