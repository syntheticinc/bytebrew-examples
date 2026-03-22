package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Product represents an item in the product catalog.
type Product struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Price    float64 `json:"price"`
	Specs    string  `json:"specs"`
}

// InventoryItem represents stock information for a product.
type InventoryItem struct {
	ProductID string `json:"product_id"`
	InStock   int    `json:"in_stock"`
	Reserved  int    `json:"reserved"`
	Available int    `json:"available"`
	Status    string `json:"status"` // "in_stock", "low_stock", "out_of_stock"
}

// QuoteItem is a line item within a quote.
type QuoteItem struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	LineTotal   float64 `json:"line_total"`
}

// Quote represents a price quote for a customer.
type Quote struct {
	QuoteID      string      `json:"quote_id"`
	CustomerName string      `json:"customer_name"`
	Items        []QuoteItem `json:"items"`
	Subtotal     float64     `json:"subtotal"`
	TaxRate      float64     `json:"tax_rate"`
	TaxAmount    float64     `json:"tax_amount"`
	Total        float64     `json:"total"`
	Discount     float64     `json:"discount_percent"`
	Status       string      `json:"status"`
	CreatedAt    string      `json:"created_at"`
	ValidUntil   string      `json:"valid_until"`
}

// Setting represents a configurable business rule.
type Setting struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

// --- Mock Product Catalog ---

var products = []Product{
	{ID: "PROD-001", Name: "ThinkPad T14", Category: "laptops", Price: 1099.00, Specs: "14\" FHD IPS, AMD Ryzen 7 PRO, 16GB RAM, 512GB SSD, Windows 11 Pro"},
	{ID: "PROD-002", Name: "Dell Latitude 5540", Category: "laptops", Price: 1149.00, Specs: "15.6\" FHD, Intel Core i7-1365U, 16GB RAM, 512GB SSD, Windows 11 Pro"},
	{ID: "PROD-003", Name: "MacBook Air M4", Category: "laptops", Price: 1299.00, Specs: "13.6\" Liquid Retina, Apple M4, 16GB RAM, 512GB SSD, macOS"},
	{ID: "PROD-004", Name: "Dell U2723QE Monitor", Category: "monitors", Price: 619.00, Specs: "27\" 4K UHD IPS, USB-C Hub, 90W PD, VESA, HDR 400"},
	{ID: "PROD-005", Name: "LG 27UK850-W Monitor", Category: "monitors", Price: 449.00, Specs: "27\" 4K UHD IPS, USB-C, HDR 10, FreeSync, VESA"},
	{ID: "PROD-006", Name: "Logitech MX Keys", Category: "keyboards", Price: 99.00, Specs: "Wireless, Backlit, Bluetooth/USB, Multi-device, Rechargeable"},
	{ID: "PROD-007", Name: "Keychron K2 V2", Category: "keyboards", Price: 79.00, Specs: "75% Mechanical, Gateron Brown, Bluetooth/USB-C, RGB, Hot-swappable"},
	{ID: "PROD-008", Name: "Logitech MX Master 3S", Category: "mice", Price: 99.00, Specs: "Wireless, 8K DPI, Quiet Clicks, USB-C, Multi-device, Ergonomic"},
	{ID: "PROD-009", Name: "Logitech M720 Triathlon", Category: "mice", Price: 49.00, Specs: "Wireless, Multi-device, 24-month battery, Bluetooth/USB"},
	{ID: "PROD-010", Name: "Sony WH-1000XM5", Category: "headsets", Price: 349.00, Specs: "Wireless ANC, 30h battery, Multipoint, Hi-Res Audio, Speak-to-Chat"},
	{ID: "PROD-011", Name: "Jabra Evolve2 75", Category: "headsets", Price: 299.00, Specs: "Wireless ANC, UC Certified, 36h battery, Busylight, Multipoint"},
	{ID: "PROD-012", Name: "Logitech C920 HD Pro", Category: "webcams", Price: 69.00, Specs: "1080p 30fps, Stereo Mic, Auto-focus, 78° FOV, USB-A"},
}

// --- Mock Inventory ---

