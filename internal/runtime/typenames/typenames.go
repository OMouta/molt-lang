package typenames

const (
	Number         = "number"
	String         = "string"
	Boolean        = "boolean"
	Nil            = "nil"
	List           = "list"
	Function       = "function"
	NativeFunction = "native-function"
	Code           = "code"
	Mutation       = "mutation"
)

func All() []string {
	return []string{
		Number,
		String,
		Boolean,
		Nil,
		List,
		Function,
		NativeFunction,
		Code,
		Mutation,
	}
}
