////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package incentives

import (
	"git.xx.network/elixxir/incentives-bot/storage"
	"github.com/golang/protobuf/proto"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"time"
)

type listener struct {
	delay time.Duration
	s     *storage.Storage
	c     *api.Client
}

// Hear messages from users to the incentives bot & respond appropriately
func (l *listener) Hear(item message.Receive) {
	// Confirm that authenticated channels
	if !l.c.HasAuthenticatedChannel(item.Sender) {
		jww.ERROR.Printf("No authenticated channel exists to %+v", item.Sender)
		return
	}

	// Parse the trigger
	in := &CMIXText{}
	var trigger string
	err := proto.Unmarshal(item.Payload, in)
	if err != nil {
		jww.ERROR.Printf("Could not unmarshal message from messenger: %+v", err)
		return
	} else {
		trigger = in.Text
	}

	jww.INFO.Printf("Received trigger %s [%+v]", trigger, in)
	var strResponse string

	// PROCESSING
	uid := item.Sender
	strResponse = l.s.Register(uid, trigger)

	// Respond to message
	payload := &CMIXText{
		Version: 0,
		Text:    strResponse,
		Reply: &TextReply{
			MessageId: item.ID.Marshal(),
			SenderId:  item.Sender.Marshal(),
		},
	}
	marshalled, err := proto.Marshal(payload)
	if err != nil {
		jww.ERROR.Printf("Failed to marshal payload: %+v", err)
		return
	}
	// Create response message
	resp := message.Send{
		Recipient:   item.Sender,
		Payload:     marshalled,
		MessageType: message.XxMessage,
	}

	// Send response message to sender over cmix
	rids, mid, t, err := l.c.SendE2E(resp, params.GetDefaultE2E())
	if err != nil {
		jww.ERROR.Printf("Failed to send message: %+v", err)
	} else {
		jww.INFO.Printf("Sent response %s [%+v] to %+v on rounds %+v [%+v]", strResponse, mid, item.Sender.String(), rids, t)
	}
}

// Name returns a name, used for debugging
func (l *listener) Name() string {
	return "Incentives-bot-listener"
}
