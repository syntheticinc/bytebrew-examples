package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Employee represents a company employee.
type Employee struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Department string `json:"department"`
	Title      string `json:"title"`
	Manager    string `json:"manager"`
	StartDate  string `json:"start_date"`
	Location   string `json:"location"`
}

// LeaveBalance represents an employee's leave balance.
type LeaveBalance struct {
	EmployeeID string `json:"employee_id"`
	Vacation   int    `json:"vacation_days"`
	Sick       int    `json:"sick_days"`
	Personal   int    `json:"personal_days"`
	Used       int    `json:"used_days"`
	Pending    int    `json:"pending_days"`
}

// Ticket represents an IT support ticket.
type Ticket struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// KBArticle represents a knowledge base article.
type KBArticle struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

// --- Mock Data ---

var employees = []Employee{
	{ID: "EMP001", Name: "Alice Johnson", Email: "alice@company.com", Department: "Engineering", Title: "Senior Developer", Manager: "Bob Smith", StartDate: "2022-03-15", Location: "New York"},
	{ID: "EMP002", Name: "Bob Smith", Email: "bob@company.com", Department: "Engineering", Title: "Engineering Manager", Manager: "Diana Prince", StartDate: "2020-01-10", Location: "New York"},
	{ID: "EMP003", Name: "Carol Williams", Email: "carol@company.com", Department: "Marketing", Title: "Marketing Lead", Manager: "Diana Prince", StartDate: "2021-06-01", Location: "San Francisco"},
	{ID: "EMP004", Name: "Diana Prince", Email: "diana@company.com", Department: "Executive", Title: "VP of Operations", Manager: "", StartDate: "2019-08-20", Location: "New York"},
	{ID: "EMP005", Name: "Eve Martinez", Email: "eve@company.com", Department: "HR", Title: "HR Specialist", Manager: "Diana Prince", StartDate: "2023-01-15", Location: "Remote"},
	{ID: "EMP006", Name: "Frank Chen", Email: "frank@company.com", Department: "Engineering", Title: "DevOps Engineer", Manager: "Bob Smith", StartDate: "2023-09-01", Location: "San Francisco"},
	{ID: "EMP007", Name: "Grace Kim", Email: "grace@company.com", Department: "Design", Title: "UX Designer", Manager: "Carol Williams", StartDate: "2024-02-01", Location: "Remote"},
}

var leaveBalances = map[string]LeaveBalance{
	"EMP001": {EmployeeID: "EMP001", Vacation: 15, Sick: 10, Personal: 3, Used: 5, Pending: 2},
	"EMP002": {EmployeeID: "EMP002", Vacation: 20, Sick: 10, Personal: 3, Used: 8, Pending: 0},
	"EMP003": {EmployeeID: "EMP003", Vacation: 18, Sick: 10, Personal: 3, Used: 12, Pending: 1},
	"EMP004": {EmployeeID: "EMP004", Vacation: 25, Sick: 10, Personal: 5, Used: 3, Pending: 0},
	"EMP005": {EmployeeID: "EMP005", Vacation: 15, Sick: 10, Personal: 3, Used: 2, Pending: 0},
	"EMP006": {EmployeeID: "EMP006", Vacation: 12, Sick: 10, Personal: 3, Used: 1, Pending: 3},
	"EMP007": {EmployeeID: "EMP007", Vacation: 10, Sick: 10, Personal: 3, Used: 0, Pending: 0},
}

