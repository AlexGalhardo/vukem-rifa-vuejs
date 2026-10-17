package jobs

// SendPaymentEmails sends the "payment confirmed" mail to the buyer and the
// "number sold" mail to the owner of a paid reservation (dispatched in the
// webhook transaction).
type SendPaymentEmails struct {
	ReservationID int64 `json:"reservation_id"`
}

// Kind implements queue.Job.
func (SendPaymentEmails) Kind() string { return "payment.emails" }
