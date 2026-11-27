package qqapi

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestGroupControlEventProjection(t *testing.T) {
	for _, kind := range []string{GroupAddRobot, GroupDelRobot, GroupMsgReceive, GroupMsgReject} {
		for _, ids := range []struct {
			json string
			want string
		}{
			{`"$id":"event-v2","id":"event-v1",`, "event-v2"},
			{`"id":"event-v1",`, "event-v1"},
		} {
			raw := fmt.Sprintf(`{"op":0,"t":%q,%s"d":{"group_openid":"group","timestamp":1784570534,"op_member_openid":"member","apply_at":false}}`, kind, ids.json)
			got, err := parseGatewayEvent(kind, []byte(raw))
			want := Event{ID: ids.want, Type: kind, GroupOpenID: "group", At: time.Unix(1784570534, 0).UTC()}
			if err != nil || got != want {
				t.Errorf("%s projection = %+v, %v; want %+v", kind, got, err, want)
			}
		}
	}
}

func TestGatewayEventFallbackIDPreservesCanonicalIdentity(t *testing.T) {
	parse := func(kind, raw string) string {
		t.Helper()
		event, err := parseGatewayEvent(kind, []byte(raw))
		if err != nil || !strings.HasPrefix(event.ID, "sha256:") {
			t.Fatalf("fallback ID = %q, %v", event.ID, err)
		}
		return event.ID
	}
	first := parse(GroupMsgReceive, `{"op":0,"s":17,"d":{"group_openid":"group","timestamp":1784570534,"op_member_openid":"member"}}`)
	retry := parse(GroupMsgReceive, `{"op":0, "s":19, "d":{"timestamp":1784570534,"group_openid":"group","op_member_openid":"member","ignored":true}}`)
	different := parse(GroupMsgReceive, `{"op":0,"s":20,"d":{"group_openid":"group","timestamp":1784570534,"op_member_openid":"other"}}`)
	if first != retry || first == different {
		t.Fatal("fallback ID did not preserve the canonical event identity")
	}
}
