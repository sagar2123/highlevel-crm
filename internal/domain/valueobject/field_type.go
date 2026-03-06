package valueobject

type FieldType string

const (
	FieldTypeText     FieldType = "text"
	FieldTypeTextArea FieldType = "textarea"
	FieldTypeNumber   FieldType = "number"
	FieldTypeDate     FieldType = "date"
	FieldTypePhone    FieldType = "phone"
	FieldTypeEmail    FieldType = "email"
	FieldTypeDropdown FieldType = "dropdown"
	FieldTypeBoolean  FieldType = "boolean"
)

func (f FieldType) IsValid() bool {
	switch f {
	case FieldTypeText, FieldTypeTextArea, FieldTypeNumber, FieldTypeDate,
		FieldTypePhone, FieldTypeEmail, FieldTypeDropdown, FieldTypeBoolean:
		return true
	}
	return false
}
