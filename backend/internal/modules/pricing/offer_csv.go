package pricing

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var csvPricePattern = regexp.MustCompile(`^(?:[0-9]+|[0-9]{1,3}(?:,[0-9]{3})+|[0-9]{1,2}(?:,[0-9]{2})+,[0-9]{3})(?:\.[0-9]{1,2})?$`)

func parseOfferCSV(raw string) ([]ObservedOfferPrice, error) {
	if len(raw) == 0 || len(raw) > 200000 {
		return nil, errInvalidObservation
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(raw, "\ufeff")))
	header, err := reader.Read()
	if err != nil {
		return nil, errInvalidObservation
	}
	date, price, source := -1, -1, -1
	for i, value := range header {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "date", "observed_on", "observed on":
			if date >= 0 {
				return nil, errInvalidObservation
			}
			date = i
		case "price", "item_price", "item price":
			if price >= 0 {
				return nil, errInvalidObservation
			}
			price = i
		case "source":
			if source >= 0 {
				return nil, errInvalidObservation
			}
			source = i
		}
	}
	if date < 0 || price < 0 {
		return nil, errInvalidObservation
	}
	result := []ObservedOfferPrice{}
	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil || len(result) == 1000 {
			return nil, errInvalidObservation
		}
		rowDate := strings.TrimSpace(row[date])
		parsedDate, dateErr := time.Parse(time.DateOnly, rowDate)
		if dateErr != nil {
			parsedDate, dateErr = time.Parse("2/1/2006", rowDate)
		}
		paise, amountErr := parseCSVPrice(row[price])
		if dateErr != nil || amountErr != nil {
			line, _ := reader.FieldPos(0)
			return nil, fmt.Errorf("%w: CSV line %d", errInvalidObservation, line)
		}
		item := ObservedOfferPrice{
			ObservedOn: parsedDate.Format(time.DateOnly), ItemPaise: paise, Availability: "unknown",
		}
		if source >= 0 {
			item.SourceLabel = row[source]
		}
		result = append(result, item)
	}
	if len(result) == 0 {
		return nil, errInvalidObservation
	}
	return result, nil
}

func parseCSVPrice(raw string) (int64, error) {
	value := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "₹"))
	if !csvPricePattern.MatchString(value) {
		return 0, errInvalidObservation
	}
	rupees, decimal, _ := strings.Cut(strings.ReplaceAll(value, ",", ""), ".")
	units, err := strconv.ParseInt(rupees, 10, 64)
	if err != nil || units > maxOfferPaise/100 {
		return 0, errInvalidObservation
	}
	cents, err := strconv.ParseInt((decimal + "00")[:2], 10, 64)
	if err != nil || units*100+cents <= 0 || units*100+cents > maxOfferPaise {
		return 0, errInvalidObservation
	}
	return units*100 + cents, nil
}
