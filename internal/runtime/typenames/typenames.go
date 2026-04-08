package typenames

const (
	Number         = "number"
	String         = "string"
	Boolean        = "boolean"
	Nil            = "nil"
	List           = "list"
	Record         = "record"
	Error          = "error"
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
		Error,
		Function,
		NativeFunction,
		Code,
		Mutation,
	}
}
