package repository

import (
	"amiya-eden/global"
	"strings"
)

func (r *SdeRepository) SearchSolarSystemsByKeyword(keyword string, exactSolarSystemID int64, limit int) ([]GalaxyRegistrySdeSystem, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := r.searchSolarSystemsWithNaming(newSDENaming(true), keyword, exactSolarSystemID, limit)
	if err == nil || exactSolarSystemID > 0 || !shouldRetrySDELowercase(err) {
		if err != nil {
			reportSDEQueryError("search_solar_systems", err)
		}
		return rows, err
	}

	fallbackRows, fallbackErr := r.searchSolarSystemsWithNaming(newSDENaming(false), keyword, exactSolarSystemID, limit)
	if fallbackErr != nil {
		return nil, wrapAndReportSDEFallbackError("search_solar_systems", err, fallbackErr)
	}
	return fallbackRows, nil
}

func shouldRetrySDELowercase(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "does not exist") || strings.Contains(msg, "no such table")
}

func (r *SdeRepository) searchSolarSystemsWithNaming(
	naming sdeNaming,
	keyword string,
	exactSolarSystemID int64,
	limit int,
) ([]GalaxyRegistrySdeSystem, error) {
	rows := make([]GalaxyRegistrySdeSystem, 0)
	query := global.DB.Table(naming.table("mapSolarSystems", "ms")).
		Select(strings.Join([]string{
			naming.col("ms", "solarSystemID") + " AS solar_system_id",
			naming.col("ms", "solarSystemName") + " AS solar_system_name",
			naming.col("ms", "regionID") + " AS region_id",
			naming.col("mr", "regionName") + " AS region_name",
			naming.col("ms", "constellationID") + " AS constellation_id",
			naming.col("mc", "constellationName") + " AS constellation_name",
			naming.col("ms", "security") + " AS security",
		}, ", ")).
		Joins("JOIN " + naming.table("mapRegions", "mr") + " ON " + naming.col("mr", "regionID") + " = " + naming.col("ms", "regionID")).
		Joins("JOIN " + naming.table("mapConstellations", "mc") + " ON " + naming.col("mc", "constellationID") + " = " + naming.col("ms", "constellationID"))

	if exactSolarSystemID > 0 {
		query = query.Where(naming.col("ms", "solarSystemID")+" = ?", exactSolarSystemID)
	} else {
		normalizedKeyword := strings.TrimSpace(keyword)
		if normalizedKeyword == "" {
			return rows, nil
		}
		like := "%" + strings.ToLower(normalizedKeyword) + "%"
		query = query.Where(
			"LOWER("+naming.col("ms", "solarSystemName")+") LIKE ? OR LOWER("+naming.col("mr", "regionName")+") LIKE ? OR LOWER("+naming.col("mc", "constellationName")+") LIKE ?",
			like, like, like,
		)
	}

	err := query.Order(naming.col("ms", "solarSystemName") + " ASC").Limit(limit).Scan(&rows).Error
	return rows, err
}
