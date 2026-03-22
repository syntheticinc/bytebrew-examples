// Package main implements a Model Context Protocol (MCP) stdio server
// that provides customer support data tools: customer lookup, ticket management,
// knowledge base search, service health checks, error logs, billing operations.
//
// Protocol: JSON-RPC 2.0 over stdin/stdout (one JSON object per line).
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
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
		Name:        "get_customer",
		Description: "Look up a CloudSync customer by ID (e.g. CUST-001), email address, or name (case-insensitive partial match). Returns customer details including plan, MRR, signup date, and status.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"identifier": map[string]interface{}{
					"type":        "string",
					"description": "Customer ID (e.g. CUST-001), email address, or name (partial match supported)",
				},
			},
			"required": []string{"identifier"},
		},
	},
	{
		Name:        "get_ticket",
		Description: "Get details of a specific support ticket by ticket ID (e.g. TKT-001). Returns title, description, priority, category, status, and assignment info.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"ticket_id": map[string]interface{}{
					"type":        "string",
					"description": "Ticket ID (e.g. TKT-001)",
				},
			},
			"required": []string{"ticket_id"},
		},
	},
	{
		Name:        "create_ticket",
		Description: "Create a new support ticket for a customer. Returns the created ticket with an assigned ticket ID and 'open' status.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"customer_id": map[string]interface{}{
					"type":        "string",
					"description": "Customer ID (e.g. CUST-001)",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Short summary of the issue",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Detailed description of the issue",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"description": "Ticket priority",
					"enum":        []string{"low", "medium", "high", "critical"},
				},
				"category": map[string]interface{}{
					"type":        "string",
					"description": "Ticket category",
					"enum":        []string{"technical", "billing"},
				},
			},
			"required": []string{"customer_id", "title", "description", "priority", "category"},
		},
	},
	{
		Name:        "search_kb",
		Description: "Search the CloudSync knowledge base for articles matching a query. Returns matching articles with title, category, content, and tags. Use for troubleshooting common issues before escalating.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query (keywords related to the issue)",
				},
			},
			"required": []string{"query"},
		},
	},
	{
		Name:        "check_service_status",
		Description: "Check the health status of a CloudSync microservice. Returns uptime, latency, error rate, and any active incidents. Available services: api-gateway, auth-service, storage, billing, notifications.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"service_name": map[string]interface{}{
					"type":        "string",
					"description": "Service name: api-gateway, auth-service, storage, billing, or notifications",
				},
			},
			"required": []string{"service_name"},
		},
	},
	{
		Name:        "get_error_logs",
		Description: "Get recent error log entries for a specific customer. Returns timestamped log entries with service, level, message, and trace ID for debugging.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"customer_id": map[string]interface{}{
					"type":        "string",
					"description": "Customer ID (e.g. CUST-001)",
				},
				"hours_back": map[string]interface{}{
					"type":        "number",
					"description": "How many hours back to search (default: 24)",
				},
			},
			"required": []string{"customer_id"},
		},
	},
	{
		Name:        "update_subscription",
		Description: "Change a customer's subscription plan. Validates that the target plan exists and the customer is active. Upgrades are prorated; downgrades take effect at end of billing cycle.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"customer_id": map[string]interface{}{
					"type":        "string",
					"description": "Customer ID (e.g. CUST-001)",
				},
				"new_plan": map[string]interface{}{
					"type":        "string",
					"description": "Target plan name",
					"enum":        []string{"Starter", "Pro", "Enterprise"},
				},
			},
			"required": []string{"customer_id", "new_plan"},
		},
	},
	{
		Name:        "process_refund",
		Description: "Issue a refund for a specific invoice. Validates the invoice exists and has been paid. Refunds over $100 require manager approval and will be set to pending_manager_approval status.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"invoice_id": map[string]interface{}{
					"type":        "string",
					"description": "Invoice ID (e.g. INV-2026-001)",
				},
				"amount": map[string]interface{}{
					"type":        "number",
					"description": "Refund amount in cents (e.g. 1900 for $19.00)",
				},
				"reason": map[string]interface{}{
					"type":        "string",
					"description": "Reason for the refund",
				},
			},
			"required": []string{"invoice_id", "amount", "reason"},
		},
	},
}

