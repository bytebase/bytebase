package pg

import "strings"

func IsSystemUser(user string) bool {
	return strings.HasPrefix(user, "alloydb")
}

func IsSystemView(view string) bool {
	if strings.HasPrefix(view, "g_columnar_") {
		return true
	}
	if strings.HasPrefix(view, "google_db_advisor_") {
		return true
	}
	if strings.HasPrefix(view, "g_agg_stat_") {
		return true
	}
	if strings.HasPrefix(view, "g_agg_stat_") {
		return true
	}
	if strings.HasPrefix(view, "hypopg") {
		return true
	}
	return false
}

func IsSystemFunctions(function string) bool {
	if strings.HasPrefix(function, "g_columnar_") {
		return true
	}
	if strings.HasPrefix(function, "google_columnar_") {
		return true
	}
	if strings.HasPrefix(function, "google_db_advisor_") {
		return true
	}
	if strings.HasPrefix(function, "g_agg_stat_") {
		return true
	}
	if strings.HasPrefix(function, "hypopg") {
		return true
	}
	if function == "pg_stat_statements_wrapper" {
		return true
	}
	return false
}
