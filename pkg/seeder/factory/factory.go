package factory

import "time"

// Factory is a generic builder for generating model instances with fake data.
// It supports a default definition, named states that override fields, and
// batch creation via MakeMany.
type Factory[T any] struct {
	definition   func(faker Faker) T
	states       map[string]func(faker Faker, base T) T
	activeStates []string
	faker        Faker
}

// NewFactory creates a new Factory with the given definition function.
// A default Faker seeded from the current time is used.
func NewFactory[T any](definition func(faker Faker) T) *Factory[T] {
	return &Factory[T]{
		definition: definition,
		states:     make(map[string]func(faker Faker, base T) T),
		faker:      NewFaker(time.Now().UnixNano()),
	}
}

// WithFaker returns a copy of the factory that uses the provided Faker.
func (f *Factory[T]) WithFaker(faker Faker) *Factory[T] {
	cp := f.copy()
	cp.faker = faker
	return cp
}

// State registers a named state modifier on the factory. The modifier receives
// the Faker and the base instance produced by the definition, and returns a
// modified instance. State returns the same factory for chaining.
func (f *Factory[T]) State(name string, fn func(faker Faker, base T) T) *Factory[T] {
	f.states[name] = fn
	return f
}

// WithState returns a new Factory copy that will apply the named state when
// Make or MakeMany is called. The original factory is not modified.
func (f *Factory[T]) WithState(name string) *Factory[T] {
	cp := f.copy()
	cp.activeStates = append(cp.activeStates, name)
	return cp
}

// Make creates a single instance using the definition, then applies any active
// states in the order they were added.
func (f *Factory[T]) Make() T {
	instance := f.definition(f.faker)
	for _, name := range f.activeStates {
		if fn, ok := f.states[name]; ok {
			instance = fn(f.faker, instance)
		}
	}
	return instance
}

// MakeMany creates count instances. Each instance is independently generated
// through Make.
func (f *Factory[T]) MakeMany(count int) []T {
	if count <= 0 {
		return nil
	}
	results := make([]T, count)
	for i := range results {
		results[i] = f.Make()
	}
	return results
}

// copy returns a shallow copy of the factory so that WithState and WithFaker
// do not mutate the original.
func (f *Factory[T]) copy() *Factory[T] {
	activeStates := make([]string, len(f.activeStates))
	copy(activeStates, f.activeStates)
	return &Factory[T]{
		definition:   f.definition,
		states:       f.states,
		activeStates: activeStates,
		faker:        f.faker,
	}
}
