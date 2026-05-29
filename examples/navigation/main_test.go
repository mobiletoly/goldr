package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/examples/navigation/app/urls"
)

func TestNavigationExampleDestinationsAndTrails(t *testing.T) {
	handler := exampleHandler()

	hqAnalytics := httptest.NewRecorder()
	handler.ServeHTTP(hqAnalytics, httptest.NewRequest(http.MethodGet, urls.Main.Hq.Teams.ByTeamID.Bind("hq-team").Analytics.Path(), nil))
	if hqAnalytics.Code != http.StatusOK {
		t.Fatalf("hq analytics status = %d, want 200", hqAnalytics.Code)
	}
	for _, want := range []string{
		`aria-label="Breadcrumb"`,
		`href="/main/hq/teams/hq-team/analytics/customers/contoso/report?_goldr_trail=hq-analytics"`,
		`class="back"`,
	} {
		if !strings.Contains(hqAnalytics.Body.String(), want) {
			t.Fatalf("hq analytics body missing %q:\n%s", want, hqAnalytics.Body.String())
		}
	}

	hqReportURL := urls.Main.Hq.Teams.ByTeamID.Analytics.Destinations.CustomerReport.Bind("hq-team").Bind("contoso").Href()
	if !strings.Contains(hqReportURL, "_goldr_trail=hq-analytics") {
		t.Fatalf("hq destination href = %q, want selected trail", hqReportURL)
	}

	hqReport := httptest.NewRecorder()
	handler.ServeHTTP(hqReport, httptest.NewRequest(http.MethodGet, hqReportURL, nil))
	if hqReport.Code != http.StatusOK {
		t.Fatalf("hq report status = %d, want 200", hqReport.Code)
	}
	for _, want := range []string{"Home", "HQ", "HQ Team", "Analytics", "Contoso Retail", "Report"} {
		if !strings.Contains(hqReport.Body.String(), want) {
			t.Fatalf("hq report body missing %q:\n%s", want, hqReport.Body.String())
		}
	}

	hqSharedReportURL := urls.Main.Hq.Teams.ByTeamID.Customers.ByCustomerID.Destinations.SharedReport.Bind("contoso").Href()
	if !strings.Contains(hqSharedReportURL, "_goldr_trail=hq-customer") {
		t.Fatalf("hq shared report href = %q, want selected trail", hqSharedReportURL)
	}
	hqSharedReport := httptest.NewRecorder()
	handler.ServeHTTP(hqSharedReport, httptest.NewRequest(http.MethodGet, hqSharedReportURL, nil))
	if hqSharedReport.Code != http.StatusOK {
		t.Fatalf("hq shared report status = %d, want 200", hqSharedReport.Code)
	}
	for _, want := range []string{"Home", "HQ", "HQ Team", "Contoso Retail", "Report"} {
		if !strings.Contains(hqSharedReport.Body.String(), want) {
			t.Fatalf("hq shared report body missing %q:\n%s", want, hqSharedReport.Body.String())
		}
	}

	regionalSharedReportURL := urls.Main.Regional.Offices.ByOfficeID.Teams.ByTeamID.Customers.ByCustomerID.Destinations.SharedReport.Bind("northwind").Href()
	if !strings.Contains(regionalSharedReportURL, "_goldr_trail=regional-customer") {
		t.Fatalf("regional shared report href = %q, want selected trail", regionalSharedReportURL)
	}
	regionalSharedReport := httptest.NewRecorder()
	handler.ServeHTTP(regionalSharedReport, httptest.NewRequest(http.MethodGet, regionalSharedReportURL, nil))
	if regionalSharedReport.Code != http.StatusOK {
		t.Fatalf("regional shared report status = %d, want 200", regionalSharedReport.Code)
	}
	for _, want := range []string{"Home", "Regional", "Seattle", "Regional Team", "Northwind Supply", "Report"} {
		if !strings.Contains(regionalSharedReport.Body.String(), want) {
			t.Fatalf("regional shared report body missing %q:\n%s", want, regionalSharedReport.Body.String())
		}
	}

	cleanReportPath := urls.Main.Hq.Teams.ByTeamID.Bind("hq-team").Customers.ByCustomerID.Bind("contoso").Report.Path()
	if strings.Contains(cleanReportPath, "_goldr_trail") {
		t.Fatalf("clean route path = %q, must not include nav trail", cleanReportPath)
	}
}
