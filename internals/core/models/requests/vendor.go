package requests

type VendorDashboard struct {
	TotalClientsServed int64 `json:"total_clients_served"`
	TotalBookings      int64 `json:"total_bookings"`
	TotalRevenue       int64 `json:"total_revenue"`
}
