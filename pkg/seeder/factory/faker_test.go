package factory

import (
	"strings"
	"testing"
	"time"
)

func newTestFaker() *DefaultFaker {
	return NewFaker(42)
}

func TestNewFaker(t *testing.T) {
	f := NewFaker(42)
	if f == nil {
		t.Errorf("expected non-nil")
	}
}

func TestFakerImplementsInterface(t *testing.T) {
	var _ Faker = NewFaker(1)
}

func TestName_ReturnsNonEmpty(t *testing.T) {
	f := newTestFaker()
	name := f.Name()
	if name == "" {
		t.Errorf("expected non-empty string")
	}
	if !strings.Contains(name, " ") {
		t.Errorf("%q not in %q", " ", name)
	}
}

func TestFirstName_ReturnsNonEmpty(t *testing.T) {
	f := newTestFaker()
	if f.FirstName() == "" {
		t.Errorf("expected non-empty string")
	}
}

func TestLastName_ReturnsNonEmpty(t *testing.T) {
	f := newTestFaker()
	if f.LastName() == "" {
		t.Errorf("expected non-empty string")
	}
}

func TestEmail_ReturnsValidFormat(t *testing.T) {
	f := newTestFaker()
	email := f.Email()
	if email == "" {
		t.Errorf("expected non-empty string")
	}
	if !strings.Contains(email, "@") {
		t.Errorf("%q not in %q", "@", email)
	}
	if !strings.Contains(email, ".") {
		t.Errorf("%q not in %q", ".", email)
	}
}

func TestPhone_ReturnsNonEmpty(t *testing.T) {
	f := newTestFaker()
	phone := f.Phone()
	if phone == "" {
		t.Errorf("expected non-empty string")
	}
	if !strings.HasPrefix(phone, "+1-") {
		t.Errorf("expected true")
	}
}

func TestAddress_ReturnsNonEmpty(t *testing.T) {
	f := newTestFaker()
	if f.Address() == "" {
		t.Errorf("expected non-empty string")
	}
}

func TestCity_ReturnsNonEmpty(t *testing.T) {
	f := newTestFaker()
	if f.City() == "" {
		t.Errorf("expected non-empty string")
	}
}

func TestCountry_ReturnsNonEmpty(t *testing.T) {
	f := newTestFaker()
	if f.Country() == "" {
		t.Errorf("expected non-empty string")
	}
}

func TestUUID_ReturnsValidFormat(t *testing.T) {
	f := newTestFaker()
	uuid := f.UUID()
	if uuid == "" {
		t.Errorf("expected non-empty string")
	}
	// UUID v4 format: 8-4-4-4-12 hex chars
	parts := strings.Split(uuid, "-")
	if len(parts) != 5 {
		t.Errorf("len = %d, want %d", len(parts), 5)
	}
	if len(parts[0]) != 8 {
		t.Errorf("len = %d, want %d", len(parts[0]), 8)
	}
	if len(parts[1]) != 4 {
		t.Errorf("len = %d, want %d", len(parts[1]), 4)
	}
	if len(parts[2]) != 4 {
		t.Errorf("len = %d, want %d", len(parts[2]), 4)
	}
	if len(parts[3]) != 4 {
		t.Errorf("len = %d, want %d", len(parts[3]), 4)
	}
	if len(parts[4]) != 12 {
		t.Errorf("len = %d, want %d", len(parts[4]), 12)
	}
}

func TestParagraph_ReturnsNonEmpty(t *testing.T) {
	f := newTestFaker()
	p := f.Paragraph()
	if p == "" {
		t.Errorf("expected non-empty string")
	}
	if !strings.Contains(p, ".") {
		t.Errorf("%q not in %q", ".", p)
	}
}

func TestSentence_ReturnsNonEmpty(t *testing.T) {
	f := newTestFaker()
	s := f.Sentence()
	if s == "" {
		t.Errorf("expected non-empty string")
	}
	if !strings.HasSuffix(s, ".") {
		t.Errorf("expected true")
	}
	// First character should be uppercase
	if strings.ToUpper(s[:1]) != s[:1] {
		t.Errorf("got %v, want %v", s[:1], strings.ToUpper(s[:1]))
	}
}

