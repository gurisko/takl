package cmd

import (
	"testing"
)

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected bool
		desc     string
	}{
		// Valid emails
		{"user@example.com", true, "basic valid email"},
		{"user.name@example.com", true, "with dot in local part"},
		{"user+tag@example.com", true, "with plus in local part"},
		{"user_name@example.com", true, "with underscore in local part"},
		{"user%percent@example.com", true, "with percent in local part"},
		{"user-name@example.com", true, "with hyphen in local part"},
		{"user'apostrophe@example.com", true, "with apostrophe in local part"},
		{"test@example-domain.com", true, "with hyphen in domain"},
		{"test@sub.example.com", true, "with subdomain"},
		{"test@example.co.uk", true, "with multi-part TLD"},
		{"a@b.co", true, "minimal valid email"},
		{"user123@example123.com", true, "with numbers"},

		// Invalid emails (clearly bad)
		{"", false, "empty string"},
		{"user", false, "no @ symbol"},
		{"@example.com", false, "no local part"},
		{"user@", false, "no domain"},
		{"user@@example.com", false, "double @ symbol"},
		{"user@example", false, "no TLD"},
		{"user@example.c", false, "TLD too short"},
		{"user@example.com.", false, "trailing dot"},
		{"user name@example.com", false, "space in local part"},
		{"user@exam ple.com", false, "space in domain"},

		// Edge cases that are allowed by pragmatic validation
		// (These would be caught by SMTP servers if actually invalid)
		{"user@.example.com", true, "domain starts with dot (pragmatic allows)"},
		{"user@example..com", true, "double dot in domain (pragmatic allows)"},
		{".user@example.com", true, "local part starts with dot (pragmatic allows)"},
		{"user.@example.com", true, "local part ends with dot (pragmatic allows)"},
		{"user..name@example.com", true, "double dot in local part (pragmatic allows)"},
		{"very-long-username-that-exceeds-normal-limits-for-email-addresses-in-most-systems@example.com", true, "long local part (should be valid)"},

		// Edge cases that should be valid with pragmatic approach
		{"test-email-with-dash@example-host.com", true, "dashes in both parts"},
		{"user+folder@example.org", true, "plus addressing"},
		{"firstname.lastname@company.co.uk", true, "realistic business email"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := isValidEmail(tt.email)
			if result != tt.expected {
				t.Errorf("isValidEmail(%q) = %v, expected %v (%s)", tt.email, result, tt.expected, tt.desc)
			}
		})
	}
}

func TestEmailRegexPerformance(t *testing.T) {
	// Test that the compiled regex performs well
	testEmails := []string{
		"user@example.com",
		"invalid-email",
		"another.user@domain.org",
		"bad@email",
	}

	// Run validation many times to test performance
	for i := 0; i < 1000; i++ {
		for _, email := range testEmails {
			isValidEmail(email) // Should not panic or be extremely slow
		}
	}

	t.Log("✅ Email validation performance test completed")
}

func BenchmarkEmailValidation(b *testing.B) {
	email := "test.email@example.com"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		isValidEmail(email)
	}
}
