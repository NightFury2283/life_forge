package auth_repository_google

import (
	"encoding/json"
	"fmt"
	"os"
	"golang.org/x/oauth2"
)

func saveToken(path string, token *oauth2.Token) error {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("save toke: %w", err)
	}
	defer file.Close()

	json.NewEncoder(file).Encode(token)

	return nil
}
