package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/examples/navigation/app/urls"
)

func TestNavigationExampleHomeAndMainAreDistinct(t *testing.T) {
	handler := exampleHandler()

	home := httptest.NewRecorder()
	handler.ServeHTTP(home, httptest.NewRequest(http.MethodGet, urls.Root.Path(), nil))
	if home.Code != http.StatusOK {
		t.Fatalf("home status = %d, want 200", home.Code)
	}
	homeBody := home.Body.String()
	for _, want := range []string{
		`<h1>Home</h1>`,
		`href="/main">Main</a>`,
		`href="/about">About</a>`,
		`Start at Main to choose a route owner.`,
	} {
		if !strings.Contains(homeBody, want) {
			t.Fatalf("home body missing %q:\n%s", want, homeBody)
		}
	}
	if regexp.MustCompile(`href="/main/hq">HQ</a>|href="/main/regional">Regional</a>`).MatchString(homeBody) {
		t.Fatalf("home body should not link directly to owner sections, got:\n%s", homeBody)
	}

	about := httptest.NewRecorder()
	handler.ServeHTTP(about, httptest.NewRequest(http.MethodGet, "/about", nil))
	if about.Code != http.StatusOK {
		t.Fatalf("about status = %d, want 200", about.Code)
	}
	aboutBody := about.Body.String()
	for _, want := range []string{
		`<h1>About</h1>`,
		`href="/">Home</a>`,
		`<span aria-current="page">About</span>`,
		`href="/main">Main</a>`,
	} {
		if !strings.Contains(aboutBody, want) {
			t.Fatalf("about body missing %q:\n%s", want, aboutBody)
		}
	}

	main := httptest.NewRecorder()
	handler.ServeHTTP(main, httptest.NewRequest(http.MethodGet, urls.Main.Path(), nil))
	if main.Code != http.StatusOK {
		t.Fatalf("main status = %d, want 200", main.Code)
	}
	mainBody := main.Body.String()
	for _, want := range []string{
		`<h1>Main</h1>`,
		`href="/main/hq">HQ</a>`,
		`href="/main/regional">Regional</a>`,
	} {
		if !strings.Contains(mainBody, want) {
			t.Fatalf("main body missing %q:\n%s", want, mainBody)
		}
	}
}

func TestNavigationExampleTemplateInspection(t *testing.T) {
	t.Setenv("GOLDR_TEMPLATE_INSPECTION", "overlay")
	handler := exampleHandler()

	overlay := httptest.NewRecorder()
	handler.ServeHTTP(overlay, httptest.NewRequest(http.MethodGet, urls.Root.Path(), nil))
	if overlay.Code != http.StatusOK {
		t.Fatalf("overlay page status = %d, want 200", overlay.Code)
	}
	overlayBody := overlay.Body.String()
	for _, want := range []string{
		`<!--goldr:start`,
		`source=app/routes/route.go`,
		`<script src="/goldr/goldr-template-inspector.js" defer></script>`,
	} {
		if !strings.Contains(overlayBody, want) {
			t.Fatalf("overlay body missing %q:\n%s", want, overlayBody)
		}
	}

	helper := httptest.NewRecorder()
	handler.ServeHTTP(helper, httptest.NewRequest(http.MethodGet, "/goldr/goldr-template-inspector.js", nil))
	if helper.Code != http.StatusOK {
		t.Fatalf("inspector helper status = %d, want 200", helper.Code)
	}
	if !strings.Contains(helper.Body.String(), "data-goldr-template-inspector") {
		t.Fatalf("inspector helper missing overlay script marker:\n%s", helper.Body.String())
	}

	t.Setenv("GOLDR_TEMPLATE_INSPECTION", "comments")
	commentsHandler := exampleHandler()
	comments := httptest.NewRecorder()
	commentsHandler.ServeHTTP(comments, httptest.NewRequest(http.MethodGet, urls.Root.Path(), nil))
	if comments.Code != http.StatusOK {
		t.Fatalf("comments page status = %d, want 200", comments.Code)
	}
	commentsBody := comments.Body.String()
	if !strings.Contains(commentsBody, "<!--goldr:start") {
		t.Fatalf("comments body missing inspector comments:\n%s", commentsBody)
	}
	if strings.Contains(commentsBody, "goldr-template-inspector.js") {
		t.Fatalf("comments body must not include overlay helper:\n%s", commentsBody)
	}
}

