// Package marketdata fetches and parses Treasury.gov par yield curve data
package marketdata

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/christophernnh/curve-engine/internal/curve"
)

// treasuryFeed mirrors the Atom <feed> wrapper Treasury.gov returns.
type treasuryFeed struct {
	XMLName xml.Name        `xml:"feed"`
	Entries []treasuryEntry `xml:"entry"`
}

type treasuryEntry struct {
	Content treasuryContent `xml:"content"`
}

type treasuryContent struct {
	Properties treasuryProperties `xml:"properties"`
}

type treasuryProperties struct {
	Date     string  `xml:"NEW_DATE"`
	Y1Month  float64 `xml:"BC_1MONTH"`
	Y1_5Month float64 `xml:"BC_1_5MONTH"`
	Y2Month  float64 `xml:"BC_2MONTH"`
	Y3Month  float64 `xml:"BC_3MONTH"`
	Y4Month  float64 `xml:"BC_4MONTH"`
	Y6Month  float64 `xml:"BC_6MONTH"`
	Y1Year   float64 `xml:"BC_1YEAR"`
	Y2Year   float64 `xml:"BC_2YEAR"`
	Y3Year   float64 `xml:"BC_3YEAR"`
	Y5Year   float64 `xml:"BC_5YEAR"`
	Y7Year   float64 `xml:"BC_7YEAR"`
	Y10Year  float64 `xml:"BC_10YEAR"`
	Y20Year  float64 `xml:"BC_20YEAR"`
	Y30Year  float64 `xml:"BC_30YEAR"`
}

// maturityField pairs a maturity (in years) with the value extracted from a parsed treasuryProperties record.
type maturityField struct {
	years float64
	value float64
}

func (p treasuryProperties) maturityFields() []maturityField {
	return []maturityField{
		{years: 1.0 / 12.0, value: p.Y1Month},
		{years: 1.5 / 12.0, value: p.Y1_5Month},
		{years: 2.0 / 12.0, value: p.Y2Month},
		{years: 3.0 / 12.0, value: p.Y3Month},
		{years: 4.0 / 12.0, value: p.Y4Month},
		{years: 6.0 / 12.0, value: p.Y6Month},
		{years: 1.0, value: p.Y1Year},
		{years: 2.0, value: p.Y2Year},
		{years: 3.0, value: p.Y3Year},
		{years: 5.0, value: p.Y5Year},
		{years: 7.0, value: p.Y7Year},
		{years: 10.0, value: p.Y10Year},
		{years: 20.0, value: p.Y20Year},
		{years: 30.0, value: p.Y30Year},
	}
}

// FetchLatestParBonds fetches the latest Treasury par yield curve data
func FetchLatestParBonds(yearMonth int) ([]curve.BootstrapInstrument, error) {
	url := fmt.Sprintf(
		"https://home.treasury.gov/resource-center/data-chart-center/interest-rates/pages/xml?data=daily_treasury_yield_curve&field_tdr_date_value_month=%d",
		yearMonth,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("marketdata: failed to fetch Treasury feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketdata: Treasury feed returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("marketdata: failed to read Treasury feed body: %w", err)
	}

	feed, err := parseTreasuryXML(body)
	if err != nil {
		return nil, err
	}

	latest, err := latestEntry(feed)
	if err != nil {
		return nil, fmt.Errorf("marketdata: %w (month %d)", err, yearMonth)
	}

	return buildParBonds(latest), nil
}

// parseTreasuryXML unmarshals raw Treasury XML bytes into a treasuryFeed.
func parseTreasuryXML(body []byte) (treasuryFeed, error) {
	var feed treasuryFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return treasuryFeed{}, fmt.Errorf("marketdata: failed to parse Treasury XML: %w", err)
	}
	return feed, nil
}

// latestEntry returns the most recent valid trading-day record from the parsed feed.
func latestEntry(feed treasuryFeed) (treasuryProperties, error) {
	var valid []treasuryProperties
	for _, e := range feed.Entries {
		if e.Content.Properties.Date == "" {
			continue
		}
		valid = append(valid, e.Content.Properties)
	}

	if len(valid) == 0 {
		return treasuryProperties{}, fmt.Errorf("no valid (non-empty) entries found in feed")
	}

	sort.Slice(valid, func(i, j int) bool {
		return valid[i].Date < valid[j].Date
	})

	return valid[len(valid)-1], nil
}

// builds ParBond instruments, skipping any maturity with no quote.
func buildParBonds(p treasuryProperties) []curve.BootstrapInstrument {
	var bonds []curve.BootstrapInstrument
	for _, f := range p.maturityFields() {
		if f.value <= 0 {
			continue
		}
		yieldDecimal := f.value / 100.0 // Treasury quotes in percent, e.g. 4.37 -> 0.0437
		bonds = append(bonds, curve.NewParBond(f.years, yieldDecimal))
	}
	return bonds
}