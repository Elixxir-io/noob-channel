package noobChannel

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/catalog"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/e2e"
	"gitlab.com/elixxir/client/v4/e2e/receive"
	"gitlab.com/elixxir/client/v4/single"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/crypto/contact"
	"time"
)

/* CMIX LISTENER IMPLEMENTATION */

// Hear is called to exercise the listener, passing in the
// data as an item
func (m *manager) Hear(item receive.Message) {
	// Confirm the partner has been estabilshed
	if !m.e2eClient.GetE2E().HasAuthenticatedChannel(item.Sender) {
		jww.WARN.Printf("Should have authenticated channel to respond")
		return
	}
	partner, err := m.e2eClient.GetE2E().GetPartner(item.Sender)
	if err != nil {
		jww.WARN.Printf("Could not get partner for sender")
		return
	}

	channelInfo, err := m.respondToHello()
	if err != nil {
		jww.ERROR.Printf("Failed to respond to hello with a noob channel: %+v", err)
		return
	}

	sr, err := m.e2eClient.GetE2E().SendE2E(catalog.XxMessage, partner.PartnerId(), channelInfo, e2e.GetDefaultParams())
	if err != nil {
		jww.ERROR.Printf("Failed to send noob channel to user %s: %+v", partner.PartnerId().String(), err)
		return
	}

	jww.INFO.Printf("Sent hello channel to %s on rounds %+v", partner.PartnerId().String(), sr.RoundList)

}

// Name returns a name, used for debugging
func (m *manager) Name() string {
	return "noob-channel-bot"
}

/* CMIX AUTH IMPLEMENTATION */

func (m *manager) Request(partner contact.Contact,
	rid receptionID.EphemeralIdentity, r rounds.Round, e2e *xxdk.E2e) {
	_, err := e2e.GetAuth().Confirm(partner)
	if err != nil {
		jww.ERROR.Printf("Failed to confirm auth for %s: %+v", partner.ID.String(), err)
	}
}

func (m *manager) Confirm(partner contact.Contact, receptionID receptionID.EphemeralIdentity,
	round rounds.Round, user *xxdk.E2e) {

}

func (m *manager) Reset(partner contact.Contact, receptionID receptionID.EphemeralIdentity,
	round rounds.Round, user *xxdk.E2e) {

}

/* CMIX SINGLE USE IMPLEMENTATION */

func (m *manager) Callback(req *single.Request, reqId receptionID.EphemeralIdentity, _ []rounds.Round) {
	jww.INFO.Printf("Received hello from %d", reqId.EphId.Int64())
	channelInfo, err := m.respondToHello()
	if err != nil {
		jww.ERROR.Printf("Failed to respond to hello with a noob channel: %+v", err)
		return
	}

	rl, err := req.Respond(channelInfo, cmix.GetDefaultCMIXParams(), time.Minute)
	if err != nil {
		jww.ERROR.Printf("Failed to send noob channel to user %d: %+v", reqId.EphId, err)
		return
	}
	jww.INFO.Printf("Sent hello channel to %d on rounds %+v", reqId.EphId, rl)
}
