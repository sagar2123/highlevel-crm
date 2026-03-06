package valueobject

type Cardinality string

const (
	OneToOne   Cardinality = "one_to_one"
	OneToMany  Cardinality = "one_to_many"
	ManyToMany Cardinality = "many_to_many"
)
