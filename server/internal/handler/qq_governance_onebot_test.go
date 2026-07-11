package handler

import "testing"

func TestParseOneBotGovernanceEvent(t *testing.T) {
	event, ok := parseOneBotGovernanceEvent([]byte(`{"post_type":"request","request_type":"group","sub_type":"add","group_id":100,"user_id":200,"self_id":300,"time":400,"flag":"request-flag"}`))
	if !ok {
		t.Fatal("event was not parsed")
	}
	if event.EventType != "request/group_add" || event.EventKey != "request/group_add:100:200:request-flag" || event.RequestFlag != "request-flag" {
		t.Fatalf("event = %#v", event)
	}
}
