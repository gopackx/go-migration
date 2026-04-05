package factory

import (
	"testing"
)

// User is a simple test struct for factory tests.
type User struct {
	Name  string
	Email string
	Age   int
}

func userFactory() *Factory[User] {
	faker := NewFaker(42)
	return NewFactory[User](func(f Faker) User {
		return User{
			Name:  f.Name(),
			Email: f.Email(),
			Age:   f.IntBetween(18, 65),
		}
	}).WithFaker(faker)
}

func TestMake_ReturnsPopulatedInstance(t *testing.T) {
	f := userFactory()
	user := f.Make()

	if user.Name == "" {
		t.Errorf("expected non-empty string")
	}
	if user.Email == "" {
		t.Errorf("expected non-empty string")
	}
	if !(user.Age >= 18 && user.Age <= 65) {
		t.Errorf("Age should be in range [18, 65]")
	}
}

func TestMakeMany_ReturnsCorrectCount(t *testing.T) {
	f := userFactory()

	users := f.MakeMany(5)
	if len(users) != 5 {
		t.Fatalf("len = %d, want %d", len(users), 5)
	}

	for _, u := range users {
		if u.Name == "" {
			t.Errorf("expected non-empty string")
		}
		if u.Email == "" {
			t.Errorf("expected non-empty string")
		}
	}
}

func TestMakeMany_ZeroReturnsNil(t *testing.T) {
	f := userFactory()
	if f.MakeMany(0) != nil {
		t.Errorf("expected nil, got %v", f.MakeMany(0))
	}
}

func TestMakeMany_NegativeReturnsNil(t *testing.T) {
	f := userFactory()
	if f.MakeMany(-1) != nil {
		t.Errorf("expected nil, got %v", f.MakeMany(-1))
	}
}

func TestState_OverridesSpecificFields(t *testing.T) {
	f := userFactory()
	f.State("admin", func(faker Faker, base User) User {
		base.Name = "Admin User"
		return base
	})

	admin := f.WithState("admin").Make()
	if "Admin User" != admin.Name {
		t.Errorf("State should override Name")
	}
	if admin.Email == "" {
		t.Errorf("expected non-empty string")
	}
	if !(admin.Age >= 18 && admin.Age <= 65) {
		t.Errorf("Age should still be in range")
	}
}

func TestWithState_DoesNotModifyOriginalFactory(t *testing.T) {
	f := userFactory()
	f.State("senior", func(faker Faker, base User) User {
		base.Age = 99
		return base
	})

	withSenior := f.WithState("senior")

	// Original factory should produce normal instances
	original := f.Make()
	if !(original.Age >= 18 && original.Age <= 65) {
		t.Errorf("Original factory should not be affected by WithState")
	}

	// Derived factory should apply the state
	senior := withSenior.Make()
	if 99 != senior.Age {
		t.Errorf("WithState factory should apply state override")
	}
}

func TestWithState_UnknownStateIsIgnored(t *testing.T) {
	f := userFactory()
	// Applying a state that was never registered should not panic
	user := f.WithState("nonexistent").Make()
	if user.Name == "" {
		t.Errorf("expected non-empty string")
	}
}

func TestMultipleStates_AppliedInOrder(t *testing.T) {
	f := userFactory()
	f.State("named", func(faker Faker, base User) User {
		base.Name = "Custom Name"
		return base
	})
	f.State("aged", func(faker Faker, base User) User {
		base.Age = 100
		return base
	})

	user := f.WithState("named").WithState("aged").Make()
	if "Custom Name" != user.Name {
		t.Errorf("got %v, want %v", user.Name, "Custom Name")
	}
	if 100 != user.Age {
		t.Errorf("got %v, want %v", user.Age, 100)
	}
}

func TestWithFaker_UsesProvidedFaker(t *testing.T) {
	f := NewFactory[User](func(faker Faker) User {
		return User{
			Name:  faker.Name(),
			Email: faker.Email(),
			Age:   faker.IntBetween(18, 65),
		}
	})

	// Two factories with the same seed should produce identical results
	f1 := f.WithFaker(NewFaker(123))
	f2 := f.WithFaker(NewFaker(123))

	u1 := f1.Make()
	u2 := f2.Make()

	if u1.Name != u2.Name {
		t.Errorf("got %v, want %v", u2.Name, u1.Name)
	}
	if u1.Email != u2.Email {
		t.Errorf("got %v, want %v", u2.Email, u1.Email)
	}
	if u1.Age != u2.Age {
		t.Errorf("got %v, want %v", u2.Age, u1.Age)
	}
}

func TestStateChaining(t *testing.T) {
	f := NewFactory[User](func(faker Faker) User {
		return User{Name: faker.Name(), Email: faker.Email(), Age: faker.IntBetween(18, 65)}
	}).WithFaker(NewFaker(42))

	// State() returns the same factory for chaining
	result := f.State("a", func(faker Faker, base User) User {
		return base
	}).State("b", func(faker Faker, base User) User {
		return base
	})

	if f != result {
		t.Errorf("State() should return the same factory for chaining")
	}
}
