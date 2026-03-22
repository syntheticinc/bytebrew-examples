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
	// Laptops (5)
	{ID: "PROD-001", Name: "ThinkPad T14", Category: "laptops", Price: 1099.00, Specs: "14\" FHD IPS, AMD Ryzen 7 PRO, 16GB RAM, 512GB SSD, Windows 11 Pro"},
	{ID: "PROD-002", Name: "Dell Latitude 5540", Category: "laptops", Price: 1149.00, Specs: "15.6\" FHD, Intel Core i7-1365U, 16GB RAM, 512GB SSD, Windows 11 Pro"},
	{ID: "PROD-003", Name: "MacBook Air M4", Category: "laptops", Price: 1299.00, Specs: "13.6\" Liquid Retina, Apple M4, 16GB RAM, 512GB SSD, macOS"},
	{ID: "PROD-004", Name: "HP EliteBook 840 G11", Category: "laptops", Price: 1199.00, Specs: "14\" WUXGA IPS, Intel Core Ultra 7, 16GB RAM, 512GB SSD, Windows 11 Pro"},
	{ID: "PROD-005", Name: "ThinkPad X1 Carbon Gen 12", Category: "laptops", Price: 1649.00, Specs: "14\" 2.8K OLED, Intel Core Ultra 7, 32GB RAM, 1TB SSD, Windows 11 Pro, 1.08kg"},
	// Monitors (4)
	{ID: "PROD-010", Name: "Dell U2723QE", Category: "monitors", Price: 619.00, Specs: "27\" 4K UHD IPS, USB-C Hub, 90W PD, VESA, HDR 400"},
	{ID: "PROD-011", Name: "LG 27UK850-W", Category: "monitors", Price: 449.00, Specs: "27\" 4K UHD IPS, USB-C, HDR 10, FreeSync, VESA"},
	{ID: "PROD-012", Name: "Samsung ViewFinity S8", Category: "monitors", Price: 549.00, Specs: "27\" 4K UHD IPS, USB-C 90W, HDR 600, 98% DCI-P3, Matte display"},
	{ID: "PROD-013", Name: "Dell P3223QE", Category: "monitors", Price: 729.00, Specs: "32\" 4K UHD IPS, USB-C Hub, 65W PD, KVM Switch, VESA, Speakers"},
	// Keyboards (3)
	{ID: "PROD-020", Name: "Logitech MX Keys S", Category: "keyboards", Price: 109.00, Specs: "Wireless, Backlit, Bluetooth/USB, Multi-device, Rechargeable, Smart Actions"},
	{ID: "PROD-021", Name: "Keychron K2 V2", Category: "keyboards", Price: 79.00, Specs: "75% Mechanical, Gateron Brown, Bluetooth/USB-C, RGB, Hot-swappable"},
	{ID: "PROD-022", Name: "Apple Magic Keyboard", Category: "keyboards", Price: 129.00, Specs: "Wireless, Touch ID, Numeric keypad, USB-C, macOS/iPadOS"},
	// Mice (3)
	{ID: "PROD-030", Name: "Logitech MX Master 3S", Category: "mice", Price: 99.00, Specs: "Wireless, 8K DPI, Quiet Clicks, USB-C, Multi-device, Ergonomic"},
	{ID: "PROD-031", Name: "Logitech M720 Triathlon", Category: "mice", Price: 49.00, Specs: "Wireless, Multi-device, 24-month battery, Bluetooth/USB"},
	{ID: "PROD-032", Name: "Apple Magic Mouse", Category: "mice", Price: 79.00, Specs: "Wireless, Multi-Touch, USB-C, Rechargeable, macOS"},
	// Headsets (4)
	{ID: "PROD-040", Name: "Sony WH-1000XM5", Category: "headsets", Price: 349.00, Specs: "Wireless ANC, 30h battery, Multipoint, Hi-Res Audio, Speak-to-Chat"},
	{ID: "PROD-041", Name: "Jabra Evolve2 75", Category: "headsets", Price: 299.00, Specs: "Wireless ANC, UC Certified, 36h battery, Busylight, Multipoint"},
	{ID: "PROD-042", Name: "Poly Voyager Focus 2", Category: "headsets", Price: 249.00, Specs: "Wireless ANC, Teams/Zoom Certified, 19h battery, USB-A/C dongle"},
	{ID: "PROD-043", Name: "Jabra Engage 50 II", Category: "headsets", Price: 179.00, Specs: "Wired USB-C, Stereo, ANC, UC Certified, Super-wideband audio"},
	// Webcams (4)
	{ID: "PROD-050", Name: "Logitech C920 HD Pro", Category: "webcams", Price: 69.00, Specs: "1080p 30fps, Stereo Mic, Auto-focus, 78° FOV, USB-A"},
	{ID: "PROD-051", Name: "Logitech Brio 500", Category: "webcams", Price: 129.00, Specs: "1080p 30fps, Auto Light Correction, Show Mode, USB-C, Privacy Shutter"},
	{ID: "PROD-052", Name: "Dell UltraSharp WB7022", Category: "webcams", Price: 199.00, Specs: "4K 30fps, Sony STARVIS, AI Auto-Framing, HDR, USB-C, 90° FOV"},
	{ID: "PROD-053", Name: "Poly Studio P5", Category: "webcams", Price: 149.00, Specs: "1080p 30fps, Auto Light, Privacy Shutter, USB-A/C, Teams Certified"},
	// Docking Stations (3)
	{ID: "PROD-060", Name: "CalDigit TS4", Category: "docks", Price: 399.00, Specs: "Thunderbolt 4, 18 ports, 98W PD, 2.5GbE, SD/microSD, 3x DisplayPort"},
	{ID: "PROD-061", Name: "Dell WD22TB4", Category: "docks", Price: 299.00, Specs: "Thunderbolt 4, 130W PD, 2x HDMI, 1x DP, USB-C, Ethernet, Dell only"},
	{ID: "PROD-062", Name: "Anker 568 USB-C Dock", Category: "docks", Price: 199.00, Specs: "USB-C, 100W PD, 2x HDMI 4K, 1x DP, Ethernet, 4x USB-A, SD slot"},
}

