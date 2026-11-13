package holiday

// Ekspor helper pengujian di backend/tests/holiday.
func ParseAPICoIDHoliday(item APICoIDHoliday, year int, seen map[string]bool) (Entry, bool, error) {
	return parseAPICoIDHoliday(item, year, seen)
}

type APICoIDHoliday = apiCoIDHoliday

const TypePublicHoliday = typePublicHoliday
