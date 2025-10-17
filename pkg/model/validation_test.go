package model

import (
	"testing"
)

func TestValidateRecipientDomains(t *testing.T) {
	tests := []struct {
		name           string
		recipients     Recipients
		allowedDomains []string
		expectError    bool
		errorContains  string
	}{
		{
			name: "empty whitelist allows all domains",
			recipients: Recipients{
				To: []string{"user@example.com", "admin@company.org"},
			},
			allowedDomains: []string{},
			expectError:    false,
		},
		{
			name: "exact domain match - single recipient",
			recipients: Recipients{
				To: []string{"user@example.com"},
			},
			allowedDomains: []string{"example.com"},
			expectError:    false,
		},
		{
			name: "exact domain match - multiple recipients",
			recipients: Recipients{
				To:  []string{"user1@example.com", "user2@example.com"},
				CC:  []string{"cc@example.com"},
				BCC: []string{"bcc@example.com"},
			},
			allowedDomains: []string{"example.com"},
			expectError:    false,
		},
		{
			name: "multiple allowed domains",
			recipients: Recipients{
				To: []string{"user1@example.com", "user2@company.org"},
			},
			allowedDomains: []string{"example.com", "company.org"},
			expectError:    false,
		},
		{
			name: "wildcard domain match",
			recipients: Recipients{
				To: []string{"user@subdomain.example.com"},
			},
			allowedDomains: []string{"*.example.com"},
			expectError:    false,
		},
		{
			name: "wildcard matches base domain too",
			recipients: Recipients{
				To: []string{"user@example.com"},
			},
			allowedDomains: []string{"*.example.com"},
			expectError:    false,
		},
		{
			name: "wildcard matches nested subdomains",
			recipients: Recipients{
				To: []string{"user@dev.staging.example.com"},
			},
			allowedDomains: []string{"*.example.com"},
			expectError:    false,
		},
		{
			name: "domain not in whitelist",
			recipients: Recipients{
				To: []string{"user@forbidden.com"},
			},
			allowedDomains: []string{"example.com"},
			expectError:    true,
			errorContains:  "not allowed",
		},
		{
			name: "one recipient allowed, one not",
			recipients: Recipients{
				To: []string{"user@example.com", "user@forbidden.com"},
			},
			allowedDomains: []string{"example.com"},
			expectError:    true,
			errorContains:  "forbidden.com",
		},
		{
			name: "invalid email format",
			recipients: Recipients{
				To: []string{"not-an-email"},
			},
			allowedDomains: []string{"example.com"},
			expectError:    true,
			errorContains:  "invalid email",
		},
		{
			name: "email with spaces",
			recipients: Recipients{
				To: []string{"  user@example.com  "},
			},
			allowedDomains: []string{"example.com"},
			expectError:    false,
		},
		{
			name: "case insensitive domain matching",
			recipients: Recipients{
				To: []string{"User@Example.COM"},
			},
			allowedDomains: []string{"example.com"},
			expectError:    false,
		},
		{
			name: "empty recipient strings are ignored",
			recipients: Recipients{
				To: []string{"", "user@example.com", ""},
			},
			allowedDomains: []string{"example.com"},
			expectError:    false,
		},
		{
			name: "wildcard with exact base domain allowed",
			recipients: Recipients{
				To: []string{"user@example.com", "user@sub.example.com"},
			},
			allowedDomains: []string{"*.example.com"},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecipientDomains(tt.recipients, tt.allowedDomains)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("error message '%s' does not contain '%s'", err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		email          string
		expectedDomain string
	}{
		{"user@example.com", "example.com"},
		{"admin@company.org", "company.org"},
		{"test@subdomain.example.com", "subdomain.example.com"},
		{"User@Example.COM", "example.com"},
		{"  user@example.com  ", "example.com"},
		{"invalid-email", ""},
		{"no-at-sign", ""},
		{"multiple@at@signs.com", ""},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			domain := extractDomain(tt.email)
			if domain != tt.expectedDomain {
				t.Errorf("extractDomain(%q) = %q, want %q", tt.email, domain, tt.expectedDomain)
			}
		})
	}
}

func TestIsDomainAllowed(t *testing.T) {
	tests := []struct {
		domain         string
		allowedDomains []string
		expected       bool
	}{
		// Exact matches
		{"example.com", []string{"example.com"}, true},
		{"company.org", []string{"example.com", "company.org"}, true},
		{"forbidden.com", []string{"example.com"}, false},

		// Case insensitive
		{"Example.COM", []string{"example.com"}, true},
		{"EXAMPLE.COM", []string{"example.com"}, true},

		// Wildcard patterns
		{"subdomain.example.com", []string{"*.example.com"}, true},
		{"dev.subdomain.example.com", []string{"*.example.com"}, true},
		{"example.com", []string{"*.example.com"}, true}, // Base domain matches wildcard
		{"notexample.com", []string{"*.example.com"}, false},

		// Mixed patterns
		{"example.com", []string{"example.com", "*.other.com"}, true},
		{"sub.other.com", []string{"example.com", "*.other.com"}, true},
		{"forbidden.com", []string{"example.com", "*.other.com"}, false},

		// Empty whitelist
		{"anything.com", []string{}, false},
	}

	for _, tt := range tests {
		testName := tt.domain
		if len(tt.allowedDomains) > 0 {
			testName += "_vs_" + tt.allowedDomains[0]
		} else {
			testName += "_vs_empty"
		}
		t.Run(testName, func(t *testing.T) {
			result := isDomainAllowed(tt.domain, tt.allowedDomains)
			if result != tt.expected {
				t.Errorf("isDomainAllowed(%q, %v) = %v, want %v",
					tt.domain, tt.allowedDomains, result, tt.expected)
			}
		})
	}
}

func TestValidateCronExpression(t *testing.T) {
	tests := []struct {
		name          string
		cronExpr      string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid daily cron (midnight)",
			cronExpr:    "0 0 * * *",
			expectError: false,
		},
		{
			name:        "valid weekly cron (Monday midnight)",
			cronExpr:    "0 0 * * 1",
			expectError: false,
		},
		{
			name:        "valid monthly cron (1st midnight)",
			cronExpr:    "0 0 1 * *",
			expectError: false,
		},
		{
			name:        "valid hourly cron",
			cronExpr:    "0 * * * *",
			expectError: false,
		},
		{
			name:        "valid every 15 minutes",
			cronExpr:    "*/15 * * * *",
			expectError: false,
		},
		{
			name:          "invalid - empty expression",
			cronExpr:      "",
			expectError:   true,
			errorContains: "cannot be empty",
		},
		{
			name:          "invalid - too few fields",
			cronExpr:      "0 0 *",
			expectError:   true,
			errorContains: "invalid cron expression",
		},
		{
			name:          "invalid - bad syntax",
			cronExpr:      "invalid cron",
			expectError:   true,
			errorContains: "invalid cron expression",
		},
		{
			name:          "invalid - out of range minute",
			cronExpr:      "60 0 * * *",
			expectError:   true,
			errorContains: "invalid cron expression",
		},
		{
			name:          "invalid - out of range hour",
			cronExpr:      "0 24 * * *",
			expectError:   true,
			errorContains: "invalid cron expression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCronExpression(tt.cronExpr)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("error message '%s' does not contain '%s'", err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
