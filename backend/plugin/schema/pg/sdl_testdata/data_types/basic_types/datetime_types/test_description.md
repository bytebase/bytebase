# DateTime Data Types Test

This test validates the SDL migration handling of PostgreSQL date and time data types:
- DATE for date values
- TIME and TIME WITH TIME ZONE
- TIMESTAMP and TIMESTAMP WITH TIME ZONE (TIMESTAMPTZ)
- INTERVAL for time intervals

Tests creation with various defaults and constraints.