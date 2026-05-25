package caldav

import (
	"encoding/xml"
	"fmt"
	"strings"

	"calendaradvanced/internal/domain"
)

func PrincipalResponse(baseURL, email string) string {
	base := strings.TrimRight(baseURL, "/")
	return xmlHeader(fmt.Sprintf(`<d:multistatus xmlns:d="DAV:" xmlns:cs="http://calendarserver.org/ns/" xmlns:cal="urn:ietf:params:xml:ns:caldav"><d:response><d:href>/dav/principals/%s/</d:href><d:propstat><d:prop><d:current-user-principal><d:href>/dav/principals/%s/</d:href></d:current-user-principal><cal:calendar-home-set><d:href>/dav/calendars/%s/</d:href></cal:calendar-home-set><d:displayname>%s</d:displayname><cs:getctag>%s</cs:getctag></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response></d:multistatus>`, esc(email), esc(email), esc(email), esc(email), esc(base)))
}

func CalendarHomeResponse(email string, calendars []domain.Calendar) string {
	var b strings.Builder
	b.WriteString(`<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav" xmlns:cs="http://calendarserver.org/ns/">`)
	b.WriteString(fmt.Sprintf(`<d:response><d:href>/dav/calendars/%s/</d:href><d:propstat><d:prop><d:resourcetype><d:collection/></d:resourcetype><d:displayname>%s</d:displayname></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`, esc(email), esc(email)))
	for _, calendar := range calendars {
		b.WriteString(fmt.Sprintf(`<d:response><d:href>/dav/calendars/%s/%d/</d:href><d:propstat><d:prop><d:resourcetype><d:collection/><cal:calendar/></d:resourcetype><d:displayname>%s</d:displayname><cal:supported-calendar-component-set><cal:comp name="VEVENT"/></cal:supported-calendar-component-set><cs:getctag>%d</cs:getctag></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`, esc(email), calendar.ID, esc(calendar.Name), calendar.UpdatedAt.Unix()))
	}
	b.WriteString(`</d:multistatus>`)
	return xmlHeader(b.String())
}

func CalendarQueryResponse(email string, calendarID int64, events []domain.Event) string {
	var b strings.Builder
	b.WriteString(`<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">`)
	for _, event := range events {
		b.WriteString(fmt.Sprintf(`<d:response><d:href>/dav/calendars/%s/%d/%s.ics</d:href><d:propstat><d:prop><d:getetag>%s</d:getetag><cal:calendar-data>%s</cal:calendar-data></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`, esc(email), calendarID, esc(event.UID), esc(event.ETag), escText(EventData(event))))
	}
	b.WriteString(`</d:multistatus>`)
	return xmlHeader(b.String())
}

func xmlHeader(body string) string {
	return `<?xml version="1.0" encoding="utf-8"?>` + body
}

func esc(v string) string     { return xmlEscape(v) }
func escText(v string) string { return xmlEscape(v) }

func xmlEscape(v string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(v))
	return b.String()
}
