package validate

type Validator interface {
	Validate() error
}

func Validate(obj any) error {
	validator, isImplement := obj.(Validator)
	if isImplement {
		return validator.Validate()
	}
	return nil
}
