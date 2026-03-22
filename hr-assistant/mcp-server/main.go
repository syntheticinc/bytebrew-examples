// Package main implements a Model Context Protocol (MCP) stdio server
// that provides HR data tools: employee lookup, leave balance check,
// and leave request submission.
//
// Protocol: JSON-RPC 2.0 over stdin/stdout (one JSON object per line).
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// --- JSON-RPC 2.0 types -----------------------------------------------------

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// --- MCP types ---------------------------------------------------------------

type toolInfo struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolCallResult struct {
	Content []toolContent `json:"content"`
}

// --- Tool definitions --------------------------------------------------------

var tools = []toolInfo{
	{
		Name:        "get_employee",
		Description: "Look up an employee by ID (e.g. EMP001), email address, or name (case-insensitive partial match). Returns employee details including department, title, start date, and manager.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"identifier": map[string]interface{}{
					"type":        "string",
					"description": "Employee ID (e.g. EMP001), email address, or name (partial match supported)",
				},
			},
			"required": []string{"identifier"},
		},
	},
	{
		Name:        "get_leave_balance",
		Description: "Get the leave balance for a specific employee, including vacation days, sick days, personal days, and how many of each have been used this year.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"employee_id": map[string]interface{}{
					"type":        "string",
					"description": "Employee ID (e.g. EMP001)",
				},
			},
			"required": []string{"employee_id"},
		},
	},
	{
		Name:        "submit_leave_request",
		Description: "Submit a new leave request for an employee. Validates that the employee exists, has enough balance for the requested type, and dates are not in the past. Returns the created leave request with a request ID and pending_approval status.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"employee_id": map[string]interface{}{
					"type":        "string",
					"description": "Employee ID (e.g. EMP001)",
				},
				"start_date": map[string]interface{}{
					"type":        "string",
					"description": "Start date in YYYY-MM-DD format",
				},
				"end_date": map[string]interface{}{
					"type":        "string",
					"description": "End date in YYYY-MM-DD format",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "Leave type: vacation, sick, or personal",
					"enum":        []string{"vacation", "sick", "personal"},
				},
				"reason": map[string]interface{}{
					"type":        "string",
					"description": "Reason for the leave request",
				},
			},
			"required": []string{"employee_id", "start_date", "end_date", "type", "reason"},
		},
	},
}

// --- Tool execution ----------------------------------------------------------

