package httpapi

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIncomingTraceID(t *testing.T) {
	traceID := "4bf92f3577b34da6a3ce929d0e0e4736"
	if got := incomingTraceID("00-" + traceID + "-00f067aa0ba902b7-01"); got != traceID {
		t.Fatalf("valid trace ID rejected: %q", got)
	}
	for _, invalid := range []string{"", "00-short-00f067aa0ba902b7-01", "00-00000000000000000000000000000000-00f067aa0ba902b7-01", "00-zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz-00f067aa0ba902b7-01"} {
		if got := incomingTraceID(invalid); got != "" {
			t.Fatalf("invalid trace accepted: %q", invalid)
		}
	}
}

func TestParseNewsFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest("GET", "/api/v1/news?q=cloud&stock=infy&officialOnly=true&page=2&pageSize=25", nil)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req
	filter, err := parseNewsFilter(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if filter.Query != "cloud" || filter.Stock != "infy" || !filter.OfficialOnly || filter.Page != 2 || filter.PageSize != 25 {
		t.Fatalf("unexpected filter: %#v", filter)
	}
}

func TestParseNewsFilterRejectsInvalidPageSize(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/news?pageSize=1000", nil)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req
	if _, err := parseNewsFilter(ctx, ""); err == nil {
		t.Fatal("expected pageSize validation error")
	}
}
