package jobs

// ExpireReservation releases a payment link's number when its ten minutes
// are over (dispatched with queue.Delay when the link is created; the
// every-minute sweeper covers lost or early jobs).
type ExpireReservation struct {
	ReservationID int64 `json:"reservation_id"`
}

// Kind implements queue.Job.
func (ExpireReservation) Kind() string { return "reservation.expire" }

// RunDraw draws the winner of a raffle (deadline reached or sold out).
type RunDraw struct {
	RaffleID int64  `json:"raffle_id"`
	Trigger  string `json:"trigger"`
}

// Kind implements queue.Job.
func (RunDraw) Kind() string { return "raffle.draw" }