func TestWord_ReturnsNonEmpty(t *testing.T) {
	f := newTestFaker()
	if f.Word() == "" {
		t.Errorf("expected non-empty string")
	}
}

func TestIntBetween_ReturnsValueInRange(t *testing.T) {
	f := newTestFaker()
	for i := 0; i < 100; i++ {
		v := f.IntBetween(5, 15)
		if v < 5 {
			t.Errorf("got %v, want >= %v", v, 5)
		}
		if v > 15 {
			t.Errorf("got %v, want <= %v", v, 15)
		}
	}
}

func TestIntBetween_MinEqualsMax(t *testing.T) {
	f := newTestFaker()
	v := f.IntBetween(7, 7)
	if 7 != v {
		t.Errorf("got %v, want %v", v, 7)
	}
}

func TestIntBetween_MinGreaterThanMax(t *testing.T) {
	f := newTestFaker()
	v := f.IntBetween(10, 5)
	if 10 != v {
		t.Errorf("got %v, want %v", v, 10)
	}
}

func TestFloat64Between_ReturnsValueInRange(t *testing.T) {
	f := newTestFaker()
	for i := 0; i < 100; i++ {
		v := f.Float64Between(1.5, 9.5)
		if v < 1.5 {
			t.Errorf("got %v, want >= %v", v, 1.5)
		}
		if v >= 9.5 {
			t.Errorf("got %v, want < %v", v, 9.5)
		}
	}
}

func TestFloat64Between_MinEqualsMax(t *testing.T) {
	f := newTestFaker()
	v := f.Float64Between(3.14, 3.14)
	if 3.14 != v {
		t.Errorf("got %v, want %v", v, 3.14)
	}
}

func TestBool_ReturnsBoolean(t *testing.T) {
	f := newTestFaker()
	// Run enough times to see both values
	seenTrue := false
	seenFalse := false
	for i := 0; i < 100; i++ {
		if f.Bool() {
			seenTrue = true
		} else {
			seenFalse = true
		}
	}
	if !seenTrue {
		t.Errorf("should produce true at least once")
	}
	if !seenFalse {
		t.Errorf("should produce false at least once")
	}
}

func TestDate_ReturnsNonZero(t *testing.T) {
	f := newTestFaker()
	d := f.Date()
	if d.IsZero() {
		t.Errorf("expected false")
	}
}

func TestDateBetween_ReturnsDateInRange(t *testing.T) {
	f := newTestFaker()
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	for i := 0; i < 100; i++ {
		d := f.DateBetween(start, end)
		if d.Before(start) {
			t.Errorf("expected false")
		}
		if d.After(end) {
			t.Errorf("expected false")
		}
	}
}

func TestDateBetween_StartEqualsEnd(t *testing.T) {
	f := newTestFaker()
	t0 := time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC)
	d := f.DateBetween(t0, t0)
	if t0 != d {
		t.Errorf("got %v, want %v", d, t0)
	}
}

func TestPick_ReturnsElementFromSlice(t *testing.T) {
	f := newTestFaker()
	items := []string{"apple", "banana", "cherry"}
	for i := 0; i < 50; i++ {
		v := f.Pick(items)
		found := false
		for _, item := range items {
			if item == v {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%q not in %v", v, items)
		}
	}
}

func TestPick_EmptySlice(t *testing.T) {
	f := newTestFaker()
	v := f.Pick([]string{})
	if len(v) != 0 {
		t.Errorf("expected empty, got %v", v)
	}
}

func TestReproducibility(t *testing.T) {
	f1 := NewFaker(123)
	f2 := NewFaker(123)
	// Same seed should produce same sequence for deterministic methods
	if f1.FirstName() != f2.FirstName() {
		t.Errorf("got %v, want %v", f2.FirstName(), f1.FirstName())
	}
	if f1.LastName() != f2.LastName() {
		t.Errorf("got %v, want %v", f2.LastName(), f1.LastName())
	}
	if f1.IntBetween(0, 100) != f2.IntBetween(0, 100) {
		t.Errorf("got %v, want %v", f2.IntBetween(0, 100), f1.IntBetween(0, 100))
	}
	if f1.Word() != f2.Word() {
		t.Errorf("got %v, want %v", f2.Word(), f1.Word())
	}
}
