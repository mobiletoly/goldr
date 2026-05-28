package reports

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

type ReportData struct {
	Audience      string
	Heading       string
	Description   string
	URLs          GoldrMountURLs
	OwnerToolsURL string
	Periods       []PeriodOption
	Rows          []Row
}

type PeriodOption struct {
	Value string
	Label string
}

type Row struct {
	Metric string
	Value  string
	Note   string
}

type Kit struct {
	data ReportData
}

func New(data ReportData) Kit {
	return Kit{data: data}
}

func (kit Kit) Page(_ *http.Request) goldr.PageRouteResponse {
	return goldr.NewPage(
		PageView(kit.data),
		goldr.PageMetadata{
			Title:       kit.data.Heading,
			Description: kit.data.Description,
		},
	)
}

func (kit Kit) Table(_ *http.Request) goldr.FragmentRouteResponse {
	return goldr.NewFragment(TableView(kit.data))
}
