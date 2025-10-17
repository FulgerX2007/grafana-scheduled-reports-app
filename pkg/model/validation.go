package model

import (
	"fmt"
	"strings"

	"github.com/gorhill/cronexpr"
)

// ValidateRecipientDomains validates that all recipient email addresses match the allowed domain whitelist.
// If allowedDomains is empty, all domains are allowed.
// Returns an error if any recipient email has a domain not in the whitelist.
func ValidateRecipientDomains(recipients Recipients, allowedDomains []string) error {
	// If no domain whitelist is configured, allow all domains
	if len(allowedDomains) == 0 {
		return nil
	}

	// Collect all email addresses from all recipient fields
	allEmails := make([]string, 0)
	allEmails = append(allEmails, recipients.To...)
	allEmails = append(allEmails, recipients.CC...)
	allEmails = append(allEmails, recipients.BCC...)

	// Validate each email address
	for _, email := range allEmails {
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}

		// Extract domain from email address
		domain := extractDomain(email)
		if domain == "" {
			return fmt.Errorf("invalid email address format: %s", email)
		}

		// Check if domain is in the whitelist
		if !isDomainAllowed(domain, allowedDomains) {
			return fmt.Errorf("email domain '%s' is not allowed (email: %s). Allowed domains: %v", domain, email, allowedDomains)
		}
	}

	return nil
}

// extractDomain extracts the domain part from an email address
func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(parts[1]))
}

// isDomainAllowed checks if a domain matches any entry in the allowed domains list
// Supports exact matches and wildcard patterns (e.g., "*.example.com")
func isDomainAllowed(domain string, allowedDomains []string) bool {
	domain = strings.ToLower(domain)

	for _, allowed := range allowedDomains {
		allowed = strings.ToLower(strings.TrimSpace(allowed))

		// Check for exact match
		if domain == allowed {
			return true
		}

		// Check for wildcard pattern (e.g., "*.example.com")
		if strings.HasPrefix(allowed, "*.") {
			baseDomain := allowed[2:] // Remove "*."
			if domain == baseDomain || strings.HasSuffix(domain, "."+baseDomain) {
				return true
			}
		}
	}

	return false
}

// ValidateCronExpression validates a cron expression format.
// Returns an error if the expression cannot be parsed.
func ValidateCronExpression(cronExpr string) error {
	if cronExpr == "" {
		return fmt.Errorf("cron expression cannot be empty")
	}

	_, err := cronexpr.Parse(cronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression '%s': %v", cronExpr, err)
	}

	return nil
}
