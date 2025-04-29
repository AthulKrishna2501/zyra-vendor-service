package requests

type VendorDashboard struct {
	TotalClientsServed    int32 `json:"total_clients_served"`
	TotalBookings         int32 `json:"total_bookings"`
	TotalRevenue          int64 `json:"total_revenue"`
	CurrentMonthBookings  int64 `json:"current_month_booking"`
	PreviousMonthBookings int64 `json:"prev_month_booking"`
	CurrentMonthClients   int64 `json:"current_month_clients"`
	PreviousMonthClients  int64 `json:"prev_month_clients"`
	PendingPayments       int32 `json:"pending_payments"`
	AverageRating         float64 `json:"average_ratings"`
	TotalReviews          int32 `json:"total_reviews"`
}
