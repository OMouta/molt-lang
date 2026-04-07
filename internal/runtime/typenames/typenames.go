package typenames

const (
	Number         = "number"
	String         = "string"
	Boolean        = "boolean"
	Nil            = "nil"
	List           = "list"
	Record         = "record"
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
		Record,
		Function,
		NativeFunction,
		Code,
		Mutation,
	}
}