// --- Tool execution ----------------------------------------------------------

func executeTool(name string, args json.RawMessage) (string, error) {
	switch name {
	case "get_customer":
		return executeGetCustomer(args)
	case "get_ticket":
		return executeGetTicket(args)
	case "create_ticket":
		return executeCreateTicket(args)
	case "search_kb":
		return executeSearchKB(args)
	case "check_service_status":
		return executeCheckServiceStatus(args)
	case "get_error_logs":
		return executeGetErrorLogs(args)
	case "update_subscription":
		return executeUpdateSubscription(args)
	case "process_refund":
		return executeProcessRefund(args)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func executeGetCustomer(args json.RawMessage) (string, error) {
	var p struct {
		Identifier string `json:"identifier"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	if p.Identifier == "" {
		return "Error: identifier is required.", nil
	}

	cust := findCustomer(p.Identifier)
	if cust == nil {
		return fmt.Sprintf("No customer found matching %q. Try searching by name, email, or customer ID (e.g. CUST-001).", p.Identifier), nil
	}

	return marshalResult(cust)
}

func executeGetTicket(args json.RawMessage) (string, error) {
	var p struct {
		TicketID string `json:"ticket_id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	if p.TicketID == "" {
		return "Error: ticket_id is required.", nil
	}

	ticket := findTicket(p.TicketID)
	if ticket == nil {
		return fmt.Sprintf("No ticket found with ID %q.", p.TicketID), nil
	}

	return marshalResult(ticket)
}

func executeCreateTicket(args json.RawMessage) (string, error) {
	var p struct {
		CustomerID  string `json:"customer_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
		Category    string `json:"category"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	if p.CustomerID == "" || p.Title == "" || p.Description == "" {
		return "Error: customer_id, title, and description are required.", nil
	}

	cust := findCustomer(p.CustomerID)
	if cust == nil {
		return fmt.Sprintf("Error: customer %q not found.", p.CustomerID), nil
	}

	validPriorities := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
	if !validPriorities[p.Priority] {
		return fmt.Sprintf("Error: invalid priority %q. Must be one of: low, medium, high, critical.", p.Priority), nil
	}

	validCategories := map[string]bool{"technical": true, "billing": true}
	if !validCategories[p.Category] {
		return fmt.Sprintf("Error: invalid category %q. Must be one of: technical, billing.", p.Category), nil
	}

	ticket := ticketStore.Create(p.CustomerID, p.Title, p.Description, p.Priority, p.Category)
	return marshalResult(ticket)
}

func executeSearchKB(args json.RawMessage) (string, error) {
	var p struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	if p.Query == "" {
		return "Error: query is required.", nil
	}

	results := searchKB(p.Query)
	if len(results) == 0 {
		return fmt.Sprintf("No knowledge base articles found matching %q.", p.Query), nil
	}

	return marshalResult(results)
}

func executeCheckServiceStatus(args json.RawMessage) (string, error) {
	var p struct {
		ServiceName string `json:"service_name"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	if p.ServiceName == "" {
		return "Error: service_name is required.", nil
	}

	status, ok := serviceStatuses[p.ServiceName]
	if !ok {
		available := make([]string, 0, len(serviceStatuses))
		for name := range serviceStatuses {
			available = append(available, name)
		}
		return fmt.Sprintf("Error: unknown service %q. Available services: %s.", p.ServiceName, strings.Join(available, ", ")), nil
	}

	return marshalResult(status)
}

func executeGetErrorLogs(args json.RawMessage) (string, error) {
	var p struct {
		CustomerID string `json:"customer_id"`
		HoursBack  int    `json:"hours_back"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	if p.CustomerID == "" {
		return "Error: customer_id is required.", nil
	}

	if p.HoursBack <= 0 {
		p.HoursBack = 24
	}

	cust := findCustomer(p.CustomerID)
	if cust == nil {
		return fmt.Sprintf("Error: customer %q not found.", p.CustomerID), nil
	}

	logs := getErrorLogs(p.CustomerID, p.HoursBack)
	if len(logs) == 0 {
		return fmt.Sprintf("No error logs found for customer %s in the last %d hours.", p.CustomerID, p.HoursBack), nil
	}

	result := map[string]interface{}{
		"customer_id": p.CustomerID,
		"hours_back":  p.HoursBack,
		"count":       len(logs),
		"entries":     logs,
	}
	return marshalResult(result)
}

func executeUpdateSubscription(args json.RawMessage) (string, error) {
	var p struct {
		CustomerID string `json:"customer_id"`
		NewPlan    string `json:"new_plan"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	if p.CustomerID == "" || p.NewPlan == "" {
		return "Error: customer_id and new_plan are required.", nil
	}

	cust := findCustomer(p.CustomerID)
	if cust == nil {
		return fmt.Sprintf("Error: customer %q not found.", p.CustomerID), nil
	}

	if cust.Status != "active" {
		return fmt.Sprintf("Error: customer %s has status %q. Only active customers can change plans.", p.CustomerID, cust.Status), nil
	}

	if !validPlans[p.NewPlan] {
		return fmt.Sprintf("Error: invalid plan %q. Available plans: Starter, Pro, Enterprise.", p.NewPlan), nil
	}

	if strings.EqualFold(cust.Plan, p.NewPlan) {
		return fmt.Sprintf("Customer %s is already on the %s plan.", p.CustomerID, p.NewPlan), nil
	}

	oldPlan := cust.Plan
	cust.Plan = p.NewPlan

	// Update MRR based on plan.
	planPrices := map[string]int{"Starter": 1900, "Pro": 7900, "Enterprise": 49900}
	cust.MRR = planPrices[p.NewPlan]

	changeType := "upgrade"
	if planPrices[p.NewPlan] < planPrices[oldPlan] {
		changeType = "downgrade"
	}

	result := map[string]interface{}{
		"customer_id": cust.ID,
		"old_plan":    oldPlan,
		"new_plan":    p.NewPlan,
		"change_type": changeType,
		"message":     fmt.Sprintf("Subscription changed from %s to %s. %s", oldPlan, p.NewPlan, subscriptionChangeMessage(changeType)),
	}
	return marshalResult(result)
}

func subscriptionChangeMessage(changeType string) string {
	if changeType == "upgrade" {
		return "Upgrade is effective immediately. Prorated charges will appear on the next invoice."
	}
	return "Downgrade takes effect at the end of the current billing cycle."
}

func executeProcessRefund(args json.RawMessage) (string, error) {
	var p struct {
		InvoiceID string `json:"invoice_id"`
		Amount    int    `json:"amount"`
		Reason    string `json:"reason"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	if p.InvoiceID == "" || p.Amount <= 0 || p.Reason == "" {
		return "Error: invoice_id, amount (positive integer in cents), and reason are required.", nil
	}

	inv := findInvoice(p.InvoiceID)
	if inv == nil {
		return fmt.Sprintf("Error: invoice %q not found.", p.InvoiceID), nil
	}

	if inv.Status != "paid" {
		return fmt.Sprintf("Error: invoice %s has status %q. Only paid invoices can be refunded.", p.InvoiceID, inv.Status), nil
	}

	if p.Amount > inv.Amount {
		return fmt.Sprintf("Error: refund amount (%d cents) exceeds invoice amount (%d cents).", p.Amount, inv.Amount), nil
	}

	refund := refundStore.Create(p.InvoiceID, p.Amount, p.Reason)

	result := map[string]interface{}{
		"refund":  refund,
		"invoice": inv,
	}
	if refund.Status == "pending_manager_approval" {
		result["note"] = "Refunds over $100.00 require manager approval. A notification has been sent to the billing manager."
	}

	return marshalResult(result)
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
				"name":    "support-data-mcp",
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
	log.Println("support-data-mcp server starting (stdio mode)")

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
