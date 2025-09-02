# Network and Geometric Data Types Test

This test validates the SDL migration handling of specialized PostgreSQL data types:

**Network Address Types**:
- **INET**: IPv4 and IPv6 addresses with optional netmask
- **CIDR**: Network addresses
- **MACADDR**: MAC addresses

**Geometric Types**:
- **POINT**: Points in 2D space
- **LINE**: Infinite lines  
- **LSEG**: Line segments
- **BOX**: Rectangular boxes
- **PATH**: Geometric paths
- **POLYGON**: Closed geometric paths
- **CIRCLE**: Circles

Tests creation, indexing, and constraints for these specialized data types.