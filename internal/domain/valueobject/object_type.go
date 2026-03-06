package valueobject

var BuiltInObjectTypes = map[string]bool{
	"contacts":      true,
	"companies":     true,
	"opportunities": true,
	"pipelines":     true,
}

func IsBuiltInObjectType(t string) bool {
	return BuiltInObjectTypes[t]
}
