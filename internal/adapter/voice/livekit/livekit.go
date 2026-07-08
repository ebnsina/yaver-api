// Package livekit implements domain.VoiceProvider over LiveKit SIP: it dials the
// customer out through a configured outbound trunk (the BD carrier) into a
// per-call LiveKit room. This is the "place the call" half of the port; the
// media pipeline that plays IVR prompts and gathers DTMF runs as a LiveKit Agent
// in that room — the next phase behind the same outcome contract.
package livekit

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/pkg/id"
)

// Provider places outbound calls via a LiveKit SIP outbound trunk.
type Provider struct {
	sip     *lksdk.SIPClient
	trunkID string
	log     *slog.Logger
}

// New builds the provider. url/apiKey/apiSecret authenticate to the LiveKit
// server; trunkID is the outbound SIP trunk that reaches the PSTN via the carrier.
func New(url, apiKey, apiSecret, trunkID string, log *slog.Logger) *Provider {
	return &Provider{
		sip:     lksdk.NewSIPClient(url, apiKey, apiSecret),
		trunkID: trunkID,
		log:     log,
	}
}

// PlaceCall dials req.ToPhone through the outbound trunk into a fresh room named
// for the call, and returns LiveKit's SIP call id as the provider call id.
func (p *Provider) PlaceCall(ctx context.Context, req domain.CallRequest) (domain.ProviderCallID, error) {
	room := "call-" + id.New("rm")
	info, err := p.sip.CreateSIPParticipant(ctx, &livekit.CreateSIPParticipantRequest{
		SipTrunkId:          p.trunkID,
		SipCallTo:           req.ToPhone,
		RoomName:            room,
		ParticipantIdentity: "customer",
		ParticipantName:     req.ToPhone,
	})
	if err != nil {
		return "", fmt.Errorf("livekit create sip participant: %w", err)
	}
	p.log.Info("livekit place_call",
		"sip_call_id", info.SipCallId,
		"room", room,
		"to", req.ToPhone,
		"flow", req.Flow.Name,
	)
	callID := info.SipCallId
	if callID == "" {
		callID = info.ParticipantId
	}
	return domain.ProviderCallID(callID), nil
}
