// Package main implements a Model Context Protocol (MCP) stdio server
// that provides sales tools: product search, inventory check, quote creation,
// discount application, and settings retrieval.
//
// Protocol: JSON-RPC 2.0 over stdin/stdout (one JSON object per line).
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
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
		Name:        "search_products",
		Description: "Search the product catalog by keyword, category, and/or price range. Returns matching products with names, prices, specs, and IDs.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search keyword (matches product name, specs, or category)",
				},
				"category": map[string]interface{}{
					"type":        "string",
					"description": "Filter by category: laptops, monitors, keyboards, mice, headsets, webcams",
					"enum":        []string{"laptops", "monitors", "keyboards", "mice", "headsets", "webcams"},
				},
				"min_price": map[string]interface{}{
					"type":        "number",
					"description": "Minimum price filter (USD)",
				},
				"max_price": map[string]interface{}{
					"type":        "number",
					"description": "Maximum price filter (USD)",
				},
			},
		},
	},
	{
		Name:        "check_inventory",
		Description: "Check current stock availability for a specific product by its ID (e.g. PROD-001). Returns in-stock quantity, reserved count, available units, and stock status.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"product_id": map[string]interface{}{
					"type":        "string",
					"description": "Product ID (e.g. PROD-001)",
				},
			},
			"required": []string{"product_id"},
		},
	},
	{
		Name:        "create_quote",
		Description: "Create a price quote for a customer. Calculates subtotal, tax (from sales_tax_rate setting), and total. Returns the quote with a QT-XXX ID. IMPORTANT: This action should be confirmed by the user before execution.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"customer_name": map[string]interface{}{
					"type":        "string",
					"description": "Customer name for the quote",
				},
				"items": map[string]interface{}{
					"type":        "array",
					"description": "List of items to include in the quote",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"product_id": map[string]interface{}{
								"type":        "string",
								"description": "Product ID (e.g. PROD-001)",
							},
							"quantity": map[string]interface{}{
								"type":        "integer",
								"description": "Number of units",
							},
						},
						"required": []string{"product_id", "quantity"},
					},
				},
			},
			"required": []string{"customer_name", "items"},
		},
	},
	{
		Name:        "apply_discount",
		Description: "Apply a discount percentage to an existing quote. Validates against the max_discount_percent setting. Recalculates total with the discount applied. IMPORTANT: This action should be confirmed by the user before execution.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"quote_id": map[string]interface{}{
					"type":        "string",
					"description": "Quote ID (e.g. QT-001)",
				},
				"discount_percent": map[string]interface{}{
					"type":        "number",
					"description": "Discount percentage to apply (e.g. 10 for 10%)",
				},
				"reason": map[string]interface{}{
					"type":        "string",
					"description": "Reason for the discount (e.g. bulk order, returning customer)",
				},
			},
			"required": []string{"quote_id", "discount_percent", "reason"},
		},
	},
	{
		Name:        "get_settings",
		Description: "Read a business rule setting by key. Available keys: max_discount_percent, free_shipping_min, quote_validity_days, sales_tax_rate.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"key": map[string]interface{}{
					"type":        "string",
					"description": "Setting key to look up",
					"enum":        []string{"max_discount_percent", "free_shipping_min", "quote_validity_days", "sales_tax_rate"},
				},
			},
			"required": []string{"key"},
		},
	},
}

// --- Tool execution ----------------------------------------------------------