func TestNavigationExampleDestinationsAndTrails(t *testing.T) {
	handler := exampleHandler()

	hqAnalytics := httptest.NewRecorder()
	handler.ServeHTTP(hqAnalytics, httptest.NewRequest(http.MethodGet, urls.Main.Hq.Teams.ByTeamID.Bind("hq-team").Analytics.Path(), nil))
	if hqAnalytics.Code != http.StatusOK {
		t.Fatalf("hq analytics status = %d, want 200", hqAnalytics.Code)
	}
	for _, want := range []string{
		`aria-label="Breadcrumb"`,
		`href="/main/hq/teams/hq-team/analytics/customers/contoso/report?_goldr_nav_trail_key=hq-analytics&amp;_goldr_return_to=%2Fmain%2Fhq%2Fteams%2Fhq-team%2Fanalytics"`,
		`class="back"`,
	} {
		if !strings.Contains(hqAnalytics.Body.String(), want) {
			t.Fatalf("hq analytics body missing %q:\n%s", want, hqAnalytics.Body.String())
		}
	}

	hqReportURL := urls.Main.Hq.Teams.ByTeamID.Analytics.Destinations.CustomerReport.Bind("hq-team").Bind("contoso").Href()
	if !strings.Contains(hqReportURL, "_goldr_nav_trail_key=hq-analytics") {
		t.Fatalf("hq destination href = %q, want selected trail", hqReportURL)
	}
	if strings.Contains(hqReportURL, "_goldr_return_to") {
		t.Fatalf("hq destination href = %q, want no return-to without navigation value", hqReportURL)
	}

	hqReport := httptest.NewRecorder()
	handler.ServeHTTP(hqReport, httptest.NewRequest(http.MethodGet, hqReportURL, nil))
	if hqReport.Code != http.StatusOK {
		t.Fatalf("hq report status = %d, want 200", hqReport.Code)
	}
	for _, want := range []string{"Home", "Main", "HQ", "HQ Team", "Analytics", "Contoso Retail", "Report"} {
		if !strings.Contains(hqReport.Body.String(), want) {
			t.Fatalf("hq report body missing %q:\n%s", want, hqReport.Body.String())
		}
	}

	hqFilteredAnalytics := httptest.NewRecorder()
	handler.ServeHTTP(hqFilteredAnalytics, httptest.NewRequest(http.MethodGet, urls.Main.Hq.Teams.ByTeamID.Bind("hq-team").Analytics.Path()+"?risk=high&page=2", nil))
	if hqFilteredAnalytics.Code != http.StatusOK {
		t.Fatalf("hq filtered analytics status = %d, want 200", hqFilteredAnalytics.Code)
	}
	for _, want := range []string{
		`<option value="high" selected>High</option>`,
		`_goldr_return_to=%2Fmain%2Fhq%2Fteams%2Fhq-team%2Fanalytics%3Fpage%3D2%26risk%3Dhigh`,
	} {
		if !strings.Contains(hqFilteredAnalytics.Body.String(), want) {
			t.Fatalf("hq filtered analytics body missing %q:\n%s", want, hqFilteredAnalytics.Body.String())
		}
	}

	hqFilteredReportURL := "/main/hq/teams/hq-team/analytics/customers/contoso/report?_goldr_nav_trail_key=hq-analytics&_goldr_return_to=%2Fmain%2Fhq%2Fteams%2Fhq-team%2Fanalytics%3Fpage%3D2%26risk%3Dhigh"
	hqFilteredReport := httptest.NewRecorder()
	handler.ServeHTTP(hqFilteredReport, httptest.NewRequest(http.MethodGet, hqFilteredReportURL, nil))
	if hqFilteredReport.Code != http.StatusOK {
		t.Fatalf("hq filtered report status = %d, want 200", hqFilteredReport.Code)
	}
	if want := `class="back" href="/main/hq/teams/hq-team/analytics?page=2&amp;risk=high"`; !strings.Contains(hqFilteredReport.Body.String(), want) {
		t.Fatalf("hq filtered report body missing %q:\n%s", want, hqFilteredReport.Body.String())
	}

	hqSharedReportURL := urls.Main.Hq.Teams.ByTeamID.Customers.ByCustomerID.Destinations.SharedReport.Bind("contoso").Href()
	if !strings.Contains(hqSharedReportURL, "_goldr_nav_trail_key=hq-customer") {
		t.Fatalf("hq shared report href = %q, want selected trail", hqSharedReportURL)
	}
	hqSharedReport := httptest.NewRecorder()
	handler.ServeHTTP(hqSharedReport, httptest.NewRequest(http.MethodGet, hqSharedReportURL, nil))
	if hqSharedReport.Code != http.StatusOK {
		t.Fatalf("hq shared report status = %d, want 200", hqSharedReport.Code)
	}
	for _, want := range []string{"Home", "Main", "HQ", "HQ Team", "Contoso Retail", "Report"} {
		if !strings.Contains(hqSharedReport.Body.String(), want) {
			t.Fatalf("hq shared report body missing %q:\n%s", want, hqSharedReport.Body.String())
		}
	}

	regionalSharedReportURL := urls.Main.Regional.Offices.ByOfficeID.Teams.ByTeamID.Customers.ByCustomerID.Destinations.SharedReport.Bind("northwind").Href()
	if !strings.Contains(regionalSharedReportURL, "_goldr_nav_trail_key=regional-customer") {
		t.Fatalf("regional shared report href = %q, want selected trail", regionalSharedReportURL)
	}
	regionalSharedReport := httptest.NewRecorder()
	handler.ServeHTTP(regionalSharedReport, httptest.NewRequest(http.MethodGet, regionalSharedReportURL, nil))
	if regionalSharedReport.Code != http.StatusOK {
		t.Fatalf("regional shared report status = %d, want 200", regionalSharedReport.Code)
	}
	for _, want := range []string{"Home", "Main", "Regional", "Seattle", "Regional Team", "Northwind Supply", "Report"} {
		if !strings.Contains(regionalSharedReport.Body.String(), want) {
			t.Fatalf("regional shared report body missing %q:\n%s", want, regionalSharedReport.Body.String())
		}
	}

	cleanReportPath := urls.Main.Hq.Teams.ByTeamID.Bind("hq-team").Customers.ByCustomerID.Bind("contoso").Report.Path()
	if strings.Contains(cleanReportPath, "_goldr_nav_trail_key") {
		t.Fatalf("clean route path = %q, must not include nav trail", cleanReportPath)
	}

	briefReportPath := urls.Main.Hq.Teams.ByTeamID.
		Bind("hq-team").
		Customers.ByCustomerID.
		Bind("contoso").
		Report.
		Brief.
		Path()
	briefReport := httptest.NewRecorder()
	handler.ServeHTTP(briefReport, httptest.NewRequest(http.MethodGet, briefReportPath, nil))
	if briefReport.Code != http.StatusOK {
		t.Fatalf("brief report status = %d, want 200", briefReport.Code)
	}
	for _, want := range []string{
		`aria-label="Report views"`,
		`href="/main/hq/teams/hq-team/customers/contoso/report">Customer report</a>`,
		`href="/main/hq/teams/hq-team/customers/contoso/report/brief">Brief customer report</a>`,
		`href="/main/hq/teams/hq-team/customers/contoso/report/detailed">Detailed customer report</a>`,
		`<h1>Brief Customer Report</h1>`,
	} {
		if !strings.Contains(briefReport.Body.String(), want) {
			t.Fatalf("brief report body missing %q:\n%s", want, briefReport.Body.String())
		}
	}
}