func executeTool(name string, args json.RawMessage) (string, error) {
	switch name {
	case "get_employee":
		return executeGetEmployee(args)
	case "get_leave_balance":
		return executeGetLeaveBalance(args)
	case "submit_leave_request":
		return executeSubmitLeaveRequest(args)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func executeGetEmployee(args json.RawMessage) (string, error) {
	var p struct {
		Identifier string `json:"identifier"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	if p.Identifier == "" {
		return "Error: identifier is required.", nil
	}

	emp := findEmployee(p.Identifier)
	if emp == nil {
		return fmt.Sprintf("No employee found matching %q. Try searching by name, email, or employee ID (e.g. EMP001).", p.Identifier), nil
	}

	return marshalResult(emp)
}

func executeGetLeaveBalance(args json.RawMessage) (string, error) {
	var p struct {
		EmployeeID string `json:"employee_id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	bal, ok := leaveBalances[p.EmployeeID]
	if !ok {
		return fmt.Sprintf("No leave balance found for employee %q. Make sure to use the employee ID (e.g. EMP001).", p.EmployeeID), nil
	}

	// Add computed remaining balances to the output.
	type balanceWithRemaining struct {
		LeaveBalance
		RemainingVacation int `json:"remaining_vacation"`
		RemainingSick     int `json:"remaining_sick"`
		RemainingPersonal int `json:"remaining_personal"`
	}

	result := balanceWithRemaining{
		LeaveBalance:      bal,
		RemainingVacation: bal.Vacation - bal.UsedVacation,
		RemainingSick:     bal.Sick - bal.UsedSick,
		RemainingPersonal: bal.Personal - bal.UsedPersonal,
	}

	return marshalResult(result)
}

func executeSubmitLeaveRequest(args json.RawMessage) (string, error) {
	var p struct {
		EmployeeID string `json:"employee_id"`
		StartDate  string `json:"start_date"`
		EndDate    string `json:"end_date"`
		Type       string `json:"type"`
		Reason     string `json:"reason"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	// Validate employee exists.
	emp := findEmployee(p.EmployeeID)
	if emp == nil {
		return fmt.Sprintf("Error: employee %q not found.", p.EmployeeID), nil
	}

	// Validate leave type.
	if !validLeaveTypes[p.Type] {
		return fmt.Sprintf("Error: invalid leave type %q. Must be one of: vacation, sick, personal.", p.Type), nil
	}

	// Parse and validate dates.
	startDate, err := time.Parse("2006-01-02", p.StartDate)
	if err != nil {
		return fmt.Sprintf("Error: invalid start_date format %q. Use YYYY-MM-DD.", p.StartDate), nil
	}

	endDate, err := time.Parse("2006-01-02", p.EndDate)
	if err != nil {
		return fmt.Sprintf("Error: invalid end_date format %q. Use YYYY-MM-DD.", p.EndDate), nil
	}

	if endDate.Before(startDate) {
		return "Error: end_date must be on or after start_date.", nil
	}

	// Check dates are not in the past (allow today).
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if startDate.Before(today) {
		return "Error: start_date cannot be in the past.", nil
	}

	// Count weekdays.
	daysCount := countWeekdays(startDate, endDate)
	if daysCount == 0 {
		return "Error: the selected date range contains no weekdays.", nil
	}

	// Check leave balance.
	bal, ok := leaveBalances[p.EmployeeID]
	if !ok {
		return fmt.Sprintf("Error: no leave balance record for employee %q.", p.EmployeeID), nil
	}

	var remaining int
	switch p.Type {
	case "vacation":
		remaining = bal.Vacation - bal.UsedVacation
	case "sick":
		remaining = bal.Sick - bal.UsedSick
	case "personal":
		remaining = bal.Personal - bal.UsedPersonal
	}

	if daysCount > remaining {
		return fmt.Sprintf("Error: insufficient %s balance. Requested %d days but only %d remaining.", p.Type, daysCount, remaining), nil
	}

	// Create the leave request.
	lr := leaveRequestStore.Create(p.EmployeeID, p.StartDate, p.EndDate, p.Type, p.Reason, daysCount)
	return marshalResult(lr)
}

func marshalResult(v interface{}) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// --- MCP request handlers ----------------------------------------------------

func handleInitialize(id interface{}) response {
	return response{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "hr-data-mcp",
				"version": "1.0.0",
			},
		},
	}
}

func handleToolsList(id interface{}) response {
	return response{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

func handleToolsCall(id interface{}, params json.RawMessage) response {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return response{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &rpcError{Code: -32602, Message: fmt.Sprintf("invalid params: %v", err)},
		}
	}

	result, err := executeTool(p.Name, p.Arguments)
	if err != nil {
		return response{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &rpcError{Code: -32000, Message: err.Error()},
		}
	}

	return response{
		JSONRPC: "2.0",
		ID:      id,
		Result: toolCallResult{
			Content: []toolContent{
				{Type: "text", Text: result},
			},
		},
	}
}

// --- Main loop ---------------------------------------------------------------

func main() {
	log.SetOutput(os.Stderr) // Keep stdout clean for JSON-RPC
	log.Println("hr-data-mcp server starting (stdio mode)")

	scanner := bufio.NewScanner(os.Stdin)
	// Allow up to 1 MB per line (large tool results).
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			log.Printf("failed to parse request: %v", err)
			continue
		}

		// Notifications (no ID) -- just acknowledge silently.
		if req.ID == nil {
			log.Printf("notification: %s", req.Method)
			continue
		}

		var resp response
		switch req.Method {
		case "initialize":
			resp = handleInitialize(req.ID)
		case "tools/list":
			resp = handleToolsList(req.ID)
		case "tools/call":
			resp = handleToolsCall(req.ID, req.Params)
		default:
			resp = response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rpcError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)},
			}
		}

		if err := encoder.Encode(resp); err != nil {
			log.Printf("failed to write response: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("stdin read error: %v", err)
	}
}
