package valueobject

type LifecycleState string

const (
	LifecycleActive   LifecycleState = "active"
	LifecycleArchived LifecycleState = "archived"
	LifecycleDeleted  LifecycleState = "deleted"
)

func (l LifecycleState) IsValid() bool {
	switch l {
	case LifecycleActive, LifecycleArchived, LifecycleDeleted:
		return true
	}
	return false
}
