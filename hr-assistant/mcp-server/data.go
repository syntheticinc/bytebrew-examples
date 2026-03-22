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
	StartDate  string `json:"start_date"`
	Manager    string `json:"manager"`
}

// LeaveBalance represents an employee's leave balance.
type LeaveBalance struct {
	EmployeeID   string `json:"employee_id"`
	Vacation     int    `json:"vacation_days"`
	Sick         int    `json:"sick_days"`
	Personal     int    `json:"personal_days"`
	UsedVacation int    `json:"used_vacation"`
	UsedSick     int    `json:"used_sick"`
	UsedPersonal int    `json:"used_personal"`
}

// LeaveRequest represents a submitted leave request.
type LeaveRequest struct {
	RequestID  string `json:"request_id"`
	EmployeeID string `json:"employee_id"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	Type       string `json:"type"`
	Reason     string `json:"reason"`
	Status     string `json:"status"`
	DaysCount  int    `json:"days_count"`
	CreatedAt  string `json:"created_at"`
}

// --- Mock Data ---

var employees = []Employee{
	{ID: "EMP001", Name: "Alice Johnson", Email: "alice.johnson@acmecorp.com", Department: "Engineering", Title: "Senior Software Engineer", StartDate: "2021-03-15", Manager: "David Kim"},
	{ID: "EMP002", Name: "Bob Martinez", Email: "bob.martinez@acmecorp.com", Department: "Engineering", Title: "Engineering Manager", StartDate: "2019-06-01", Manager: "Sarah Chen"},
	{ID: "EMP003", Name: "Carol Williams", Email: "carol.williams@acmecorp.com", Department: "Marketing", Title: "Marketing Director", StartDate: "2020-01-20", Manager: "Sarah Chen"},
	{ID: "EMP004", Name: "David Kim", Email: "david.kim@acmecorp.com", Department: "Engineering", Title: "VP of Engineering", StartDate: "2018-09-10", Manager: "Sarah Chen"},
	{ID: "EMP005", Name: "Emily Davis", Email: "emily.davis@acmecorp.com", Department: "HR", Title: "HR Manager", StartDate: "2020-04-15", Manager: "Sarah Chen"},
	{ID: "EMP006", Name: "Frank Thompson", Email: "frank.thompson@acmecorp.com", Department: "Sales", Title: "Account Executive", StartDate: "2023-02-01", Manager: "Lisa Park"},
	{ID: "EMP007", Name: "Grace Lee", Email: "grace.lee@acmecorp.com", Department: "Finance", Title: "Financial Analyst", StartDate: "2022-07-11", Manager: "Michael Brown"},
	{ID: "EMP008", Name: "Henry Wilson", Email: "henry.wilson@acmecorp.com", Department: "Engineering", Title: "Junior Developer", StartDate: "2025-01-06", Manager: "Bob Martinez"},
	{ID: "EMP009", Name: "Lisa Park", Email: "lisa.park@acmecorp.com", Department: "Sales", Title: "Sales Director", StartDate: "2019-11-18", Manager: "Sarah Chen"},
	{ID: "EMP010", Name: "Michael Brown", Email: "michael.brown@acmecorp.com", Department: "Finance", Title: "Finance Director", StartDate: "2020-08-03", Manager: "Sarah Chen"},
}

var leaveBalances = map[string]LeaveBalance{
	"EMP001": {EmployeeID: "EMP001", Vacation: 20, Sick: 10, Personal: 5, UsedVacation: 8, UsedSick: 2, UsedPersonal: 1},
	"EMP002": {EmployeeID: "EMP002", Vacation: 25, Sick: 10, Personal: 5, UsedVacation: 12, UsedSick: 3, UsedPersonal: 2},
	"EMP003": {EmployeeID: "EMP003", Vacation: 25, Sick: 10, Personal: 5, UsedVacation: 10, UsedSick: 1, UsedPersonal: 3},
	"EMP004": {EmployeeID: "EMP004", Vacation: 25, Sick: 10, Personal: 5, UsedVacation: 5, UsedSick: 0, UsedPersonal: 1},
	"EMP005": {EmployeeID: "EMP005", Vacation: 25, Sick: 10, Personal: 5, UsedVacation: 7, UsedSick: 4, UsedPersonal: 2},
	"EMP006": {EmployeeID: "EMP006", Vacation: 15, Sick: 10, Personal: 5, UsedVacation: 3, UsedSick: 1, UsedPersonal: 0},
	"EMP007": {EmployeeID: "EMP007", Vacation: 20, Sick: 10, Personal: 5, UsedVacation: 6, UsedSick: 2, UsedPersonal: 1},
	"EMP008": {EmployeeID: "EMP008", Vacation: 15, Sick: 10, Personal: 5, UsedVacation: 0, UsedSick: 1, UsedPersonal: 0},
	"EMP009": {EmployeeID: "EMP009", Vacation: 25, Sick: 10, Personal: 5, UsedVacation: 15, UsedSick: 3, UsedPersonal: 4},
	"EMP010": {EmployeeID: "EMP010", Vacation: 25, Sick: 10, Personal: 5, UsedVacation: 9, UsedSick: 0, UsedPersonal: 2},
}

// LeaveRequestStore is a thread-safe in-memory leave request store.
type LeaveRequestStore struct {
	mu       sync.Mutex
	requests []LeaveRequest
	nextID   int
}

var leaveRequestStore = &LeaveRequestStore{nextID: 0}

func (s *LeaveRequestStore) Create(employeeID, startDate, endDate, leaveType, reason string, daysCount int) LeaveRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	lr := LeaveRequest{
		RequestID:  fmt.Sprintf("LR-%03d", s.nextID),
		EmployeeID: employeeID,
		StartDate:  startDate,
		EndDate:    endDate,
		Type:       leaveType,
		Reason:     reason,
		Status:     "pending_approval",
		DaysCount:  daysCount,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	s.requests = append(s.requests, lr)
	return lr
}

// findEmployee searches for an employee by ID, email, or name (case-insensitive partial match).
func findEmployee(identifier string) *Employee {
	lower := strings.ToLower(identifier)

	// Exact ID match first.
	for i := range employees {
		if strings.EqualFold(employees[i].ID, identifier) {
			return &employees[i]
		}
	}

	// Exact email match.
	for i := range employees {
		if strings.EqualFold(employees[i].Email, identifier) {
			return &employees[i]
		}
	}

	// Partial name match.
	for i := range employees {
		if strings.Contains(strings.ToLower(employees[i].Name), lower) {
			return &employees[i]
		}
	}

	return nil
}

// countWeekdays counts the number of weekdays between two dates (inclusive).
func countWeekdays(start, end time.Time) int {
	if end.Before(start) {
		return 0
	}

	days := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		wd := d.Weekday()
		if wd != time.Saturday && wd != time.Sunday {
			days++
		}
	}
	return days
}

// validLeaveTypes defines the allowed leave request types.
var validLeaveTypes = map[string]bool{
	"vacation": true,
	"sick":     true,
	"personal": true,
}
