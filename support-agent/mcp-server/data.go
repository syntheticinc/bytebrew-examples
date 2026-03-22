package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// --- Domain types ------------------------------------------------------------

// Customer represents a CloudSync SaaS customer.
type Customer struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Company   string `json:"company"`
	Plan      string `json:"plan"`
	MRR       int    `json:"mrr_cents"`
	SignupDate string `json:"signup_date"`
	Status    string `json:"status"`
}

// Ticket represents a support ticket.
type Ticket struct {
	ID          string `json:"id"`
	CustomerID  string `json:"customer_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Category    string `json:"category"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	AssignedTo  string `json:"assigned_to,omitempty"`
}

// KBArticle represents a knowledge base article.
type KBArticle struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Content  string   `json:"content"`
	Tags     []string `json:"tags"`
}

// ServiceStatus represents the health of a CloudSync microservice.
type ServiceStatus struct {
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	Uptime    string  `json:"uptime"`
	Latency   string  `json:"latency_p99"`
	ErrorRate float64 `json:"error_rate_pct"`
	Message   string  `json:"message,omitempty"`
}

// ErrorLogEntry represents a single error log entry.
type ErrorLogEntry struct {
	Timestamp string `json:"timestamp"`
	Service   string `json:"service"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	TraceID   string `json:"trace_id"`
}

// Invoice represents a billing invoice.
type Invoice struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	Amount     int    `json:"amount_cents"`
	Status     string `json:"status"`
	IssuedAt   string `json:"issued_at"`
	PaidAt     string `json:"paid_at,omitempty"`
	Plan       string `json:"plan"`
}

// --- Mock data ---------------------------------------------------------------

var customers = []Customer{
	{ID: "CUST-001", Name: "Alice Chen", Email: "alice@techstartup.io", Company: "TechStartup Inc", Plan: "Pro", MRR: 7900, SignupDate: "2024-06-15", Status: "active"},
	{ID: "CUST-002", Name: "Bob Williams", Email: "bob@megacorp.com", Company: "MegaCorp", Plan: "Enterprise", MRR: 49900, SignupDate: "2023-11-01", Status: "active"},
	{ID: "CUST-003", Name: "Carol Martinez", Email: "carol@designlab.co", Company: "DesignLab Co", Plan: "Starter", MRR: 1900, SignupDate: "2025-01-10", Status: "active"},
	{ID: "CUST-004", Name: "David Park", Email: "david@finserv.com", Company: "FinServ Solutions", Plan: "Enterprise", MRR: 49900, SignupDate: "2024-03-22", Status: "active"},
	{ID: "CUST-005", Name: "Emily Johnson", Email: "emily@freelance.me", Company: "", Plan: "Starter", MRR: 1900, SignupDate: "2025-02-01", Status: "active"},
	{ID: "CUST-006", Name: "Frank Lee", Email: "frank@ecommshop.com", Company: "E-Comm Shop", Plan: "Pro", MRR: 7900, SignupDate: "2024-09-05", Status: "active"},
	{ID: "CUST-007", Name: "Grace Kim", Email: "grace@dataflows.io", Company: "DataFlows", Plan: "Pro", MRR: 7900, SignupDate: "2024-07-18", Status: "churned"},
	{ID: "CUST-008", Name: "Henry Davis", Email: "henry@agency360.com", Company: "Agency360", Plan: "Enterprise", MRR: 49900, SignupDate: "2023-05-30", Status: "active"},
}

var tickets = []Ticket{
	{ID: "TKT-001", CustomerID: "CUST-001", Title: "Cannot sync large files", Description: "Files over 500MB fail to sync with timeout error. Started happening yesterday.", Priority: "high", Category: "technical", Status: "open", CreatedAt: "2026-03-20T10:30:00Z", UpdatedAt: "2026-03-20T10:30:00Z"},
	{ID: "TKT-002", CustomerID: "CUST-002", Title: "API rate limit too low", Description: "Our integration is hitting the 1000 req/min rate limit. Need Enterprise+ limits.", Priority: "medium", Category: "technical", Status: "open", CreatedAt: "2026-03-19T14:20:00Z", UpdatedAt: "2026-03-20T09:00:00Z", AssignedTo: "tech-team"},
	{ID: "TKT-003", CustomerID: "CUST-003", Title: "Billing discrepancy", Description: "Charged $29 instead of $19 on my last invoice. Plan is Starter.", Priority: "high", Category: "billing", Status: "open", CreatedAt: "2026-03-21T08:15:00Z", UpdatedAt: "2026-03-21T08:15:00Z"},
	{ID: "TKT-004", CustomerID: "CUST-004", Title: "SSO integration not working", Description: "SAML SSO returns invalid_response error after IdP upgrade.", Priority: "critical", Category: "technical", Status: "in_progress", CreatedAt: "2026-03-18T11:00:00Z", UpdatedAt: "2026-03-20T16:30:00Z", AssignedTo: "tech-team"},
	{ID: "TKT-005", CustomerID: "CUST-005", Title: "How to upgrade plan", Description: "I want to move from Starter to Pro. What are the steps?", Priority: "low", Category: "billing", Status: "open", CreatedAt: "2026-03-21T15:00:00Z", UpdatedAt: "2026-03-21T15:00:00Z"},
	{ID: "TKT-006", CustomerID: "CUST-001", Title: "Webhook delivery delays", Description: "Webhooks are arriving 30-60 seconds late since the maintenance window.", Priority: "medium", Category: "technical", Status: "open", CreatedAt: "2026-03-21T09:45:00Z", UpdatedAt: "2026-03-21T09:45:00Z"},
	{ID: "TKT-007", CustomerID: "CUST-006", Title: "Invoice PDF not downloading", Description: "Clicking download on invoice INV-2026-042 returns 404.", Priority: "low", Category: "billing", Status: "resolved", CreatedAt: "2026-03-15T12:00:00Z", UpdatedAt: "2026-03-16T10:00:00Z", AssignedTo: "billing-team"},
	{ID: "TKT-008", CustomerID: "CUST-002", Title: "Data export taking too long", Description: "Full account export has been running for 6 hours. Usually takes 30 min.", Priority: "high", Category: "technical", Status: "in_progress", CreatedAt: "2026-03-20T07:00:00Z", UpdatedAt: "2026-03-20T13:00:00Z", AssignedTo: "tech-team"},
	{ID: "TKT-009", CustomerID: "CUST-007", Title: "Request account deletion", Description: "Please delete all our data per GDPR. Account already cancelled.", Priority: "medium", Category: "billing", Status: "open", CreatedAt: "2026-03-19T16:30:00Z", UpdatedAt: "2026-03-19T16:30:00Z"},
	{ID: "TKT-010", CustomerID: "CUST-008", Title: "Custom domain SSL error", Description: "Custom domain sync.agency360.com showing ERR_CERT_DATE_INVALID.", Priority: "critical", Category: "technical", Status: "open", CreatedAt: "2026-03-22T06:00:00Z", UpdatedAt: "2026-03-22T06:00:00Z"},
	{ID: "TKT-011", CustomerID: "CUST-004", Title: "Need additional seats", Description: "Adding 50 more users. Need updated quote for Enterprise plan.", Priority: "medium", Category: "billing", Status: "open", CreatedAt: "2026-03-21T11:00:00Z", UpdatedAt: "2026-03-21T11:00:00Z"},
	{ID: "TKT-012", CustomerID: "CUST-003", Title: "Mobile app crash on upload", Description: "iOS app crashes when uploading photos. Version 3.2.1, iPhone 15.", Priority: "high", Category: "technical", Status: "open", CreatedAt: "2026-03-22T08:30:00Z", UpdatedAt: "2026-03-22T08:30:00Z"},
	{ID: "TKT-013", CustomerID: "CUST-006", Title: "Refund for downtime", Description: "Requesting refund for 4-hour outage on March 10. SLA breach.", Priority: "high", Category: "billing", Status: "open", CreatedAt: "2026-03-17T09:00:00Z", UpdatedAt: "2026-03-17T09:00:00Z"},
	{ID: "TKT-014", CustomerID: "CUST-001", Title: "Two-factor auth lockout", Description: "Lost my 2FA device. Cannot log in. Need account recovery.", Priority: "critical", Category: "technical", Status: "resolved", CreatedAt: "2026-03-14T18:00:00Z", UpdatedAt: "2026-03-15T09:00:00Z", AssignedTo: "tech-team"},
	{ID: "TKT-015", CustomerID: "CUST-008", Title: "Annual billing switch", Description: "Want to switch from monthly to annual billing for 20% discount.", Priority: "low", Category: "billing", Status: "open", CreatedAt: "2026-03-22T10:00:00Z", UpdatedAt: "2026-03-22T10:00:00Z"},
}

var kbArticles = []KBArticle{
	{
		ID:       "KB-001",
		Title:    "File Sync Troubleshooting",
		Category: "technical",
		Tags:     []string{"sync", "files", "timeout", "upload"},
		Content: `# File Sync Troubleshooting

## Common Issues

### Large File Timeouts
Files over 500MB may timeout on slower connections. Solutions:
1. Enable chunked upload in Settings > Sync > Advanced
2. Check your network bandwidth (minimum 10 Mbps recommended)
3. Ensure the CloudSync desktop agent is updated to v3.2+

### Sync Conflicts
When the same file is edited on multiple devices:
- CloudSync keeps both versions with a conflict suffix
- Open Settings > Sync > Conflict Resolution to set preferred behavior
- Options: keep-newest, keep-both, ask-every-time

### Selective Sync
To exclude large folders: Settings > Sync > Selective Sync.
Pro and Enterprise plans support regex-based exclusion patterns.`,
	},
	{
		ID:       "KB-002",
		Title:    "API Rate Limits and Quotas",
		Category: "technical",
		Tags:     []string{"api", "rate-limit", "quota", "throttle"},
		Content: `# API Rate Limits

## Limits by Plan
| Plan       | Requests/min | Concurrent | Daily Quota  |
|------------|-------------|------------|--------------|
| Starter    | 100         | 5          | 10,000       |
| Pro        | 1,000       | 25         | 100,000      |
| Enterprise | 10,000      | 100        | Unlimited    |

## Rate Limit Headers
All API responses include:
- X-RateLimit-Limit: your plan limit
- X-RateLimit-Remaining: requests left in window
- X-RateLimit-Reset: UTC epoch when window resets

## Handling 429 Errors
Implement exponential backoff: wait 1s, 2s, 4s, 8s (max 30s).
Contact support for temporary limit increases during migrations.`,
	},
	{
		ID:       "KB-003",
		Title:    "Billing and Subscription Management",
		Category: "billing",
		Tags:     []string{"billing", "subscription", "invoice", "upgrade", "downgrade", "refund"},
		Content: `# Billing and Subscriptions

## Plans
- **Starter** ($19/mo): 5 users, 50GB storage, basic sync
- **Pro** ($79/mo): 25 users, 500GB storage, API access, priority support
- **Enterprise** ($499/mo): Unlimited users, 5TB storage, SSO, SLA, dedicated support

## Upgrading
Upgrades are prorated. You only pay the difference for the remaining billing cycle.
Go to Account > Subscription > Change Plan.

## Downgrading
Downgrades take effect at the end of the current billing cycle.
Data exceeding the new plan's storage limit must be removed first.

## Refund Policy
- Full refund within 14 days of initial purchase
- SLA credit: 10x the hourly cost for each hour of downtime exceeding SLA
- Refunds over $100 require manager approval
- Processing time: 5-10 business days`,
	},
	{
		ID:       "KB-004",
		Title:    "SSO and Authentication Setup",
		Category: "technical",
		Tags:     []string{"sso", "saml", "oauth", "2fa", "authentication"},
		Content: `# SSO and Authentication

## SAML SSO (Enterprise only)
1. Go to Admin > Security > SSO
2. Upload your IdP metadata XML or enter the SSO URL and certificate
3. Map attributes: email, firstName, lastName, groups
4. Test with a single user before enabling for all

### Common SAML Errors
- **invalid_response**: IdP certificate mismatch or clock skew >5 minutes
- **audience_mismatch**: Check Entity ID matches exactly
- **signature_error**: Re-upload the IdP signing certificate

## Two-Factor Authentication
All plans support TOTP (Google Authenticator, Authy).
Enterprise plans also support hardware keys (FIDO2/WebAuthn).

## Account Recovery
If locked out of 2FA:
1. Use one of your backup recovery codes
2. Contact support with account verification (company email + last 4 of card)
3. Recovery takes 24-48 hours for security verification`,
	},
	{
		ID:       "KB-005",
		Title:    "Webhooks and Integrations",
		Category: "technical",
		Tags:     []string{"webhooks", "integrations", "api", "events"},
		Content: `# Webhooks and Integrations

## Webhook Events
- file.created, file.updated, file.deleted
- user.added, user.removed
- sync.completed, sync.failed

## Webhook Configuration
1. Go to Settings > Integrations > Webhooks
2. Add endpoint URL (must be HTTPS)
3. Select events to subscribe to
4. Optionally set a signing secret for verification

## Delivery Guarantees
- At-least-once delivery with automatic retry
- Retry schedule: 1min, 5min, 30min, 2hr, 12hr (then marked failed)
- Webhook timeout: 30 seconds
- Failed deliveries visible in Settings > Integrations > Delivery Log

## Known Issues
- After maintenance windows, webhook delivery may be delayed up to 5 minutes
- This is due to the event queue draining backlog and is expected behavior`,
	},
}

var serviceStatuses = map[string]ServiceStatus{
	"api-gateway":   {Name: "api-gateway", Status: "healthy", Uptime: "99.99%", Latency: "45ms", ErrorRate: 0.01, Message: ""},
	"auth-service":  {Name: "auth-service", Status: "healthy", Uptime: "99.98%", Latency: "32ms", ErrorRate: 0.02, Message: ""},
	"storage":       {Name: "storage", Status: "degraded", Uptime: "99.5%", Latency: "890ms", ErrorRate: 2.1, Message: "Elevated latency on large file operations. Engineering investigating. ETA: 2 hours."},
	"billing":       {Name: "billing", Status: "healthy", Uptime: "99.99%", Latency: "28ms", ErrorRate: 0.0, Message: ""},
	"notifications": {Name: "notifications", Status: "healthy", Uptime: "99.95%", Latency: "120ms", ErrorRate: 0.05, Message: ""},
}

// errorLogsByCustomer provides realistic error logs per customer.
var errorLogsByCustomer = map[string][]ErrorLogEntry{
	"CUST-001": {
		{Timestamp: "2026-03-22T09:45:12Z", Service: "storage", Level: "ERROR", Message: "Upload timeout: file size 650MB exceeded chunk timeout of 300s", TraceID: "tr-a1b2c3"},
		{Timestamp: "2026-03-22T09:30:05Z", Service: "storage", Level: "WARN", Message: "Slow upload detected: 2.3 MB/s for file project-assets.zip", TraceID: "tr-d4e5f6"},
		{Timestamp: "2026-03-22T08:15:00Z", Service: "notifications", Level: "ERROR", Message: "Webhook delivery delayed: 45s latency to endpoint https://techstartup.io/hooks", TraceID: "tr-g7h8i9"},
		{Timestamp: "2026-03-21T22:00:00Z", Service: "storage", Level: "ERROR", Message: "Upload timeout: file size 512MB failed after 3 retries", TraceID: "tr-j0k1l2"},
	},
	"CUST-002": {
		{Timestamp: "2026-03-22T07:30:00Z", Service: "api-gateway", Level: "WARN", Message: "Rate limit approaching: 950/1000 requests in current window", TraceID: "tr-m3n4o5"},
		{Timestamp: "2026-03-22T06:00:00Z", Service: "storage", Level: "ERROR", Message: "Export job EXP-7821 timed out after 6h. Data size: 2.1TB", TraceID: "tr-p6q7r8"},
		{Timestamp: "2026-03-21T18:00:00Z", Service: "api-gateway", Level: "ERROR", Message: "Rate limit exceeded: 429 returned for 47 requests", TraceID: "tr-s9t0u1"},
	},
	"CUST-003": {
		{Timestamp: "2026-03-22T08:32:00Z", Service: "api-gateway", Level: "ERROR", Message: "Client crash report: iOS app v3.2.1, upload_controller.dart:142, NullPointerException", TraceID: "tr-v2w3x4"},
		{Timestamp: "2026-03-22T08:31:55Z", Service: "storage", Level: "ERROR", Message: "Upload interrupted: client disconnected during multipart upload", TraceID: "tr-y5z6a7"},
	},
	"CUST-004": {
		{Timestamp: "2026-03-20T16:00:00Z", Service: "auth-service", Level: "ERROR", Message: "SAML assertion validation failed: certificate fingerprint mismatch", TraceID: "tr-b8c9d0"},
		{Timestamp: "2026-03-20T15:55:00Z", Service: "auth-service", Level: "ERROR", Message: "SSO login failed for user david@finserv.com: invalid_response", TraceID: "tr-e1f2g3"},
		{Timestamp: "2026-03-20T15:50:00Z", Service: "auth-service", Level: "WARN", Message: "Clock skew detected: IdP timestamp 6 minutes ahead of server", TraceID: "tr-h4i5j6"},
	},
	"CUST-008": {
		{Timestamp: "2026-03-22T06:05:00Z", Service: "api-gateway", Level: "ERROR", Message: "TLS handshake failed for custom domain sync.agency360.com: certificate expired", TraceID: "tr-k7l8m9"},
		{Timestamp: "2026-03-22T06:04:00Z", Service: "api-gateway", Level: "WARN", Message: "SSL certificate for sync.agency360.com expires in 0 days", TraceID: "tr-n0o1p2"},
	},
}

var invoices = []Invoice{
	{ID: "INV-2026-001", CustomerID: "CUST-001", Amount: 7900, Status: "paid", IssuedAt: "2026-03-01T00:00:00Z", PaidAt: "2026-03-01T00:00:00Z", Plan: "Pro"},
	{ID: "INV-2026-002", CustomerID: "CUST-002", Amount: 49900, Status: "paid", IssuedAt: "2026-03-01T00:00:00Z", PaidAt: "2026-03-01T00:00:00Z", Plan: "Enterprise"},
	{ID: "INV-2026-003", CustomerID: "CUST-003", Amount: 2900, Status: "paid", IssuedAt: "2026-03-01T00:00:00Z", PaidAt: "2026-03-02T00:00:00Z", Plan: "Starter"},
	{ID: "INV-2026-004", CustomerID: "CUST-004", Amount: 49900, Status: "paid", IssuedAt: "2026-03-01T00:00:00Z", PaidAt: "2026-03-01T00:00:00Z", Plan: "Enterprise"},
	{ID: "INV-2026-005", CustomerID: "CUST-005", Amount: 1900, Status: "paid", IssuedAt: "2026-03-01T00:00:00Z", PaidAt: "2026-03-03T00:00:00Z", Plan: "Starter"},
	{ID: "INV-2026-006", CustomerID: "CUST-006", Amount: 7900, Status: "paid", IssuedAt: "2026-03-01T00:00:00Z", PaidAt: "2026-03-01T00:00:00Z", Plan: "Pro"},
	{ID: "INV-2026-007", CustomerID: "CUST-007", Amount: 7900, Status: "refunded", IssuedAt: "2026-02-01T00:00:00Z", PaidAt: "2026-02-01T00:00:00Z", Plan: "Pro"},
	{ID: "INV-2026-008", CustomerID: "CUST-008", Amount: 49900, Status: "paid", IssuedAt: "2026-03-01T00:00:00Z", PaidAt: "2026-03-01T00:00:00Z", Plan: "Enterprise"},
	{ID: "INV-2026-042", CustomerID: "CUST-006", Amount: 7900, Status: "paid", IssuedAt: "2026-02-01T00:00:00Z", PaidAt: "2026-02-01T00:00:00Z", Plan: "Pro"},
}

var validPlans = map[string]bool{
	"Starter":    true,
	"Pro":        true,
	"Enterprise": true,
}

// --- Thread-safe stores ------------------------------------------------------

// TicketStore is a thread-safe in-memory ticket store.
type TicketStore struct {
	mu     sync.Mutex
	nextID int
}

var ticketStore = &TicketStore{nextID: 15}

func (s *TicketStore) Create(customerID, title, description, priority, category string) Ticket {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	t := Ticket{
		ID:          fmt.Sprintf("TKT-%03d", s.nextID),
		CustomerID:  customerID,
		Title:       title,
		Description: description,
		Priority:    priority,
		Category:    category,
		Status:      "open",
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	tickets = append(tickets, t)
	return t
}

// RefundRecord represents a processed refund.
type RefundRecord struct {
	RefundID  string `json:"refund_id"`
	InvoiceID string `json:"invoice_id"`
	Amount    int    `json:"amount_cents"`
	Reason    string `json:"reason"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// RefundStore is a thread-safe in-memory refund store.
type RefundStore struct {
	mu      sync.Mutex
	refunds []RefundRecord
	nextID  int
}

var refundStore = &RefundStore{nextID: 0}

func (s *RefundStore) Create(invoiceID string, amount int, reason string) RefundRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	status := "approved"
	if amount > 10000 { // > $100
		status = "pending_manager_approval"
	}

	r := RefundRecord{
		RefundID:  fmt.Sprintf("REF-%03d", s.nextID),
		InvoiceID: invoiceID,
		Amount:    amount,
		Reason:    reason,
		Status:    status,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	s.refunds = append(s.refunds, r)
	return r
}

// --- Lookup helpers ----------------------------------------------------------

// findCustomer searches by ID, email, or name (case-insensitive partial match).
func findCustomer(identifier string) *Customer {
	lower := strings.ToLower(identifier)

	for i := range customers {
		if strings.EqualFold(customers[i].ID, identifier) {
			return &customers[i]
		}
	}

	for i := range customers {
		if strings.EqualFold(customers[i].Email, identifier) {
			return &customers[i]
		}
	}

	for i := range customers {
		if strings.Contains(strings.ToLower(customers[i].Name), lower) {
			return &customers[i]
		}
	}

	return nil
}

// findTicket returns a ticket by ID.
func findTicket(ticketID string) *Ticket {
	for i := range tickets {
		if strings.EqualFold(tickets[i].ID, ticketID) {
			return &tickets[i]
		}
	}
	return nil
}

// findInvoice returns an invoice by ID.
func findInvoice(invoiceID string) *Invoice {
	for i := range invoices {
		if strings.EqualFold(invoices[i].ID, invoiceID) {
			return &invoices[i]
		}
	}
	return nil
}

// searchKB performs a simple keyword search over knowledge base articles.
func searchKB(query string) []KBArticle {
	lower := strings.ToLower(query)
	words := strings.Fields(lower)

	var results []KBArticle
	for _, article := range kbArticles {
		score := 0
		titleLower := strings.ToLower(article.Title)
		contentLower := strings.ToLower(article.Content)

		for _, word := range words {
			if strings.Contains(titleLower, word) {
				score += 3
			}
			if strings.Contains(contentLower, word) {
				score += 1
			}
			for _, tag := range article.Tags {
				if strings.Contains(strings.ToLower(tag), word) {
					score += 2
				}
			}
		}

		if score > 0 {
			results = append(results, article)
		}
	}

	return results
}

// getErrorLogs returns error logs for a customer within the last N hours.
func getErrorLogs(customerID string, hoursBack int) []ErrorLogEntry {
	logs, ok := errorLogsByCustomer[customerID]
	if !ok {
		return nil
	}

	cutoff := time.Now().UTC().Add(-time.Duration(hoursBack) * time.Hour)
	var filtered []ErrorLogEntry
	for _, entry := range logs {
		ts, err := time.Parse(time.RFC3339, entry.Timestamp)
		if err != nil {
			continue
		}
		if ts.After(cutoff) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
