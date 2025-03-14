package consts

// This is expected behaviour of enum. Standard `type ComponentType string`
// makes it possible to pass any string value to fields expecting ComponentType.

// ComponentType is an enum of Slurm component types
type ComponentType interface {
	ct()
	String() string
}

type BaseComponentType struct {
	Value string
}

func (b BaseComponentType) ct() {}
func (b BaseComponentType) String() string {
	return b.Value
}

var (
	ComponentTypeController ComponentType = BaseComponentType{"controller"}
	ComponentTypeAccounting ComponentType = BaseComponentType{"accounting"}
	ComponentTypeWeb        ComponentType = BaseComponentType{"web"}
	ComponentTypeComputing  ComponentType = BaseComponentType{"computing"}
	ComponentTypeLogin      ComponentType = BaseComponentType{"login"}
	ComponentTypeMariaDb    ComponentType = BaseComponentType{"mariadb"}
	ComponentTypeMunge      ComponentType = BaseComponentType{"munge"}
)
