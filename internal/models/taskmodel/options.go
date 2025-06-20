package taskmodel

type Option func(*Task)

func WithName(name string) Option {
	return func(t *Task) {
		t.Name = name
	}
}
