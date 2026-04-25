package constants

type GeoType int16

const (
	GeoTypeID          GeoType = 1
	GeoTypeName        GeoType = 2
	GeoTypeIcon        GeoType = 3
	GeoTypeVariant     GeoType = 4
	GeoTypeDescription GeoType = 5
	GeoTypeUnknow      GeoType = 0
)

func (t GeoType) String() string {
	switch t {
	case GeoTypeID:
		return "ID"
	case GeoTypeName:
		return "NAME"
	case GeoTypeIcon:
		return "ICON"
	case GeoTypeVariant:
		return "VARIANT"
	case GeoTypeDescription:
		return "DESCRIPTION"
	default:
		return "UNKNOW"
	}
}

func ParseGeoTypeText(v string) GeoType {
	switch v {
	case "ID":
		return GeoTypeID
	case "NAME":
		return GeoTypeName
	case "ICON":
		return GeoTypeIcon
	case "VARIANT":
		return GeoTypeVariant
	case "DESCRIPTION":
		return GeoTypeDescription
	default:
		return GeoTypeUnknow
	}
}
func ParseGeoType(v int16) GeoType {
	switch v {
	case 1:
		return GeoTypeID
	case 2:
		return GeoTypeName
	case 3:
		return GeoTypeIcon
	case 4:
		return GeoTypeVariant
	case 5:
		return GeoTypeDescription
	default:
		return GeoTypeUnknow
	}
}

func (t GeoType) Int16() int16 {
	return int16(t)
}