// --- Mock Inventory ---

var inventory = map[string]InventoryItem{
	// Laptops
	"PROD-001": {ProductID: "PROD-001", InStock: 45, Reserved: 3, Available: 42, Status: "in_stock"},
	"PROD-002": {ProductID: "PROD-002", InStock: 30, Reserved: 5, Available: 25, Status: "in_stock"},
	"PROD-003": {ProductID: "PROD-003", InStock: 8, Reserved: 2, Available: 6, Status: "low_stock"},
	"PROD-004": {ProductID: "PROD-004", InStock: 22, Reserved: 1, Available: 21, Status: "in_stock"},
	"PROD-005": {ProductID: "PROD-005", InStock: 5, Reserved: 3, Available: 2, Status: "low_stock"},
	// Monitors
	"PROD-010": {ProductID: "PROD-010", InStock: 20, Reserved: 1, Available: 19, Status: "in_stock"},
	"PROD-011": {ProductID: "PROD-011", InStock: 15, Reserved: 0, Available: 15, Status: "in_stock"},
	"PROD-012": {ProductID: "PROD-012", InStock: 12, Reserved: 2, Available: 10, Status: "in_stock"},
	"PROD-013": {ProductID: "PROD-013", InStock: 7, Reserved: 1, Available: 6, Status: "low_stock"},
	// Keyboards
	"PROD-020": {ProductID: "PROD-020", InStock: 60, Reserved: 4, Available: 56, Status: "in_stock"},
	"PROD-021": {ProductID: "PROD-021", InStock: 3, Reserved: 1, Available: 2, Status: "low_stock"},
	"PROD-022": {ProductID: "PROD-022", InStock: 25, Reserved: 0, Available: 25, Status: "in_stock"},
	// Mice
	"PROD-030": {ProductID: "PROD-030", InStock: 50, Reserved: 2, Available: 48, Status: "in_stock"},
	"PROD-031": {ProductID: "PROD-031", InStock: 35, Reserved: 0, Available: 35, Status: "in_stock"},
	"PROD-032": {ProductID: "PROD-032", InStock: 18, Reserved: 1, Available: 17, Status: "in_stock"},
	// Headsets
	"PROD-040": {ProductID: "PROD-040", InStock: 12, Reserved: 3, Available: 9, Status: "in_stock"},
	"PROD-041": {ProductID: "PROD-041", InStock: 0, Reserved: 0, Available: 0, Status: "out_of_stock"},
	"PROD-042": {ProductID: "PROD-042", InStock: 20, Reserved: 2, Available: 18, Status: "in_stock"},
	"PROD-043": {ProductID: "PROD-043", InStock: 30, Reserved: 0, Available: 30, Status: "in_stock"},
	// Webcams
	"PROD-050": {ProductID: "PROD-050", InStock: 80, Reserved: 5, Available: 75, Status: "in_stock"},
	"PROD-051": {ProductID: "PROD-051", InStock: 25, Reserved: 3, Available: 22, Status: "in_stock"},
	"PROD-052": {ProductID: "PROD-052", InStock: 4, Reserved: 1, Available: 3, Status: "low_stock"},
	"PROD-053": {ProductID: "PROD-053", InStock: 0, Reserved: 0, Available: 0, Status: "out_of_stock"},
	// Docks
	"PROD-060": {ProductID: "PROD-060", InStock: 10, Reserved: 2, Available: 8, Status: "in_stock"},
	"PROD-061": {ProductID: "PROD-061", InStock: 15, Reserved: 0, Available: 15, Status: "in_stock"},
	"PROD-062": {ProductID: "PROD-062", InStock: 40, Reserved: 5, Available: 35, Status: "in_stock"},
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
