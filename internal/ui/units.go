package ui

import "fmt"

type unitSystem int

const (
	unitMetric unitSystem = iota
	unitImperial
)

const (
	kmToMiles = 0.621371
)

func (u unitSystem) next() unitSystem {
	if u == unitImperial {
		return unitMetric
	}
	return unitImperial
}

func (u unitSystem) name() string {
	if u == unitImperial {
		return "imperial"
	}
	return "metric"
}

func (u unitSystem) compactName() string {
	if u == unitImperial {
		return "imp"
	}
	return "met"
}

func (u unitSystem) distanceUnit() string {
	if u == unitImperial {
		return "mi"
	}
	return "km"
}

func (u unitSystem) speedUnit() string {
	if u == unitImperial {
		return "mi/s"
	}
	return "km/s"
}

func distanceInUnits(km float64, units unitSystem) float64 {
	if units == unitImperial {
		return km * kmToMiles
	}
	return km
}

func speedInUnits(kmps float64, units unitSystem) float64 {
	if units == unitImperial {
		return kmps * kmToMiles
	}
	return kmps
}

func formatDistForUnits(km float64, units unitSystem) string {
	value := distanceInUnits(km, units)
	unit := units.distanceUnit()
	if value >= 1e6 {
		return fmt.Sprintf("%.1f M %s", value/1e6, unit)
	}
	if value >= 1000 {
		return fmt.Sprintf("%.0f %s", value, unit)
	}
	return fmt.Sprintf("%.1f %s", value, unit)
}

func formatCompactDistForUnits(km float64, units unitSystem) string {
	value := distanceInUnits(km, units)
	unit := units.distanceUnit()
	if value >= 1e6 {
		return fmt.Sprintf("%.1fM %s", value/1e6, unit)
	}
	if value >= 1000 {
		return fmt.Sprintf("%.0fk %s", value/1e3, unit)
	}
	return fmt.Sprintf("%.0f %s", value, unit)
}

func formatSpeedForUnits(kmps float64, units unitSystem) string {
	speed := speedInUnits(kmps, units)
	primaryUnit := units.speedUnit()
	if units == unitImperial {
		return fmt.Sprintf("%.3f %s  (%.0f mph)", speed, primaryUnit, speed*3600)
	}
	return fmt.Sprintf("%.3f %s  (%.0f km/h)", speed, primaryUnit, speed*3600)
}

func speedCompanionUnit(kmps float64, units unitSystem) string {
	if units == unitImperial {
		return fmt.Sprintf("(%.0f mph)", speedInUnits(kmps, units)*3600)
	}
	return fmt.Sprintf("(%.0f km/h)", kmps*3600)
}

func formatRateForUnits(kmps float64, units unitSystem) string {
	return fmt.Sprintf("%+.3f %s", speedInUnits(kmps, units), units.speedUnit())
}

func formatVectorForUnits(v float64, units unitSystem) string {
	return fmt.Sprintf("%+.2f", speedInUnits(v, units))
}

func formatWindSpeedForUnits(kmps float64, units unitSystem) string {
	value := speedInUnits(kmps, units)
	if units == unitImperial {
		return fmt.Sprintf("%.0f mi/s", value)
	}
	return fmt.Sprintf("%.0f km/s", value)
}
