package calendar_repository_google

func (gcs *GoogleCalendarStorage) CreateEvent(ctx context.Context, event models.EventRequest) (*calendar.Event, error) {
	if event.StartTime == nil {
		log.Println("start time is required")
	}

	startTime := *event.StartTime
	var endTime time.Time

	if event.DurationHours != nil {
		endTime = startTime.Add(time.Duration(*event.DurationHours * float64(time.Hour)))
	} else {
		endTime = startTime.Add(time.Hour)
	}

	googleEvent := &calendar.Event{
		Summary: event.Title,
		Start: &calendar.EventDateTime{
			DateTime: startTime.Format(time.RFC3339),
			TimeZone: "Europe/Moscow",
		},
		End: &calendar.EventDateTime{
			DateTime: endTime.Format(time.RFC3339),
			TimeZone: "Europe/Moscow",
		},
	}

	if event.Recurrence != nil && *event.Recurrence != "" {
		recurrenceRule := formatRecurrenceRule(*event.Recurrence)
		if recurrenceRule != "" {
			googleEvent.Recurrence = []string{recurrenceRule}
			log.Printf("📅 Устанавливаем рекуррентность: %s", recurrenceRule)
		}
	}

	if event.Description != nil {
		googleEvent.Description = *event.Description
	}

	return gcs.service.Events.Insert("primary", googleEvent).Context(ctx).Do()
}


func (gcs *GoogleCalendarStorage) ListEvents(ctx context.Context, timeMin, timeMax time.Time, calendarIDs ...string) ([]*calendar.Event, error) {
	if gcs.service == nil {
		return nil, fmt.Errorf("Календарь не подключен. Перейдите по /auth/google для авторизации.")
	}
	timeMinStr := timeMin.Format(time.RFC3339)
	timeMaxStr := timeMax.Format(time.RFC3339)

	if len(calendarIDs) == 0 {
		calendarIDs = []string{"primary"}
	}

	var allEvents []*calendar.Event

	for _, cid := range calendarIDs {
		events, err := gcs.service.Events.List(cid).
			TimeMin(timeMinStr).
			TimeMax(timeMaxStr).
			SingleEvents(true).
			OrderBy("startTime").
			Context(ctx).
			Do()

		if err != nil {
			log.Printf("failed to list events for %s: %v", cid, err)
			continue
		}
		allEvents = append(allEvents, events.Items...)
	}

	return allEvents, nil
}


// GetUserCalendars returns user's calendar list
func (gcs *GoogleCalendarStorage) GetUserCalendars(ctx context.Context) ([]*calendar.CalendarListEntry, error) {
	if gcs.service == nil {
		return nil, fmt.Errorf("Календарь не авторизован")
	}
	list, err := gcs.service.CalendarList.List().Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}


func (gcs *GoogleCalendarStorage) GetCalendarPreview(ctx context.Context, days int) string {
	timeMin := time.Now().UTC()
	timeMax := time.Now().AddDate(0, 0, days).UTC()
	events, err := gcs.ListEvents(ctx, timeMin, timeMax)
	if err != nil {
		return fmt.Sprintf("Error to load calendar: %v", err)
	}

	if len(events) == 0 {
		return "No events"
	}

	var preview strings.Builder
	preview.WriteString("Closest events:\n")

	for _, event := range events {
		start := event.Start.DateTime
		if start == "" {
			start = event.Start.Date
		}
		preview.WriteString(fmt.Sprintf("- %s: %s\n", start, event.Summary))
	}

	return preview.String()
}