func executeTool(name string, args json.RawMessage) (string, error) {
	switch name {
	case "search_products":
		return executeSearchProducts(args)
	case "check_inventory":
		return executeCheckInventory(args)
	case "create_quote":
		return executeCreateQuote(args)
	case "apply_discount":
		return executeApplyDiscount(args)
	case "get_settings":
		return executeGetSettings(args)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func executeSearchProducts(args json.RawMessage) (string, error) {
	var p struct {
		Query    string  `json:"query"`
		Category string  `json:"category"`
		MinPrice float64 `json:"min_price"`
		MaxPrice float64 `json:"max_price"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	results := searchProducts(p.Query, p.Category, p.MinPrice, p.MaxPrice)
	if len(results) == 0 {
		return "No products found matching your criteria. Try broadening your search.", nil
	}

	return marshalResult(map[string]interface{}{
		"count":    len(results),
		"products": results,
	})
}

func executeCheckInventory(args json.RawMessage) (string, error) {
	var p struct {
		ProductID string `json:"product_id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	if p.ProductID == "" {
		return "Error: product_id is required.", nil
	}

	inv, ok := inventory[p.ProductID]
	if !ok {
		return fmt.Sprintf("No inventory record found for product %q. Use search_products to find valid product IDs.", p.ProductID), nil
	}

	product := findProductByID(p.ProductID)
	result := map[string]interface{}{
		"inventory": inv,
	}
	if product != nil {
		result["product_name"] = product.Name
	}

	return marshalResult(result)
}

func executeCreateQuote(args json.RawMessage) (string, error) {
	var p struct {
		CustomerName string `json:"customer_name"`
		Items        []struct {
			ProductID string `json:"product_id"`
			Quantity  int    `json:"quantity"`
		} `json:"items"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	if p.CustomerName == "" {
		return "Error: customer_name is required.", nil
	}
	if len(p.Items) == 0 {
		return "Error: at least one item is required.", nil
	}

	// Look up tax rate from settings.
	taxRate := 8.25
	if s, ok := settings["sales_tax_rate"]; ok {
		if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
			taxRate = v
		}
	}

	var quoteItems []QuoteItem
	var subtotal float64

	for _, item := range p.Items {
		product := findProductByID(item.ProductID)
		if product == nil {
			return fmt.Sprintf("Error: product %q not found. Use search_products to find valid product IDs.", item.ProductID), nil
		}
		if item.Quantity <= 0 {
			return fmt.Sprintf("Error: quantity for %s must be greater than 0.", product.Name), nil
		}

		lineTotal := product.Price * float64(item.Quantity)
		quoteItems = append(quoteItems, QuoteItem{
			ProductID:   product.ID,
			ProductName: product.Name,
			Quantity:    item.Quantity,
			UnitPrice:   product.Price,
			LineTotal:   lineTotal,
		})
		subtotal += lineTotal
	}

	taxAmount := roundTo2(subtotal * taxRate / 100)
	total := roundTo2(subtotal + taxAmount)

	quote := quoteStore.Create(p.CustomerName, quoteItems, subtotal, taxRate, taxAmount, total)
	return marshalResult(quote)
}

func executeApplyDiscount(args json.RawMessage) (string, error) {
	var p struct {
		QuoteID         string  `json:"quote_id"`
		DiscountPercent float64 `json:"discount_percent"`
		Reason          string  `json:"reason"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	if p.QuoteID == "" {
		return "Error: quote_id is required.", nil
	}
	if p.DiscountPercent <= 0 {
		return "Error: discount_percent must be greater than 0.", nil
	}
	if p.Reason == "" {
		return "Error: reason is required when applying a discount.", nil
	}

	// Check max discount from settings.
	maxDiscount := 15.0
	if s, ok := settings["max_discount_percent"]; ok {
		if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
			maxDiscount = v
		}
	}

	if p.DiscountPercent > maxDiscount {
		return fmt.Sprintf("Error: discount of %.1f%% exceeds the maximum allowed discount of %.1f%%. Contact a manager for higher discounts.", p.DiscountPercent, maxDiscount), nil
	}

	quote := quoteStore.Get(p.QuoteID)
	if quote == nil {
		return fmt.Sprintf("Error: quote %q not found.", p.QuoteID), nil
	}

	updated := quoteStore.ApplyDiscount(p.QuoteID, p.DiscountPercent)
	result := map[string]interface{}{
		"quote":   updated,
		"reason":  p.Reason,
		"message": fmt.Sprintf("%.1f%% discount applied. New total: $%.2f (was $%.2f)", p.DiscountPercent, updated.Total, quote.Total),
	}

	return marshalResult(result)
}

func executeGetSettings(args json.RawMessage) (string, error) {
	var p struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	if p.Key == "" {
		return "Error: key is required. Available keys: max_discount_percent, free_shipping_min, quote_validity_days, sales_tax_rate.", nil
	}

	setting, ok := settings[p.Key]
	if !ok {
		return fmt.Sprintf("Error: setting %q not found. Available keys: max_discount_percent, free_shipping_min, quote_validity_days, sales_tax_rate.", p.Key), nil
	}

	return marshalResult(setting)
}

// --- Helpers -----------------------------------------------------------------

func marshalResult(v interface{}) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func roundTo2(f float64) float64 {
	return math.Round(f*100) / 100
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
				"name":    "sales-data-mcp",
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
	log.Println("sales-data-mcp server starting (stdio mode)")

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
