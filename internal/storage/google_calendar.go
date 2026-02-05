package storage

import (
    "context"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "google.golang.org/api/calendar/v3"
    "golang.org/x/oauth2/google"
    "golang.org/x/oauth2"
)

type GoogleCalendarStorage struct {
	service *calendar.Service
}

func NewGoogleCalendarStorage() *GoogleCalendarStorage {
	

}