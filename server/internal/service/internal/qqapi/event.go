package qqapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

const (
	GroupAddRobot    = "GROUP_ADD_ROBOT"
	GroupDelRobot    = "GROUP_DEL_ROBOT"
	GroupMsgReceive  = "GROUP_MSG_RECEIVE"
	GroupMsgReject   = "GROUP_MSG_REJECT"
	GroupJoinRequest = "GROUP_JOIN_REQUEST"

	maxEventIDBytes = 255
)

// Event is the allowlisted projection of a QQ group control or admission event.
// ID is stable across retries and can be used for deduplication by the caller.
type Event struct {
	ID          string
	Type        string
	GroupOpenID string
	At          time.Time
	JoinRequest *JoinRequest
}

type gatewayEventIDs struct {
	ID   string `json:"id"`
	IDV2 string `json:"$id"`
}

func (ids gatewayEventIDs) eventID() string {
	if ids.IDV2 != "" {
		return ids.IDV2
	}
	return ids.ID
}

func parseGatewayEvent(kind string, raw []byte) (Event, error) {
	event := Event{Type: kind}
	var operator string
	var id string
	switch kind {
	case GroupAddRobot, GroupDelRobot, GroupMsgReceive, GroupMsgReject:
		var envelope struct {
			gatewayEventIDs
			Data struct {
				GroupOpenID string `json:"group_openid"`
				Operator    string `json:"op_member_openid"`
				Timestamp   int64  `json:"timestamp"`
			} `json:"d"`
		}
		if json.Unmarshal(raw, &envelope) != nil || envelope.Data.Timestamp <= 0 || envelope.Data.Timestamp > 253402300799 {
			return Event{}, errGatewayMalformedEvent
		}
		body := envelope.Data
		id = envelope.eventID()
		event.GroupOpenID, event.At = body.GroupOpenID, time.Unix(body.Timestamp, 0).UTC()
		if urlSafeID(body.Operator, maxOpenIDBytes) {
			operator = body.Operator
		}
	case GroupJoinRequest:
		// QQ documents this under GROUP_MEMBER_EVENT, not the selected intent.
		var envelope struct {
			gatewayEventIDs
			Data struct {
				GroupOpenID string `json:"group_openid"`
				wireJoinRequest
			} `json:"d"`
		}
		if json.Unmarshal(raw, &envelope) != nil {
			return Event{}, errGatewayMalformedEvent
		}
		request, ok := envelope.Data.joinRequest()
		if !ok {
			return Event{}, errGatewayMalformedEvent
		}
		id = envelope.eventID()
		event.GroupOpenID, event.At, event.JoinRequest = envelope.Data.GroupOpenID, request.AppliedAt, &request
	default:
		// Don't decode message content or other non-allowlisted payloads.
		return Event{}, nil
	}
	if !urlSafeID(event.GroupOpenID, maxOpenIDBytes) || (id != "" && !visibleASCII(id, maxEventIDBytes)) {
		return Event{}, errGatewayMalformedEvent
	}
	if id == "" {
		// Hash only canonical, allowlisted identity fields, not provider JSON
		// or the connection-specific sequence number.
		identity := []string{event.Type, event.GroupOpenID, strconv.FormatInt(event.At.Unix(), 10), operator}
		if event.JoinRequest != nil {
			identity = append(identity, event.JoinRequest.ID, event.JoinRequest.MemberOpenID, strconv.FormatBool(event.JoinRequest.AutoApproved))
		}
		hash := sha256.Sum256([]byte(strings.Join(identity, "\n")))
		id = "sha256:" + hex.EncodeToString(hash[:])
	}
	event.ID = id
	return event, nil
}
