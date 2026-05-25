package ics

import (
	"bufio"
	"bytes"
	"errors"
	"strconv"
	"strings"
	"time"
)

type Event struct {
	UID         string
	Title       string
	Description string
	Location    string
	StartsAt    time.Time
	EndsAt      time.Time
	Timezone    string
	AllDay      bool
	RRule       string
	ReminderMin []int
}

type ParseResult struct {
	Events   []Event
	Warnings []string
}

func ParseCalendar(data []byte, defaultTimezone string) (ParseResult, error) {
	if !bytes.Contains(data, []byte("BEGIN:VCALENDAR")) {
		return ParseResult{}, errors.New("not an ics calendar")
	}
	location := time.UTC
	if defaultTimezone != "" {
		if loaded, err := time.LoadLocation(defaultTimezone); err == nil {
			location = loaded
		}
	}

	lines := unfoldLines(data)
	result := ParseResult{}
	var stack []string
	var current *Event
	inAlarm := false

	for _, raw := range lines {
		name, params, value := parseLine(raw)
		switch name {
		case "BEGIN":
			component := strings.ToUpper(value)
			stack = append(stack, component)
			if component == "VEVENT" {
				current = &Event{Timezone: location.String()}
			}
			if component == "VALARM" && current != nil {
				inAlarm = true
			}
			continue
		case "END":
			component := strings.ToUpper(value)
			if component == "VALARM" {
				inAlarm = false
			}
			if component == "VEVENT" && current != nil {
				normalizeEvent(current)
				if current.Title == "" {
					current.Title = "Ohne Titel"
				}
				if !current.StartsAt.IsZero() && current.EndsAt.After(current.StartsAt) {
					result.Events = append(result.Events, *current)
				} else {
					result.Warnings = append(result.Warnings, "VEVENT ohne gültige Start-/Endzeit übersprungen")
				}
				current = nil
			}
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			continue
		}

		if current == nil {
			continue
		}
		if inAlarm {
			if name == "TRIGGER" {
				if minutes, ok := parseTriggerMinutes(value); ok {
					current.ReminderMin = append(current.ReminderMin, minutes)
				}
			}
			continue
		}

		switch name {
		case "UID":
			current.UID = strings.TrimSpace(value)
		case "SUMMARY":
			current.Title = unescapeText(value)
		case "DESCRIPTION":
			current.Description = unescapeText(value)
		case "LOCATION":
			current.Location = unescapeText(value)
		case "DTSTART":
			start, allDay, timezone, err := parseDateTime(value, params, location)
			if err == nil {
				current.StartsAt = start
				current.AllDay = allDay
				current.Timezone = timezone
			}
		case "DTEND":
			end, _, timezone, err := parseDateTime(value, params, location)
			if err == nil {
				current.EndsAt = end
				if timezone != "" {
					current.Timezone = timezone
				}
			}
		case "RRULE":
			current.RRule = strings.TrimSpace(value)
		}
	}

	return result, nil
}

func unfoldLines(data []byte) []string {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	var lines []string
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && len(lines) > 0 {
			lines[len(lines)-1] += strings.TrimLeft(line, " \t")
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func parseLine(line string) (string, map[string]string, string) {
	left, value, _ := strings.Cut(line, ":")
	parts := strings.Split(left, ";")
	name := strings.ToUpper(parts[0])
	params := map[string]string{}
	for _, part := range parts[1:] {
		key, val, ok := strings.Cut(part, "=")
		if ok {
			params[strings.ToUpper(key)] = strings.Trim(val, `"`)
		}
	}
	return name, params, value
}

func parseDateTime(value string, params map[string]string, fallback *time.Location) (time.Time, bool, string, error) {
	if params["VALUE"] == "DATE" || (!strings.Contains(value, "T") && len(value) == 8) {
		parsed, err := time.ParseInLocation("20060102", value, fallback)
		return parsed, true, fallback.String(), err
	}
	if strings.HasSuffix(value, "Z") {
		parsed, err := time.Parse("20060102T150405Z", value)
		return parsed, false, "UTC", err
	}
	location := fallback
	timezone := fallback.String()
	if tzid := params["TZID"]; tzid != "" {
		if loaded, err := time.LoadLocation(tzid); err == nil {
			location = loaded
			timezone = loaded.String()
		} else {
			timezone = tzid
		}
	}
	parsed, err := time.ParseInLocation("20060102T150405", value, location)
	return parsed, false, timezone, err
}

func normalizeEvent(event *Event) {
	if event.EndsAt.IsZero() {
		if event.AllDay {
			event.EndsAt = event.StartsAt.AddDate(0, 0, 1)
		} else {
			event.EndsAt = event.StartsAt.Add(time.Hour)
		}
	}
	if !event.EndsAt.After(event.StartsAt) {
		if event.AllDay {
			event.EndsAt = event.StartsAt.AddDate(0, 0, 1)
		} else {
			event.EndsAt = event.StartsAt.Add(time.Hour)
		}
	}
}

func parseTriggerMinutes(value string) (int, bool) {
	negative := strings.HasPrefix(value, "-")
	value = strings.TrimPrefix(strings.TrimPrefix(value, "-"), "+")
	if !strings.HasPrefix(value, "PT") {
		return 0, false
	}
	value = strings.TrimPrefix(value, "PT")
	multiplier := 1
	if strings.HasSuffix(value, "H") {
		multiplier = 60
		value = strings.TrimSuffix(value, "H")
	} else if strings.HasSuffix(value, "M") {
		value = strings.TrimSuffix(value, "M")
	} else {
		return 0, false
	}
	amount, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	if negative {
		return amount * multiplier, true
	}
	return -amount * multiplier, true
}

func unescapeText(value string) string {
	value = strings.ReplaceAll(value, `\n`, "\n")
	value = strings.ReplaceAll(value, `\N`, "\n")
	value = strings.ReplaceAll(value, `\,`, ",")
	value = strings.ReplaceAll(value, `\;`, ";")
	value = strings.ReplaceAll(value, `\\`, `\`)
	return value
}
