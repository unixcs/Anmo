package transaction

import (
	"context"

	"anmo/server/internal/modules/appointment"
	"anmo/server/internal/shared"
)

// workbench.go — the admin "today" view (plan §73-75): one screen answering
// "who comes today, when, what, settled or not". Lives in the transaction
// module because it enriches appointments with settlement data (one-way
// dependency transaction → appointment, no cycle).

// WorkbenchCard is one appointment card with member and settlement info.
type WorkbenchCard struct {
	Appointment appointment.Appointment         `json:"appointment"`
	Service     *appointment.AppointmentService `json:"service"`
	MemberName  string                          `json:"member_name"`
	Payment     *Payment                        `json:"payment"` // VALID payment, if any
}

// Workbench returns the summary and enriched cards for a business date.
func (p *Provider) Workbench(ctx context.Context, date string) (*appointment.TodaySummary, []WorkbenchCard, error) {
	summary, details, err := p.appointments.Today(ctx, date)
	if err != nil {
		return nil, nil, err
	}
	cards := make([]WorkbenchCard, 0, len(details))
	for i := range details {
		d := details[i]
		card := WorkbenchCard{Appointment: d.Appointment, Service: d.Service}
		if m, err := p.members.Get(ctx, d.MemberID); err == nil {
			card.MemberName = m.Name
		}
		card.Payment = p.validPaymentFor(ctx, d.ID)
		cards = append(cards, card)
	}
	return summary, cards, nil
}

// validPaymentFor returns the appointment's VALID payment, if any.
func (p *Provider) validPaymentFor(ctx context.Context, aptID string) *Payment {
	rows, err := p.db.QueryContext(ctx,
		`SELECT `+paymentColumns+` FROM payment WHERE appointment_id = ? AND status = 'VALID' LIMIT 1`, aptID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	if rows.Next() {
		py, err := scanPayment(rows)
		if err == nil {
			return py
		}
	}
	return nil
}

var _ = shared.OK