var inventory = map[string]InventoryItem{
	"PROD-001": {ProductID: "PROD-001", InStock: 45, Reserved: 3, Available: 42, Status: "in_stock"},
	"PROD-002": {ProductID: "PROD-002", InStock: 30, Reserved: 5, Available: 25, Status: "in_stock"},
	"PROD-003": {ProductID: "PROD-003", InStock: 8, Reserved: 2, Available: 6, Status: "low_stock"},
	"PROD-004": {ProductID: "PROD-004", InStock: 20, Reserved: 1, Available: 19, Status: "in_stock"},
	"PROD-005": {ProductID: "PROD-005", InStock: 15, Reserved: 0, Available: 15, Status: "in_stock"},
	"PROD-006": {ProductID: "PROD-006", InStock: 60, Reserved: 4, Available: 56, Status: "in_stock"},
	"PROD-007": {ProductID: "PROD-007", InStock: 3, Reserved: 1, Available: 2, Status: "low_stock"},
	"PROD-008": {ProductID: "PROD-008", InStock: 50, Reserved: 2, Available: 48, Status: "in_stock"},
	"PROD-009": {ProductID: "PROD-009", InStock: 35, Reserved: 0, Available: 35, Status: "in_stock"},
	"PROD-010": {ProductID: "PROD-010", InStock: 12, Reserved: 3, Available: 9, Status: "in_stock"},
	"PROD-011": {ProductID: "PROD-011", InStock: 0, Reserved: 0, Available: 0, Status: "out_of_stock"},
	"PROD-012": {ProductID: "PROD-012", InStock: 80, Reserved: 5, Available: 75, Status: "in_stock"},
}

// --- Settings ---

var settings = map[string]Setting{
	"max_discount_percent": {Key: "max_discount_percent", Value: "15", Description: "Maximum discount percentage allowed on quotes"},
	"free_shipping_min":    {Key: "free_shipping_min", Value: "100", Description: "Minimum order amount for free shipping (USD)"},
	"quote_validity_days":  {Key: "quote_validity_days", Value: "7", Description: "Number of days a quote remains valid"},
	"sales_tax_rate":       {Key: "sales_tax_rate", Value: "8.25", Description: "Sales tax rate percentage"},
}

// --- Quote Store ---

// QuoteStore is a thread-safe in-memory quote store.
type QuoteStore struct {
	mu     sync.Mutex
	quotes map[string]*Quote
	nextID int
}

var quoteStore = &QuoteStore{
	quotes: make(map[string]*Quote),
	nextID: 0,
}

func (s *QuoteStore) Create(customerName string, items []QuoteItem, subtotal, taxRate, taxAmount, total float64) *Quote {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	now := time.Now().UTC()
	validityDays := 7

	q := &Quote{
		QuoteID:      fmt.Sprintf("QT-%03d", s.nextID),
		CustomerName: customerName,
		Items:        items,
		Subtotal:     subtotal,
		TaxRate:      taxRate,
		TaxAmount:    taxAmount,
		Total:        total,
		Discount:     0,
		Status:       "draft",
		CreatedAt:    now.Format(time.RFC3339),
		ValidUntil:   now.AddDate(0, 0, validityDays).Format("2006-01-02"),
	}
	s.quotes[q.QuoteID] = q
	return q
}

func (s *QuoteStore) Get(quoteID string) *Quote {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.quotes[quoteID]
}

func (s *QuoteStore) ApplyDiscount(quoteID string, discountPercent float64) *Quote {
	s.mu.Lock()
	defer s.mu.Unlock()

	q, ok := s.quotes[quoteID]
	if !ok {
		return nil
	}

	q.Discount = discountPercent
	discountedSubtotal := q.Subtotal * (1 - discountPercent/100)
	q.TaxAmount = discountedSubtotal * q.TaxRate / 100
	q.Total = discountedSubtotal + q.TaxAmount
	q.Status = "discount_applied"
	return q
}

// --- Search helpers ---

func searchProducts(query, category string, minPrice, maxPrice float64) []Product {
	var results []Product
	lowerQuery := strings.ToLower(query)

	for _, p := range products {
		if category != "" && !strings.EqualFold(p.Category, category) {
			continue
		}
		if minPrice > 0 && p.Price < minPrice {
			continue
		}
		if maxPrice > 0 && p.Price > maxPrice {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(p.Name), lowerQuery) &&
			!strings.Contains(strings.ToLower(p.Specs), lowerQuery) &&
			!strings.Contains(strings.ToLower(p.Category), lowerQuery) {
			continue
		}
		results = append(results, p)
	}

	return results
}

func findProductByID(id string) *Product {
	for i := range products {
		if strings.EqualFold(products[i].ID, id) {
			return &products[i]
		}
	}
	return nil
}
