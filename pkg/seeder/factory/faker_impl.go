package factory

import (
	"crypto/rand"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"strings"
	"time"
)

// Data pools for realistic-looking fake data.
var (
	firstNames = []string{
		"Alice", "Bob", "Charlie", "Diana", "Edward",
		"Fiona", "George", "Hannah", "Ivan", "Julia",
		"Kevin", "Laura", "Michael", "Nina", "Oscar",
	}

	lastNames = []string{
		"Smith", "Johnson", "Williams", "Brown", "Jones",
		"Garcia", "Miller", "Davis", "Rodriguez", "Martinez",
		"Anderson", "Taylor", "Thomas", "Moore", "Jackson",
	}

	cities = []string{
		"New York", "London", "Tokyo", "Paris", "Berlin",
		"Sydney", "Toronto", "Mumbai", "São Paulo", "Cairo",
		"Seoul", "Rome", "Bangkok", "Istanbul", "Lagos",
	}

	countries = []string{
		"United States", "United Kingdom", "Japan", "France", "Germany",
		"Australia", "Canada", "India", "Brazil", "Egypt",
		"South Korea", "Italy", "Thailand", "Turkey", "Nigeria",
	}

	streetNames = []string{
		"Main St", "Oak Ave", "Elm St", "Park Blvd", "Cedar Ln",
		"Maple Dr", "Pine Rd", "Washington Ave", "Lake St", "Hill Rd",
	}

	emailDomains = []string{
		"example.com", "test.org", "mail.net", "demo.io", "sample.dev",
	}

	words = []string{
		"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog",
		"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing",
		"elit", "sed", "do", "eiusmod", "tempor",
	}
)

// DefaultFaker implements the Faker interface using a seeded random source.
type DefaultFaker struct {
	rng *mathrand.Rand
}

// NewFaker creates a new DefaultFaker with the given seed for reproducibility.
func NewFaker(seed int64) *DefaultFaker {
	return &DefaultFaker{
		rng: mathrand.New(mathrand.NewSource(seed)),
	}
}

// NewFakerWithRand creates a new DefaultFaker using the provided *math/rand.Rand.
func NewFakerWithRand(rng *mathrand.Rand) *DefaultFaker {
	return &DefaultFaker{rng: rng}
}

func (f *DefaultFaker) pick(pool []string) string {
	return pool[f.rng.Intn(len(pool))]
}

func (f *DefaultFaker) Name() string {
	return f.FirstName() + " " + f.LastName()
}

func (f *DefaultFaker) FirstName() string {
	return f.pick(firstNames)
}

func (f *DefaultFaker) LastName() string {
	return f.pick(lastNames)
}

func (f *DefaultFaker) Email() string {
	first := strings.ToLower(f.FirstName())
	last := strings.ToLower(f.LastName())
	domain := f.pick(emailDomains)
	return fmt.Sprintf("%s.%s@%s", first, last, domain)
}

func (f *DefaultFaker) Phone() string {
	// Format: +1-XXX-XXX-XXXX
	var b strings.Builder
	b.WriteString("+1-")
	for i := 0; i < 3; i++ {
		b.WriteByte(byte('0' + f.rng.Intn(10)))
	}
	b.WriteByte('-')
	for i := 0; i < 3; i++ {
		b.WriteByte(byte('0' + f.rng.Intn(10)))
	}
	b.WriteByte('-')
	for i := 0; i < 4; i++ {
		b.WriteByte(byte('0' + f.rng.Intn(10)))
	}
	return b.String()
}

func (f *DefaultFaker) Address() string {
	num := f.rng.Intn(9999) + 1
	street := f.pick(streetNames)
	return fmt.Sprintf("%d %s", num, street)
}

func (f *DefaultFaker) City() string {
	return f.pick(cities)
}

func (f *DefaultFaker) Country() string {
	return f.pick(countries)
}

// UUID generates a v4 UUID using crypto/rand for proper randomness.
func (f *DefaultFaker) UUID() string {
	var uuid [16]byte
	_, err := rand.Read(uuid[:])
	if err != nil {
		// Fallback to math/rand if crypto/rand fails
		for i := range uuid {
			uuid[i] = byte(f.rng.Intn(256))
		}
	}
	// Set version 4
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	// Set variant bits
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

func (f *DefaultFaker) Paragraph() string {
	count := f.rng.Intn(3) + 3 // 3-5 sentences
	sentences := make([]string, count)
	for i := range sentences {
		sentences[i] = f.Sentence()
	}
	return strings.Join(sentences, " ")
}

func (f *DefaultFaker) Sentence() string {
	count := f.rng.Intn(6) + 4 // 4-9 words
	w := make([]string, count)
	for i := range w {
		w[i] = f.Word()
	}
	// Capitalize first word
	w[0] = strings.ToUpper(w[0][:1]) + w[0][1:]
	return strings.Join(w, " ") + "."
}

func (f *DefaultFaker) Word() string {
	return f.pick(words)
}

func (f *DefaultFaker) IntBetween(min, max int) int {
	if min >= max {
		return min
	}
	return min + f.rng.Intn(max-min+1)
}

func (f *DefaultFaker) Float64Between(min, max float64) float64 {
	if min >= max {
		return min
	}
	return min + f.rng.Float64()*(max-min)
}

func (f *DefaultFaker) Bool() bool {
	return f.rng.Intn(2) == 1
}

func (f *DefaultFaker) Date() time.Time {
	// Random date between 2000-01-01 and 2030-12-31
	start := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC)
	return f.DateBetween(start, end)
}

func (f *DefaultFaker) DateBetween(start, end time.Time) time.Time {
	if !start.Before(end) {
		return start
	}
	delta := end.Unix() - start.Unix()
	n, err := rand.Int(rand.Reader, big.NewInt(delta))
	if err != nil {
		// Fallback to math/rand
		offset := f.rng.Int63n(delta)
		return start.Add(time.Duration(offset) * time.Second)
	}
	return start.Add(time.Duration(n.Int64()) * time.Second)
}

func (f *DefaultFaker) Pick(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[f.rng.Intn(len(items))]
}
