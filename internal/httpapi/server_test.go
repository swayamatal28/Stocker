package httpapi

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

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