var kbArticles = []KBArticle{
	{
		ID:       "KB001",
		Title:    "How to reset your password",
		Content:  "To reset your password: 1) Go to https://sso.company.com/reset. 2) Enter your email address. 3) Click 'Send Reset Link'. 4) Check your email for the reset link. 5) Create a new password (minimum 12 characters, must include uppercase, lowercase, number, and special character). If you don't receive the email within 5 minutes, contact IT support.",
		Category: "IT",
		Tags:     []string{"password", "reset", "login", "access"},
	},
	{
		ID:       "KB002",
		Title:    "VPN Setup Guide",
		Content:  "To set up the company VPN: 1) Download the VPN client from https://vpn.company.com/download. 2) Install and open the client. 3) Enter server address: vpn.company.com. 4) Use your SSO credentials to log in. 5) Select 'Remember credentials' for convenience. Troubleshooting: If connection fails, check your internet connection first. If the issue persists, try switching to the backup server: vpn2.company.com.",
		Category: "IT",
		Tags:     []string{"vpn", "remote", "access", "network"},
	},
	{
		ID:       "KB003",
		Title:    "Leave Policy",
		Content:  "Annual leave policy: Full-time employees receive 15 vacation days per year (prorated for first year). After 3 years of service, this increases to 20 days. Managers and above receive 25 days. Sick leave: 10 days per year (unused days do not carry over). Personal days: 3 per year. Leave requests must be submitted at least 2 weeks in advance for vacation, except in emergencies. All leave is subject to manager approval.",
		Category: "HR",
		Tags:     []string{"leave", "vacation", "sick", "policy", "time-off"},
	},
	{
		ID:       "KB004",
		Title:    "Remote Work Policy",
		Content:  "Our hybrid work policy: Employees may work remotely up to 3 days per week. Core in-office days are Tuesday and Thursday. Remote-first positions (marked in job description) have no in-office requirement. All employees must be available during core hours (10am-4pm local time) regardless of work location. Equipment: the company provides a laptop, monitor, and $500 home office stipend for new hires.",
		Category: "HR",
		Tags:     []string{"remote", "wfh", "hybrid", "work-from-home", "policy"},
	},
	{
		ID:       "KB005",
		Title:    "Requesting new software or hardware",
		Content:  "To request new software: Submit a ticket via IT support with the software name, business justification, and number of licenses needed. Standard software is typically approved within 2 business days. Enterprise software may require manager and budget approval (5-10 business days). For hardware requests: Submit a ticket with specifications and justification. Standard hardware (keyboard, mouse, headset) is approved automatically. Laptops and monitors require manager approval.",
		Category: "IT",
		Tags:     []string{"software", "hardware", "request", "equipment", "procurement"},
	},
	{
		ID:       "KB006",
		Title:    "Benefits Overview",
		Content:  "Employee benefits include: Health insurance (medical, dental, vision) with 80% company contribution. 401(k) with 4% company match (vested after 1 year). Life insurance: 2x annual salary. Employee Assistance Program (EAP): free counseling and support services. Professional development: $2,000/year budget for courses, conferences, and certifications. Gym membership discount: 50% off at partner gyms.",
		Category: "HR",
		Tags:     []string{"benefits", "insurance", "401k", "health", "professional-development"},
	},
}

// TicketStore is a thread-safe in-memory ticket store.
type TicketStore struct {
	mu      sync.Mutex
	tickets []Ticket
	nextID  int
}

var ticketStore = &TicketStore{nextID: 1000}

func (s *TicketStore) Create(title, description, priority string) Ticket {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	t := Ticket{
		ID:          fmt.Sprintf("TKT-%04d", s.nextID),
		Title:       title,
		Description: description,
		Priority:    priority,
		Status:      "open",
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	s.tickets = append(s.tickets, t)
	return t
}

// searchKB performs a simple keyword search over knowledge base articles.
func searchKB(query string) []KBArticle {
	query = strings.ToLower(query)
	words := strings.Fields(query)

	var results []KBArticle
	for _, article := range kbArticles {
		text := strings.ToLower(article.Title + " " + article.Content + " " + strings.Join(article.Tags, " "))
		matched := false
		for _, w := range words {
			if strings.Contains(text, w) {
				matched = true
				break
			}
		}
		if matched {
			results = append(results, article)
		}
	}
	return results
}

// findEmployee finds an employee by ID.
func findEmployee(id string) *Employee {
	for _, e := range employees {
		if e.ID == id {
			return &e
		}
	}
	return nil
}
