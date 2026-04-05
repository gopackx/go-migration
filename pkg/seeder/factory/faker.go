package factory

import "time"

// Faker defines the contract for generating fake data values.
// Implementations provide realistic-looking data for use in factories and seeders.
type Faker interface {
	// Name returns a full name (first + last).
	Name() string
	// FirstName returns a random first name.
	FirstName() string
	// LastName returns a random last name.
	LastName() string
	// Email returns a random email address.
	Email() string
	// Phone returns a random phone number string.
	Phone() string
	// Address returns a random street address.
	Address() string
	// City returns a random city name.
	City() string
	// Country returns a random country name.
	Country() string
	// UUID returns a random v4 UUID string.
	UUID() string
	// Paragraph returns a random paragraph of text.
	Paragraph() string
	// Sentence returns a random sentence.
	Sentence() string
	// Word returns a random word.
	Word() string
	// IntBetween returns a random integer in [min, max].
	IntBetween(min, max int) int
	// Float64Between returns a random float64 in [min, max).
	Float64Between(min, max float64) float64
	// Bool returns a random boolean.
	Bool() bool
	// Date returns a random date.
	Date() time.Time
	// DateBetween returns a random date between start and end.
	DateBetween(start, end time.Time) time.Time
	// Pick returns a random element from the given slice.
	Pick(items []string) string
}
